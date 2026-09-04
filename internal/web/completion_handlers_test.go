package web

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"kredit/internal/config"
	"log/slog"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNotificationReceiptsAuthenticateExactBody(t *testing.T) {
	cfg := config.Config{Environment: "development", Currency: "NGN", MoneyUnit: "kobo", TokenHashKey: "test-only", CollectionProvider: "mock", NotificationEmailToken: strings.Repeat("e", 32)}
	s := NewServer(cfg, slog.Default())
	original := `{"event_id":"receipt","notification_event_id":"notice","message_id":"message","delivered_at":"2026-01-01T00:00:00Z"}`
	mac := hmac.New(sha256.New, []byte(cfg.NotificationEmailToken))
	mac.Write([]byte(original))
	signature := hex.EncodeToString(mac.Sum(nil))
	for _, test := range []struct {
		body, signature string
		status          int
	}{{original, "", 401}, {strings.Replace(original, "message\"", "changed\"", 1), signature, 401}, {original, signature, 409}} {
		request := httptest.NewRequest("POST", "/api/v1/webhooks/notifications/email", strings.NewReader(test.body))
		request.SetPathValue("channel", "email")
		request.Header.Set("X-Notification-Signature", test.signature)
		recorder := httptest.NewRecorder()
		s.notificationDeliveryReceipt(recorder, request)
		if recorder.Code != test.status {
			t.Fatalf("got %d wanted %d: %s", recorder.Code, test.status, recorder.Body.String())
		}
	}
}
func TestScrapeCredentialOnlyGrantsMetricsAccess(t *testing.T) {
	cfg := config.Config{Environment: "development", Currency: "NGN", MoneyUnit: "kobo", TokenHashKey: "test-only", CollectionProvider: "mock", MetricsScrapeToken: strings.Repeat("m", 32)}
	s := NewServer(cfg, slog.Default())
	request := httptest.NewRequest("GET", "/api/v1/ops/metrics/prometheus", nil)
	request.Header.Set("Authorization", "Bearer "+cfg.MetricsScrapeToken)
	recorder := httptest.NewRecorder()
	s.metricsPrometheus(recorder, request)
	if recorder.Code != 200 {
		t.Fatalf("scrape failed %d", recorder.Code)
	}
	request = httptest.NewRequest("GET", "/api/v1/ops/financial-reconciliation", nil)
	request.Header.Set("Authorization", "Bearer "+cfg.MetricsScrapeToken)
	recorder = httptest.NewRecorder()
	s.financialReviews(recorder, request)
	if recorder.Code == 200 {
		t.Fatal("scrape credential opened financial cases")
	}
}
