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
		SessionSigningKey: "short", OTPHMACKey: "short", TokenHashKey: "short", DatabaseURL: "postgres://db.example/kredit?sslmode=verify-full", DatabaseDirectURL: "postgres://db.example/kredit?sslmode=verify-full", RiverDatabaseURL: "postgres://db.example/kredit?sslmode=verify-full",
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

// productionBase is a production configuration whose infrastructure is complete
// and whose capabilities are all switched off. Every gate test starts here and
// turns on the one thing it is about.
//
// This exists because the earlier gate tests asserted on the first error a bare
// Config produced, which made them tests of the order of the checks rather than
// of the gates they were named after: moving one unrelated check to the top of
// the block failed two tests that had nothing to do with it. A test named for a
// gate should fail for that gate.
func productionBase() Config {
	return Config{
		Environment:        "production",
		Version:            "1",
		APIListenAddr:      ":8080",
		Currency:           "NGN",
		MoneyUnit:          "kobo",
		CollectionProvider: "provider",
		AdminSurfaces:      []string{"all"},

		SessionSigningKey:       "session-signing-key-fixture-0123456789abcdef",
		OTPHMACKey:              "otp-hmac-key-fixture-0123456789abcdef0123",
		TokenHashKey:            "token-hash-key-fixture-0123456789abcdef01",
		SettingsEncryptionKey:   "settings-encryption-fixture-0123456789abcd",
		FieldEncryptionKey:      "field-encryption-key-fixture-0123456789ab",
		FieldEncryptionKeyID:    "kms-key-001",
		FrontendProxySigningKey: "frontend-proxy-fixture-key-0123456789abcdef",
		ObjectStorageSecretKey:  "object-storage-secret-fixture-0123456789ab",
		ObjectStorageAccessKey:  "storage-access-001",
		ObjectStorageEndpoint:   "https://s3.kredit.test",
		ObjectStorageBucket:     "kredit",
		ObjectStorageRegion:     "eu-west-1",

		PublicBaseURL:  "https://kredit.test",
		AppBaseURL:     "https://app.kredit.test",
		APIInternalURL: "https://api.kredit.test",
		OTelEndpoint:   "https://otel.kredit.test",

		DatabaseURL:       "postgres://db.kredit.test/kredit?sslmode=verify-full",
		DatabaseDirectURL: "postgres://db.kredit.test/kredit?sslmode=verify-full",
		RiverDatabaseURL:  "postgres://db.kredit.test/kredit?sslmode=verify-full",

		// Holding records is unconditional in production, so the base satisfies it.
		ApprovedRetentionPolicy:    true,
		RetentionApprovalReference: "retention-approval-001",
	}
}

// The public site and the bookkeeping product must be able to run in production
// with no bank provider, no identity provider and no messaging provider — that
// separation is the whole point of splitting infrastructure from capabilities.
func TestProductionRunsWithoutAnyExternalProvider(t *testing.T) {
	if err := productionBase().Validate(); err != nil {
		t.Fatalf("production with no external capability should be valid, got %v", err)
	}
}

func TestProductionAlwaysRequiresApprovedRetention(t *testing.T) {
	cfg := productionBase()
	cfg.ApprovedRetentionPolicy = false
	cfg.RetentionApprovalReference = ""
	// Not conditional on collections: the deployment retains personal records
	// from its first sale whether or not it ever debits an account.
	if err := cfg.Validate(); err == nil || !strings.Contains(err.Error(), "FEATURE_APPROVED_RETENTION_POLICY") {
		t.Fatalf("expected approved retention gate, got %v", err)
	}
	cfg.ApprovedRetentionPolicy = true
	if err := cfg.Validate(); err == nil || !strings.Contains(err.Error(), "written approval reference") {
		t.Fatalf("retention enabled without its reference should fail closed, got %v", err)
	}
}

