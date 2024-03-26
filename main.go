// SPDX-License-Identifier: MIT

package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/digineo/go-ping"
	mon "github.com/digineo/go-ping/monitor"

	"github.com/czerwonk/ping_exporter/config"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
	"gopkg.in/alecthomas/kingpin.v2"
	inotify "gopkg.in/fsnotify.v1"
)

const version string = "1.1.1"

var (
	showVersion             = kingpin.Flag("version", "Print version information").Default().Bool()
	listenAddress           = kingpin.Flag("web.listen-address", "Address on which to expose metrics and web interface").Default(":9427").String()
	metricsPath             = kingpin.Flag("web.telemetry-path", "Path under which to expose metrics").Default("/metrics").String()
	serverUseTLS            = kingpin.Flag("web.tls.enabled", "Enable TLS for web server, default is false").Default().Bool()
	serverTlsCertFile       = kingpin.Flag("web.tls.cert-file", "The certificate file for the web server").Default("").String()
	serverTlsKeyFile        = kingpin.Flag("web.tls.key-file", "The key file for the web server").Default("").String()
	serverMutualAuthEnabled = kingpin.Flag("web.tls.mutual-auth-enabled", "Enable TLS client mutual authentication, default is false").Default().Bool()
	serverTlsCAFile         = kingpin.Flag("web.tls.ca-file", "The certificate authority file for client's certificate verification").Default("").String()
	configFile              = kingpin.Flag("config.path", "Path to config file").Default("").String()
	pingInterval            = kingpin.Flag("ping.interval", "Interval for ICMP echo requests").Default("5s").Duration()
	pingTimeout             = kingpin.Flag("ping.timeout", "Timeout for ICMP echo request").Default("4s").Duration()
	pingSize                = kingpin.Flag("ping.size", "Payload size for ICMP echo requests").Default("56").Uint16()
	historySize             = kingpin.Flag("ping.history-size", "Number of results to remember per target").Default("10").Int()
	dnsRefresh              = kingpin.Flag("dns.refresh", "Interval for refreshing DNS records and updating targets accordingly (0 if disabled)").Default("1m").Duration()
	dnsNameServer           = kingpin.Flag("dns.nameserver", "DNS server used to resolve hostname of targets").Default("").String()
	disableIPv6             = kingpin.Flag("options.disable-ipv6", "Disable DNS from resolving IPv6 AAAA records").Default().Bool()
	disableIPv4             = kingpin.Flag("options.disable-ipv4", "Disable DNS from resolving IPv4 A records").Default().Bool()
	logLevel                = kingpin.Flag("log.level", "Only log messages with the given severity or above. Valid levels: [debug, info, warn, error, fatal]").Default("info").String()
	targetFlag              = kingpin.Arg("targets", "A list of targets to ping").Strings()

	tailnet = kingpin.Flag("ts.tailnet", "tailnet name").String()
)

var (
	enableDeprecatedMetrics = true // default may change in future
	deprecatedMetrics       = kingpin.Flag("metrics.deprecated", "Enable or disable deprecated metrics (`ping_rtt_ms{type=best|worst|mean|std_dev}`). Valid choices: [enable, disable]").Default("disable").String()

	rttMetricsScale = rttInMills // might change in future
	rttMode         = kingpin.Flag("metrics.rttunit", "Export ping results as either seconds (default), or milliseconds (deprecated), or both (for migrations). Valid choices: [s, ms, both]").Default("s").String()
	desiredTargets  *targets
)

