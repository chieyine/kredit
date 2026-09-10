package paymentclaims

import (
	"context"
	"fmt"
	"math"
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
	if _, err := store.Decide(context.Background(), expiring.ID, "supplier-1", Rejected, "Transfer could not be verified", ""); err != nil {
		t.Fatal("expired collection hold prevented claim review", err)
	}
}

func TestClaimHoldsCannotOverflowAndReturnedReviewsCannotChangeHistory(t *testing.T) {
	ctx := context.Background()
	store := NewStore(func(id string) (ObligationSnapshot, error) {
		return ObligationSnapshot{ID: id, BuyerUserID: "buyer", OutstandingKobo: ledger.Money(math.MaxInt64), Currency: "NGN"}, nil
	})
	var claim Claim
	for i := 0; i < 2; i++ {
		var err error
		claim, err = store.Create(ctx, CreateInput{ObligationID: "debt", BuyerUserID: "buyer", AmountKobo: ledger.Money(math.MaxInt64), TransferReference: fmt.Sprint(i), IdempotencyKey: fmt.Sprint(i)})
		if err != nil {
			t.Fatal(err)
		}
	}
	if hold := store.ActiveHold(ctx, "debt", time.Now()); hold != ledger.Money(math.MaxInt64) {
		t.Fatalf("overflowed hold: %d", hold)
	}
	decided, err := store.Decide(ctx, claim.ID, "reviewer", Rejected, "Transfer could not be verified", "")
	if err != nil {
		t.Fatal(err)
	}
	original := *decided.ReviewedAt
	*decided.ReviewedAt = time.Time{}
	loaded, err := store.Get(ctx, claim.ID)
	if err != nil || !loaded.ReviewedAt.Equal(original) {
		t.Fatalf("external mutation changed review: %+v %v", loaded, err)
	}
	*loaded.ReviewedAt = time.Time{}
	again := store.ListForObligation(ctx, "debt")
	for _, item := range again {
		if item.ID == claim.ID && !item.ReviewedAt.Equal(original) {
			t.Fatal("read exposed mutable review")
		}
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
