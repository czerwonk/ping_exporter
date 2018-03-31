package main

import (
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/digineo/go-ping"
	mon "github.com/digineo/go-ping/monitor"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/log"
)

const version string = "0.3.1"

var (
	showVersion   = flag.Bool("version", false, "Print version information.")
	listenAddress = flag.String("web.listen-address", ":9427", "Address on which to expose metrics and web interface.")
	metricsPath   = flag.String("web.telemetry-path", "/metrics", "Path under which to expose metrics.")

	pingInterval = flag.Duration("ping.interval", time.Duration(5)*time.Second, "Interval for ICMP echo requests")
	pingTimeout  = flag.Duration("ping.timeout", time.Duration(4)*time.Second, "Timeout for ICMP echo request")
)

func init() {
	flag.Usage = func() {
		fmt.Println("Usage:", os.Args[0], "[options] target1 target2 ...")
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

	var targets = flag.Args()

	if len(targets) == 0 {
		flag.Usage()
		os.Exit(1)
	}

	m, err := startMonitor(targets)
	if err != nil {
		log.Errorln(err)
		os.Exit(2)
	}

	startServer(m, targets)
}

func printVersion() {
	fmt.Println("ping-exporter")
	fmt.Printf("Version: %s\n", version)
	fmt.Println("Author(s): Philip Berndroth, Daniel Czerwonk")
	fmt.Println("Metric exporter for go-icmp")
}

func startMonitor(targets []string) (*mon.Monitor, error) {
	pinger, err := ping.New("0.0.0.0", "::")
	if err != nil {
		return nil, err
	}

	monitor := mon.New(pinger, *pingInterval, *pingTimeout)

	for i, target := range targets {
		err := addTarget(target, i, monitor)
		if err != nil {
			log.Errorln(err)
		}
	}

	return monitor, nil
}

func addTarget(target string, pos int, monitor *mon.Monitor) error {
	addrs, err := net.LookupIP(target)
	if err != nil {
		return err
	}

	for _, addr := range addrs {
		t := fmt.Sprintf("%s %s ", target, addr)

		if addr.To4() == nil {
			t += "6"
		} else {
			t += "4"
		}

		log.Infoln("adding target", target)
		monitor.AddTargetDelayed(t, net.IPAddr{IP: addr, Zone: ""}, 10*time.Millisecond*time.Duration(pos))
	}

	return nil
}

func startServer(monitor *mon.Monitor, targets []string) {
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
