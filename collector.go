package main

import (
	"strings"
	"sync"

	mon "github.com/digineo/go-ping/monitor"
	"github.com/prometheus/client_golang/prometheus"
)

var labelConfigs = make(map[string]map[string]string, 0)
var labelExtends = make(map[string][]string, 0)

func newDesc(name, help string, variableLabels []string, constLabels prometheus.Labels) *prometheus.Desc {
	return prometheus.NewDesc("ping_"+name, help, variableLabels, constLabels)
}

var (
	labelNames []string
	rttDesc    scaledMetrics
	bestDesc   scaledMetrics
	worstDesc  scaledMetrics
	meanDesc   scaledMetrics
	stddevDesc scaledMetrics
	lossDesc   *prometheus.Desc
	progDesc   *prometheus.Desc
	mutex      *sync.Mutex
)

func createDesc() {

	labelSet := make(map[string]int)
	for _, labels := range labelConfigs {
		for label := range labels {
			labelSet[label] = 0
		}
	}

	var labelExtend []string
	for label := range labelSet {
		labelExtend = append(labelExtend, label)
	}

	for host, labels := range labelConfigs {
		for _, label := range labelExtend {
			if _, ok := labels[label]; ok {
				labelExtends[host] = append(labelExtends[host], labels[label])
			} else {
				labelExtends[host] = append(labelExtends[host], "")
			}
		}
	}

	labelNames = []string{"target", "ip", "ip_version"}
	labelNames = append(labelNames, labelExtend...)

	rttDesc = newScaledDesc("rtt", "Round trip time", append(labelNames, "type"))
	bestDesc = newScaledDesc("rtt_best", "Best round trip time", labelNames)
	worstDesc = newScaledDesc("rtt_worst", "Worst round trip time", labelNames)
	meanDesc = newScaledDesc("rtt_mean", "Mean round trip time", labelNames)
	stddevDesc = newScaledDesc("rtt_std_deviation", "Standard deviation", labelNames)
	lossDesc = newDesc("loss_ratio", "Packet loss from 0.0 to 1.0", labelNames, nil)
	progDesc = newDesc("up", "ping_exporter version", nil, prometheus.Labels{"version": version})
	mutex = &sync.Mutex{}
}

type pingCollector struct {
	monitor *mon.Monitor
	metrics map[string]*mon.Metrics
}

func (p *pingCollector) Describe(ch chan<- *prometheus.Desc) {
	createDesc()

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

	if m := p.monitor.Export(); len(m) > 0 {
		p.metrics = m
	}

	ch <- prometheus.MustNewConstMetric(progDesc, prometheus.GaugeValue, 1)

	for target, metrics := range p.metrics {
		l := strings.SplitN(target, " ", 3)
		l = append(l, labelExtends[l[0]]...)

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
