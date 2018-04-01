package config

import (
	"bytes"
	"io/ioutil"
	"reflect"
	"testing"
)

func TestParseConfig(t *testing.T) {
	b, err := ioutil.ReadFile("tests/config_test.yml")
	if err != nil {
		t.Error("failed to read file", err)
		t.FailNow()
	}

	c, err := FromYAML(bytes.NewReader(b))
	if err != nil {
		t.Error("failed to parse", err)
		t.FailNow()
	}

	targets := []string{
		"8.8.8.8",
		"8.8.4.4",
		"2001:4860:4860::8888",
		"2001:4860:4860::8844",
	}

	if !reflect.DeepEqual(targets, c.Targets) {
		t.Errorf("expected 4 targets (%v) but got %d (%v)", targets, len(c.Targets), c.Targets)
		t.FailNow()
	}
}
