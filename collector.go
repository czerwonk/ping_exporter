package main

import (
	"strings"
	"sync"

	mon "github.com/digineo/go-ping/monitor"
	"github.com/prometheus/client_golang/prometheus"
)

const prefix = "ping_"

var (
	labelNames = []string{"target", "ip", "ip_version"}
	rttDesc    = prometheus.NewDesc(prefix+"rtt_ms", "Round trip time in millis", append(labelNames, "type"), nil)
	lossDesc   = prometheus.NewDesc(prefix+"loss_percent", "Packet loss in percent", labelNames, nil)
	mutex      = &sync.Mutex{}
)

type pingCollector struct {
	monitor *mon.Monitor
	metrics map[string]*mon.Metrics
}

func (p *pingCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- rttDesc
	ch <- lossDesc
}

func (p *pingCollector) Collect(ch chan<- prometheus.Metric) {
	mutex.Lock()
	defer mutex.Unlock()

	metrics := p.monitor.ExportAndClear()

	if len(metrics) > 0 {
		p.metrics = metrics
	}

	if p.metrics == nil || len(p.metrics) == 0 {
		return
	}

	for target, metrics := range p.metrics {
		t := strings.Split(target, " ")
		l := []string{t[0], t[1], t[2]}

		ch <- prometheus.MustNewConstMetric(rttDesc, prometheus.GaugeValue, float64(metrics.Best), append(l, "best")...)
		ch <- prometheus.MustNewConstMetric(rttDesc, prometheus.GaugeValue, float64(metrics.Worst), append(l, "worst")...)
		ch <- prometheus.MustNewConstMetric(rttDesc, prometheus.GaugeValue, float64(metrics.Mean), append(l, "mean")...)
		ch <- prometheus.MustNewConstMetric(rttDesc, prometheus.GaugeValue, float64(metrics.StdDev), append(l, "std_dev")...)

		loss := float64(metrics.PacketsLost) / float64(metrics.PacketsSent)
		ch <- prometheus.MustNewConstMetric(lossDesc, prometheus.GaugeValue, loss, l...)
	}
}
