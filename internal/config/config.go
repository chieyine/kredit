package config

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

const defaultTimezone = "Africa/Lagos"

// Config contains deployment configuration shared by the API and worker.
// Secrets are read from the environment and are never logged or serialized.
type Config struct {
	AdminConnectionVersions    map[string]int `json:"-"`
	MetricsScrapeToken         string
	MonoSweepEnabled           bool
	PartialSweepEnabled        bool
	CollectionNoticeMinHours   int64
	DeemedAcceptanceMinHours   int64
	AdminSurfaces              []string
	AutomaticCollectionEnabled bool
	AutomaticRetryEnabled      bool
	MonoSecretKey              string
	MonoWebhookSecret          string
	MonoRedirectURL            string

	Environment                    string
	Version                        string
	PublicBaseURL                  string
	AppBaseURL                     string
	APIInternalURL                 string
	APIListenAddr                  string
	DatabaseURL                    string
	DatabaseDirectURL              string
	RiverDatabaseURL               string
	ObjectStorageEndpoint          string
	ObjectStorageBucket            string
	ObjectStorageRegion            string
	ObjectStorageAccessKey         string
	ObjectStorageSecretKey         string
	DocumentScannerEnabled         bool
	DocumentScannerEndpoint        string
	DocumentScannerToken           string
	SessionSigningKey              string
	FieldEncryptionKeyID           string
	FieldEncryptionKey             string
	OTPHMACKey                     string
	TokenHashKey                   string
	SettingsEncryptionKey          string
	OTelEndpoint                   string
	Timezone                       string
	Currency                       string
	MoneyUnit                      string
	RealCollections                bool
	CollectionProvider             string
	CollectionProviderEndpoint     string
	CollectionProviderToken        string
	CollectionWebhookSecret        string
	ProviderApprovedAt             string
	ProviderApprovalReference      string
	ProviderApprovedBy             string
	MultiAccountApprovalReference  string
	DirectSettlementReference      string
	BillingTaxApprovalReference    string
	IdentityApprovalReference      string
	RetentionApprovalReference     string
	PilotApprovalReference         string
	SecurityReviewReference        string
	DPIAReference                  string
	LegalApprovalReference         string
	PenTestReference               string
	BackupRestoreReference         string
	ProviderCertificationReference string
	SupportTrainingReference       string
	LaunchApprovalReference        string
	PilotMaxSupplierOrganizations  int64
	PilotMaxBuyerBusinesses        int64
	PilotMaxPrincipalKobo          int64
	PilotMaxActiveExposureKobo     int64
	PilotMaxDrawdownsPerLineDay    int64
	PilotMaxCollectionRetries      int64
	PilotEnhancedReviewKobo        int64
	PilotAllowedProviderAccounts   string
	PilotAllowedIndustries         string
	RealIdentity                   bool
	WhatsApp                       bool
	OffPlatformPaymentClaims       bool
	MultiAccountCollections        bool
	DirectSupplierSettlement       bool
	LiveSupplierBilling            bool
	ApprovedRetentionPolicy        bool
	ProductionPilot                bool
	NotificationEmailEndpoint      string
	NotificationEmailToken         string
	NotificationSMSEndpoint        string
	NotificationSMSToken           string
	NotificationWhatsAppEndpoint   string
	NotificationWhatsAppToken      string
	IdentityProvider               string
	IdentityProviderEndpoint       string
	IdentityProviderToken          string
	IdentityWebhookSecret          string
}

