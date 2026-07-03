// SPDX-License-Identifier: MIT

package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/czerwonk/ping_exporter/config"

	"github.com/alecthomas/kingpin/v2"
	log "github.com/sirupsen/logrus"
)

const version string = "1.2.1"
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

var (
	showVersion             = kingpin.Flag("version", "Print version information").Default().Bool()
	listenAddress           = kingpin.Flag("web.listen-address", "Address on which to expose metrics and web interface").Default(":9427").String()
	metricsPath             = kingpin.Flag("web.telemetry-path", "Path under which to expose metrics").Default("/metrics").String()
	metricsToken            = kingpin.Flag("web.token", "Token (in http request headers of queries) which to expose metrics").Default("").String()
	serverUseTLS            = kingpin.Flag("web.tls.enabled", "Enable TLS for web server, default is false").Default().Bool()
	serverTLSCertFile       = kingpin.Flag("web.tls.cert-file", "The certificate file for the web server").Default("").String()
	serverTLSKeyFile        = kingpin.Flag("web.tls.key-file", "The key file for the web server").Default("").String()
	serverMutualAuthEnabled = kingpin.Flag("web.tls.mutual-auth-enabled", "Enable TLS client mutual authentication, default is false").Default().Bool()
	serverTLSCAFile         = kingpin.Flag("web.tls.ca-file", "The certificate authority file for client's certificate verification").Default("").String()
	configFile              = kingpin.Flag("config.path", "Path to config file").Default("").String()
	pingInterval            = kingpin.Flag("ping.interval", "Interval for ICMP echo requests").Default("5s").Duration()
	pingTimeout             = kingpin.Flag("ping.timeout", "Timeout for ICMP echo request").Default("4s").Duration()
	pingSize                = kingpin.Flag("ping.size", "Payload size for ICMP echo requests").Default("56").Uint16()
	firewallMark            = kingpin.Flag("ping.fw-mark", "set socket mark (SO_MARK) to this value").Default("0").Uint()
	historySize             = kingpin.Flag("ping.history-size", "Number of results to remember per target").Default("10").Int()
	dnsRefresh              = kingpin.Flag("dns.refresh", "Interval for refreshing DNS records and updating targets accordingly (0 if disabled)").Default("1m").Duration()
	dnsNameServer           = kingpin.Flag("dns.nameserver", "DNS server used to resolve hostname of targets").Default("").String()
	dnsLookupTimeout        = kingpin.Flag("dns.timeout", "Timeout for DNS resolution").Default("0s").Duration()
	disableIPv6             = kingpin.Flag("options.disable-ipv6", "Disable DNS from resolving IPv6 AAAA records").Default().Bool()
	disableIPv4             = kingpin.Flag("options.disable-ipv4", "Disable DNS from resolving IPv4 A records").Default().Bool()
	logLevel                = kingpin.Flag("log.level", "Only log messages with the given severity or above. Valid levels: [debug, info, warn, error, fatal]").Default("info").String()
	targetFlag              = kingpin.Arg("targets", "A list of targets to ping").Strings()
)

var (
	enableDeprecatedMetrics = true // default may change in future
	deprecatedMetrics       = kingpin.Flag("metrics.deprecated", "Enable or disable deprecated metrics (`ping_rtt_ms{type=best|worst|mean|std_dev}`). Valid choices: [enable, disable]").Default("disable").String()

	rttMetricsScale = rttInMills // might change in future
	rttMode         = kingpin.Flag("metrics.rttunit", "Export ping results as either seconds (default), or milliseconds (deprecated), or both (for migrations). Valid choices: [s, ms, both]").Default("s").String()
)

func main() {
	kingpin.Parse()

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

	runExporter(cfg)
}

func runInteractive(cfg *config.Config) {
	exporter := NewExporter(cfg)

	err := exporter.Start()
	if err != nil {
		log.Fatalf("failed to start exporter: %v", err)
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Info("Shutting down...")
	exporter.Stop()
}

func printVersion() {
	fmt.Println("ping-exporter")
	fmt.Printf("Version: %s\n", version)
	fmt.Println("Author(s): Philip Berndroth, Daniel Czerwonk")
	fmt.Println("Metric exporter for go-icmp")
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
	defer func() {
		if err := f.Close(); err != nil {
			log.Errorf("failed to close config file: %v", err)
		}
	}()

	cfg, err := config.FromYAML(f)
	if err == nil {
		addFlagToConfig(cfg)
	}

	return cfg, err
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
	if cfg.Ping.FirewallMark == 0 {
		cfg.Ping.FirewallMark = *firewallMark
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
	if cfg.DNS.Timeout == 0 {
		cfg.DNS.Timeout.Set(*dnsLookupTimeout)
	}
}
