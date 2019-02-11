package config

import (
	"io"
	"time"

	yaml "gopkg.in/yaml.v2"
)

// Config represents configuration for the exporter
type Config struct {
	Targets []string `yaml:"targets"`

	Ping struct {
		Interval duration `yaml:"interval"`
		Timeout  duration `yaml:"timeout"`
		History  int      `yaml:"history-size"`
		Size     uint16   `yaml:"payload-size"`
	} `yaml:"ping"`

	DNS struct {
		Refresh    duration `yaml:"refresh"`
		Nameserver string   `yaml:"nameserver"`
	} `yaml:"dns"`
}

type duration time.Duration

// UnmarshalYAML implements yaml.Unmarshaler interface.
func (d *duration) UnmarshalYAML(unmashal func(interface{}) error) error {
	var s string
	if err := unmashal(&s); err != nil {
		return err
	}
	dur, err := time.ParseDuration(s)
	if err != nil {
		return err
	}
	*d = duration(dur)
	return nil
}

// Duration is a convenience getter.
func (d duration) Duration() time.Duration {
	return time.Duration(d)
}

// Set updates the underlying duration.
func (d *duration) Set(dur time.Duration) {
	*d = duration(dur)
}

// FromYAML reads YAML from reader and unmarshals it to Config
func FromYAML(r io.Reader) (*Config, error) {
	c := &Config{}
	err := yaml.NewDecoder(r).Decode(c)
	if err != nil {
		return nil, err
	}
	return c, nil
}
