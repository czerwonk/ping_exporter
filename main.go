package main

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/czerwonk/ping_exporter/config"
	"github.com/digineo/go-ping"
	mon "github.com/digineo/go-ping/monitor"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/log"
)

const version string = "0.4.3"

var (
	showVersion   = flag.Bool("version", false, "Print version information")
	listenAddress = flag.String("web.listen-address", ":9427", "Address on which to expose metrics and web interface")
	metricsPath   = flag.String("web.telemetry-path", "/metrics", "Path under which to expose metrics")
	configFile    = flag.String("config.path", "", "Path to config file")
	pingInterval  = flag.Duration("ping.interval", time.Duration(5)*time.Second, "Interval for ICMP echo requests")
	pingTimeout   = flag.Duration("ping.timeout", time.Duration(4)*time.Second, "Timeout for ICMP echo request")
	dnsRefresh    = flag.Duration("dns.refresh", time.Duration(1)*time.Minute, "Interval for refreshing DNS records and updating targets accordingly (0 if disabled)")
	dnsNameServer = flag.String("dns.nameserver", "", "DNS server used to resolve hostname of targets")
	logLevel      = flag.String("log.level", "info", "Only log messages with the given severity or above. Valid levels: [debug, info, warn, error, fatal]")
)

func init() {
	flag.Usage = func() {
		fmt.Println("Usage:", os.Args[0], "-config.path=$my-config-file [options]")
		fmt.Println()
		flag.PrintDefaults()
	}
}

func main() {
	flag.Parse()

	if *showVersion {
		printVersion()
		os.Exit(0)
	}

	err := log.Logger.SetLevel(log.Base(), *logLevel)
	if err != nil {
		log.Errorln(err)
		os.Exit(1)
	}

	cfg, err := loadConfig()
	if err != nil {
		log.Errorln(err)
		os.Exit(1)
	}

	if len(cfg.Targets) == 0 {
		flag.Usage()
		os.Exit(1)
	}

	m, err := startMonitor(cfg)
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
	fmt.Println("Metric exporter for go-icmp")
}

func startMonitor(cfg *config.Config) (*mon.Monitor, error) {
	pinger, err := ping.New("0.0.0.0", "::")
	if err != nil {
		return nil, err
	}

	monitor := mon.New(pinger, *pingInterval, *pingTimeout)
	targets := make([]*target, len(cfg.Targets))
	for i, host := range cfg.Targets {
		t := &target{
			host:      host,
			addresses: make([]net.IP, 0),
			delay:     time.Duration(10*i) * time.Millisecond,
			dns:       *dnsNameServer,
		}
		targets[i] = t

		err := t.addOrUpdateMonitor(monitor)
		if err != nil {
			log.Errorln(err)
		}
	}

	go startDNSAutoRefresh(targets, monitor)

	return monitor, nil
}

func startDNSAutoRefresh(targets []*target, monitor *mon.Monitor) {
	if *dnsRefresh == 0 {
		return
	}

	for {
		select {
		case <-time.After(*dnsRefresh):
			refreshDNS(targets, monitor)
		}
	}
}

func refreshDNS(targets []*target, monitor *mon.Monitor) {
	for _, t := range targets {
		log.Infoln("refreshing DNS")

		go func(ta *target) {
			err := ta.addOrUpdateMonitor(monitor)
			if err != nil {
				log.Errorf("could refresh dns: %v", err)
			}
		}(t)
	}
}

func startServer(monitor *mon.Monitor) {
	log.Infof("Starting ping exporter (Version: %s)", version)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html>
			<head><title>ping Exporter (Version ` + version + `)</title></head>
			<body>
			<h1>ping Exporter</h1>
			<p><a href="` + *metricsPath + `">Metrics</a></p>
			<h2>More information:</h2>
			<p><a href="https://github.com/czerwonk/ping_exporter">github.com/czerwonk/ping_exporter</a></p>
			</body>
			</html>`))
	})

	reg := prometheus.NewRegistry()
	reg.MustRegister(&pingCollector{monitor: monitor})
	h := promhttp.HandlerFor(reg, promhttp.HandlerOpts{
		ErrorLog:      log.NewErrorLogger(),
		ErrorHandling: promhttp.ContinueOnError})
	http.HandleFunc("/metrics", h.ServeHTTP)

	log.Infof("Listening for %s on %s", *metricsPath, *listenAddress)
	log.Fatal(http.ListenAndServe(*listenAddress, nil))
}

func loadConfig() (*config.Config, error) {
	if *configFile == "" {
		return &config.Config{Targets: flag.Args()}, nil
	}

	b, err := ioutil.ReadFile(*configFile)
	if err != nil {
		return nil, err
	}

	return config.FromYAML(bytes.NewReader(b))
}
