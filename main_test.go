package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/czerwonk/ping_exporter/config"
)

func TestLoadConfigDNSRefresh(t *testing.T) {
	oldPath, oldRefresh := *configFile, *dnsRefresh
	t.Cleanup(func() { *configFile, *dnsRefresh = oldPath, oldRefresh })
	*dnsRefresh = time.Minute
	for _, tc := range []struct {
		name, yaml string
		want       time.Duration
	}{
		{"omitted", "targets: [127.0.0.1]\n", time.Minute},
		{"disabled", "targets: [127.0.0.1]\ndns:\n  refresh: 0s\n", 0},
		{"numeric zero", "targets: [127.0.0.1]\ndns:\n  refresh: 0\n", 0},
		{"null", "targets: [127.0.0.1]\ndns:\n  refresh: null\n", time.Minute},
		{"configured", "targets: [127.0.0.1]\ndns:\n  refresh: 2m15s\n", 135 * time.Second},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "config.yaml")
			if err := os.WriteFile(path, []byte(tc.yaml), 0600); err != nil {
				t.Fatal(err)
			}
			*configFile = path
			cfg, err := loadConfig()
			if err != nil {
				t.Fatal(err)
			}
			if got := cfg.DNS.Refresh.Duration(); got != tc.want {
				t.Fatalf("effective DNS refresh = %v, want %v", got, tc.want)
			}
		})
	}

	*configFile = ""
	*dnsRefresh = 0
	cfg, err := loadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if got := cfg.DNS.Refresh.Duration(); got != 0 {
		t.Fatalf("CLI-only DNS refresh = %v, want 0s", got)
	}
}

func TestAddFlagToConfigListenAddressFromYAML(t *testing.T) {
	orig := *listenAddress
	t.Cleanup(func() { *listenAddress = orig })

	*listenAddress = ":9427"
	cfg := &config.Config{}
	cfg.Web.ListenAddress = "127.0.0.1:9999"
	addFlagToConfig(cfg)

	if *listenAddress != "127.0.0.1:9999" {
		t.Fatalf("listen address = %q, want 127.0.0.1:9999", *listenAddress)
	}
}

func TestAddFlagToConfigListenAddressKeepsFlagWhenYAMLEmpty(t *testing.T) {
	orig := *listenAddress
	t.Cleanup(func() { *listenAddress = orig })

	*listenAddress = "0.0.0.0:1234"
	cfg := &config.Config{}
	addFlagToConfig(cfg)

	if *listenAddress != "0.0.0.0:1234" {
		t.Fatalf("listen address = %q, want 0.0.0.0:1234", *listenAddress)
	}
}
