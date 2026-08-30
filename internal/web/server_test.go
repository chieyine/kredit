package web

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"kredit/internal/config"
)

func TestDecodeJSONRequestWritesClearClientError(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/test", bytes.NewBufferString(`{"known":true,"unexpected":1}`))
	response := httptest.NewRecorder()
	var input struct {
		Known bool `json:"known"`
	}
	if decodeJSONRequest(response, request, &input) {
		t.Fatal("malformed request was accepted")
	}
	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", response.Code)
	}
	var problem map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &problem); err != nil {
		t.Fatalf("response was not valid problem JSON: %v", err)
	}
	if problem["title"] != "invalid_request" {
		t.Fatalf("unexpected problem response: %#v", problem)
	}
}

func TestHealthEndpoint(t *testing.T) {
	cfg, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	server := NewServer(cfg, slog.New(slog.NewTextHandler(httptest.NewRecorder(), nil)))
	req := httptest.NewRequest(http.MethodGet, "/api/v1/healthz", nil)
	res := httptest.NewRecorder()
	server.Handler().ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.Code)
	}
	if res.Header().Get("X-Request-ID") == "" {
		t.Fatal("expected request id")
	}
}

func TestInvitationTokenIsRedactedFromAccessPath(t *testing.T) {
	if got := safeRequestPath("/api/v1/buyer-invitations/secret-token/accept"); got != "/api/v1/buyer-invitations/[redacted]/accept" {
		t.Fatalf("unexpected redacted path: %q", got)
	}
}

func TestInvalidRequestIDIsReplacedAndSecurityHeadersArePresent(t *testing.T) {
	cfg, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	server := NewServer(cfg, slog.New(slog.NewTextHandler(httptest.NewRecorder(), nil)))
	req := httptest.NewRequest(http.MethodGet, "/api/v1/healthz", nil)
	req.Header.Set("X-Request-ID", "bad\nvalue")
	res := httptest.NewRecorder()
	server.Handler().ServeHTTP(res, req)
	if res.Header().Get("X-Request-ID") == "bad\nvalue" || res.Header().Get("X-Request-ID") == "" {
		t.Fatal("unsafe request id was accepted")
	}
	for _, header := range []string{"Cross-Origin-Opener-Policy", "Cross-Origin-Resource-Policy", "X-Permitted-Cross-Domain-Policies"} {
		if res.Header().Get(header) == "" {
			t.Fatalf("missing security header %s", header)
		}
	}
}

func TestTraceContextIsPropagatedAndRequestsAreMeasured(t *testing.T) {
	cfg, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	server := NewServer(cfg, slog.New(slog.NewTextHandler(httptest.NewRecorder(), nil)))
	req := httptest.NewRequest(http.MethodGet, "/api/v1/healthz", nil)
	req.Header.Set("traceparent", "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01")
	res := httptest.NewRecorder()
	server.Handler().ServeHTTP(res, req)
	if res.Header().Get("X-Trace-ID") != "4bf92f3577b34da6a3ce929d0e0e4736" {
		t.Fatalf("trace ID was not propagated: %q", res.Header().Get("X-Trace-ID"))
	}
	snapshot := server.runtime.Metrics.Snapshot()
	if snapshot.Counters["http_requests_total"] == 0 || snapshot.Durations["http_request_duration"].Count == 0 {
		t.Fatalf("request metrics were not recorded: %#v", snapshot)
	}
}
