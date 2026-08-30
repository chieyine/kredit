package disputes

import (
	"testing"

	"kredit/internal/ledger"
)

func TestPartialDisputeBlocksContestedAmountAndRecordsAdjustment(t *testing.T) {
	snapshot := ObligationSnapshot{OutstandingKobo: 1000000, SupplierOrganizationID: "org", BuyerUserID: "buyer"}
	apply := func(_ string, amount ledger.Money) error { snapshot.OutstandingKobo -= amount; return nil }
	store := NewStore(func(_ string) (ObligationSnapshot, error) { return snapshot, nil }, ledger.NewStore(), apply)
	dispute, err := store.Open(OpenInput{ObligationID: "obl", OpenedBy: "buyer", DisputedAmountKobo: 200000, Reason: "wrong quantity", CollectionEffect: EffectContestedOnly})
	if err != nil {
		t.Fatal(err)
	}
	blocked, err := store.BlockedAmount("obl")
	if err != nil || blocked != 200000 {
		t.Fatalf("blocked=%d err=%v", blocked, err)
	}
	if _, err := store.AddEvidence(dispute.ID, "buyer", "doc-1", "delivery note differs"); err != nil {
		t.Fatal(err)
	}
	updated, decision, err := store.Decide(DecideInput{DisputeID: dispute.ID, ReviewerID: "reviewer", Outcome: "partial", ValidPrincipalKobo: 100000, AdjustmentKobo: 100000, RemainingDisputedKobo: 100000, Reason: "half accepted"})
	if err != nil {
		t.Fatal(err)
	}
	if decision.AdjustmentKobo != 100000 || updated.RemainingDisputedKobo != 100000 || snapshot.OutstandingKobo != 900000 {
		t.Fatalf("updated=%+v decision=%+v outstanding=%d", updated, decision, snapshot.OutstandingKobo)
	}
	blocked, _ = store.BlockedAmount("obl")
	if blocked != 100000 {
		t.Fatalf("blocked after decision=%d", blocked)
	}
}

func TestFullBlockStopsUntilDecision(t *testing.T) {
	snapshot := ObligationSnapshot{OutstandingKobo: 500}
	store := NewStore(func(_ string) (ObligationSnapshot, error) { return snapshot, nil }, ledger.NewStore(), func(string, ledger.Money) error { return nil })
	if _, err := store.Open(OpenInput{ObligationID: "obl", OpenedBy: "buyer", DisputedAmountKobo: 100, Reason: "not received", CollectionEffect: EffectFullBlock}); err != nil {
		t.Fatal(err)
	}
	blocked, _ := store.BlockedAmount("obl")
	if blocked != 500 {
		t.Fatalf("blocked=%d", blocked)
	}
}
