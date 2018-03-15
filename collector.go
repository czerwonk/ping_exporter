package main

import (
	"strings"
	"sync"

	mon "github.com/digineo/go-ping/monitor"
	"github.com/prometheus/client_golang/prometheus"
)

const prefix = "ping_"

var (
	bestDesc   = prometheus.NewDesc(prefix+"best_ms", "Best round trip time in millis", labelNames, nil)
	worstDesc  = prometheus.NewDesc(prefix+"worst_ms", "Worst round trip time in millis", labelNames, nil)
	meanDesc   = prometheus.NewDesc(prefix+"mean_ms", "Mean round trip time in millis", labelNames, nil)
	stddevDesc = prometheus.NewDesc(prefix+"std_deviation_ms", "Standard deviation in millis", labelNames, nil)
	lossDesc   = prometheus.NewDesc(prefix+"loss_percent", "Packet loss in percent", labelNames, nil)
	labelNames = []string{"target", "ip", "ip_version"}
	mutex      = &sync.Mutex{}
)

type pingCollector struct {
	monitor *mon.Monitor
	metrics map[string]*mon.Metrics
}

func (p *pingCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- bestDesc
	ch <- worstDesc
	ch <- meanDesc
	ch <- stddevDesc
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

		ch <- prometheus.MustNewConstMetric(bestDesc, prometheus.GaugeValue, float64(metrics.Best), l...)
		ch <- prometheus.MustNewConstMetric(worstDesc, prometheus.GaugeValue, float64(metrics.Worst), l...)
		ch <- prometheus.MustNewConstMetric(meanDesc, prometheus.GaugeValue, float64(metrics.Mean), l...)
		ch <- prometheus.MustNewConstMetric(stddevDesc, prometheus.GaugeValue, float64(metrics.StdDev), l...)
		ch <- prometheus.MustNewConstMetric(lossDesc, prometheus.GaugeValue, float64(metrics.PacketsLost)/float64(metrics.PacketsSent), l...)
	}
}
