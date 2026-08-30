package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSimulatorCoversProviderContractsAndIdempotency(t *testing.T) {
	handler := newSimulator().handler()

	mandate := postJSON(t, handler, "/mandates", map[string]any{"user_id": "buyer-1", "business_id": "business-1", "amount_ceiling": 120000000}, nil)
	if mandate["status"] != "active" || mandate["provider_id"] == "" {
		t.Fatalf("unexpected mandate: %#v", mandate)
	}
	cancelled := postJSON(t, handler, "/mandates/"+mandate["provider_id"].(string)+"/cancel", map[string]string{"reason": "buyer request"}, nil)
	if cancelled["status"] != "cancelled" {
		t.Fatalf("unexpected cancelled mandate: %#v", cancelled)
	}
	restored := postJSON(t, handler, "/mandates/"+mandate["provider_id"].(string)+"/restore", map[string]string{}, nil)
	if restored["status"] != "active" || restored["provider_id"] == mandate["provider_id"] {
		t.Fatalf("unexpected restored mandate: %#v", restored)
	}

	partial := postJSON(t, handler, "/collections", map[string]any{"external_reference": "attempt-1", "amount_kobo": 1000}, map[string]string{"X-Simulator-Scenario": "partial"})
	if partial["state"] != "partial" || partial["succeeded_amount_kobo"] != float64(500) {
		t.Fatalf("unexpected partial collection: %#v", partial)
	}

	first := postJSON(t, handler, "/notifications", map[string]any{"event_id": "event-1", "channel": "email"}, map[string]string{"Idempotency-Key": "event-1:email"})
	second := postJSON(t, handler, "/notifications", map[string]any{"event_id": "event-1", "channel": "email"}, map[string]string{"Idempotency-Key": "event-1:email"})
	if first["message_id"] != second["message_id"] {
		t.Fatalf("notification replay changed provider id: %#v %#v", first, second)
	}

	scan := postJSON(t, handler, "/documents/scan?scenario=quarantine", map[string]any{"document_id": "doc-1"}, nil)
	if scan["state"] != "QUARANTINED" {
		t.Fatalf("unexpected scan response: %#v", scan)
	}
}

func TestCancelMandateMalformedJSONWritesOneError(t *testing.T) {
	handler := newSimulator().handler()
	request := httptest.NewRequest(http.MethodPost, "/mandates/missing/cancel", strings.NewReader("{"))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
	}
	decoder := json.NewDecoder(recorder.Body)
	var problem map[string]string
	if err := decoder.Decode(&problem); err != nil {
		t.Fatal(err)
	}
	if problem["error"] != "invalid JSON" {
		t.Fatalf("error = %q, want invalid JSON", problem["error"])
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		t.Fatalf("response contained more than one JSON value: %v", err)
	}
}

func TestSimulatorRejectsOversizedAndTrailingJSON(t *testing.T) {
	handler := newSimulator().handler()
	tests := []struct {
		name       string
		body       string
		wantStatus int
	}{
		{name: "oversized", body: `{"event_id":"` + strings.Repeat("a", (1<<20)+1) + `"}`, wantStatus: http.StatusRequestEntityTooLarge},
		{name: "trailing value", body: `{} {}`, wantStatus: http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, "/notifications", strings.NewReader(tt.body))
			request.Header.Set("Content-Type", "application/json")
			request.Header.Set("Idempotency-Key", "test")
			response := httptest.NewRecorder()

			handler.ServeHTTP(response, request)

			if response.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", response.Code, tt.wantStatus)
			}
		})
	}
}

func TestSimulatorHealthcheckUsesConfiguredAddress(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/healthz" {
			t.Fatalf("path = %q, want /healthz", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	t.Setenv("SIMULATOR_ADDR", strings.TrimPrefix(server.URL, "http://"))

	if status := runSelfHealthcheck(); status != 0 {
		t.Fatalf("runSelfHealthcheck() = %d, want 0", status)
	}
}

func postJSON(t *testing.T, handler http.Handler, path string, input any, headers map[string]string) map[string]any {
	t.Helper()
	body, _ := json.Marshal(input)
	request := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	for key, value := range headers {
		request.Header.Set(key, value)
	}
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	response := recorder.Result()
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		t.Fatalf("unexpected status %d", response.StatusCode)
	}
	var result map[string]any
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		t.Fatal(err)
	}
	return result
}
