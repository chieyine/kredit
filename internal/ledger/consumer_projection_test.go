package ledger

import (
	"context"
	"reflect"
	"testing"
	"time"
)

func TestConsumerProjectionPreservesCashReceivableAndCustomerFunds(t *testing.T) {
	balances := map[string]Money{}
	apply := func(action string, amount Money, released bool, price, net Money) {
		t.Helper()
		postings, err := consumerPostings(action, amount, released, price, net)
		if err != nil {
			t.Fatal(err)
		}
		if err := validateTransaction(Transaction{EventType: "consumer_" + action, ReferenceType: "consumer_event", ReferenceID: "event", IdempotencyKey: "consumer:event", Postings: postings}); err != nil {
			t.Fatalf("unbalanced %s: %v", action, err)
		}
		for _, p := range postings {
			balances[p.Account] += p.Debit - p.Credit
		}
	}
	assert := func(cash, receivable, funds, sales Money) {
		t.Helper()
		want := map[string]Money{"CONSUMER_PAYMENT_CONTROL": cash, "CONSUMER_RECEIVABLE": receivable, "CONSUMER_CUSTOMER_FUNDS": -funds, "CONSUMER_SALES_CONTROL": -sales}
		for account, amount := range want {
			if balances[account] != amount {
				t.Errorf("%s balance = %d, want %d", account, balances[account], amount)
			}
		}
		for account := range balances {
			if _, ok := want[account]; !ok {
				t.Errorf("consumer event touched non-consumer account %s", account)
			}
		}
	}
	apply("payment", 40, false, 100, 0)
	assert(40, 0, 40, 0)
	apply("release", 0, false, 100, 40)
	assert(40, 60, 0, 100)
	apply("payment", 50, true, 100, 40)
	assert(90, 10, 0, 100)
	apply("reduce_price", 20, true, 100, 90)
	assert(90, 0, 10, 80)
	apply("refund", 10, true, 80, 90)
	assert(80, 0, 0, 80)
	apply("approve_return", 0, true, 80, 80)
	assert(80, 0, 80, 0)
	// The cancelled sale is no longer released for accounting purposes.
	apply("refund", 80, false, 80, 80)
	assert(0, 0, 0, 0)
}

func TestConsumerPaymentAndReversalAreExactInverses(t *testing.T) {
	for _, released := range []bool{false, true} {
		for _, net := range []Money{0, 40, 100, 120} {
			paid, err := consumerPostings("payment", 75, released, 100, net)
			if err != nil {
				t.Fatal(err)
			}
			reversed, err := consumerPostings("reverse_payment", 75, released, 100, net+75)
			if err != nil {
				t.Fatal(err)
			}
			balance := map[string]Money{}
			for _, p := range append(paid, reversed...) {
				balance[p.Account] += p.Debit - p.Credit
			}
			for account, amount := range balance {
				if amount != 0 {
					t.Errorf("released=%v net=%d: %s changed by %d after reversal", released, net, account, amount)
				}
			}
		}
	}
}

func TestConsumerProjectionRejectsInvalidFinancialIntent(t *testing.T) {
	for _, tc := range []struct {
		action   string
		amount   Money
		released bool
		price    Money
		net      Money
	}{
		{"pyaument", 1, false, 100, 0},
		{"", 0, false, 0, 0},
		{"payment", 0, false, 100, 0},
		{"payment", -1, false, 100, 0},
		{"payment", 1, true, -1, 0},
		{"release", 0, false, 100, -1},
		{"reduce_price", 101, true, 100, 0},
		{"reverse_payment", 41, true, 100, 40},
		{"refund", 41, false, 100, 40},
		{"refund", 1, true, 100, 40},
		{"refund", 21, true, 100, 120},
	} {
		if postings, err := consumerPostings(tc.action, tc.amount, tc.released, tc.price, tc.net); err == nil || len(postings) != 0 {
			t.Errorf("accepted invalid financial intent: %+v", tc)
		}
	}
	for _, action := range []string{"accept", "decline", "claim", "reject_claim", "received", "cancel", "request_return", "escalate", "reject_return"} {
		postings, err := consumerPostings(action, 0, false, 100, 0)
		if err != nil || len(postings) != 0 {
			t.Errorf("nonfinancial %s created a journal or failed: %v", action, err)
		}
	}
	for _, action := range []string{"reduce_price", "approve_return"} {
		postings, err := consumerPostings(action, 10, false, 100, 40)
		if err != nil || len(postings) != 0 {
			t.Errorf("unreleased %s incorrectly changed the journal: %v", action, err)
		}
	}
	store := NewPostgresStore(nil)
	if err := store.PostConsumerEventTx(context.Background(), nil, "event", "payment", 1, false, 100, 0, time.Time{}); err == nil {
		t.Fatal("financial event accepted without a transaction")
	}
	if err := store.PostConsumerEventTx(context.Background(), nil, "", "accept", 0, false, 100, 0, time.Time{}); err == nil {
		t.Fatal("event accepted without a reference")
	}
}

func TestConsumerProjectionKeepsPostingIntentDeterministic(t *testing.T) {
	first, err := consumerPostings("reduce_price", 20, true, 100, 90)
	if err != nil {
		t.Fatal(err)
	}
	second, err := consumerPostings("reduce_price", 20, true, 100, 90)
	if err != nil || !reflect.DeepEqual(first, second) {
		t.Fatal("identical consumer intents produced different postings")
	}
}
