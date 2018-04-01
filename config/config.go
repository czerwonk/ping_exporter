package config

import (
	"io"
	"io/ioutil"

	yaml "gopkg.in/yaml.v2"
)

// Config represents configuration for the exporter
type Config struct {
	Targets []string `yaml:"targets"`
}

// FromYAML reads YAML from reader and unmarshals it to Config
func FromYAML(r io.Reader) (*Config, error) {
	b, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}

	c := &Config{}
	err = yaml.Unmarshal(b, c)
	if err != nil {
		return nil, err
	}

	return c, nil
}
