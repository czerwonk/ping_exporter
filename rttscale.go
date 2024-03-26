// SPDX-License-Identifier: MIT

package main

import "github.com/prometheus/client_golang/prometheus"

type rttUnit int

const (
	rttInvalid rttUnit = iota
	rttInMills
	rttInSeconds
	rttBoth
)

func rttUnitFromString(s string) rttUnit {
	switch s {
	case "s":
		return rttInSeconds
	case "ms":
		return rttInMills
	case "both":
		return rttBoth
	default:
		return rttInvalid
	}
}

type scaledMetrics struct {
	Millis  *prometheus.Desc
	Seconds *prometheus.Desc
	scale   rttUnit
}

func (s *scaledMetrics) Describe(ch chan<- *prometheus.Desc) {
	if s.scale == rttInMills || s.scale == rttBoth {
		ch <- s.Millis
	}
	if s.scale == rttInSeconds || s.scale == rttBoth {
		ch <- s.Seconds
	}
}

func (s *scaledMetrics) Collect(ch chan<- prometheus.Metric, value float32, labelValues ...string) {
	if s.scale == rttInMills || s.scale == rttBoth {
		ch <- prometheus.MustNewConstMetric(s.Millis, prometheus.GaugeValue, float64(value), labelValues...)
	}
	if s.scale == rttInSeconds || s.scale == rttBoth {
		ch <- prometheus.MustNewConstMetric(s.Seconds, prometheus.GaugeValue, float64(value)/1000, labelValues...)
	}
}

func newScaledDesc(name, help string, scale rttUnit, variableLabels []string) scaledMetrics {
	return scaledMetrics{
		scale:   scale,
		Millis:  newDesc(name+"_ms", help+" in millis (deprecated)", variableLabels, nil),
		Seconds: newDesc(name+"_seconds", help+" in seconds", variableLabels, nil),
	}
}