func Load() (Config, error) {
	c := Config{
		MetricsScrapeToken:             envOr("METRICS_SCRAPE_TOKEN", ""),
		MonoSecretKey:                  envOr("MONO_SECRET_KEY", ""),
		MonoWebhookSecret:              envOr("MONO_WEBHOOK_SECRET", ""),
		MonoRedirectURL:                envOr("MONO_REDIRECT_URL", ""),
		Environment:                    strings.ToLower(strings.TrimSpace(envOr("APP_ENV", "development"))),
		Version:                        envOr("APP_VERSION", "0.1.0-dev"),
		PublicBaseURL:                  envOr("PUBLIC_BASE_URL", "http://localhost:5173"),
		AppBaseURL:                     envOr("APP_BASE_URL", "http://localhost:5173"),
		APIInternalURL:                 envOr("API_INTERNAL_URL", "http://localhost:8080"),
		APIListenAddr:                  envOr("API_ADDR", ":8080"),
		DatabaseURL:                    envOr("DATABASE_URL", "postgres://kredit_app_login:kredit-app-development-only@localhost:5432/kredit?sslmode=disable"),
		DatabaseDirectURL:              envOr("DATABASE_DIRECT_URL", ""),
		RiverDatabaseURL:               envOr("RIVER_DATABASE_URL", "postgres://kredit_worker_login:kredit-worker-development-only@localhost:5432/kredit?sslmode=disable"),
		ObjectStorageEndpoint:          envOr("OBJECT_STORAGE_ENDPOINT", "http://localhost:9000"),
		ObjectStorageBucket:            envOr("OBJECT_STORAGE_BUCKET", "kredit-local"),
		ObjectStorageRegion:            envOr("OBJECT_STORAGE_REGION", "us-east-1"),
		ObjectStorageAccessKey:         envOr("OBJECT_STORAGE_ACCESS_KEY", "minioadmin"),
		ObjectStorageSecretKey:         envOr("OBJECT_STORAGE_SECRET_KEY", "minioadmin"),
		DocumentScannerEndpoint:        envOr("DOCUMENT_SCANNER_ENDPOINT", ""),
		DocumentScannerToken:           envOr("DOCUMENT_SCANNER_TOKEN", ""),
		SessionSigningKey:              envOr("SESSION_SIGNING_KEY", "development-only-change-me"),
		FieldEncryptionKeyID:           envOr("FIELD_ENCRYPTION_KEY_ID", "development-only"),
		FieldEncryptionKey:             envOr("FIELD_ENCRYPTION_KEY", "development-only-change-me"),
		OTPHMACKey:                     envOr("OTP_HMAC_KEY", "development-only-change-me"),
		TokenHashKey:                   envOr("TOKEN_HASH_KEY", "development-only-change-me"),
		SettingsEncryptionKey:          envOr("SETTINGS_ENCRYPTION_KEY", ""),
		OTelEndpoint:                   envOr("OTEL_EXPORTER_OTLP_ENDPOINT", "http://localhost:4318"),
		Timezone:                       envOr("BUSINESS_TIMEZONE", defaultTimezone),
		Currency:                       "NGN",
		MoneyUnit:                      "kobo",
		CollectionProvider:             envOr("COLLECTION_PROVIDER", "mock-collection"),
		CollectionProviderEndpoint:     envOr("COLLECTION_PROVIDER_ENDPOINT", ""),
		CollectionProviderToken:        envOr("COLLECTION_PROVIDER_TOKEN", ""),
		CollectionWebhookSecret:        envOr("COLLECTION_WEBHOOK_SECRET", ""),
		ProviderApprovedAt:             envOr("PROVIDER_APPROVED_AT", ""),
		ProviderApprovalReference:      envOr("PROVIDER_APPROVAL_REFERENCE", ""),
		ProviderApprovedBy:             envOr("PROVIDER_APPROVED_BY", ""),
		MultiAccountApprovalReference:  envOr("MULTI_ACCOUNT_APPROVAL_REFERENCE", ""),
		DirectSettlementReference:      envOr("DIRECT_SETTLEMENT_APPROVAL_REFERENCE", ""),
		BillingTaxApprovalReference:    envOr("BILLING_TAX_APPROVAL_REFERENCE", ""),
		IdentityApprovalReference:      envOr("IDENTITY_APPROVAL_REFERENCE", ""),
		RetentionApprovalReference:     envOr("RETENTION_APPROVAL_REFERENCE", ""),
		PilotApprovalReference:         envOr("PILOT_APPROVAL_REFERENCE", ""),
		SecurityReviewReference:        envOr("SECURITY_REVIEW_REFERENCE", ""),
		DPIAReference:                  envOr("DPIA_REFERENCE", ""),
		LegalApprovalReference:         envOr("LEGAL_APPROVAL_REFERENCE", ""),
		PenTestReference:               envOr("PEN_TEST_REFERENCE", ""),
		BackupRestoreReference:         envOr("BACKUP_RESTORE_REFERENCE", ""),
		ProviderCertificationReference: envOr("PROVIDER_CERTIFICATION_REFERENCE", ""),
		SupportTrainingReference:       envOr("SUPPORT_TRAINING_REFERENCE", ""),
		LaunchApprovalReference:        envOr("LAUNCH_APPROVAL_REFERENCE", ""),
		PilotAllowedProviderAccounts:   envOr("PILOT_ALLOWED_PROVIDER_ACCOUNTS", ""),
		PilotAllowedIndustries:         envOr("PILOT_ALLOWED_INDUSTRIES", ""),
		NotificationEmailEndpoint:      envOr("NOTIFICATION_EMAIL_ENDPOINT", ""),
		NotificationEmailToken:         envOr("NOTIFICATION_EMAIL_TOKEN", ""),
		NotificationSMSEndpoint:        envOr("NOTIFICATION_SMS_ENDPOINT", ""),
		NotificationSMSToken:           envOr("NOTIFICATION_SMS_TOKEN", ""),
		NotificationWhatsAppEndpoint:   envOr("NOTIFICATION_WHATSAPP_ENDPOINT", ""),
		NotificationWhatsAppToken:      envOr("NOTIFICATION_WHATSAPP_TOKEN", ""),
		IdentityProvider:               envOr("IDENTITY_PROVIDER", "mock-identity"),
		IdentityProviderEndpoint:       envOr("IDENTITY_PROVIDER_ENDPOINT", ""),
		IdentityProviderToken:          envOr("IDENTITY_PROVIDER_TOKEN", ""),
		IdentityWebhookSecret:          envOr("IDENTITY_WEBHOOK_SECRET", ""),
	}

	var err error
	c.CollectionNoticeMinHours, err = int64Env("COLLECTION_NOTICE_MIN_HOURS", 24)
	if err != nil {
		return Config{}, err
	}
	// Three days, not one: goods released on a Friday afternoon must not be
	// deemed accepted before the buyer reopens on Monday.
	c.DeemedAcceptanceMinHours, err = int64Env("DEEMED_ACCEPTANCE_MIN_HOURS", 72)
	if err != nil {
		return Config{}, err
	}
	c.AdminSurfaces = splitList(envOr("ADMIN_SURFACES", ""))
	if c.RealCollections, err = boolEnv("FEATURE_REAL_COLLECTIONS", false); err != nil {
		return Config{}, err
	}
	if c.RealIdentity, err = boolEnv("FEATURE_REAL_IDENTITY", false); err != nil {
		return Config{}, err
	}
	if c.WhatsApp, err = boolEnv("FEATURE_WHATSAPP", false); err != nil {
		return Config{}, err
	}
	if c.OffPlatformPaymentClaims, err = boolEnv("OFF_PLATFORM_PAYMENT_CLAIMS_ENABLED", c.Environment == "development"); err != nil {
		return Config{}, err
	}
	for name, target := range map[string]*bool{
		"MONO_SWEEP_ENABLED":                 &c.MonoSweepEnabled,
		"PARTIAL_SWEEP_ENABLED":              &c.PartialSweepEnabled,
		"AUTOMATIC_COLLECTION_ENABLED":       &c.AutomaticCollectionEnabled,
		"AUTOMATIC_RETRY_ENABLED":            &c.AutomaticRetryEnabled,
		"FEATURE_MULTI_ACCOUNT_COLLECTIONS":  &c.MultiAccountCollections,
		"FEATURE_DIRECT_SUPPLIER_SETTLEMENT": &c.DirectSupplierSettlement,
		"FEATURE_LIVE_SUPPLIER_BILLING":      &c.LiveSupplierBilling,
		"FEATURE_APPROVED_RETENTION_POLICY":  &c.ApprovedRetentionPolicy,
		"FEATURE_PRODUCTION_PILOT":           &c.ProductionPilot,
	} {
		if *target, err = boolEnv(name, false); err != nil {
			return Config{}, err
		}
	}
	for name, target := range map[string]*int64{
		"PILOT_MAX_SUPPLIER_ORGANIZATIONS": &c.PilotMaxSupplierOrganizations,
		"PILOT_MAX_BUYER_BUSINESSES":       &c.PilotMaxBuyerBusinesses,
		"PILOT_MAX_PRINCIPAL_KOBO":         &c.PilotMaxPrincipalKobo,
		"PILOT_MAX_ACTIVE_EXPOSURE_KOBO":   &c.PilotMaxActiveExposureKobo,
		"PILOT_MAX_DRAWDOWNS_PER_LINE_DAY": &c.PilotMaxDrawdownsPerLineDay,
		"PILOT_MAX_COLLECTION_RETRIES":     &c.PilotMaxCollectionRetries,
		"PILOT_ENHANCED_REVIEW_KOBO":       &c.PilotEnhancedReviewKobo,
	} {
		if *target, err = int64Env(name, 0); err != nil {
			return Config{}, err
		}
	}
	if err := c.Validate(); err != nil {
		return Config{}, err
	}
	return c, nil
}

