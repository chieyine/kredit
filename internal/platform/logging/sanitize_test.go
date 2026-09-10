package logging

import (
	"bytes"
	"errors"
	"log/slog"
	"strings"
	"testing"
)

func TestSafePathRemovesQueriesAndOpaqueIdentifiers(t *testing.T) {
	if got := SafePath("/api/v1/buyer-invitations/raw-secret/accept?phone=2348000000000"); got != "/api/v1/buyer-invitations/[redacted]/accept" {
		t.Fatalf("unexpected safe path: %q", got)
	}
	if got := SafePath("/api/v1/credit-requests?account_number=1234"); got != "/api/v1/credit-requests" {
		t.Fatalf("query leaked into safe path: %q", got)
	}
}

func TestSafeAttributesRedactsRestrictedKeys(t *testing.T) {
	got := SafeAttributes(map[string]string{"phone": "2348000000000", "operation": "token=abc"})
	if got["phone"] != "[redacted]" || got["operation"] != "token=[redacted]" {
		t.Fatalf("sensitive attributes were not redacted: %#v", got)
	}
}

func TestSanitizingHandlerRedactsErrorAndSensitiveAttributes(t *testing.T) {
	var output bytes.Buffer
	logger := slog.New(NewSanitizingHandler(slog.NewTextHandler(&output, nil)))
	logger.Error("request failed", "error", errors.New("postgres://user:password@db.internal/secret"), "phone", "2348000000000")
	text := output.String()
	if strings.Contains(text, "password") || strings.Contains(text, "2348000000000") {
		t.Fatalf("sensitive log data leaked: %s", text)
	}
}

func TestSafePathRedactsPublicFinancialLinks(t *testing.T) {
	for _, prefix := range []string{"/api/v1/public/receipts/", "/api/v1/public/payment-intents/", "/receipt/", "/pay/", "/c/"} {
		if got := SafePath(prefix + "signed-bearer-link?source=message"); got != prefix+"[redacted]" {
			t.Errorf("public financial token leaked: %q", got)
		}
	}
}

type secretLogValue struct{}

func (secretLogValue) LogValue() slog.Value {
	return slog.GroupValue(slog.String("token", "nested-private-value"))
}

func TestSanitizingHandlerRedactsGroupsAndDeferredValues(t *testing.T) {
	var output bytes.Buffer
	logger := slog.New(NewSanitizingHandler(slog.NewJSONHandler(&output, nil)))
	logger.With(slog.Group("context", slog.String("password", "private-password"))).Info("event", slog.Any("details", secretLogValue{}))
	for _, secret := range []string{"private-password", "nested-private-value"} {
		if strings.Contains(output.String(), secret) {
			t.Fatal("structured log leaked sensitive value")
		}
	}
	attributes := SafeAttributes(map[string]string{"authorization": "private-auth", "cookie": "private-cookie"})
	if attributes["authorization"] != "[redacted]" || attributes["cookie"] != "[redacted]" {
		t.Fatal("credential headers were not redacted")
	}
}
