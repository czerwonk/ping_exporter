package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"github.com/czerwonk/ping_exporter/config"
	"github.com/digineo/go-ping"
	mon "github.com/digineo/go-ping/monitor"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
	"gopkg.in/alecthomas/kingpin.v2"
)

const version string = "0.4.8"

var (
	showVersion   = kingpin.Flag("version", "Print version information").Default().Bool()
	listenAddress = kingpin.Flag("web.listen-address", "Address on which to expose metrics and web interface").Default(":9427").String()
	metricsPath   = kingpin.Flag("web.telemetry-path", "Path under which to expose metrics").Default("/metrics").String()
	configFile    = kingpin.Flag("config.path", "Path to config file").Default("").String()
	pingInterval  = kingpin.Flag("ping.interval", "Interval for ICMP echo requests").Default("5s").Duration()
	pingTimeout   = kingpin.Flag("ping.timeout", "Timeout for ICMP echo request").Default("4s").Duration()
	pingSize      = kingpin.Flag("ping.size", "Payload size for ICMP echo requests").Default("56").Uint16()
	historySize   = kingpin.Flag("ping.history-size", "Number of results to remember per target").Default("10").Int()
	dnsRefresh    = kingpin.Flag("dns.refresh", "Interval for refreshing DNS records and updating targets accordingly (0 if disabled)").Default("1m").Duration()
	dnsNameServer = kingpin.Flag("dns.nameserver", "DNS server used to resolve hostname of targets").Default("").String()
	logLevel      = kingpin.Flag("log.level", "Only log messages with the given severity or above. Valid levels: [debug, info, warn, error, fatal]").Default("info").String()
	targets       = kingpin.Arg("targets", "A list of targets to ping").Strings()
)

var (
	enableDeprecatedMetrics = true // default may change in future
	deprecatedMetrics       = kingpin.Flag("metrics.deprecated", "Enable or disable deprecated metrics (`ping_rtt_ms{type=best|worst|mean|std_dev}`). Valid choices: [enable, disable]").Default("enable").String()

	rttMetricsScale = rttInMills // might change in future
	rttMode         = kingpin.Flag("metrics.rttunit", "Export ping results as either millis (default), or seconds (best practice), or both (for migrations). Valid choices: [ms, s, both]").Default("ms").String()
)

// The following are used to keep track of the last used ping ID field value,
// and to pick a new one.  Each new ping ID is incremented by PINGID_INCR,
// which is a large relatively-prime value chosen to distribute the ID values
// as evenly as possible over the entire space in a deterministic manner.  This
// is slightly better than just picking random values as it guarantees a
// maximum interval before reuse, while still making accidental conflicts
// unlikely.
const PINGID_INCR = 29479
// (this is set up so that the first value chosen will always end up being the
// PID.  If multiple monitors and id-change-intervals are not being used, this
// is consistent with the old behavior and it is also a good way to make sure
// that if somebody's running multiple copies of this program they are unlikely
// to overlap with each other (even when periodically generating new IDs).)
var lastPingId = uint32(os.Getpid() - PINGID_INCR)

func init() {
	kingpin.Parse()
}

func main() {
	if *showVersion {
		printVersion()
		os.Exit(0)
	}

	setLogLevel(*logLevel)

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

	if len(cfg.Monitors) == 0 {
		if len(cfg.Targets) == 0 {
			kingpin.FatalUsage("either 'monitors' or 'targets' must be specified")
		} else {
			// Legacy single-target-list format.  Create a single
			// monitor entry under cfg.Monitors with the specified
			// targets.
			cfg.Monitors = make([]config.MonitorConfig, 1)
			cfg.Monitors[0].Targets = cfg.Targets
		}
	} else if len(cfg.Targets) != 0 {
		kingpin.FatalUsage("you must specify either 'monitors' or 'targets', not both")
	}

	// Go through all the defined monitors and fill in any undefined values
	// from the global defaults
	for i := range cfg.Monitors {
		setMonitorConfigDefaults(cfg, &cfg.Monitors[i])
	}

	m, err := startMonitors(cfg)
	if err != nil {
		log.Errorln(err)
		os.Exit(2)
	}

	startServer(m)
}

func printVersion() {
	fmt.Println("ping-exporter")
	fmt.Printf("Version: %s\n", version)
	fmt.Println("Author(s): Philip Berndroth, Daniel Czerwonk")
	fmt.Println("Metric exporter for go-ping")
}

// Return a new ID value which won't overlap with any recent previous values.
// There is unfortunately no easy way to make sure something else isn't using
// this value, so we just have to try to spread them around as much as possible
// to try to avoid conflicts with other things.
// Note: In many OSes, the kernel and things like the `ping` command often
// choose id values starting at low numbers and work up, so we avoid using the
// first 1024 values, just to reduce the likelihood of conflicts with people
// running arbitrary `ping` commands on the command line from time to time.
func newPingId() uint16 {
	for {
		if id := uint16(atomic.AddUint32(&lastPingId, PINGID_INCR)); id >= 1024 {
			return id
		}
	}
}

