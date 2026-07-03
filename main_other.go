//go:build !windows

package main

import "github.com/czerwonk/ping_exporter/config"

func runExporter(cfg *config.Config) {
	runInteractive(cfg)
}