func (c Config) Validate() error {
	if strings.TrimSpace(c.Environment) == "" || strings.TrimSpace(c.Version) == "" || strings.TrimSpace(c.APIListenAddr) == "" {
		return errors.New("APP_ENV, APP_VERSION, and API_ADDR are required")
	}
	switch c.Environment {
	case "development", "staging", "production":
	default:
		return fmt.Errorf("APP_ENV must be development, staging, or production, got %q", c.Environment)
	}
	if c.MetricsScrapeToken != "" && len(c.MetricsScrapeToken) < 32 {
		return errors.New("METRICS_SCRAPE_TOKEN must contain at least 32 characters")
	}
	if c.Currency != "NGN" || c.MoneyUnit != "kobo" {
		return errors.New("money configuration must remain NGN/kobo")
	}
	if c.MonoSweepEnabled || c.CollectionProvider == "mono-sweep" {
		if c.CollectionProvider != "mono-sweep" || strings.TrimSpace(c.MonoWebhookSecret) == "" || strings.TrimSpace(c.MonoRedirectURL) == "" {
			return errors.New("mono Sweep requires COLLECTION_PROVIDER=mono-sweep, webhook secret, and redirect URL")
		}
		if c.Environment == "production" {
			if c.MonoSweepEnabled && strings.TrimSpace(c.ProviderCertificationReference) == "" {
				return errors.New("mono Sweep production requires PROVIDER_CERTIFICATION_REFERENCE")
			}
			if strings.TrimSpace(c.MonoSecretKey) == "" || strings.HasPrefix(c.MonoSecretKey, "test_sk_") {
				return errors.New("mono Sweep production requires a live Mono secret key; sandbox keys are refused")
			}
			if err := validateSecret("MONO_SECRET_KEY", c.MonoSecretKey); err != nil {
				return err
			}
			if err := validateSecret("MONO_WEBHOOK_SECRET", c.MonoWebhookSecret); err != nil {
				return err
			}
			if err := validateProductionURL("MONO_REDIRECT_URL", c.MonoRedirectURL); err != nil {
				return err
			}
		} else if !strings.HasPrefix(c.MonoSecretKey, "test_sk_") {
			return errors.New("mono sandbox requires a test secret key")
		}
	}
	if c.PartialSweepEnabled && !c.MonoSweepEnabled {
		return errors.New("PARTIAL_SWEEP_ENABLED requires MONO_SWEEP_ENABLED")
	}
	if c.AutomaticRetryEnabled && !c.AutomaticCollectionEnabled {
		return errors.New("automatic retries require automatic collection")
	}
	if strings.TrimSpace(c.CollectionProvider) == "" {
		return errors.New("COLLECTION_PROVIDER is required")
	}
	// Zero retains the application's default while collections are disabled.
	// Bound explicit values before runtime converts hours to time.Duration.
	if c.CollectionNoticeMinHours < 0 || c.CollectionNoticeMinHours > 720 {
		return errors.New("COLLECTION_NOTICE_MIN_HOURS must be between 0 and 720")
	}
	if c.DeemedAcceptanceMinHours < 0 || c.DeemedAcceptanceMinHours > 720 {
		return errors.New("DEEMED_ACCEPTANCE_MIN_HOURS must be between 0 and 720")
	}
	if (c.RealCollections || c.MonoSweepEnabled) && (c.CollectionNoticeMinHours < 1 || c.CollectionNoticeMinHours > 720) {
		return errors.New("real collections require COLLECTION_NOTICE_MIN_HOURS between 1 and 720")
	}
	// Buyer silence may only become a collectable obligation where the notice
	// has demonstrably been with the buyer long enough to answer it. Zero would
	// disable that evidence requirement, so it is refused wherever real money
	// can move.
	if (c.RealCollections || c.MonoSweepEnabled) && (c.DeemedAcceptanceMinHours < 24 || c.DeemedAcceptanceMinHours > 720) {
		return errors.New("real collections require DEEMED_ACCEPTANCE_MIN_HOURS between 24 and 720")
	}
	if c.RealCollections || (c.MonoSweepEnabled && c.Environment == "production") {
		if strings.TrimSpace(c.ProviderApprovalReference) == "" || strings.TrimSpace(c.ProviderApprovedBy) == "" || strings.TrimSpace(c.ProviderApprovedAt) == "" {
			return errors.New("real collections require a written provider approval reference, approver, and approval time")
		}
		approvedAt, err := time.Parse(time.RFC3339, c.ProviderApprovedAt)
		if err != nil {
			return fmt.Errorf("PROVIDER_APPROVED_AT must be an RFC3339 timestamp: %w", err)
		}
		if approvedAt.After(time.Now()) {
			return errors.New("PROVIDER_APPROVED_AT must not be in the future")
		}
	}
	for _, gate := range []struct {
		enabled   bool
		name      string
		reference string
	}{
		{c.MultiAccountCollections, "FEATURE_MULTI_ACCOUNT_COLLECTIONS", c.MultiAccountApprovalReference},
		{c.DirectSupplierSettlement, "FEATURE_DIRECT_SUPPLIER_SETTLEMENT", c.DirectSettlementReference},
		{c.LiveSupplierBilling, "FEATURE_LIVE_SUPPLIER_BILLING", c.BillingTaxApprovalReference},
		{c.RealIdentity, "FEATURE_REAL_IDENTITY", c.IdentityApprovalReference},
		{c.ApprovedRetentionPolicy, "FEATURE_APPROVED_RETENTION_POLICY", c.RetentionApprovalReference},
		{c.ProductionPilot, "FEATURE_PRODUCTION_PILOT", c.PilotApprovalReference},
	} {
		if gate.enabled && strings.TrimSpace(gate.reference) == "" {
			return fmt.Errorf("%s requires its written approval reference", gate.name)
		}
	}
	if c.MultiAccountCollections && !c.RealCollections {
		return errors.New("FEATURE_MULTI_ACCOUNT_COLLECTIONS requires FEATURE_REAL_COLLECTIONS")
	}
	if c.DirectSupplierSettlement && !c.RealCollections {
		return errors.New("FEATURE_DIRECT_SUPPLIER_SETTLEMENT requires FEATURE_REAL_COLLECTIONS")
	}
	// Production hardening has two halves and they are deliberately separate.
	//
	// Infrastructure requirements always apply: without them the deployment is
	// not safe to run at all. Capability requirements apply only to the
	// capabilities this deployment has actually switched on. That separation is
	// what lets the public site and the bookkeeping product go live before an
	// external bank, identity, messaging or scanning provider is contracted,
	// without weakening a single security control. Running as "staging" to skip
	// a provider is not an alternative: it would drop every check below at once.
	if c.Environment == "production" {
		// --- Infrastructure: always required. ---
		if len(c.AdminSurfaces) == 0 {
			return errors.New("ADMIN_SURFACES must list the operations surfaces this deployment enables, or 'all'")
		}
		// A deployment holds people's records from its first sale — names, phone
		// numbers, what they owe — whether or not it ever debits a bank account.
		// How long those are kept is therefore not a collections decision, and
		// this gate does not move with the collections capability.
		if !c.ApprovedRetentionPolicy {
			return errors.New("production requires FEATURE_APPROVED_RETENTION_POLICY and its written approval reference: the deployment retains personal records from its first sale")
		}
		for name, value := range map[string]string{
			"SESSION_SIGNING_KEY":         c.SessionSigningKey,
			"OTP_HMAC_KEY":                c.OTPHMACKey,
			"TOKEN_HASH_KEY":              c.TokenHashKey,
			"SETTINGS_ENCRYPTION_KEY":     c.SettingsEncryptionKey,
			"FIELD_ENCRYPTION_KEY":        c.FieldEncryptionKey,
			"FIELD_ENCRYPTION_KEY_ID":     c.FieldEncryptionKeyID,
			"DATABASE_URL":                c.DatabaseURL,
			"DATABASE_DIRECT_URL":         c.DatabaseDirectURL,
			"RIVER_DATABASE_URL":          c.RiverDatabaseURL,
			"OBJECT_STORAGE_ENDPOINT":     c.ObjectStorageEndpoint,
			"OBJECT_STORAGE_BUCKET":       c.ObjectStorageBucket,
			"OBJECT_STORAGE_REGION":       c.ObjectStorageRegion,
			"OBJECT_STORAGE_ACCESS_KEY":   c.ObjectStorageAccessKey,
			"OBJECT_STORAGE_SECRET_KEY":   c.ObjectStorageSecretKey,
			"OTEL_EXPORTER_OTLP_ENDPOINT": c.OTelEndpoint,
		} {
			if strings.TrimSpace(value) == "" {
				return fmt.Errorf("%s must be set to a production value", name)
			}
		}
		for name, value := range map[string]string{
			"SESSION_SIGNING_KEY":       c.SessionSigningKey,
			"OTP_HMAC_KEY":              c.OTPHMACKey,
			"TOKEN_HASH_KEY":            c.TokenHashKey,
			"SETTINGS_ENCRYPTION_KEY":   c.SettingsEncryptionKey,
			"FIELD_ENCRYPTION_KEY":      c.FieldEncryptionKey,
			"OBJECT_STORAGE_SECRET_KEY": c.ObjectStorageSecretKey,
		} {
			if err := validateSecret(name, value); err != nil {
				return err
			}
		}
		for name, value := range map[string]string{"FIELD_ENCRYPTION_KEY_ID": c.FieldEncryptionKeyID, "OBJECT_STORAGE_ACCESS_KEY": c.ObjectStorageAccessKey} {
			if err := validateIdentifier(name, value); err != nil {
				return err
			}
		}
		for name, value := range map[string]string{"PUBLIC_BASE_URL": c.PublicBaseURL, "APP_BASE_URL": c.AppBaseURL, "API_INTERNAL_URL": c.APIInternalURL, "OBJECT_STORAGE_ENDPOINT": c.ObjectStorageEndpoint, "OTEL_EXPORTER_OTLP_ENDPOINT": c.OTelEndpoint} {
			if err := validateProductionURL(name, value); err != nil {
				return err
			}
		}
		for name, value := range map[string]string{"DATABASE_URL": c.DatabaseURL, "DATABASE_DIRECT_URL": c.DatabaseDirectURL, "RIVER_DATABASE_URL": c.RiverDatabaseURL} {
			if err := validateProductionDatabaseURL(name, value); err != nil {
				return err
			}
		}

		// --- Sign-in delivery: validated when configured, never half-configured. ---
		// Neither channel is mandatory to boot: a deployment may publish the
		// public site before a messaging provider is contracted. What is refused
		// is a channel that is half set up, which would fail at send time with no
		// warning. Whether anyone can sign in is reported by readiness, not by
		// refusing to start.
		for _, channel := range []struct{ name, endpoint, token string }{
			{"NOTIFICATION_EMAIL", c.NotificationEmailEndpoint, c.NotificationEmailToken},
			{"NOTIFICATION_SMS", c.NotificationSMSEndpoint, c.NotificationSMSToken},
		} {
			endpoint, token := strings.TrimSpace(channel.endpoint), strings.TrimSpace(channel.token)
			if endpoint == "" && token == "" {
				continue
			}
			if endpoint == "" || token == "" {
				return fmt.Errorf("%s_ENDPOINT and %s_TOKEN must be set together", channel.name, channel.name)
			}
			if err := validateProductionURL(channel.name+"_ENDPOINT", endpoint); err != nil {
				return err
			}
			if err := validateSecret(channel.name+"_TOKEN", token); err != nil {
				return err
			}
		}

		// --- Capabilities: required only where the capability is switched on. ---
		if c.WhatsApp {
			if strings.TrimSpace(c.NotificationWhatsAppEndpoint) == "" || strings.TrimSpace(c.NotificationWhatsAppToken) == "" {
				return errors.New("FEATURE_WHATSAPP requires NOTIFICATION_WHATSAPP_ENDPOINT and NOTIFICATION_WHATSAPP_TOKEN")
			}
			if err := validateProductionURL("NOTIFICATION_WHATSAPP_ENDPOINT", c.NotificationWhatsAppEndpoint); err != nil {
				return err
			}
			if err := validateSecret("NOTIFICATION_WHATSAPP_TOKEN", c.NotificationWhatsAppToken); err != nil {
				return err
			}
		}

		// Document uploads are refused when no scanner is configured, so the
		// scanner is optional here. A half-configured scanner is not.
		if strings.TrimSpace(c.DocumentScannerEndpoint) != "" || strings.TrimSpace(c.DocumentScannerToken) != "" {
			if strings.TrimSpace(c.DocumentScannerEndpoint) == "" || strings.TrimSpace(c.DocumentScannerToken) == "" {
				return errors.New("DOCUMENT_SCANNER_ENDPOINT and DOCUMENT_SCANNER_TOKEN must be set together")
			}
			if err := validateProductionURL("DOCUMENT_SCANNER_ENDPOINT", c.DocumentScannerEndpoint); err != nil {
				return err
			}
			if err := validateSecret("DOCUMENT_SCANNER_TOKEN", c.DocumentScannerToken); err != nil {
				return err
			}
		}

		if c.RealIdentity {
			if strings.TrimSpace(c.IdentityProvider) == "" || strings.Contains(strings.ToLower(c.IdentityProvider), "mock") || strings.TrimSpace(c.IdentityProviderEndpoint) == "" || strings.TrimSpace(c.IdentityProviderToken) == "" || strings.TrimSpace(c.IdentityWebhookSecret) == "" {
				return errors.New("FEATURE_REAL_IDENTITY requires a certified IDENTITY_PROVIDER and its endpoint, token, and webhook secret")
			}
			if err := validateProductionURL("IDENTITY_PROVIDER_ENDPOINT", c.IdentityProviderEndpoint); err != nil {
				return err
			}
			for name, value := range map[string]string{"IDENTITY_PROVIDER_TOKEN": c.IdentityProviderToken, "IDENTITY_WEBHOOK_SECRET": c.IdentityWebhookSecret} {
				if err := validateSecret(name, value); err != nil {
					return err
				}
			}
		}

		// Moving other people's money is the line that requires recorded
		// approvals and bounded exposure. Everything above can run without them.
		if c.RealCollections || c.MonoSweepEnabled {
			if strings.Contains(strings.ToLower(c.CollectionProvider), "mock") {
				return errors.New("FEATURE_REAL_COLLECTIONS requires a certified COLLECTION_PROVIDER")
			}
			if !c.MonoSweepEnabled {
				if strings.TrimSpace(c.CollectionProviderEndpoint) == "" || strings.TrimSpace(c.CollectionProviderToken) == "" || strings.TrimSpace(c.CollectionWebhookSecret) == "" {
					return errors.New("FEATURE_REAL_COLLECTIONS requires the collection connector endpoint, token, and webhook secret")
				}
				if err := validateProductionURL("COLLECTION_PROVIDER_ENDPOINT", c.CollectionProviderEndpoint); err != nil {
					return err
				}
				for name, value := range map[string]string{"COLLECTION_PROVIDER_TOKEN": c.CollectionProviderToken, "COLLECTION_WEBHOOK_SECRET": c.CollectionWebhookSecret} {
					if err := validateSecret(name, value); err != nil {
						return err
					}
				}
			}

			if !c.RealIdentity {
				return errors.New("FEATURE_REAL_COLLECTIONS requires FEATURE_REAL_IDENTITY; money must not move against an unverified party")
			}
			if !c.ProductionPilot {
				return errors.New("FEATURE_REAL_COLLECTIONS requires FEATURE_PRODUCTION_PILOT and its bounded limits")
			}
			for name, value := range map[string]string{"SECURITY_REVIEW_REFERENCE": c.SecurityReviewReference, "DPIA_REFERENCE": c.DPIAReference, "LEGAL_APPROVAL_REFERENCE": c.LegalApprovalReference, "PEN_TEST_REFERENCE": c.PenTestReference, "BACKUP_RESTORE_REFERENCE": c.BackupRestoreReference, "PROVIDER_CERTIFICATION_REFERENCE": c.ProviderCertificationReference, "SUPPORT_TRAINING_REFERENCE": c.SupportTrainingReference, "LAUNCH_APPROVAL_REFERENCE": c.LaunchApprovalReference, "PILOT_ALLOWED_PROVIDER_ACCOUNTS": c.PilotAllowedProviderAccounts, "PILOT_ALLOWED_INDUSTRIES": c.PilotAllowedIndustries} {
				if strings.TrimSpace(value) == "" {
					return fmt.Errorf("%s is required before live collections are enabled", name)
				}
			}
			for name, value := range map[string]int64{"PILOT_MAX_SUPPLIER_ORGANIZATIONS": c.PilotMaxSupplierOrganizations, "PILOT_MAX_BUYER_BUSINESSES": c.PilotMaxBuyerBusinesses, "PILOT_MAX_PRINCIPAL_KOBO": c.PilotMaxPrincipalKobo, "PILOT_MAX_ACTIVE_EXPOSURE_KOBO": c.PilotMaxActiveExposureKobo, "PILOT_MAX_DRAWDOWNS_PER_LINE_DAY": c.PilotMaxDrawdownsPerLineDay, "PILOT_MAX_COLLECTION_RETRIES": c.PilotMaxCollectionRetries, "PILOT_ENHANCED_REVIEW_KOBO": c.PilotEnhancedReviewKobo} {
				if value <= 0 {
					return fmt.Errorf("%s must be positive before live collections are enabled", name)
				}
			}
		}
	}
	return nil
}

