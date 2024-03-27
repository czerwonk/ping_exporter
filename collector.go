// SPDX-License-Identifier: MIT

package main

import (
	"strings"
	"sync"

	mon "github.com/digineo/go-ping/monitor"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/czerwonk/ping_exporter/config"
)

type pingCollector struct {
	monitor                 *mon.Monitor
	enableDeprecatedMetrics bool
	rttUnit                 rttUnit

	cfg *config.Config

	mutex sync.RWMutex

	customLabels *customLabelSet
	metrics      map[string]*mon.Metrics

	rttDesc    scaledMetrics
	bestDesc   scaledMetrics
	worstDesc  scaledMetrics
	meanDesc   scaledMetrics
	stddevDesc scaledMetrics
	lossDesc   *prometheus.Desc
	progDesc   *prometheus.Desc
}

func NewPingCollector(enableDeprecatedMetrics bool, unit rttUnit, monitor *mon.Monitor, cfg *config.Config) *pingCollector {
	ret := &pingCollector{
		monitor:                 monitor,
		enableDeprecatedMetrics: enableDeprecatedMetrics,
		rttUnit:                 unit,
		cfg:                     cfg,
	}
	ret.customLabels = newCustomLabelSet(cfg.Targets)
	ret.createDesc()
	return ret
}

func (p *pingCollector) UpdateConfig(cfg *config.Config) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	p.cfg.Targets = cfg.Targets
}

func (p *pingCollector) Describe(ch chan<- *prometheus.Desc) {
	if p.enableDeprecatedMetrics {
		p.rttDesc.Describe(ch)
	}
	p.bestDesc.Describe(ch)
	p.worstDesc.Describe(ch)
	p.meanDesc.Describe(ch)
	p.stddevDesc.Describe(ch)
	ch <- p.lossDesc
	ch <- p.progDesc
}

func (p *pingCollector) Collect(ch chan<- prometheus.Metric) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if m := p.monitor.Export(); len(m) > 0 {
		p.metrics = m
	}

	ch <- prometheus.MustNewConstMetric(p.progDesc, prometheus.GaugeValue, 1)

	for target, metrics := range p.metrics {
		l := strings.SplitN(target, " ", 3)

		targetConfig := p.cfg.TargetConfigByAddr(l[0])
		l = append(l, p.customLabels.labelValues(targetConfig)...)

		if metrics.PacketsSent > metrics.PacketsLost {
			if enableDeprecatedMetrics {
				p.rttDesc.Collect(ch, metrics.Best, append(l, "best")...)
				p.rttDesc.Collect(ch, metrics.Worst, append(l, "worst")...)
				p.rttDesc.Collect(ch, metrics.Mean, append(l, "mean")...)
				p.rttDesc.Collect(ch, metrics.StdDev, append(l, "std_dev")...)
			}

			p.bestDesc.Collect(ch, metrics.Best, l...)
			p.worstDesc.Collect(ch, metrics.Worst, l...)
			p.meanDesc.Collect(ch, metrics.Mean, l...)
			p.stddevDesc.Collect(ch, metrics.StdDev, l...)
		}

		loss := float64(metrics.PacketsLost) / float64(metrics.PacketsSent)
		ch <- prometheus.MustNewConstMetric(p.lossDesc, prometheus.GaugeValue, loss, l...)
	}
}

func (p *pingCollector) createDesc() {
	labelNames := []string{"target", "ip", "ip_version"}
	labelNames = append(labelNames, p.customLabels.labelNames()...)

	p.rttDesc = newScaledDesc("rtt", "Round trip time", p.rttUnit, append(labelNames, "type"))
	p.bestDesc = newScaledDesc("rtt_best", "Best round trip time", p.rttUnit, labelNames)
	p.worstDesc = newScaledDesc("rtt_worst", "Worst round trip time", p.rttUnit, labelNames)
	p.meanDesc = newScaledDesc("rtt_mean", "Mean round trip time", p.rttUnit, labelNames)
	p.stddevDesc = newScaledDesc("rtt_std_deviation", "Standard deviation", p.rttUnit, labelNames)
	p.lossDesc = newDesc("loss_ratio", "Packet loss from 0.0 to 1.0", labelNames, nil)
	p.progDesc = newDesc("up", "ping_exporter version", nil, prometheus.Labels{"version": version})
}

func newDesc(name, help string, variableLabels []string, constLabels prometheus.Labels) *prometheus.Desc {
	return prometheus.NewDesc("ping_"+name, help, variableLabels, constLabels)
}
