package ledger

import (
	"context"
	"math"
	"testing"
	"time"
)

func TestDSATransactionValidatesSignedEventIntent(t *testing.T) {
	at := time.Date(2026, 9, 21, 0, 0, 0, 0, time.UTC)
	for _, tc := range []struct {
		name   string
		kind   string
		amount int64
		debit  string
		credit string
	}{
		{"earning", "earning", 100, accountDSAAcquisitionExpense, accountDSACommissionPayable},
		{"reversal", "earning", -100, accountDSACommissionPayable, accountDSAAcquisitionExpense},
		{"payout", "payout", 100, accountDSACommissionPayable, accountDSAPayoutCash},
		{"maximum earning", "earning", math.MaxInt64, accountDSAAcquisitionExpense, accountDSACommissionPayable},
		{"maximum reversal", "earning", -math.MaxInt64, accountDSACommissionPayable, accountDSAAcquisitionExpense},
	} {
		t.Run(tc.name, func(t *testing.T) {
			transaction, err := dsaTransaction("referral-reference", tc.kind, tc.amount, at)
			if err != nil {
				t.Fatal(err)
			}
			if err := validateTransaction(transaction); err != nil {
				t.Fatalf("invalid double-entry journal: %v", err)
			}
			magnitude := tc.amount
			if magnitude < 0 {
				magnitude = -magnitude
			}
			if len(transaction.Postings) != 2 || transaction.Postings[0] != (Posting{Account: tc.debit, Debit: Money(magnitude)}) || transaction.Postings[1] != (Posting{Account: tc.credit, Credit: Money(magnitude)}) {
				t.Fatalf("incorrect posting direction: %+v", transaction.Postings)
			}
			if transaction.ReferenceID != "referral-reference" || transaction.ReferenceType != "dsa_"+tc.kind || transaction.EventType != "dsa_"+tc.kind || transaction.IdempotencyKey != "dsa:"+tc.kind+":referral-reference" || !transaction.EffectiveAt.Equal(at) {
				t.Fatalf("journal identity changed: %+v", transaction)
			}
		})
	}
}

func TestDSATransactionRejectsInvalidEventsBeforePosting(t *testing.T) {
	for _, tc := range []struct {
		id     string
		kind   string
		amount int64
	}{
		{"", "earning", 1},
		{" reference", "earning", 1},
		{"reference ", "earning", 1},
		{"reference", "", 1},
		{"reference", "pyaout", 1},
		{"reference", "earning", 0},
		{"reference", "earning", math.MinInt64},
		{"reference", "payout", -1},
		{"reference", "payout", math.MinInt64},
	} {
		if _, err := dsaTransaction(tc.id, tc.kind, tc.amount, time.Time{}); err == nil {
			t.Errorf("accepted invalid event: %+v", tc)
		}
	}
	if err := NewPostgresStore(nil).PostDSATx(context.Background(), nil, "reference", "earning", 1, time.Time{}); err == nil {
		t.Fatal("accepted a missing transaction")
	}
}
