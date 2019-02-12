package config

import (
	"os"
	"reflect"
	"testing"
	"time"
)

func TestParseConfig(t *testing.T) {
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

	if expected := 2*time.Minute + 15*time.Second; time.Duration(c.DNS.Refresh) != expected {
		t.Errorf("expected dns.refresh to be %v, got %v", expected, c.DNS.Refresh)
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

}
