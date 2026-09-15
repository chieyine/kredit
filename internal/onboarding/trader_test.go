package onboarding

import (
	"testing"
	"time"
)

func TestTraderBankNameAndVerifiedOwnerCannotBeSubstituted(t *testing.T) {
	s := NewStore()
	p, _ := s.Ensure("trader", "owner", true, true)
	p, _, _ = s.UpdateRepresentative("trader", "owner", RepresentativeInput{ExpectedVersion: p.Version, Name: "Ada Okafor", Title: "Owner"})
	p, _, _ = s.SubmitKYB("trader", "owner", "person-reference", p.Version)
	p, _, err := s.RecordKYBDecision("trader", "owner", "approved", "owner_identity_approved", time.Now().Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err = s.UpdateRepresentative("trader", "owner", RepresentativeInput{ExpectedVersion: p.Version, Name: "Someone Else", Title: "Owner"}); err == nil {
		t.Fatal("verified owner replaced")
	}
	for _, name := range []string{"Someone Else", "Okafor Ada"} {
		p = snapshotTraderBank(t, s, p, name)
		_, sum, _ := s.Get("trader")
		for _, r := range sum.Requirements {
			if r.Code == "settlement_verified" && r.Complete != (name == "Okafor Ada") {
				t.Fatalf("bank name %q readiness incorrect", name)
			}
		}
	}
}
func snapshotTraderBank(t *testing.T, s *Store, p Profile, name string) Profile {
	t.Helper()
	p, _, err := s.UpdateSettlement("trader", "owner", SettlementInput{ExpectedVersion: p.Version, Provider: "bank", ProviderReference: "destination", BankName: "Bank", AccountName: name, AccountLast4: "1234"})
	if err != nil {
		t.Fatal(err)
	}
	p, _, err = s.RecordSettlementDecision("trader", "provider", "verified", "")
	if err != nil {
		t.Fatal(err)
	}
	return p
}
