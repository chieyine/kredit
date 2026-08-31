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
	DocumentScannerEndpoint        string
	DocumentScannerToken           string
	SessionSigningKey              string
	FieldEncryptionKeyID           string
	OTPHMACKey                     string
	TokenHashKey                   string
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
		Environment:                    strings.ToLower(strings.TrimSpace(envOr("APP_ENV", "development"))),
		Version:                        envOr("APP_VERSION", "0.1.0-dev"),
		PublicBaseURL:                  envOr("PUBLIC_BASE_URL", "http://localhost:5173"),
		AppBaseURL:                     envOr("APP_BASE_URL", "http://localhost:5173"),
		APIInternalURL:                 envOr("API_INTERNAL_URL", "http://localhost:8080"),
		APIListenAddr:                  envOr("API_ADDR", ":8080"),
		DatabaseURL:                    envOr("DATABASE_URL", "postgres://kredit:kredit@localhost:5432/kredit?sslmode=disable"),
		DatabaseDirectURL:              envOr("DATABASE_DIRECT_URL", "postgres://kredit:kredit@localhost:5432/kredit?sslmode=disable"),
		RiverDatabaseURL:               envOr("RIVER_DATABASE_URL", "postgres://kredit:kredit@localhost:5432/kredit?sslmode=disable"),
		ObjectStorageEndpoint:          envOr("OBJECT_STORAGE_ENDPOINT", "http://localhost:9000"),
		ObjectStorageBucket:            envOr("OBJECT_STORAGE_BUCKET", "kredit-local"),
		ObjectStorageRegion:            envOr("OBJECT_STORAGE_REGION", "us-east-1"),
		ObjectStorageAccessKey:         envOr("OBJECT_STORAGE_ACCESS_KEY", "minioadmin"),
		ObjectStorageSecretKey:         envOr("OBJECT_STORAGE_SECRET_KEY", "minioadmin"),
		DocumentScannerEndpoint:        envOr("DOCUMENT_SCANNER_ENDPOINT", ""),
		DocumentScannerToken:           envOr("DOCUMENT_SCANNER_TOKEN", ""),
		SessionSigningKey:              envOr("SESSION_SIGNING_KEY", "development-only-change-me"),
		FieldEncryptionKeyID:           envOr("FIELD_ENCRYPTION_KEY_ID", "development-only"),
		OTPHMACKey:                     envOr("OTP_HMAC_KEY", "development-only-change-me"),
		TokenHashKey:                   envOr("TOKEN_HASH_KEY", "development-only-change-me"),
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
	if c.Currency != "NGN" || c.MoneyUnit != "kobo" {
		return errors.New("money configuration must remain NGN/kobo")
	}
	if strings.TrimSpace(c.CollectionProvider) == "" {
		return errors.New("COLLECTION_PROVIDER is required")
	}
	if c.RealCollections {
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
	if c.Environment == "production" {
		if !c.ApprovedRetentionPolicy {
			return errors.New("FEATURE_APPROVED_RETENTION_POLICY must be enabled in production")
		}
		if !c.ProductionPilot {
			return errors.New("FEATURE_PRODUCTION_PILOT must be enabled in production")
		}
		if !c.RealIdentity {
			return errors.New("FEATURE_REAL_IDENTITY must be enabled in production; mock identity is not permitted")
		}
		if strings.TrimSpace(c.IdentityProvider) == "" || strings.Contains(strings.ToLower(c.IdentityProvider), "mock") || strings.TrimSpace(c.IdentityProviderEndpoint) == "" || strings.TrimSpace(c.IdentityProviderToken) == "" || strings.TrimSpace(c.IdentityWebhookSecret) == "" {
			return errors.New("production requires a certified IDENTITY_PROVIDER and configured connector endpoint, token, and webhook secret")
		}
		if !c.RealCollections {
			return errors.New("FEATURE_REAL_COLLECTIONS must be enabled in production; collection money paths are disabled")
		}
		if strings.Contains(strings.ToLower(c.CollectionProvider), "mock") {
			return errors.New("COLLECTION_PROVIDER must name a certified production provider")
		}
		if strings.TrimSpace(c.CollectionProviderEndpoint) == "" || strings.TrimSpace(c.CollectionProviderToken) == "" || strings.TrimSpace(c.CollectionWebhookSecret) == "" {
			return errors.New("production requires collection connector endpoint, token, webhook secret, and PROVIDER_APPROVED_AT")
		}
		for name, value := range map[string]string{
			"SESSION_SIGNING_KEY":         c.SessionSigningKey,
			"OTP_HMAC_KEY":                c.OTPHMACKey,
			"TOKEN_HASH_KEY":              c.TokenHashKey,
			"DATABASE_URL":                c.DatabaseURL,
			"DATABASE_DIRECT_URL":         c.DatabaseDirectURL,
			"RIVER_DATABASE_URL":          c.RiverDatabaseURL,
			"OBJECT_STORAGE_ENDPOINT":     c.ObjectStorageEndpoint,
			"OBJECT_STORAGE_BUCKET":       c.ObjectStorageBucket,
			"OBJECT_STORAGE_REGION":       c.ObjectStorageRegion,
			"OBJECT_STORAGE_ACCESS_KEY":   c.ObjectStorageAccessKey,
			"OBJECT_STORAGE_SECRET_KEY":   c.ObjectStorageSecretKey,
			"DOCUMENT_SCANNER_ENDPOINT":   c.DocumentScannerEndpoint,
			"DOCUMENT_SCANNER_TOKEN":      c.DocumentScannerToken,
			"FIELD_ENCRYPTION_KEY_ID":     c.FieldEncryptionKeyID,
			"OTEL_EXPORTER_OTLP_ENDPOINT": c.OTelEndpoint,
			"NOTIFICATION_EMAIL_ENDPOINT": c.NotificationEmailEndpoint,
			"NOTIFICATION_EMAIL_TOKEN":    c.NotificationEmailToken,
			"NOTIFICATION_SMS_ENDPOINT":   c.NotificationSMSEndpoint,
			"NOTIFICATION_SMS_TOKEN":      c.NotificationSMSToken,
		} {
			if strings.TrimSpace(value) == "" {
				return fmt.Errorf("%s must be set to a production value", name)
			}
		}
		if c.WhatsApp && (strings.TrimSpace(c.NotificationWhatsAppEndpoint) == "" || strings.TrimSpace(c.NotificationWhatsAppToken) == "") {
			return errors.New("FEATURE_WHATSAPP requires NOTIFICATION_WHATSAPP_ENDPOINT and NOTIFICATION_WHATSAPP_TOKEN in production")
		}
		for name, value := range map[string]string{"SESSION_SIGNING_KEY": c.SessionSigningKey, "OTP_HMAC_KEY": c.OTPHMACKey, "TOKEN_HASH_KEY": c.TokenHashKey, "OBJECT_STORAGE_SECRET_KEY": c.ObjectStorageSecretKey, "DOCUMENT_SCANNER_TOKEN": c.DocumentScannerToken, "NOTIFICATION_EMAIL_TOKEN": c.NotificationEmailToken, "NOTIFICATION_SMS_TOKEN": c.NotificationSMSToken, "IDENTITY_PROVIDER_TOKEN": c.IdentityProviderToken, "IDENTITY_WEBHOOK_SECRET": c.IdentityWebhookSecret, "COLLECTION_PROVIDER_TOKEN": c.CollectionProviderToken, "COLLECTION_WEBHOOK_SECRET": c.CollectionWebhookSecret} {
			if err := validateSecret(name, value); err != nil {
				return err
			}
		}
		if c.WhatsApp {
			if err := validateSecret("NOTIFICATION_WHATSAPP_TOKEN", c.NotificationWhatsAppToken); err != nil {
				return err
			}
		}
		for name, value := range map[string]string{"FIELD_ENCRYPTION_KEY_ID": c.FieldEncryptionKeyID, "OBJECT_STORAGE_ACCESS_KEY": c.ObjectStorageAccessKey} {
			if err := validateIdentifier(name, value); err != nil {
				return err
			}
		}
		for name, value := range map[string]string{"PUBLIC_BASE_URL": c.PublicBaseURL, "APP_BASE_URL": c.AppBaseURL, "API_INTERNAL_URL": c.APIInternalURL, "OBJECT_STORAGE_ENDPOINT": c.ObjectStorageEndpoint, "DOCUMENT_SCANNER_ENDPOINT": c.DocumentScannerEndpoint, "OTEL_EXPORTER_OTLP_ENDPOINT": c.OTelEndpoint, "NOTIFICATION_EMAIL_ENDPOINT": c.NotificationEmailEndpoint, "NOTIFICATION_SMS_ENDPOINT": c.NotificationSMSEndpoint, "IDENTITY_PROVIDER_ENDPOINT": c.IdentityProviderEndpoint, "COLLECTION_PROVIDER_ENDPOINT": c.CollectionProviderEndpoint} {
			if err := validateProductionURL(name, value); err != nil {
				return err
			}
		}
		if c.WhatsApp {
			if err := validateProductionURL("NOTIFICATION_WHATSAPP_ENDPOINT", c.NotificationWhatsAppEndpoint); err != nil {
				return err
			}
		}
		for name, value := range map[string]string{"DATABASE_URL": c.DatabaseURL, "DATABASE_DIRECT_URL": c.DatabaseDirectURL, "RIVER_DATABASE_URL": c.RiverDatabaseURL} {
			if err := validateProductionDatabaseURL(name, value); err != nil {
				return err
			}
		}
		for name, value := range map[string]string{"SECURITY_REVIEW_REFERENCE": c.SecurityReviewReference, "DPIA_REFERENCE": c.DPIAReference, "LEGAL_APPROVAL_REFERENCE": c.LegalApprovalReference, "PEN_TEST_REFERENCE": c.PenTestReference, "BACKUP_RESTORE_REFERENCE": c.BackupRestoreReference, "PROVIDER_CERTIFICATION_REFERENCE": c.ProviderCertificationReference, "SUPPORT_TRAINING_REFERENCE": c.SupportTrainingReference, "LAUNCH_APPROVAL_REFERENCE": c.LaunchApprovalReference, "PILOT_ALLOWED_PROVIDER_ACCOUNTS": c.PilotAllowedProviderAccounts, "PILOT_ALLOWED_INDUSTRIES": c.PilotAllowedIndustries} {
			if strings.TrimSpace(value) == "" {
				return fmt.Errorf("%s is required for production readiness", name)
			}
		}
		for name, value := range map[string]int64{"PILOT_MAX_SUPPLIER_ORGANIZATIONS": c.PilotMaxSupplierOrganizations, "PILOT_MAX_BUYER_BUSINESSES": c.PilotMaxBuyerBusinesses, "PILOT_MAX_PRINCIPAL_KOBO": c.PilotMaxPrincipalKobo, "PILOT_MAX_ACTIVE_EXPOSURE_KOBO": c.PilotMaxActiveExposureKobo, "PILOT_MAX_DRAWDOWNS_PER_LINE_DAY": c.PilotMaxDrawdownsPerLineDay, "PILOT_MAX_COLLECTION_RETRIES": c.PilotMaxCollectionRetries, "PILOT_ENHANCED_REVIEW_KOBO": c.PilotEnhancedReviewKobo} {
			if value <= 0 {
				return fmt.Errorf("%s must be positive for production readiness", name)
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
