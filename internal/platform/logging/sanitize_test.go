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