func main() {
	desiredTargets = &targets{}
	kingpin.Parse()

	if len(*tailnet) > 0 {
		tsDiscover()
	}

	if *showVersion {
		printVersion()
		os.Exit(0)
	}

	setLogLevel(*logLevel)
	log.SetReportCaller(true)

	switch *deprecatedMetrics {
	case "enable":
		enableDeprecatedMetrics = true
	case "disable":
		enableDeprecatedMetrics = false
	default:
		kingpin.FatalUsage("metrics.deprecated must be `enable` or `disable`")
	}

	if rttMetricsScale = rttUnitFromString(*rttMode); rttMetricsScale == rttInvalid {
		kingpin.FatalUsage("metrics.rttunit must be `ms` for millis, or `s` for seconds, or `both`")
	}
	log.Infof("rtt units: %#v", rttMetricsScale)

	if mpath := *metricsPath; mpath == "" {
		log.Warnln("web.telemetry-path is empty, correcting to `/metrics`")
		mpath = "/metrics"
		metricsPath = &mpath
	} else if mpath[0] != '/' {
		mpath = "/" + mpath
		metricsPath = &mpath
	}

	cfg, err := loadConfig()
	if err != nil {
		kingpin.FatalUsage("could not load config.path: %v", err)
	}

	if cfg.Ping.History < 1 {
		kingpin.FatalUsage("ping.history-size must be greater than 0")
	}

	if cfg.Ping.Size > 65500 {
		kingpin.FatalUsage("ping.size must be between 0 and 65500")
	}

	if len(cfg.Targets) == 0 {
		kingpin.FatalUsage("No targets specified")
	}

	resolver := setupResolver(cfg)

	m, err := startMonitor(cfg, resolver)
	if err != nil {
		log.Errorln(err)
		os.Exit(2)
	}

	collector := NewPingCollector(enableDeprecatedMetrics, rttMetricsScale, m, cfg)
	go watchConfig(desiredTargets, resolver, m, collector)

	startServer(cfg, collector)
}

func printVersion() {
	fmt.Println("ping-exporter")
	fmt.Printf("Version: %s\n", version)
	fmt.Println("Author(s): Philip Berndroth, Daniel Czerwonk")
	fmt.Println("Metric exporter for go-icmp")
}

func startMonitor(cfg *config.Config, resolver *net.Resolver) (*mon.Monitor, error) {
	var bind4, bind6 string
	if ln, err := net.Listen("tcp4", "127.0.0.1:0"); err == nil {
		// ipv4 enabled
		ln.Close()
		bind4 = "0.0.0.0"
	}
	if ln, err := net.Listen("tcp6", "[::1]:0"); err == nil {
		// ipv6 enabled
		ln.Close()
		bind6 = "::"
	}
	pinger, err := ping.New(bind4, bind6)
	if err != nil {
		return nil, fmt.Errorf("cannot start monitoring: %w", err)
	}

	if pinger.PayloadSize() != cfg.Ping.Size {
		pinger.SetPayloadSize(cfg.Ping.Size)
	}

	monitor := mon.New(pinger,
		cfg.Ping.Interval.Duration(),
		cfg.Ping.Timeout.Duration())
	monitor.HistorySize = cfg.Ping.History

	err = upsertTargets(desiredTargets, resolver, cfg, monitor)
	if err != nil {
		log.Fatalln(err)
	}

	go startDNSAutoRefresh(cfg.DNS.Refresh.Duration(), desiredTargets, monitor, cfg)
	return monitor, nil
}

func upsertTargets(globalTargets *targets, resolver *net.Resolver, cfg *config.Config, monitor *mon.Monitor) error {
	oldTargets := globalTargets.Targets()
	newTargets := make([]*target, len(cfg.Targets))
	var wg sync.WaitGroup
	for i, t := range cfg.Targets {
		newTarget := globalTargets.Get(t.Addr)
		if newTarget == nil {
			newTarget = &target{
				host:      t.Addr,
				addresses: make([]net.IPAddr, 0),
				delay:     time.Duration(10*i) * time.Millisecond,
				resolver:  resolver,
			}
		}

		newTargets[i] = newTarget

		wg.Add(1)
		go func() {
			err := newTarget.addOrUpdateMonitor(monitor, targetOpts{
				disableIPv4: cfg.Options.DisableIPv4,
				disableIPv6: cfg.Options.DisableIPv6,
			})
			if err != nil {
				log.Errorf("failed to setup target: %v", err)
			}
			wg.Done()
		}()
	}
	wg.Wait()
	globalTargets.SetTargets(newTargets)

	removed := removedTargets(oldTargets, globalTargets)
	for _, removedTarget := range removed {
		log.Infof("remove target: %s", removedTarget.host)
		removedTarget.removeFromMonitor(monitor)
	}
	return nil
}

