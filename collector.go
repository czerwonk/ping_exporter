package main

import (
	"strings"
	"sync"

	mon "github.com/digineo/go-ping/monitor"
	"github.com/prometheus/client_golang/prometheus"
)

func newDesc(name, help string, variableLabels []string, constLabels prometheus.Labels) *prometheus.Desc {
	return prometheus.NewDesc("ping_"+name, help, variableLabels, constLabels)
}

var (
	labelNames = []string{"target", "ip", "ip_version"}
	rttDesc    = newScaledDesc("rtt", "Round trip time", append(labelNames, "type"))
	bestDesc   = newScaledDesc("rtt_best", "Best round trip time", labelNames)
	worstDesc  = newScaledDesc("rtt_worst", "Worst round trip time", labelNames)
	meanDesc   = newScaledDesc("rtt_mean", "Mean round trip time", labelNames)
	stddevDesc = newScaledDesc("rtt_std_deviation", "Standard deviation", labelNames)
	lossDesc   = newDesc("loss_percent", "Packet loss in percent", labelNames, nil)
	progDesc   = newDesc("up", "ping_exporter version", nil, prometheus.Labels{"version": version})
	mutex      = &sync.Mutex{}
)

type pingCollector struct {
	monitors []*mon.Monitor
	metrics  map[string]*mon.Metrics
}

func (p *pingCollector) Describe(ch chan<- *prometheus.Desc) {
	if enableDeprecatedMetrics {
		rttDesc.Describe(ch)
	}
	bestDesc.Describe(ch)
	worstDesc.Describe(ch)
	meanDesc.Describe(ch)
	stddevDesc.Describe(ch)
	ch <- lossDesc
	ch <- progDesc
}

func (p *pingCollector) Collect(ch chan<- prometheus.Metric) {
	mutex.Lock()
	defer mutex.Unlock()

	if p.metrics == nil {
		p.metrics = make(map[string]*mon.Metrics)
	}
	for _, mon := range p.monitors {
		for target, metrics := range mon.Export() {
			p.metrics[target] = metrics
		}
	}

	ch <- prometheus.MustNewConstMetric(progDesc, prometheus.GaugeValue, 1)

	for target, metrics := range p.metrics {
		l := strings.SplitN(target, " ", 3)

		if metrics.PacketsSent > metrics.PacketsLost {
			if enableDeprecatedMetrics {
				rttDesc.Collect(ch, metrics.Best, append(l, "best")...)
				rttDesc.Collect(ch, metrics.Worst, append(l, "worst")...)
				rttDesc.Collect(ch, metrics.Mean, append(l, "mean")...)
				rttDesc.Collect(ch, metrics.StdDev, append(l, "std_dev")...)
			}

			bestDesc.Collect(ch, metrics.Best, l...)
			worstDesc.Collect(ch, metrics.Worst, l...)
			meanDesc.Collect(ch, metrics.Mean, l...)
			stddevDesc.Collect(ch, metrics.StdDev, l...)
		}

		loss := float64(metrics.PacketsLost) / float64(metrics.PacketsSent)
		ch <- prometheus.MustNewConstMetric(lossDesc, prometheus.GaugeValue, loss, l...)
	}
}
