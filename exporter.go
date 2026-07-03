// SPDX-License-Identifier: MIT

package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/czerwonk/ping_exporter/config"
	"github.com/digineo/go-ping"
	mon "github.com/digineo/go-ping/monitor"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
	inotify "gopkg.in/fsnotify.v1"
)

type Exporter struct {
	cfg            *config.Config
	monitor        *mon.Monitor
	collector      *pingCollector
	server         *http.Server
	desiredTargets *targets
	resolver       Resolver
	ctx            context.Context
	cancel         context.CancelFunc
	wg             sync.WaitGroup
	listener       net.Listener
}

func NewExporter(cfg *config.Config) *Exporter {
	return &Exporter{
		cfg:            cfg,
		desiredTargets: &targets{},
	}
}

func (e *Exporter) Start() error {
	e.ctx, e.cancel = context.WithCancel(context.Background())

	e.resolver = setupGlobalResolver(e.cfg)

	var err error
	e.monitor, err = e.startMonitor()
	if err != nil {
		return err
	}

	err = e.upsertTargets(e.cfg)
	if err != nil {
		return err
	}

	e.collector = NewPingCollector(enableDeprecatedMetrics, rttMetricsScale, e.monitor, e.cfg)

	e.wg.Go(func() {
		e.watchConfig(e.ctx)
	})

	if e.cfg.DNS.Refresh.Duration() > 0 {
		e.wg.Go(func() {
			e.startDNSAutoRefresh(e.ctx)
		})
	}

	err = e.startServer()
	if err != nil {
		e.Stop()
		return err
	}

	return nil
}

func (e *Exporter) Stop() {
	if e.cancel != nil {
		e.cancel()
	}

	if e.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := e.server.Shutdown(ctx); err != nil {
			log.Errorf("Error shutting down HTTP server: %v", err)
		}
	}

	if e.monitor != nil {
		e.monitor.Stop()
	}

	e.wg.Wait()
	log.Info("Exporter stopped")
}

func (e *Exporter) startMonitor() (*mon.Monitor, error) {
	var bind4, bind6 string
	if ln, err := net.Listen("tcp4", "127.0.0.1:0"); err == nil {
		if err := ln.Close(); err != nil {
			return nil, fmt.Errorf("failed to close tcp4 listener: %w", err)
		}
		bind4 = "0.0.0.0"
	}
	if ln, err := net.Listen("tcp6", "[::1]:0"); err == nil {
		if err := ln.Close(); err != nil {
			return nil, fmt.Errorf("failed to close tcp6 listener: %w", err)
		}
		bind6 = "::"
	}
	pinger, err := ping.New(bind4, bind6)
	if err != nil {
		return nil, fmt.Errorf("cannot start monitoring: %w", err)
	}

	if pinger.PayloadSize() != e.cfg.Ping.Size {
		pinger.SetPayloadSize(e.cfg.Ping.Size)
	}

	if e.cfg.Ping.FirewallMark > 0 {
		err := pinger.SetMark(e.cfg.Ping.FirewallMark)
		if err != nil {
			return nil, fmt.Errorf("failed to set fwmark: %w", err)
		}
	}

	monitor := mon.New(pinger,
		e.cfg.Ping.Interval.Duration(),
		e.cfg.Ping.Timeout.Duration())
	monitor.HistorySize = e.cfg.Ping.History

	return monitor, nil
}

func (e *Exporter) upsertTargets(cfg *config.Config) error {
	oldTargets := e.desiredTargets.Targets()
	newTargets := make([]*target, len(cfg.Targets))
	var wg sync.WaitGroup
	var err error
	for i, t := range cfg.Targets {
		newTarget := e.desiredTargets.Get(t.Addr)
		if newTarget == nil {
			resolver := e.resolver
			if r, ok := t.Labels["resolver"]; ok && r == "k8s" {
				resolver, err = NewK8sResolver()
				if err != nil {
					return fmt.Errorf("failed to create k8s resolver: %w", err)
				}
			}
			newTarget = &target{
				host:      t.Addr,
				addresses: make([]net.IPAddr, 0),
				delay:     time.Duration(10*i) * time.Millisecond,
				resolver:  resolver,
			}
		}

		newTargets[i] = newTarget

		wg.Go(func() {
			err := newTarget.addOrUpdateMonitor(e.monitor, targetOpts{
				disableIPv4: cfg.Options.DisableIPv4,
				disableIPv6: cfg.Options.DisableIPv6,
			}, cfg)
			if err != nil {
				log.Errorf("failed to setup target: %v", err)
			}
		})
	}
	wg.Wait()
	e.desiredTargets.SetTargets(newTargets)

	removed := removedTargets(oldTargets, e.desiredTargets)
	for _, removedTarget := range removed {
		log.Infof("remove target: %s", removedTarget.host)
		removedTarget.removeFromMonitor(e.monitor)
	}
	return nil
}

func (e *Exporter) watchConfig(ctx context.Context) {
	watcher, err := inotify.NewWatcher()
	if err != nil {
		log.Fatalf("unable to create file watcher: %v", err)
	}
	defer watcher.Close()

	err = watcher.Add(*configFile)
	if err != nil {
		log.Fatalf("unable to watch file: %v", err)
	}
	for {
		select {
		case <-ctx.Done():
			log.Info("stopping config watcher")
			return
		case event := <-watcher.Events:
			log.Debugf("Got file inotify event: %s", event)
			if event.Op == inotify.Remove {
				if err = watcher.Add(*configFile); err != nil {
					log.Fatalf("failed to renew watch for file: %v", err)
				}
			}
			cfg, err := loadConfig()
			if err != nil {
				log.Errorf("unable to load config: %v", err)
				continue
			}
			if len(cfg.Targets) == 0 {
				continue
			}
			log.Infof("reloading config file %s", *configFile)
			if err := e.upsertTargets(cfg); err != nil {
				log.Errorf("failed to reload config: %v", err)
				continue
			}
			e.collector.UpdateConfig(cfg)
		case err := <-watcher.Errors:
			log.Errorf("watching file failed: %v", err)
		}
	}
}

