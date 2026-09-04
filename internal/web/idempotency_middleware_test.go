package web

import (
	"bytes"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"kredit/internal/config"
)

func TestIdempotencyMiddlewareReplaysAndRejectsConflicts(t *testing.T) {
	cfg := config.Config{Environment: "development", Version: "test", APIListenAddr: ":0", Currency: "NGN", MoneyUnit: "kobo", CollectionProvider: "mock"}
	server := NewServer(cfg, slog.New(slog.NewTextHandler(io.Discard, nil)))
	server.mux.HandleFunc("POST /api/v1/idempotency-test", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusCreated, map[string]string{"result": "created"})
	})
	handler := server.Handler()

	request := func(payload string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/idempotency-test", bytes.NewBufferString(payload))
		req.Header.Set("Idempotency-Key", "test-key")
		req.Header.Set("Content-Type", "application/json")
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, req)
		return response
	}

	first := request(`{"amount":10}`)
	if first.Code != http.StatusCreated || first.Header().Get("X-Idempotent-Replay") != "" {
		t.Fatalf("unexpected first response: %d %q", first.Code, first.Body.String())
	}
	second := request(`{"amount":10}`)
	if second.Code != http.StatusCreated || second.Header().Get("X-Idempotent-Replay") != "true" || second.Body.String() != first.Body.String() {
		t.Fatalf("expected exact replay, first=%d/%q second=%d/%q", first.Code, first.Body.String(), second.Code, second.Body.String())
	}
	conflict := request(`{"amount":11}`)
	if conflict.Code != http.StatusConflict {
		t.Fatalf("expected idempotency conflict, got %d: %s", conflict.Code, conflict.Body.String())
	}
}

func TestFinancialMutationsRequireIdempotencyKey(t *testing.T) {
	cfg := config.Config{Environment: "development", Version: "test", APIListenAddr: ":0", Currency: "NGN", MoneyUnit: "kobo", CollectionProvider: "mock"}
	server := NewServer(cfg, slog.New(slog.NewTextHandler(io.Discard, nil)))
	server.mux.HandleFunc("POST /api/v1/organizations/org-1/credit-requests", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusCreated, map[string]string{"result": "created"})
	})
	response := httptest.NewRecorder()
	server.Handler().ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/api/v1/organizations/org-1/credit-requests", bytes.NewBufferString(`{"principal_kobo":100}`)))
	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected missing idempotency key to be rejected, got %d: %s", response.Code, response.Body.String())
	}
}

func TestFinancialMutationRouteMatrixRequiresIdempotencyKey(t *testing.T) {
	cases := []string{
		"/api/v1/credit-requests/req-1/accept",
		"/api/v1/obligations/obl-1/release",
		"/api/v1/obligations/obl-1/receipt",
		"/api/v1/payments/adjust",
		"/api/v1/settlement/change",
		"/api/v1/organizations/org-1/members",
		"/api/v1/buyer/trade-lines/line-1/drawdowns/draw-1/confirm",
		"/api/v1/organizations/org-1/credit-requests/req-1/send",
		"/api/v1/me/product-feedback",
	}
	for _, path := range cases {
		t.Run(path, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, path, bytes.NewBufferString(`{}`))
			if !requiresIdempotencyKey(req) {
				t.Fatalf("route %s is not protected", path)
			}
		})
	}
}

func TestOneTimeCredentialsAreNotStoredForReplay(t *testing.T) {
	cfg := config.Config{Environment: "development", CollectionProvider: "mock"}
	s := NewServer(cfg, slog.New(slog.NewTextHandler(io.Discard, nil)))
	s.mux.HandleFunc("POST /api/v1/mfa/test-one-time", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, map[string]string{"secret": "private-test-value"})
	})
	request := func() *httptest.ResponseRecorder {
		r := httptest.NewRequest("POST", "/api/v1/mfa/test-one-time", bytes.NewBufferString(`{}`))
		r.Header.Set("Idempotency-Key", "one-time")
		w := httptest.NewRecorder()
		s.Handler().ServeHTTP(w, r)
		return w
	}
	if first := request(); first.Code != 200 {
		t.Fatal(first.Code)
	}
	if replay := request(); replay.Code != 409 || bytes.Contains(replay.Body.Bytes(), []byte("private-test-value")) {
		t.Fatalf("one-time response leaked: %d", replay.Code)
	}
}

func TestAdminCommandPreviewDoesNotRequireMutationKey(t *testing.T) {
	if requiresIdempotencyKey(httptest.NewRequest("POST", "/api/v1/ops/commands/preview", nil)) {
		t.Fatal("read-only preview requires a key")
	}
	if !requiresIdempotencyKey(httptest.NewRequest("POST", "/api/v1/ops/commands", nil)) {
		t.Fatal("command execution lost its key requirement")
	}
}

func TestDocumentJSONCanCarryAFullTwoMiBFile(t *testing.T) {
	cfg := config.Config{Environment: "development", Version: "test", APIListenAddr: ":0", Currency: "NGN", MoneyUnit: "kobo", CollectionProvider: "mock"}
	server := NewServer(cfg, slog.New(slog.NewTextHandler(io.Discard, nil)))
	server.mux.HandleFunc("POST /api/v1/organizations/org/documents", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusCreated, map[string]bool{"ok": true})
	})
	// Base64 expands two MiB to roughly 2.67 MiB before the small JSON envelope.
	payload := `{"content_base64":"` + strings.Repeat("A", (2<<20)*4/3) + `"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/organizations/org/documents", bytes.NewBufferString(payload))
	req.Header.Set("Idempotency-Key", "large-document")
	response := httptest.NewRecorder()
	server.Handler().ServeHTTP(response, req)
	if response.Code != http.StatusCreated {
		t.Fatalf("full-size document JSON rejected: %d %s", response.Code, response.Body.String())
	}
}