func watchConfig(globalTargets *targets, resolver *net.Resolver, monitor *mon.Monitor, collector *pingCollector) {
	watcher, err := inotify.NewWatcher()
	if err != nil {
		log.Fatalf("unable to create file watcher: %v", err)
	}

	err = watcher.Add(*configFile)
	if err != nil {
		log.Fatalf("unable to watch file: %v", err)
	}
	for {
		select {
		case event := <-watcher.Events:
			log.Debugf("Got file inotify event: %s", event)
			// If the file is removed, the inotify watcher will lose track of the file. Add it again.
			if event.Op == inotify.Remove {
				if err = watcher.Add(*configFile); err != nil {
					log.Fatalf("failed to renew watch for file: %v", err)
				}
			}
			cfg, err := loadConfig()
			if err != nil {
				log.Errorf("unable to load config: %v", err)
				continue
			}
			// We get zero targets if the file was truncated. This happens if an automation tool rewrites
			// the complete file, instead of alternating only parts of it.
			if len(cfg.Targets) == 0 {
				continue
			}
			log.Infof("reloading config file %s", *configFile)
			if err := upsertTargets(globalTargets, resolver, cfg, monitor); err != nil {
				log.Errorf("failed to reload config: %v", err)
				continue
			}
			collector.UpdateConfig(cfg)
		case err := <-watcher.Errors:
			log.Errorf("watching file failed: %v", err)
		}
	}
}

func removedTargets(old []*target, new *targets) []*target {
	var ret []*target
	for _, oldTarget := range old {
		if !new.Contains(oldTarget) {
			ret = append(ret, oldTarget)
		}
	}
	return ret
}

func startDNSAutoRefresh(interval time.Duration, tar *targets, monitor *mon.Monitor, cfg *config.Config) {
	if interval <= 0 {
		return
	}

	for range time.NewTicker(interval).C {
		refreshDNS(tar, monitor, cfg)
	}
}

func refreshDNS(tar *targets, monitor *mon.Monitor, cfg *config.Config) {
	log.Infoln("refreshing DNS")
	for _, t := range tar.Targets() {
		go func(ta *target) {
			err := ta.addOrUpdateMonitor(monitor, targetOpts{
				disableIPv4: cfg.Options.DisableIPv4,
				disableIPv6: cfg.Options.DisableIPv6,
			})
			if err != nil {
				log.Errorf("could not refresh dns: %v", err)
			}
		}(t)
	}
}

func startServer(cfg *config.Config, collector *pingCollector) {
	var err error
	log.Infof("Starting ping exporter (Version: %s)", version)
	http.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprintf(w, indexHTML, *metricsPath)
	})

	reg := prometheus.NewRegistry()
	reg.MustRegister(collector)

	l := log.New()
	l.Level = log.ErrorLevel

	h := promhttp.HandlerFor(reg, promhttp.HandlerOpts{
		ErrorLog:      l,
		ErrorHandling: promhttp.ContinueOnError,
	})
	http.Handle(*metricsPath, h)

	server := http.Server{
		Addr: *listenAddress,
	}

	if *serverUseTLS {
		confureTLS(&server)
		log.Infof("Listening for %s on %s (HTTPS)", *metricsPath, *listenAddress)
		err = server.ListenAndServeTLS("", "")
	} else {
		log.Infof("Listening for %s on %s (HTTP)", *metricsPath, *listenAddress)
		err = server.ListenAndServe()
	}

	if err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}

