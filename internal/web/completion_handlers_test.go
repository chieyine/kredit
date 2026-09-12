package web

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"kredit/internal/config"
	"kredit/internal/mandates"
	"log/slog"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNotificationReceiptsAuthenticateExactBody(t *testing.T) {
	cfg := config.Config{Environment: "development", Currency: "NGN", MoneyUnit: "kobo", TokenHashKey: "test-only", CollectionProvider: "mock", NotificationEmailAdapter: "connector", NotificationEmailToken: strings.Repeat("e", 32)}
	s := NewServer(cfg, slog.Default())
	original := `{"event_id":"receipt","notification_event_id":"notice","message_id":"message","delivered_at":"2026-01-01T00:00:00Z"}`
	mac := hmac.New(sha256.New, []byte(cfg.NotificationEmailToken))
	mac.Write([]byte(original))
	signature := hex.EncodeToString(mac.Sum(nil))
	for _, test := range []struct {
		body, signature string
		status          int
	}{{original, "", 401}, {strings.Replace(original, "message\"", "changed\"", 1), signature, 401}, {original, signature, 200}} {
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

func TestBuyerCanFindStandaloneBankPermission(t *testing.T) {
	cfg := config.Config{Environment: "development", Currency: "NGN", MoneyUnit: "kobo", TokenHashKey: "test-only", CollectionProvider: "mock"}
	s := NewServer(cfg, slog.Default())
	created, err := s.runtime.Mandates.CreateAuthorizationSession(t.Context(), mandates.AuthorizationInput{UserID: "standalone-buyer", BusinessID: "business", SupplierOrganizationID: "supplier", AmountCeiling: 10000})
	if err != nil {
		t.Fatal(err)
	}
	current, views, found, err := s.findBuyerMandate(t.Context(), "standalone-buyer", created.ID)
	if err != nil || !found || current.ID != created.ID || len(views) != 0 {
		t.Fatalf("standalone permission hidden: %+v %v %v", current, found, err)
	}
	_, _, found, err = s.findBuyerMandate(t.Context(), "another-buyer", created.ID)
	if err != nil || found {
		t.Fatalf("another buyer found permission: %v %v", found, err)
	}
}
