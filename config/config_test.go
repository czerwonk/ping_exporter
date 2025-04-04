// SPDX-License-Identifier: MIT

package config

import (
	"bytes"
	"os"
	"reflect"
	"testing"
	"time"
)

func TestParseConfig(t *testing.T) {
	t.Parallel()

	f, err := os.Open("testdata/config_test.yml")
	if err != nil {
		t.Error("failed to open file", err)
		t.FailNow()
	}

	c, err := FromYAML(f)
	f.Close()
	if err != nil {
		t.Error("failed to parse", err)
		t.FailNow()
	}

	targets := []TargetConfig{
		{Addr: "8.8.8.8"},
		{Addr: "8.8.4.4"},
		{Addr: "2001:4860:4860::8888"},
		{
			Addr: "2001:4860:4860::8844",
			Labels: map[string]string{
				"foo": "bar",
			},
		},
	}

	if !reflect.DeepEqual(targets, c.Targets) {
		t.Errorf("expected 4 targets (%v) but got %d (%v)", targets, len(c.Targets), c.Targets)
		t.FailNow()
	}

	if expected := 2*time.Minute + 15*time.Second; time.Duration(c.DNS.Refresh) != expected {
		t.Errorf("expected dns.refresh to be %v, got %v", expected, c.DNS.Refresh)
	}
	if expected := 5 * time.Second; time.Duration(c.DNS.Timeout) != expected {
		t.Errorf("expected dns.timeout to be %v, got %v", expected, c.DNS.Timeout)
	}
	if expected := "1.1.1.1"; c.DNS.Nameserver != expected {
		t.Errorf("expected dns.nameserver to be %q, got %q", expected, c.DNS.Nameserver)
	}

	if expected := 2 * time.Second; time.Duration(c.Ping.Interval) != expected {
		t.Errorf("expected ping.interval to be %v, got %v", expected, c.Ping.Interval)
	}
	if expected := 3 * time.Second; time.Duration(c.Ping.Timeout) != expected {
		t.Errorf("expected ping.timeout to be %v, got %v", expected, c.Ping.Timeout)
	}
	if expected := 42; c.Ping.History != expected {
		t.Errorf("expected ping.history-size to be %d, got %d", expected, c.Ping.History)
	}
	if expected := 120; c.Ping.Size != uint16(expected) {
		t.Errorf("expected ping.payload-size to be %d, got %d", expected, c.Ping.Size)
	}
	if expected := true; c.Options.DisableIPv6 != expected {
		t.Errorf("expected options.disable-ipv6 to be %v, got %v", expected, c.Options.DisableIPv6)
	}
}

func TestRoundtrip(t *testing.T) {
	f, err := os.Open("testdata/config_test.yml")
	if err != nil {
		t.Error("failed to open file", err)
		t.FailNow()
	}

	c, err := FromYAML(f)
	if err != nil {
		t.Error("failed to read file", err)
		t.FailNow()
	}

	buf := bytes.NewBuffer(nil)
	err = ToYAML(buf, c)
	if err != nil {
		t.Error("failed to encode config", err)
		t.FailNow()
	}

	after, err := FromYAML(buf)
	if err != nil {
		t.Error("failed to read config again", err)
		t.FailNow()
	}

	if !reflect.DeepEqual(c, after) {
		t.Error("config after Decode(Encode(cfg)) didn't match")
		t.FailNow()
	}
}