func validateSecret(name, value string) error {
	if len(value) < 32 {
		return fmt.Errorf("%s must contain at least 32 bytes", name)
	}
	lower := strings.ToLower(strings.TrimSpace(value))
	for _, placeholder := range []string{"development-only", "change-me", "changeme", "replace-me", "example", "password", "minioadmin"} {
		if strings.Contains(lower, placeholder) {
			return fmt.Errorf("%s must not contain a placeholder value", name)
		}
	}
	runes := []rune(value)
	repeated := len(runes) > 0
	for _, current := range runes[1:] {
		if current != runes[0] {
			repeated = false
			break
		}
	}
	if repeated {
		return fmt.Errorf("%s must not be a repeated character", name)
	}
	return nil
}

func validateIdentifier(name, value string) error {
	lower := strings.ToLower(strings.TrimSpace(value))
	if lower == "" {
		return fmt.Errorf("%s must be set to a production value", name)
	}
	for _, placeholder := range []string{"development-only", "change-me", "changeme", "replace-me", "example", "minioadmin"} {
		if strings.Contains(lower, placeholder) {
			return fmt.Errorf("%s must not contain a placeholder value", name)
		}
	}
	return nil
}

func validateProductionURL(name, value string) error {
	parsed, err := url.Parse(value)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" || parsed.User != nil {
		return fmt.Errorf("%s must be an absolute URL", name)
	}
	host := strings.TrimSuffix(strings.ToLower(parsed.Hostname()), ".")
	ip := net.ParseIP(host)
	if host == "localhost" || (ip != nil && (ip.IsLoopback() || ip.IsUnspecified())) {
		return fmt.Errorf("%s must not point to a local host in production", name)
	}
	if parsed.Scheme != "https" {
		return fmt.Errorf("%s must use https in production", name)
	}
	return nil
}