func TestLiveCollectionsRequireBoundedPilotEnablement(t *testing.T) {
	cfg := productionBase()
	cfg.RealCollections = true
	cfg.RealIdentity = true
	cfg.IdentityApprovalReference = "identity-approval-001"
	cfg.IdentityProvider = "certified-identity"
	cfg.IdentityProviderEndpoint = "https://identity.kredit.test"
	cfg.IdentityProviderToken = "identity-provider-token-0123456789abcdef01"
	cfg.IdentityWebhookSecret = "identity-webhook-secret-0123456789abcdef01"
	cfg.CollectionProviderEndpoint = "https://collections.kredit.test"
	cfg.CollectionProviderToken = "collection-provider-token-0123456789abcdef"
	cfg.CollectionWebhookSecret = "collection-webhook-secret-0123456789abcdef"
	cfg.CollectionNoticeMinHours = 24
	cfg.DeemedAcceptanceMinHours = 72
	cfg.ProviderApprovalReference = "provider-approval-001"
	cfg.ProviderApprovedBy = "compliance"
	cfg.ProviderApprovedAt = time.Now().UTC().Add(-time.Hour).Format(time.RFC3339)

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
		{name: "wrong scheme", value: "https://db.example.com/kredit?sslmode=verify-full"},
		{name: "missing TLS mode", value: "postgres://db.example.com/kredit"},
		{name: "TLS disabled", value: "postgres://db.example.com/kredit?sslmode=disable"},
		{name: "IPv6 loopback", value: "postgres://[::1]/kredit?sslmode=verify-full"},
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

func TestMonoSandboxFlagsFailClosed(t *testing.T) {
	t.Setenv("APP_ENV", "development")
	t.Setenv("MONO_SWEEP_ENABLED", "true")
	t.Setenv("COLLECTION_PROVIDER", "mono-sweep")
	t.Setenv("MONO_WEBHOOK_SECRET", "sandbox-webhook-secret")
	t.Setenv("MONO_REDIRECT_URL", "https://example.test/return")
	t.Setenv("MONO_SECRET_KEY", "live_sk_forbidden")
	if _, err := Load(); err == nil {
		t.Fatal("live credentials accepted for sandbox")
	}
	t.Setenv("MONO_SECRET_KEY", "test_sk_fixture")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.AutomaticCollectionEnabled || cfg.AutomaticRetryEnabled || cfg.PartialSweepEnabled {
		t.Fatal("financial flags enabled by default")
	}
	t.Setenv("APP_ENV", "production")
	if _, err = Load(); err == nil {
		t.Fatal("uncertified Mono production enabled")
	}
}

func TestProductionRequiresAnEnumeratedAdminSurface(t *testing.T) {
	cfg := Config{Environment: "production", Version: "1", APIListenAddr: ":8080", Currency: "NGN", MoneyUnit: "kobo", CollectionProvider: "provider", ApprovedRetentionPolicy: true, RetentionApprovalReference: "retention-approval", ProductionPilot: true, PilotApprovalReference: "pilot-approval"}
	if err := cfg.Validate(); err == nil || !strings.Contains(err.Error(), "ADMIN_SURFACES") {
		t.Fatalf("expected the admin surface enumeration gate, got %v", err)
	}
	cfg.AdminSurfaces = []string{"cases", "disputes"}
	if err := cfg.Validate(); err != nil && strings.Contains(err.Error(), "ADMIN_SURFACES") {
		t.Fatalf("an enumerated surface list must satisfy the gate, got %v", err)
	}
}

func TestAdminSurfacesIgnoreBlankEntries(t *testing.T) {
	if surfaces := splitList("  cases , ,disputes ,"); len(surfaces) != 2 || surfaces[0] != "cases" || surfaces[1] != "disputes" {
		t.Fatalf("unexpected surface list %#v", surfaces)
	}
	if surfaces := splitList("   "); surfaces != nil {
		t.Fatalf("a blank setting must read as not configured, got %#v", surfaces)
	}
}

func TestDeemedAcceptanceWindowIsBoundedWhereMoneyCanMove(t *testing.T) {
	cfg := Config{Environment: "development", Version: "1", APIListenAddr: ":8080", Currency: "NGN", MoneyUnit: "kobo", CollectionProvider: "provider", RealCollections: true, CollectionNoticeMinHours: 24, DeemedAcceptanceMinHours: 0, ProviderApprovalReference: "ref", ProviderApprovedBy: "approver", ProviderApprovedAt: "2026-01-01T00:00:00Z"}
	if err := cfg.Validate(); err == nil || !strings.Contains(err.Error(), "DEEMED_ACCEPTANCE_MIN_HOURS") {
		t.Fatalf("a disabled waiting period must be refused where real money can move, got %v", err)
	}
	cfg.DeemedAcceptanceMinHours = 12
	if err := cfg.Validate(); err == nil || !strings.Contains(err.Error(), "DEEMED_ACCEPTANCE_MIN_HOURS") {
		t.Fatalf("a window shorter than a day must be refused, got %v", err)
	}
	cfg.DeemedAcceptanceMinHours = 72
	if err := cfg.Validate(); err != nil && strings.Contains(err.Error(), "DEEMED_ACCEPTANCE_MIN_HOURS") {
		t.Fatalf("the default window must be accepted, got %v", err)
	}
}

func TestDisabledCollectionTimingCannotOverflow(t *testing.T) {
	base := Config{Environment: "development", Version: "1", APIListenAddr: ":8080", Currency: "NGN", MoneyUnit: "kobo", CollectionProvider: "mock-collection"}
	if err := base.Validate(); err != nil {
		t.Fatalf("disabled defaults: %v", err)
	}
	for _, value := range []int64{-1, 721, 1<<63 - 1} {
		cfg := base
		cfg.CollectionNoticeMinHours = value
		if err := cfg.Validate(); err == nil || !strings.Contains(err.Error(), "COLLECTION_NOTICE_MIN_HOURS") {
			t.Fatalf("collection hours %d: %v", value, err)
		}
		cfg = base
		cfg.DeemedAcceptanceMinHours = value
		if err := cfg.Validate(); err == nil || !strings.Contains(err.Error(), "DEEMED_ACCEPTANCE_MIN_HOURS") {
			t.Fatalf("acceptance hours %d: %v", value, err)
		}
	}
	base.CollectionNoticeMinHours, base.DeemedAcceptanceMinHours = 720, 720
	if err := base.Validate(); err != nil {
		t.Fatalf("maximum bounded settings: %v", err)
	}
}