func (e *Exporter) startDNSAutoRefresh(ctx context.Context) {
	interval := e.cfg.DNS.Refresh.Duration()
	if interval <= 0 {
		return
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Info("stopping DNS auto refresh")
			return
		case <-ticker.C:
			e.refreshDNS()
		}
	}
}

func (e *Exporter) refreshDNS() {
	log.Infoln("refreshing DNS")
	for _, t := range e.desiredTargets.Targets() {
		go func(ta *target) {
			err := ta.addOrUpdateMonitor(e.monitor, targetOpts{
				disableIPv4: e.cfg.Options.DisableIPv4,
				disableIPv6: e.cfg.Options.DisableIPv6,
			}, e.cfg)
			if err != nil {
				log.Errorf("could not refresh dns: %v", err)
			}
		}(t)
	}
}

func (e *Exporter) startServer() error {
	log.Infof("Starting ping exporter (Version: %s)", version)

	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if !hasValidToken(r, w) {
			return
		}

		if _, err := fmt.Fprintf(w, indexHTML, *metricsPath); err != nil {
			log.Errorf("failed to write response: %v", err)
		}
	})

	reg := prometheus.NewRegistry()
	reg.MustRegister(e.collector)

	l := log.New()
	l.Level = log.ErrorLevel

	h := promhttp.HandlerFor(reg, promhttp.HandlerOpts{
		ErrorLog:      l,
		ErrorHandling: promhttp.ContinueOnError,
	})
	mux.HandleFunc(*metricsPath, func(w http.ResponseWriter, r *http.Request) {
		if !hasValidToken(r, w) {
			return
		}

		h.ServeHTTP(w, r)
	})

	e.server = &http.Server{
		Addr:    *listenAddress,
		Handler: mux,
	}

	ln, err := net.Listen("tcp", e.server.Addr)
	if err != nil {
		return err
	}
	e.listener = ln

	go func() {
		var err error
		if *serverUseTLS {
			err = configureTLS(e.server)
			if err != nil {
				log.Fatalf("could not configure TLS: %v", err)
			}
			log.Infof("Listening for %s on %s (HTTPS)", *metricsPath, *listenAddress)
			err = e.server.ServeTLS(e.listener, "", "")
		} else {
			log.Infof("Listening for %s on %s (HTTP)", *metricsPath, *listenAddress)
			err = e.server.Serve(e.listener)
		}

		if err != nil && err != http.ErrServerClosed {
			log.Errorf("Server error: %v", err)
		}
	}()

	return nil
}

func removedTargets(old []*target, new *targets) []*target {
	var ret []*target
	for _, oldTarget := range old {
		if !new.Contains(oldTarget) {
			ret = append(ret, oldTarget)
		}
	}
	return ret
}

func setupGlobalResolver(cfg *config.Config) Resolver {
	if cfg.DNS.Nameserver == "" {
		resolver := net.DefaultResolver
		return resolver
	}

	if _, _, err := net.SplitHostPort(cfg.DNS.Nameserver); err != nil {
		// No port present — add default DNS port, handling IPv6 correctly
		cfg.DNS.Nameserver = net.JoinHostPort(cfg.DNS.Nameserver, "53")
	}
	dialer := func(ctx context.Context, _, _ string) (net.Conn, error) {
		d := net.Dialer{}
		return d.DialContext(ctx, "udp", cfg.DNS.Nameserver)
	}

	return &net.Resolver{PreferGo: true, Dial: dialer}
}

func configureTLS(server *http.Server) error {
	if *serverTLSCertFile == "" || *serverTLSKeyFile == "" {
		return fmt.Errorf("'web.tls.cert-file' and 'web.tls.key-file' must be defined")
	}

	server.TLSConfig = &tls.Config{
		MinVersion: tls.VersionTLS12,
	}

	var err error
	server.TLSConfig.Certificates = make([]tls.Certificate, 1)
	server.TLSConfig.Certificates[0], err = tls.LoadX509KeyPair(*serverTLSCertFile, *serverTLSKeyFile)
	if err != nil {
		return fmt.Errorf("loading certificates error: %v", err)
	}

	if *serverMutualAuthEnabled {
		err = initMutualAuth(server)
		if err != nil {
			return fmt.Errorf("could not initialize mutual auth: %w", err)
		}
	} else {
		server.TLSConfig.ClientAuth = tls.NoClientCert
	}

	return nil
}

func initMutualAuth(server *http.Server) error {
	server.TLSConfig.ClientAuth = tls.RequireAndVerifyClientCert

	if *serverTLSCAFile != "" {
		var err error
		var ca []byte
		if ca, err = os.ReadFile(*serverTLSCAFile); err != nil {
			return fmt.Errorf("loading CA error: %v", err)
		} else {
			server.TLSConfig.ClientCAs = x509.NewCertPool()
			server.TLSConfig.ClientCAs.AppendCertsFromPEM(ca)
		}
	}

	return nil
}

func hasValidToken(r *http.Request, w http.ResponseWriter) bool {
	if *metricsToken == "" {
		return true
	}

	token := r.URL.Query().Get("token")
	if token == "" {
		token = r.Header.Get("token")
	}

	if token != *metricsToken {
		w.WriteHeader(http.StatusForbidden)
		if _, err := fmt.Fprint(w, "wrong token"); err != nil {
			log.Errorf("failed to write response: %v", err)
		}
		return false
	}

	return true
}