func startMonitors(cfg *config.Config) ([]*mon.Monitor, error) {
	resolver := setupResolver(cfg)
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

	result := make([]*mon.Monitor, len(cfg.Monitors))
	for mon_index, mon_cfg := range cfg.Monitors {
		pinger, err := ping.New(bind4, bind6)
		if err != nil {
			return nil, fmt.Errorf("cannot start monitoring: %w", err)
		}
		// Set a distinct ICMP identifier field for each pinger
		pinger.Id = newPingId()

		if pinger.PayloadSize() != mon_cfg.Ping.Size {
			pinger.SetPayloadSize(mon_cfg.Ping.Size)
		}

		monitor := mon.New(pinger,
			mon_cfg.Ping.Interval.Duration(),
			mon_cfg.Ping.Timeout.Duration())
		monitor.HistorySize = mon_cfg.Ping.History
		log.Infof("Created new monitor (interval=%s, timeout=%s, history=%d)",
			mon_cfg.Ping.Interval.Duration(),
			mon_cfg.Ping.Timeout.Duration(),
			mon_cfg.Ping.History)

		targets := make([]*target, len(mon_cfg.Targets))
		for i, host := range mon_cfg.Targets {
			t := &target{
				host:      host,
				addresses: make([]net.IPAddr, 0),
				delay:     time.Duration(10*i) * time.Millisecond,
				resolver:  resolver,
			}
			targets[i] = t

			err := t.addOrUpdateMonitor(monitor)
			if err != nil {
				log.Errorln(err)
			}
		}

		go startDNSAutoRefresh(mon_cfg.DNS.Refresh.Duration(), targets, monitor)
		go startPingIdAutoUpdate(mon_cfg.Ping.IDChangeInterval.Duration(),
			mon_cfg.Ping.IDChangeThreshold, pinger, monitor)

		result[mon_index] = monitor
	}

	return result, nil
}

func startDNSAutoRefresh(interval time.Duration, targets []*target, monitor *mon.Monitor) {
	if interval <= 0 {
		return
	}

	for range time.NewTicker(interval).C {
		refreshDNS(targets, monitor)
	}
}

func refreshDNS(targets []*target, monitor *mon.Monitor) {
	log.Debugln("Refreshing DNS")
	for _, t := range targets {
		go func(ta *target) {
			err := ta.addOrUpdateMonitor(monitor)
			if err != nil {
				log.Errorf("could not refresh dns: %v", err)
			}
		}(t)
	}
}

func startPingIdAutoUpdate(interval time.Duration, threshold float64, pinger *ping.Pinger, monitor *mon.Monitor) {
	if interval <= 0 {
		return
	}

	for range time.NewTicker(interval).C {
		log.Debugln("Checking for Ping ID update")
		if monitorOverLossThreshold(monitor, threshold) {
			updatePingId(pinger)
		}
	}
}

func monitorOverLossThreshold(monitor *mon.Monitor, threshold float64) bool {
	if threshold <= 0 {
		return true
	}

	sent := 0
	lost := 0

	for _, metrics := range monitor.Export() {
		sent += metrics.PacketsSent
		lost += metrics.PacketsLost
	}
	ratio := float64(lost) / float64(sent)

	log.Debugf("monitor packet loss: %f (threshold=%f)", ratio, threshold)
	return ratio >= threshold
}

func updatePingId(pinger *ping.Pinger) {
	pinger.Id = newPingId()
	log.Debugf("Setting new ping ID of %d", pinger.Id)
}

func startServer(monitors []*mon.Monitor) {
	log.Infof("Starting ping exporter (Version: %s)", version)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, indexHTML, *metricsPath)
	})

	reg := prometheus.NewRegistry()
	reg.MustRegister(&pingCollector{monitors: monitors})

	l := log.New()
	l.Level = log.ErrorLevel

	h := promhttp.HandlerFor(reg, promhttp.HandlerOpts{
		ErrorLog:      l,
		ErrorHandling: promhttp.ContinueOnError,
	})
	http.Handle(*metricsPath, h)

	log.Infof("Listening for %s on %s", *metricsPath, *listenAddress)
	log.Fatal(http.ListenAndServe(*listenAddress, nil))
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
	dialer := func(ctx context.Context, network, address string) (net.Conn, error) {
		d := net.Dialer{}

		return d.DialContext(ctx, "udp", cfg.DNS.Nameserver)
	}

	return &net.Resolver{PreferGo: true, Dial: dialer}
}

// addFlagToConfig updates cfg with command line flag values, unless the
// config has non-zero values.
func addFlagToConfig(cfg *config.Config) {
	if len(cfg.Targets) == 0 {
		cfg.Targets = *targets
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
}

func setMonitorConfigDefaults(cfg *config.Config, mon_cfg *config.MonitorConfig) {
	if mon_cfg.Ping.History == 0 {
		mon_cfg.Ping.History = cfg.Ping.History
	}
	if mon_cfg.Ping.Interval == 0 {
		mon_cfg.Ping.Interval = cfg.Ping.Interval
	}
	if mon_cfg.Ping.Timeout == 0 {
		mon_cfg.Ping.Timeout = cfg.Ping.Timeout
	}
	if mon_cfg.Ping.Size == 0 {
		mon_cfg.Ping.Size = cfg.Ping.Size
	}
	if mon_cfg.Ping.IDChangeInterval == 0 {
		mon_cfg.Ping.IDChangeInterval = cfg.Ping.IDChangeInterval
	}
	if mon_cfg.Ping.IDChangeThreshold == 0 {
		mon_cfg.Ping.IDChangeThreshold = cfg.Ping.IDChangeThreshold
	}
	if mon_cfg.DNS.Refresh == 0 {
		mon_cfg.DNS.Refresh = cfg.DNS.Refresh
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
