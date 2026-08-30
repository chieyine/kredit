package readiness

import (
	"strings"
	"time"

	"kredit/internal/config"
)

type Gate struct {
	Name   string `json:"name"`
	Passed bool   `json:"passed"`
	Detail string `json:"detail"`
}

type PilotLimits struct {
	MaxSupplierOrganizations int64  `json:"max_supplier_organizations"`
	MaxBuyerBusinesses       int64  `json:"max_buyer_businesses"`
	MaxPrincipalKobo         int64  `json:"max_principal_kobo"`
	MaxActiveExposureKobo    int64  `json:"max_active_exposure_kobo"`
	MaxDrawdownsPerLineDay   int64  `json:"max_drawdowns_per_line_day"`
	MaxCollectionRetries     int64  `json:"max_collection_retries"`
	EnhancedReviewKobo       int64  `json:"enhanced_review_kobo"`
	AllowedProviderAccounts  string `json:"allowed_provider_accounts"`
	AllowedIndustries        string `json:"allowed_industries"`
}

type Report struct {
	GeneratedAt time.Time   `json:"generated_at"`
	Environment string      `json:"environment"`
	Ready       bool        `json:"ready"`
	Gates       []Gate      `json:"gates"`
	Missing     []string    `json:"missing"`
	PilotLimits PilotLimits `json:"pilot_limits"`
}

func Evaluate(cfg config.Config) Report {
	checks := []struct{ name, value, detail string }{
		{"security_review", cfg.SecurityReviewReference, "security review evidence"},
		{"dpia", cfg.DPIAReference, "DPIA evidence"},
		{"legal_approval", cfg.LegalApprovalReference, "legal approval evidence"},
		{"penetration_test", cfg.PenTestReference, "penetration test evidence"},
		{"backup_restore", cfg.BackupRestoreReference, "backup and restore drill evidence"},
		{"provider_certification", cfg.ProviderCertificationReference, "provider certification evidence"},
		{"support_training", cfg.SupportTrainingReference, "support training evidence"},
		{"launch_approval", cfg.LaunchApprovalReference, "launch approval evidence"},
		{"pilot_provider_accounts", cfg.PilotAllowedProviderAccounts, "permitted provider/account types"},
		{"pilot_industries", cfg.PilotAllowedIndustries, "permitted pilot industries"},
	}
	if cfg.Environment == "production" {
		identityGate := ""
		if cfg.RealIdentity {
			identityGate = "enabled"
		}
		collectionGate := ""
		if cfg.RealCollections && !strings.Contains(strings.ToLower(cfg.CollectionProvider), "mock") {
			collectionGate = "enabled"
		}
		checks = append(checks,
			struct{ name, value, detail string }{"real_identity_provider", identityGate, "certified identity provider enabled"},
			struct{ name, value, detail string }{"real_collection_provider", collectionGate, "certified collection provider enabled"},
		)
	}
	gates := make([]Gate, 0, len(checks)+1)
	missing := []string{}
	for _, check := range checks {
		passed := strings.TrimSpace(check.value) != ""
		gates = append(gates, Gate{Name: check.name, Passed: passed, Detail: check.detail})
		if !passed {
			missing = append(missing, check.name)
		}
	}
	limits := PilotLimits{MaxSupplierOrganizations: cfg.PilotMaxSupplierOrganizations, MaxBuyerBusinesses: cfg.PilotMaxBuyerBusinesses, MaxPrincipalKobo: cfg.PilotMaxPrincipalKobo, MaxActiveExposureKobo: cfg.PilotMaxActiveExposureKobo, MaxDrawdownsPerLineDay: cfg.PilotMaxDrawdownsPerLineDay, MaxCollectionRetries: cfg.PilotMaxCollectionRetries, EnhancedReviewKobo: cfg.PilotEnhancedReviewKobo, AllowedProviderAccounts: cfg.PilotAllowedProviderAccounts, AllowedIndustries: cfg.PilotAllowedIndustries}
	for _, limit := range []struct {
		name  string
		value int64
	}{{"pilot_max_supplier_organizations", limits.MaxSupplierOrganizations}, {"pilot_max_buyer_businesses", limits.MaxBuyerBusinesses}, {"pilot_max_principal_kobo", limits.MaxPrincipalKobo}, {"pilot_max_active_exposure_kobo", limits.MaxActiveExposureKobo}, {"pilot_max_drawdowns_per_line_day", limits.MaxDrawdownsPerLineDay}, {"pilot_max_collection_retries", limits.MaxCollectionRetries}, {"pilot_enhanced_review_kobo", limits.EnhancedReviewKobo}} {
		passed := limit.value > 0
		gates = append(gates, Gate{Name: limit.name, Passed: passed, Detail: "positive configurable pilot limit"})
		if !passed {
			missing = append(missing, limit.name)
		}
	}
	return Report{GeneratedAt: time.Now().UTC(), Environment: cfg.Environment, Ready: len(missing) == 0, Gates: gates, Missing: missing, PilotLimits: limits}
}
