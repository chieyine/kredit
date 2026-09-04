package paymentclaims

import (
	"context"
	"testing"
	"time"

	"kredit/internal/ledger"
)

func TestClaimHoldExpiresAndDecisionRequiresPayment(t *testing.T) {
	now := time.Date(2026, 8, 21, 12, 0, 0, 0, time.UTC)
	store := NewStore(func(string) (ObligationSnapshot, error) {
		return ObligationSnapshot{ID: "obligation-1", BuyerUserID: "buyer-1", SupplierOrganizationID: "org-1", OutstandingKobo: 100_000, Currency: "NGN"}, nil
	})
	store.now = func() time.Time { return now }
	claim, err := store.Create(context.Background(), CreateInput{ObligationID: "obligation-1", BuyerUserID: "buyer-1", AmountKobo: ledger.Money(40_000), TransferReference: "bank-1", IdempotencyKey: "claim-1"})
	if err != nil {
		t.Fatal(err)
	}
	if got := store.ActiveHold(context.Background(), "obligation-1", now); got != 40_000 {
		t.Fatalf("hold=%d", got)
	}
	if _, err = store.Decide(context.Background(), claim.ID, "supplier-1", Confirmed, "matched bank statement", ""); err == nil {
		t.Fatal("confirmation without recognized payment must fail")
	}
	confirmed, err := store.Decide(context.Background(), claim.ID, "supplier-1", Confirmed, "matched bank statement", "payment-1")
	if err != nil || confirmed.PaymentID != "payment-1" {
		t.Fatalf("confirmed=%+v err=%v", confirmed, err)
	}
	if got := store.ActiveHold(context.Background(), "obligation-1", now); got != 0 {
		t.Fatalf("confirmed claim hold=%d", got)
	}

	expiring, err := store.Create(context.Background(), CreateInput{ObligationID: "obligation-1", BuyerUserID: "buyer-1", AmountKobo: 10_000, TransferReference: "bank-2", IdempotencyKey: "claim-2"})
	if err != nil {
		t.Fatal(err)
	}
	if got := store.ActiveHold(context.Background(), "obligation-1", expiring.HoldExpiresAt); got != 0 {
		t.Fatalf("expired hold=%d", got)
	}
}

func TestClaimRetryRejectsChangedOwnerAndIntent(t *testing.T) {
	store := NewStore(func(id string) (ObligationSnapshot, error) {
		return ObligationSnapshot{ID: id, BuyerUserID: "buyer", OutstandingKobo: 10000, Currency: "NGN"}, nil
	})
	input := CreateInput{ObligationID: "debt", BuyerUserID: "buyer", AmountKobo: 2500, TransferReference: "bank", IdempotencyKey: "key"}
	first, err := store.Create(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	retry, err := store.Create(context.Background(), input)
	if err != nil || retry.ID != first.ID {
		t.Fatalf("retry=%+v %v", retry, err)
	}
	for _, mutate := range []func(*CreateInput){func(i *CreateInput) { i.AmountKobo++ }, func(i *CreateInput) { i.BuyerUserID = "other" }, func(i *CreateInput) { i.ObligationID = "other" }, func(i *CreateInput) { i.TransferReference = "other" }} {
		altered := input
		mutate(&altered)
		if _, err := store.Create(context.Background(), altered); err == nil {
			t.Fatal("changed claim accepted")
		}
	}
}
