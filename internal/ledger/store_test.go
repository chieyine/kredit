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

func TestVerifyChain(t *testing.T) {
	tx1 := Transaction{
		ID:             "tx-1",
		EventType:      "principal_activated",
		ReferenceType:  "obligation",
		ReferenceID:    "ob-1",
		IdempotencyKey: "key-1",
		EffectiveAt:    time.Unix(1000, 0),
		Postings: []Posting{
			{Account: AccountTradeReceivable, Debit: 1000},
			{Account: AccountPrincipalOriginated, Credit: 1000},
		},
	}
	tx2 := Transaction{
		ID:             "tx-2",
		EventType:      "payment_recognized",
		ReferenceType:  "payment",
		ReferenceID:    "pay-1",
		IdempotencyKey: "key-2",
		EffectiveAt:    time.Unix(2000, 0),
		Postings: []Posting{
			{Account: AccountVoluntarySettlement, Debit: 500},
			{Account: AccountTradeReceivable, Credit: 500},
		},
	}
	hash1 := VerifyChain([]Transaction{tx1, tx2})
	if hash1 == "" {
		t.Fatal("expected non-empty chain hash")
	}
	hash2 := VerifyChain([]Transaction{tx1, tx2})
	if hash1 != hash2 {
		t.Fatal("expected deterministic hash chain")
	}

	// Tampered posting amount must change hash
	tx2Tampered := tx2
	tx2Tampered.Postings = []Posting{
		{Account: AccountVoluntarySettlement, Debit: 600},
		{Account: AccountTradeReceivable, Credit: 600},
	}
	hashTampered := VerifyChain([]Transaction{tx1, tx2Tampered})
	if hash1 == hashTampered {
		t.Fatal("expected tampered posting to change hash")
	}
}

func now() time.Time { return time.Now().UTC() }