func confureTLS(server *http.Server) {
	if *serverTlsCertFile == "" || *serverTlsKeyFile == "" {
		log.Error("'web.tls.cert-file' and 'web.tls.key-file' must be defined")
		return
	}

	server.TLSConfig = &tls.Config{
		MinVersion: tls.VersionTLS12,
	}

	var err error
	server.TLSConfig.Certificates = make([]tls.Certificate, 1)
	server.TLSConfig.Certificates[0], err = tls.LoadX509KeyPair(*serverTlsCertFile, *serverTlsKeyFile)
	if err != nil {
		log.Errorf("Loading certificates error: %v", err)
		return
	}

	if *serverMutualAuthEnabled {
		server.TLSConfig.ClientAuth = tls.RequireAndVerifyClientCert

		if *serverTlsCAFile != "" {
			var ca []byte
			if ca, err = os.ReadFile(*serverTlsCAFile); err != nil {
				log.Errorf("Loading CA error: %v", err)
				return
			} else {
				server.TLSConfig.ClientCAs = x509.NewCertPool()
				server.TLSConfig.ClientCAs.AppendCertsFromPEM(ca)
			}
		}
	} else {
		server.TLSConfig.ClientAuth = tls.NoClientCert
	}
}

func loadConfig() (*config.Config, error) {
	if *configFile == "" {
		cfg := config.Config{}
		addFlagToConfig(&cfg)

		return &cfg, nil
	}

	f, err := os.Open(*configFile)
	if err != nil {
		return nil, fmt.Errorf("cannot load config file: %w", err)
	}
	defer f.Close()

	cfg, err := config.FromYAML(f)
	if err == nil {
		addFlagToConfig(cfg)
	}

	return cfg, err
}

func setupResolver(cfg *config.Config) *net.Resolver {
	if cfg.DNS.Nameserver == "" {
		return net.DefaultResolver
	}

	if !strings.HasSuffix(cfg.DNS.Nameserver, ":53") {
		cfg.DNS.Nameserver += ":53"
	}
	dialer := func(ctx context.Context, _, _ string) (net.Conn, error) {
		d := net.Dialer{}
		return d.DialContext(ctx, "udp", cfg.DNS.Nameserver)
	}

	return &net.Resolver{PreferGo: true, Dial: dialer}
}

// addFlagToConfig updates cfg with command line flag values, unless the
// config has non-zero values.
func addFlagToConfig(cfg *config.Config) {
	if len(cfg.Targets) == 0 {
		cfg.Targets = make([]config.TargetConfig, len(*targetFlag))
		for i, t := range *targetFlag {
			cfg.Targets[i] = config.TargetConfig{
				Addr: t,
			}
		}
	}
	if cfg.Ping.History == 0 {
		cfg.Ping.History = *historySize
	}
	if cfg.Ping.Interval == 0 {
		cfg.Ping.Interval.Set(*pingInterval)
	}
	if cfg.Ping.Timeout == 0 {
		cfg.Ping.Timeout.Set(*pingTimeout)
	}
	if cfg.Ping.Size == 0 {
		cfg.Ping.Size = *pingSize
	}
	if cfg.DNS.Refresh == 0 {
		cfg.DNS.Refresh.Set(*dnsRefresh)
	}
	if cfg.DNS.Nameserver == "" {
		cfg.DNS.Nameserver = *dnsNameServer
	}
	if !cfg.Options.DisableIPv6 {
		cfg.Options.DisableIPv6 = *disableIPv6
	}
	if !cfg.Options.DisableIPv4 {
		cfg.Options.DisableIPv4 = *disableIPv4
	}
}

const indexHTML = `<!doctype html>
<html>
<head>
	<meta charset="UTF-8">
	<title>ping Exporter (Version ` + version + `)</title>
</head>
<body>
	<h1>ping Exporter</h1>
	<p><a href="%s">Metrics</a></p>
	<h2>More information:</h2>
	<p><a href="https://github.com/czerwonk/ping_exporter">github.com/czerwonk/ping_exporter</a></p>
</body>
</html>
`
