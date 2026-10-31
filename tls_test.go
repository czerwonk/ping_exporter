// SPDX-License-Identifier: MIT

package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func writeTestCA(t *testing.T, path string) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	tmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "test-ca"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageCertSign,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestInitMutualAuth(t *testing.T) {
	dir := t.TempDir()
	validCA := filepath.Join(dir, "ca.pem")
	writeTestCA(t, validCA)
	notPEM := filepath.Join(dir, "not-pem.pem")
	if err := os.WriteFile(notPEM, []byte("this is not a certificate\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	empty := filepath.Join(dir, "empty.pem")
	if err := os.WriteFile(empty, nil, 0o600); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name    string
		caFile  string
		wantErr bool
	}{
		{name: "valid CA", caFile: validCA},
		{name: "no CA file", caFile: ""},
		{name: "not PEM", caFile: notPEM, wantErr: true},
		{name: "empty file", caFile: empty, wantErr: true},
		{name: "missing file", caFile: filepath.Join(dir, "missing.pem"), wantErr: true},
	}

	orig := *serverTLSCAFile
	t.Cleanup(func() { *serverTLSCAFile = orig })

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			*serverTLSCAFile = tt.caFile
			server := &http.Server{TLSConfig: &tls.Config{}}
			err := initMutualAuth(server)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected an error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if server.TLSConfig.ClientAuth != tls.RequireAndVerifyClientCert {
				t.Errorf("ClientAuth = %v, want RequireAndVerifyClientCert", server.TLSConfig.ClientAuth)
			}
			if tt.caFile != "" && server.TLSConfig.ClientCAs == nil {
				t.Error("ClientCAs not set")
			}
		})
	}
}
