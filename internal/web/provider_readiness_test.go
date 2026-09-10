package web

import (
	"kredit/internal/config"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestReadinessRejectsFailedConfiguredAdapterWithoutLeakingDetails(t *testing.T) {
	cfg := config.Config{Environment: "development"}
	runtime := NewRuntime(cfg)
	runtime.ProviderFailures = []string{"connector unavailable at private-provider.internal"}
	server := NewServerWithRuntime(cfg, slog.Default(), runtime)
	response := httptest.NewRecorder()
	server.ready(response, httptest.NewRequest(http.MethodGet, "/readyz", nil))
	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("ready status=%d", response.Code)
	}
	if strings.Contains(response.Body.String(), "private-provider.internal") {
		t.Fatal("provider diagnostics exposed publicly")
	}
	runtime.ProviderFailures = nil
	response = httptest.NewRecorder()
	server.ready(response, httptest.NewRequest(http.MethodGet, "/readyz", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("unconfigured optional adapters must not block initial setup: %d", response.Code)
	}
}
