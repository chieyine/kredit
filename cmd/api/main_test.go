package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRunSelfHealthcheck(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/healthz" || r.URL.Path == "/healthz" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"status":"ok"}`))
			return
		}
		http.NotFound(w, r)
	}))
	defer ts.Close()

	addr := strings.TrimPrefix(ts.URL, "http://")
	t.Setenv("API_ADDR", addr)

	code := runSelfHealthcheck()
	if code != 0 {
		t.Fatalf("expected healthcheck code 0 for healthy server, got %d", code)
	}

	t.Setenv("API_ADDR", "127.0.0.1:1")
	codeUnhealthy := runSelfHealthcheck()
	if codeUnhealthy != 1 {
		t.Fatalf("expected healthcheck code 1 for unreachable server, got %d", codeUnhealthy)
	}
}
