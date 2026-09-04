package operations

import (
	"testing"

	"kredit/internal/ledger"
)

func TestWriteOffRequiresApprovalAtHighValueAndReducesBalance(t *testing.T) {
	outstanding := ledger.Money(2000000)
	store := NewStore(ledger.NewStore(), func(_ string, amount ledger.Money) error { outstanding -= amount; return nil })
	if _, err := store.WriteOff("actor", "org", "obl", 1000000, "approved adjustment", ""); err == nil {
		t.Fatal("expected approval requirement")
	}
	action, err := store.WriteOff("actor", "org", "obl", 500000, "approved adjustment", "")
	if err != nil {
		t.Fatal(err)
	}
	if action.ActionType != "write_off" || outstanding != 1500000 {
		t.Fatalf("action=%+v outstanding=%d", action, outstanding)
	}
}
func TestFeeWaiverIsAudited(t *testing.T) {
	store := NewStore(ledger.NewStore(), nil)
	action, err := store.WaiveFee("actor", "org", "obl", 1000, "service recovery", "")
	if err != nil {
		t.Fatal(err)
	}
	if action.ActionType != "fee_waiver" || action.LedgerTransactionID == "" {
		t.Fatalf("action=%+v", action)
	}
}

func TestClaimedApprovalCannotAuthorizeHighValueOperation(t *testing.T) {
	store := NewStore(ledger.NewStore(), func(string, ledger.Money) error { t.Fatal("unapproved write-off applied"); return nil })
	for _, amount := range []ledger.Money{1000, 1000000} {
		if _, err := store.WriteOff("actor", "org", "obl", amount, "claimed approval", "someone-else"); err == nil {
			t.Fatal("unverified approval accepted")
		}
	}
}
