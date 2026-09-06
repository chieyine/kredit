package config

import (
	"strings"
	"testing"
	"time"
)

func TestMonoProductionRequiresCertificationAndLiveCredentials(t *testing.T) {
	base := Config{
		Environment:               "production",
		Version:                   "1",
		APIListenAddr:             ":8080",
		Currency:                  "NGN",
		MoneyUnit:                 "kobo",
		MonoSweepEnabled:          true,
		CollectionProvider:        "mono-sweep",
		MonoWebhookSecret:         "mono-webhook-secret-0123456789abcdef0123456789",
		MonoRedirectURL:           "https://app.kredit.com.ng/mono/return",
		MonoSecretKey:             "live_sk_0123456789abcdef0123456789abcdef",
		RealCollections:           true,
		CollectionNoticeMinHours:  24,
		DeemedAcceptanceMinHours:  72,
		ProviderApprovalReference: "mono-provider-approval-001",
		ProviderApprovedBy:        "compliance",
		ProviderApprovedAt:        time.Now().UTC().Add(-time.Hour).Format(time.RFC3339),
	}

	if err := base.Validate(); err == nil || !strings.Contains(err.Error(), "PROVIDER_CERTIFICATION_REFERENCE") {
		t.Fatalf("expected certification evidence gate, got %v", err)
	}

	certified := base
	certified.ProviderCertificationReference = "mono-certification-001"
	certified.MonoSecretKey = "test_sk_fixture"
	if err := certified.Validate(); err == nil || !strings.Contains(err.Error(), "sandbox keys are refused") {
		t.Fatalf("expected sandbox key to be rejected in production, got %v", err)
	}

	certified.MonoSecretKey = "live_sk_0123456789abcdef0123456789abcdef"
	if err := certified.Validate(); err == nil || !strings.Contains(err.Error(), "FEATURE_APPROVED_RETENTION_POLICY") {
		t.Fatalf("expected Mono gate to pass into the broader production readiness gates, got %v", err)
	}
}

func TestMonoStagingRefusesLiveCredential(t *testing.T) {
	cfg := Config{
		Environment:        "staging",
		Version:            "1",
		APIListenAddr:      ":8080",
		Currency:           "NGN",
		MoneyUnit:          "kobo",
		MonoSweepEnabled:   true,
		CollectionProvider: "mono-sweep",
		MonoWebhookSecret:  "sandbox-webhook-secret",
		MonoRedirectURL:    "https://example.test/return",
		MonoSecretKey:      "live_sk_0123456789abcdef0123456789abcdef",
	}
	if err := cfg.Validate(); err == nil || !strings.Contains(err.Error(), "test secret key") {
		t.Fatalf("expected live credential to be refused outside production, got %v", err)
	}
}
