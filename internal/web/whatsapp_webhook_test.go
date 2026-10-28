package web

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"kredit/internal/config"
	"kredit/internal/whatsapp"
)

func TestWhatsAppWebhookStatusDistinguishesSignatureReplayAndUnreadText(t *testing.T) {
	t.Setenv("APP_ENV", "development")
	cfg, _ := config.Load()
	server := NewServer(cfg, slog.Default())
	routes := server.Handler()
	post := func(event whatsapp.Event) *httptest.ResponseRecorder {
		body, _ := json.Marshal(event)
		rec := httptest.NewRecorder()
		routes.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/messaging/whatsapp", bytes.NewReader(body)))
		return rec
	}
	if rec := post(whatsapp.Event{ID: "wamid-1", From: "2348000000000", Text: "hello", Signature: "forged"}); rec.Code != http.StatusUnauthorized {
		t.Fatalf("forged signature: got %d %s", rec.Code, rec.Body.String())
	}
	handler := server.runtime.WhatsApp
	unread := whatsapp.Event{ID: "wamid-2", From: "2348000000000", Text: "zzzz qqqq"}
	unread.Signature = handler.Sign(unread)
	rec := post(unread)
	if rec.Code != http.StatusOK || !bytes.Contains(rec.Body.Bytes(), []byte(`"understood":false`)) {
		t.Fatalf("authentic unread text must be acknowledged: got %d %s", rec.Code, rec.Body.String())
	}
	reused := whatsapp.Event{ID: "wamid-2", From: "2348000000000", Text: "different text"}
	reused.Signature = handler.Sign(reused)
	if rec := post(reused); rec.Code != http.StatusConflict {
		t.Fatalf("reused event identifier: got %d %s", rec.Code, rec.Body.String())
	}
}
