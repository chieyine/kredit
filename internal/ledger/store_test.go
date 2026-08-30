package ledger

import (
	"math"
	"testing"
	"time"
)

func TestActivationPostsBalancedPrincipalAndBaseFee(t *testing.T) {
	store := NewStore()
	transaction, err := store.PostActivation("obligation-1", Money(120000000), now(), "obligation-1:activation")
	if err != nil {
		t.Fatal(err)
	}
	if len(transaction.Postings) != 4 {
		t.Fatalf("expected principal and fee postings, got %d", len(transaction.Postings))
	}
	var debit, credit Money
	for _, posting := range transaction.Postings {
		debit += posting.Debit
		credit += posting.Credit
	}
	if debit != credit || debit != 120600000 {
		t.Fatalf("unexpected balance: debit=%d credit=%d", debit, credit)
	}
	retry, err := store.PostActivation("obligation-1", Money(120000000), now(), "obligation-1:activation")
	if err != nil {
		t.Fatal(err)
	}
	if retry.ID != transaction.ID {
		t.Fatal("idempotent activation should return the original transaction")
	}
}

func TestLedgerRejectsChangedIntentForIdempotencyKey(t *testing.T) {
	store := NewStore()
	if _, err := store.PostPayment("payment-1", 100, "cash_recorded", now(), "payment-key"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.PostPayment("payment-1", 101, "cash_recorded", now(), "payment-key"); err == nil {
		t.Fatal("expected changed amount to be rejected")
	}
	if _, err := store.PostPayment("payment-2", 100, "cash_recorded", now(), "payment-key"); err == nil {
		t.Fatal("expected changed reference to be rejected")
	}
}

func TestMoneyArithmeticRejectsOverflow(t *testing.T) {
	if _, err := CheckedAdd(Money(math.MaxInt64), 1); err == nil {
		t.Fatal("expected positive overflow to be rejected")
	}
	if _, err := CheckedAdd(Money(math.MinInt64), -1); err == nil {
		t.Fatal("expected negative overflow to be rejected")
	}
	err := validateTransaction(Transaction{EventType: "overflow", ReferenceID: "ref", IdempotencyKey: "key", Postings: []Posting{
		{Account: "a", Debit: Money(math.MaxInt64)}, {Account: "b", Debit: 1},
		{Account: "c", Credit: Money(math.MaxInt64)}, {Account: "d", Credit: 1},
	}})
	if err == nil {
		t.Fatal("expected overflowing journal totals to be rejected")
	}
}

func now() time.Time { return time.Now().UTC() }
