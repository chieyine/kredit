package onboarding

import (
	"strings"
	"testing"
	"time"
)

func TestReadinessIsDerivedAndProviderExpiryRevokesIt(t *testing.T) {
	s := NewStore()
	now := time.Date(2026, 8, 29, 10, 0, 0, 0, time.UTC)
	s.now = func() time.Time { return now }
	p, err := s.Ensure("org-1", "owner-1", true, false)
	if err != nil {
		t.Fatal(err)
	}
	if _, summary, _ := s.Get("org-1"); summary.Ready || len(summary.Missing) != 8 {
		t.Fatalf("unexpected initial readiness: %#v", summary)
	}
	p, _, err = s.RecordContactVerified("org-1", "owner-1", "phone")
	if err != nil {
		t.Fatal(err)
	}
	p, _, err = s.UpdateRepresentative("org-1", "owner-1", RepresentativeInput{ExpectedVersion: p.Version, Name: "Ada Okafor", Title: "Director"})
	if err != nil {
		t.Fatal(err)
	}
	p, _, err = s.SubmitKYB("org-1", "owner-1", "kyb-ref", p.Version)
	if err != nil {
		t.Fatal(err)
	}
	p, _, err = s.RecordKYBDecision("org-1", "provider", "approved", "", now.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	p, _, err = s.UpdateSettlement("org-1", "owner-1", SettlementInput{ExpectedVersion: p.Version, Provider: "provider", ProviderReference: "destination-ref", BankName: "Bank", AccountName: "ABC Ltd", AccountLast4: "1234"})
	if err != nil {
		t.Fatal(err)
	}
	p, _, err = s.RecordSettlementDecision("org-1", "provider", "verified", "")
	if err != nil {
		t.Fatal(err)
	}
	p, _, err = s.UpdateBilling("org-1", "owner-1", BillingInput{ExpectedVersion: p.Version, Method: "split_settlement", ProviderReference: "billing-ref", Cycle: "per_settlement"})
	if err != nil {
		t.Fatal(err)
	}
	p, _, err = s.UpdateCreditPolicy("org-1", "owner-1", CreditPolicyInput{ExpectedVersion: p.Version, CreditLimitKobo: 500_000_000, PaymentDays: 30, GraceHours: 48})
	if err != nil {
		t.Fatal(err)
	}
	p, _, err = s.AcceptConsents("org-1", "owner-1", p.Version, CurrentTermsVersion, CurrentPrivacyVersion)
	if err != nil {
		t.Fatal(err)
	}
	p, summary, err := s.SyncSecurity("org-1", "owner-1", true, true)
	if err != nil {
		t.Fatal(err)
	}
	if !summary.Ready || p.ReadinessState != StatePilotReady {
		t.Fatalf("expected pilot ready, got %#v", summary)
	}
	now = now.Add(2 * time.Hour)
	changed := s.Reconcile(now)
	if len(changed) != 1 || changed[0].KYBState != "expired" || changed[0].ReadinessState != "expired" {
		t.Fatalf("expected durable expiry transition, got %#v", changed)
	}
}

func TestProviderStatesVersionConflictsAndRestrictedSettlementInput(t *testing.T) {
	s := NewStore()
	p, _ := s.Ensure("org", "owner", true, true)
	if _, _, err := s.UpdateRepresentative("org", "owner", RepresentativeInput{ExpectedVersion: p.Version + 1, Name: "Ada", Title: "Owner"}); err == nil || !strings.Contains(err.Error(), "version conflict") {
		t.Fatalf("expected optimistic conflict, got %v", err)
	}
	if _, _, err := s.UpdateSettlement("org", "owner", SettlementInput{ExpectedVersion: p.Version, Provider: "bank", ProviderReference: "ref", AccountLast4: "1234567890"}); err == nil {
		t.Fatal("full account number must not be accepted")
	}
	p, _, _ = s.SubmitKYB("org", "owner", "provider-ref", p.Version)
	p, summary, err := s.RecordKYBDecision("org", "provider", "rejected", "registration_mismatch", time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	if p.KYBState != "rejected" || summary.State != "rejected" || p.KYBReasonCode != "registration_mismatch" {
		t.Fatalf("rejection evidence missing: %#v %#v", p, summary)
	}
	p, _, err = s.SubmitKYB("org", "owner", "provider-ref-2", p.Version)
	if err != nil {
		t.Fatal(err)
	}
	if p.KYBState != "submitted" || p.KYBReasonCode != "" {
		t.Fatalf("resubmission did not reset review state: %#v", p)
	}
}
