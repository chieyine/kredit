package web

import (
	"io"
	"log/slog"
	"net/http/httptest"
	"testing"

	"kredit/internal/config"
)

// clientIP is the key for the per-replica rate limiter, the cross-replica
// budget on sign-in routes, and the account-recovery throttle. A caller that can
// choose its own key has no limit at all, so the header is believed only when it
// is proved or when the peer is explicitly trusted.
func TestClientIPIgnoresUnsignedHeaderOnceSigningIsConfigured(t *testing.T) {
	t.Setenv("APP_ENV", "development")
	cfg, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	cfg.FrontendProxySigningKey = "a-frontend-proxy-signing-key-of-sufficient-length"
	cfg.TrustedProxies = nil
	server := NewServer(cfg, slog.New(slog.NewTextHandler(io.Discard, nil)))

	request := httptest.NewRequest("POST", "/api/v1/auth/otp/challenges", nil)
	request.RemoteAddr = "10.1.2.3:44444"
	request.Header.Set("X-Real-IP", "203.0.113.7")

	if got := server.clientIP(request); got != "10.1.2.3" {
		t.Fatalf("an unsigned forwarded address must not become the rate-limit key, got %q", got)
	}
}

func TestClientIPHonoursHeaderOnlyFromATrustedPeer(t *testing.T) {
	t.Setenv("APP_ENV", "development")
	cfg, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	cfg.FrontendProxySigningKey = "a-frontend-proxy-signing-key-of-sufficient-length"
	cfg.TrustedProxies = []string{"10.9.0.0/16"}
	server := NewServer(cfg, slog.New(slog.NewTextHandler(io.Discard, nil)))

	trusted := httptest.NewRequest("POST", "/api/v1/auth/otp/challenges", nil)
	trusted.RemoteAddr = "10.9.4.5:44444"
	trusted.Header.Set("X-Real-IP", "203.0.113.7")
	if got := server.clientIP(trusted); got != "203.0.113.7" {
		t.Fatalf("a declared ingress must be able to forward the client address, got %q", got)
	}

	// Another container on the same private network is not the ingress.
	untrusted := httptest.NewRequest("POST", "/api/v1/auth/otp/challenges", nil)
	untrusted.RemoteAddr = "10.1.2.3:44444"
	untrusted.Header.Set("X-Real-IP", "203.0.113.7")
	if got := server.clientIP(untrusted); got != "10.1.2.3" {
		t.Fatalf("an undeclared private peer must not set the rate-limit key, got %q", got)
	}
}

// Development runs without a signing key and without a declared ingress, where
// a private peer forwarding an address is the normal local setup.
func TestClientIPAcceptsPrivatePeerWhenNothingIsConfigured(t *testing.T) {
	t.Setenv("APP_ENV", "development")
	cfg, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	cfg.FrontendProxySigningKey = ""
	cfg.TrustedProxies = nil
	server := NewServer(cfg, slog.New(slog.NewTextHandler(io.Discard, nil)))

	request := httptest.NewRequest("GET", "/api/v1/healthz", nil)
	request.RemoteAddr = "127.0.0.1:44444"
	request.Header.Set("X-Real-IP", "203.0.113.7")
	if got := server.clientIP(request); got != "203.0.113.7" {
		t.Fatalf("local development must keep its forwarded address, got %q", got)
	}
}