func validateProductionDatabaseURL(name, value string) error {
	parsed, err := url.Parse(value)
	if err != nil || (parsed.Scheme != "postgres" && parsed.Scheme != "postgresql") || parsed.Host == "" {
		return fmt.Errorf("%s must be an absolute PostgreSQL URL", name)
	}
	host := strings.TrimSuffix(strings.ToLower(parsed.Hostname()), ".")
	ip := net.ParseIP(host)
	if host == "localhost" || (ip != nil && (ip.IsLoopback() || ip.IsUnspecified())) {
		return fmt.Errorf("%s must not point to a local host in production", name)
	}
	switch strings.ToLower(parsed.Query().Get("sslmode")) {
	case "require", "verify-ca", "verify-full":
		return nil
	default:
		return fmt.Errorf("%s must explicitly require TLS", name)
	}
}

// splitList reads a comma-separated setting into trimmed, non-empty entries.
// An unset or blank value yields nil rather than a single empty entry, so
// callers can distinguish "not configured" from "configured as empty".
func splitList(value string) []string {
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func int64Env(name string, fallback int64) (int64, error) {
	value := os.Getenv(name)
	if value == "" {
		return fallback, nil
	}
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil || parsed < 0 {
		if err == nil {
			err = errors.New("must be non-negative")
		}
		return 0, fmt.Errorf("%s %w", name, err)
	}
	return parsed, nil
}

func envOr(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}

func boolEnv(name string, fallback bool) (bool, error) {
	value := os.Getenv(name)
	if value == "" {
		return fallback, nil
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return false, fmt.Errorf("%s must be true or false: %w", name, err)
	}
	return parsed, nil
}
