package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
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
