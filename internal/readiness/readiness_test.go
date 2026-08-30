package readiness

import (
	"testing"

	"kredit/internal/config"
)

func TestEvaluateRequiresEvidenceAndPositivePilotLimits(t *testing.T) {
	report := Evaluate(config.Config{Environment: "development"})
	if report.Ready || len(report.Missing) == 0 {
		t.Fatalf("expected incomplete readiness report: %#v", report)
	}
	complete := config.Config{Environment: "staging", SecurityReviewReference: "sec-1", DPIAReference: "dpia-1", LegalApprovalReference: "legal-1", PenTestReference: "pentest-1", BackupRestoreReference: "restore-1", ProviderCertificationReference: "provider-1", SupportTrainingReference: "training-1", LaunchApprovalReference: "launch-1", PilotMaxSupplierOrganizations: 1, PilotMaxBuyerBusinesses: 1, PilotMaxPrincipalKobo: 100, PilotMaxActiveExposureKobo: 100, PilotMaxDrawdownsPerLineDay: 1, PilotMaxCollectionRetries: 1, PilotEnhancedReviewKobo: 100, PilotAllowedProviderAccounts: "approved", PilotAllowedIndustries: "pharmacy"}
	if report = Evaluate(complete); !report.Ready || len(report.Missing) != 0 {
		t.Fatalf("expected ready report: %#v", report)
	}
}

func TestProductionReadinessRequiresRealProviders(t *testing.T) {
	report := Evaluate(config.Config{Environment: "production", RealIdentity: true, RealCollections: true, CollectionProvider: "mock-collection"})
	for _, gate := range report.Gates {
		if gate.Name == "real_collection_provider" && gate.Passed {
			t.Fatal("mock collection provider must fail production readiness")
		}
	}
}
