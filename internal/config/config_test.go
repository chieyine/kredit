package config

import (
	"strings"
	"testing"
	"time"
)

func TestLoadUsesNigerianMoneyDefaults(t *testing.T) {
	t.Setenv("APP_ENV", "development")
	t.Setenv("FEATURE_REAL_COLLECTIONS", "false")
	t.Setenv("FEATURE_REAL_IDENTITY", "false")
	t.Setenv("FEATURE_WHATSAPP", "false")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Currency != "NGN" || cfg.MoneyUnit != "kobo" || cfg.Timezone != "Africa/Lagos" {
		t.Fatalf("unexpected defaults: %+v", cfg)
	}
	if cfg.MultiAccountCollections || cfg.DirectSupplierSettlement || cfg.LiveSupplierBilling || cfg.ApprovedRetentionPolicy || cfg.ProductionPilot {
		t.Fatal("externally gated capabilities must default to disabled")
	}
}

func TestLoadParsesExternalDecisionFeatureGates(t *testing.T) {
	t.Setenv("APP_ENV", "development")
	t.Setenv("FEATURE_REAL_COLLECTIONS", "true")
	t.Setenv("PROVIDER_APPROVAL_REFERENCE", "EXT-001-approved")
	t.Setenv("PROVIDER_APPROVED_BY", "compliance")
	t.Setenv("PROVIDER_APPROVED_AT", time.Now().UTC().Add(-time.Hour).Format(time.RFC3339))
	t.Setenv("FEATURE_MULTI_ACCOUNT_COLLECTIONS", "true")
	t.Setenv("FEATURE_DIRECT_SUPPLIER_SETTLEMENT", "true")
	t.Setenv("FEATURE_LIVE_SUPPLIER_BILLING", "true")
	t.Setenv("FEATURE_APPROVED_RETENTION_POLICY", "true")
	t.Setenv("FEATURE_PRODUCTION_PILOT", "true")
	t.Setenv("MULTI_ACCOUNT_APPROVAL_REFERENCE", "EXT-003-approved")
	t.Setenv("DIRECT_SETTLEMENT_APPROVAL_REFERENCE", "EXT-005-approved")
	t.Setenv("BILLING_TAX_APPROVAL_REFERENCE", "EXT-006-approved")
	t.Setenv("RETENTION_APPROVAL_REFERENCE", "EXT-008-approved")
	t.Setenv("PILOT_APPROVAL_REFERENCE", "EXT-009-approved")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.MultiAccountCollections || !cfg.DirectSupplierSettlement || !cfg.LiveSupplierBilling || !cfg.ApprovedRetentionPolicy || !cfg.ProductionPilot {
		t.Fatal("expected all explicitly enabled external decision gates")
	}
}

func TestExternalDecisionFeatureGatesFailClosedWithoutEvidence(t *testing.T) {
	t.Setenv("APP_ENV", "development")
	t.Setenv("FEATURE_LIVE_SUPPLIER_BILLING", "true")
	t.Setenv("BILLING_TAX_APPROVAL_REFERENCE", "")
	if _, err := Load(); err == nil || !strings.Contains(err.Error(), "written approval reference") {
		t.Fatalf("expected missing external decision evidence to fail closed, got %v", err)
	}
}

func TestProductionRejectsDevelopmentSecrets(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	if _, err := Load(); err == nil {
		t.Fatal("expected production configuration validation error")
	}
}

func TestRealCollectionsRequiresWrittenApproval(t *testing.T) {
	t.Setenv("APP_ENV", "development")
	t.Setenv("FEATURE_REAL_COLLECTIONS", "true")
	t.Setenv("PROVIDER_APPROVAL_REFERENCE", "")
	t.Setenv("PROVIDER_APPROVED_BY", "")
	if _, err := Load(); err == nil {
		t.Fatal("expected approval gate")
	}
}

func TestProductionRejectsWeakSecretsAndLocalEndpoints(t *testing.T) {
	cfg := Config{
		Environment: "production", Version: "1", APIListenAddr: ":8080", Currency: "NGN", MoneyUnit: "kobo", CollectionProvider: "provider",
		PublicBaseURL: "https://app.example.com", AppBaseURL: "https://app.example.com", APIInternalURL: "https://api.example.com", ObjectStorageEndpoint: "https://s3.example.com", ObjectStorageBucket: "bucket", ObjectStorageRegion: "region", ObjectStorageAccessKey: "access-key", ObjectStorageSecretKey: "short", FieldEncryptionKeyID: "kms-key",
		SessionSigningKey: "short", OTPHMACKey: "short", TokenHashKey: "short", DatabaseURL: "postgres://db.example/kredit?sslmode=require", DatabaseDirectURL: "postgres://db.example/kredit?sslmode=require", RiverDatabaseURL: "postgres://db.example/kredit?sslmode=require",
	}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected weak production secret validation error")
	}
}

