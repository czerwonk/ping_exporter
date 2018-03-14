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

const version string = "0.0.1"

var (
	showVersion     = flag.Bool("version", false, "Print version information.")
	listenAddress   = flag.String("web.listen-address", ":8080", "Address on which to expose metrics and web interface.")
	metricsPath     = flag.String("web.telemetry-path", "/metrics", "Path under which to expose metrics.")

	pingInterval = flag.Duration("pingInterval", time.Duration(5)*time.Second, "interval for ICMP echo requests")
	pingTimeout = flag.Duration("pingTimeout", time.Duration(4)*time.Second, "timeout for ICMP echo request")

	monitor *mon.Monitor
)

func init() {
	flag.Usage = func() {
		fmt.Println("Usage: ping-exporter [ ... ]\n\nParameters:")
		fmt.Println()
		flag.PrintDefaults()
	}
}

func main() {
	flag.Parse()

	//todo: remove global variable
	var targets = flag.Args()

	// Targets empty?
	if len(targets) == 0 {
		fmt.Println("Usage:", os.Args[0], "[options] target1 target2 ...")
		flag.PrintDefaults()
		os.Exit(1)
	}

	// Too many targets?
	if len(targets) > int(^byte(0)) {
		fmt.Println("Too many targets")
		os.Exit(1)
	}

	if *showVersion {
		printVersion()
		os.Exit(0)
	}

	startMonitor(targets)
	startServer()
}

func printVersion() {
	fmt.Println("ping-exporter")
	fmt.Printf("Version: %s\n", version)
	fmt.Println("Author(s): Philip Berndroth, Daniel Czerwonk")
	fmt.Println("Metric exporter for go-icmp")
}

func startMonitor(targets []string){

	pinger, err := ping.New("0.0.0.0", "::")
	if err != nil {
		panic(err)
	}

	monitor = mon.New(pinger, *pingInterval, *pingTimeout)
	defer monitor.Stop()

	// Add targets
	for i, target := range targets {
		ipAddr, err := net.ResolveIPAddr("", target)
		if err != nil {
			fmt.Printf("invalid target '%s': %s", target, err)
			continue
		}
		fmt.Printf("adding target '%s'\n", target)
		monitor.AddTargetDelayed(string([]byte{byte(i)}), *ipAddr, 10*time.Millisecond*time.Duration(i))
	}
}

func startServer() {
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

	http.HandleFunc(*metricsPath, handleMetricsRequest)

	log.Infof("Listening for %s on %s", *metricsPath, *listenAddress)
	log.Fatal(http.ListenAndServe(*listenAddress, nil))
}

func handleMetricsRequest(w http.ResponseWriter, r *http.Request) {

	reg := prometheus.NewRegistry()
	reg.MustRegister(&PingCollector{})

	promhttp.HandlerFor(reg, promhttp.HandlerOpts{
		ErrorLog:      log.NewErrorLogger(),
		ErrorHandling: promhttp.ContinueOnError}).ServeHTTP(w, r)
}
