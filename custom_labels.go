package main

import "github.com/czerwonk/ping_exporter/config"

type customLabelSet struct {
	names   []string
	nameMap map[string]interface{}
}

func newCustomLabelSet(targets []config.TargetConfig) *customLabelSet {
	cl := &customLabelSet{
		nameMap: make(map[string]interface{}),
		names:   make([]string, 0),
	}

	for _, t := range targets {
		cl.addLabelsForTarget(&t)
	}

	return cl
}

func (cl *customLabelSet) addLabelsForTarget(t *config.TargetConfig) {
	if t.Labels == nil {
		return
	}

	for name := range t.Labels {
		cl.addLabel(name)
	}
}

func (cl *customLabelSet) addLabel(name string) {
	_, exists := cl.nameMap[name]
	if exists {
		return
	}

	cl.names = append(cl.names, name)
	cl.nameMap[name] = nil
}

func (cl *customLabelSet) labelNames() []string {
	return cl.names
}

func (cl *customLabelSet) labelValues(t config.TargetConfig) []string {
	values := make([]string, len(cl.names))
	if t.Labels == nil {
		return values
	}

	for i, name := range cl.names {
		if value, isSet := t.Labels[name]; isSet {
			values[i] = value
		}
	}

	return values
}