func TestProductionRejectsMockProviders(t *testing.T) {
	cfg := Config{Environment: "production", Version: "1", APIListenAddr: ":8080", Currency: "NGN", MoneyUnit: "kobo", CollectionProvider: "mock-collection", RealIdentity: false, RealCollections: false}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected production mock-provider gate")
	}
}

func TestProductionRequiresApprovedRetentionAndPilotEnablement(t *testing.T) {
	cfg := Config{Environment: "production", Version: "1", APIListenAddr: ":8080", Currency: "NGN", MoneyUnit: "kobo", CollectionProvider: "provider"}
	if err := cfg.Validate(); err == nil || !strings.Contains(err.Error(), "FEATURE_APPROVED_RETENTION_POLICY") {
		t.Fatalf("expected approved retention gate, got %v", err)
	}
	cfg.ApprovedRetentionPolicy = true
	cfg.RetentionApprovalReference = "retention-approval"
	if err := cfg.Validate(); err == nil || !strings.Contains(err.Error(), "FEATURE_PRODUCTION_PILOT") {
		t.Fatalf("expected production pilot gate, got %v", err)
	}
}

func TestValidateProductionURLRequiresTLS(t *testing.T) {
	if err := validateProductionURL("PUBLIC_BASE_URL", "http://app.example.com"); err == nil {
		t.Fatal("expected HTTPS validation error")
	}
}

func TestLoadNormalizesEnvironment(t *testing.T) {
	t.Setenv("APP_ENV", " Development ")
	t.Setenv("FEATURE_REAL_COLLECTIONS", "false")
	t.Setenv("FEATURE_REAL_IDENTITY", "false")
	t.Setenv("FEATURE_WHATSAPP", "false")

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Environment != "development" {
		t.Fatalf("environment = %q, want development", cfg.Environment)
	}
}

func TestValidateRejectsUnknownEnvironment(t *testing.T) {
	cfg := Config{Environment: "prod", Version: "1", APIListenAddr: ":8080", Currency: "NGN", MoneyUnit: "kobo", CollectionProvider: "provider"}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected unknown environment to be rejected")
	}
}

func TestRealCollectionsRequiresValidApprovalTime(t *testing.T) {
	base := Config{Environment: "staging", Version: "1", APIListenAddr: ":8080", Currency: "NGN", MoneyUnit: "kobo", CollectionProvider: "provider", RealCollections: true, ProviderApprovalReference: "approval-1", ProviderApprovedBy: "compliance"}
	tests := []struct {
		name       string
		approvedAt string
	}{
		{name: "missing", approvedAt: ""},
		{name: "malformed", approvedAt: "yesterday"},
		{name: "future", approvedAt: time.Now().UTC().Add(time.Hour).Format(time.RFC3339)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := base
			cfg.ProviderApprovedAt = tt.approvedAt
			if err := cfg.Validate(); err == nil {
				t.Fatal("expected provider approval time to be rejected")
			}
		})
	}
}

func TestValidateProductionDatabaseURLRequiresPostgresTLSAndRemoteHost(t *testing.T) {
	tests := []struct {
		name  string
		value string
	}{
		{name: "not a URL", value: "db.internal"},
		{name: "wrong scheme", value: "https://db.example.com/kredit?sslmode=require"},
		{name: "missing TLS mode", value: "postgres://db.example.com/kredit"},
		{name: "TLS disabled", value: "postgres://db.example.com/kredit?sslmode=disable"},
		{name: "IPv6 loopback", value: "postgres://[::1]/kredit?sslmode=require"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := validateProductionDatabaseURL("DATABASE_URL", tt.value); err == nil {
				t.Fatal("expected database URL to be rejected")
			}
		})
	}
	if err := validateProductionDatabaseURL("DATABASE_URL", "postgresql://db.internal/kredit?sslmode=verify-full"); err != nil {
		t.Fatalf("valid database URL rejected: %v", err)
	}
}

func TestValidateProductionURLRejectsAllLoopbackAddresses(t *testing.T) {
	for _, value := range []string{"https://127.0.0.2/service", "https://[0:0:0:0:0:0:0:1]/service", "https://localhost./service"} {
		if err := validateProductionURL("SERVICE_URL", value); err == nil {
			t.Fatalf("expected %q to be rejected", value)
		}
	}
}

func TestValidateSecretRejectsRepeatedUnicodeCharacter(t *testing.T) {
	if err := validateSecret("TOKEN", strings.Repeat("é", 16)); err == nil {
		t.Fatal("expected repeated Unicode secret to be rejected")
	}
}
