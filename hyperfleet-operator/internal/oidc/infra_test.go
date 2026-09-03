/*
Copyright 2026.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package oidc

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"testing"
)

func rsaKeyPEM(t *testing.T, bits int) []byte {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, bits)
	if err != nil {
		t.Fatalf("generate RSA key: %v", err)
	}
	return pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	})
}

func TestValidateRSAPrivateKey(t *testing.T) {
	t.Run("accepts a key at the minimum size", func(t *testing.T) {
		if err := ValidateRSAPrivateKey(rsaKeyPEM(t, minKeyBits)); err != nil {
			t.Errorf("expected no error, got %v", err)
		}
	})

	t.Run("rejects a key below the minimum size", func(t *testing.T) {
		if err := ValidateRSAPrivateKey(rsaKeyPEM(t, 1024)); err == nil {
			t.Error("expected an error for a key below the minimum size, got nil")
		}
	})
}

func TestResolveIssuerHost(t *testing.T) {
	// IP literals resolve without a real DNS lookup, so these cases don't
	// require network access.
	tests := []struct {
		name      string
		issuerURL string
		wantErr   bool
		wantAddr  string
	}{
		{"rejects non-https scheme", "http://93.184.216.34", true, ""},
		{"rejects loopback", "https://127.0.0.1", true, ""},
		{"rejects IPv6 loopback", "https://[::1]", true, ""},
		{"rejects link-local (cloud metadata)", "https://169.254.169.254", true, ""},
		{"rejects private range", "https://10.0.0.1", true, ""},
		{"rejects unspecified", "https://0.0.0.0", true, ""},
		{"accepts a public address", "https://93.184.216.34", false, "93.184.216.34:443"},
		{"accepts a public address with explicit port", "https://93.184.216.34:8443", false, "93.184.216.34:8443"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, dialAddr, err := resolveIssuerHost(context.Background(), tt.issuerURL)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected an error for %q, got nil", tt.issuerURL)
				}
				return
			}
			if err != nil {
				t.Fatalf("expected no error for %q, got %v", tt.issuerURL, err)
			}
			if dialAddr != tt.wantAddr {
				t.Errorf("expected dial address %q, got %q", tt.wantAddr, dialAddr)
			}
		})
	}
}

// discoveryHandler returns an http.HandlerFunc that serves body only at the discovery path, and 404s everything else,
// mirroring a real OIDC issuer
func discoveryHandler(status int, body string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/"+discoveryPath {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}
}

func TestVerifyIssuerDocument(t *testing.T) {
	t.Run("succeeds for a valid discovery document with a matching issuer", func(t *testing.T) {
		var srv *httptest.Server
		srv = httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/"+discoveryPath {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]string{"issuer": srv.URL})
		}))
		defer srv.Close()

		thumbprint, err := verifyIssuerDocument(context.Background(), srv.Client(), srv.URL)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if thumbprint == "" {
			t.Error("expected a non-empty thumbprint")
		}
	})

	t.Run("succeeds when issuerURL has a trailing slash but the document reports it without one", func(t *testing.T) {
		var srv *httptest.Server
		srv = httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/"+discoveryPath {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]string{"issuer": srv.URL})
		}))
		defer srv.Close()

		if _, err := verifyIssuerDocument(context.Background(), srv.Client(), srv.URL+"/"); err != nil {
			t.Fatalf("expected no error for a trailing-slash issuerURL, got %v", err)
		}
	})

	t.Run("fails on a non-200 response", func(t *testing.T) {
		srv := httptest.NewTLSServer(discoveryHandler(http.StatusInternalServerError, `{"issuer":"ignored"}`))
		defer srv.Close()

		if _, err := verifyIssuerDocument(context.Background(), srv.Client(), srv.URL); err == nil {
			t.Error("expected an error for a non-200 response, got nil")
		}
	})

	t.Run("fails on an invalid JSON body", func(t *testing.T) {
		srv := httptest.NewTLSServer(discoveryHandler(http.StatusOK, `not json`))
		defer srv.Close()

		if _, err := verifyIssuerDocument(context.Background(), srv.Client(), srv.URL); err == nil {
			t.Error("expected an error for an invalid JSON body, got nil")
		}
	})

	t.Run("fails when the issuer field is missing", func(t *testing.T) {
		srv := httptest.NewTLSServer(discoveryHandler(http.StatusOK, `{}`))
		defer srv.Close()

		if _, err := verifyIssuerDocument(context.Background(), srv.Client(), srv.URL); err == nil {
			t.Error("expected an error for a missing issuer field, got nil")
		}
	})

	t.Run("fails when the issuer field does not match", func(t *testing.T) {
		srv := httptest.NewTLSServer(discoveryHandler(http.StatusOK, `{"issuer":"https://wrong.example.com"}`))
		defer srv.Close()

		if _, err := verifyIssuerDocument(context.Background(), srv.Client(), srv.URL); err == nil {
			t.Error("expected an error for a mismatched issuer field, got nil")
		}
	})

	t.Run("fails when the host is unreachable", func(t *testing.T) {
		srv := httptest.NewTLSServer(discoveryHandler(http.StatusOK, `{"issuer":"ignored"}`))
		issuerURL := srv.URL
		client := srv.Client()
		srv.Close()

		if _, err := verifyIssuerDocument(context.Background(), client, issuerURL); err == nil {
			t.Error("expected an error for an unreachable host, got nil")
		}
	})
}
