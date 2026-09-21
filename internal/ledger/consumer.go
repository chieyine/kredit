package ledger

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

// PostConsumerEventTx keeps retailer receipts and consumer deposits separate
// from business trade credit, platform fees and wholesaler settlements.
func (s *PostgresStore) PostConsumerEventTx(ctx context.Context, tx pgx.Tx, id, action string, amount Money, released bool, price, net Money, at time.Time) error {
	if id == "" || strings.TrimSpace(id) != id {
		return errors.New("consumer event reference is required without surrounding whitespace")
	}
	postings, err := consumerPostings(action, amount, released, price, net)
	if err != nil {
		return err
	}
	if len(postings) == 0 {
		return nil
	}
	if tx == nil {
		return errors.New("consumer posting requires a transaction")
	}
	_, err = s.postTx(ctx, tx, Transaction{EventType: "consumer_" + action, ReferenceType: "consumer_event", ReferenceID: id, IdempotencyKey: "consumer:" + id, EffectiveAt: at, Postings: postings})
	return err
}

// consumerPostings is an accounting projection, not an authorization decision.
// The sale aggregate authorizes actions and supplies its pre-event price/net.
// Enumerate nonfinancial events explicitly so a new or misspelled financial
// event cannot silently succeed without its required journal entry.
func consumerPostings(action string, amount Money, released bool, price, net Money) ([]Posting, error) {
	switch action {
	case "accept", "decline", "claim", "reject_claim", "received", "cancel", "request_return", "escalate", "reject_return":
		return nil, nil
	case "payment", "reverse_payment", "release", "approve_return", "reduce_price", "refund":
	default:
		return nil, errors.New("unsupported consumer journal event")
	}
	if price < 0 || net < 0 {
		return nil, errors.New("consumer balances must not be negative")
	}
	switch action {
	case "payment", "reverse_payment", "reduce_price", "refund":
		if amount <= 0 {
			return nil, errors.New("consumer journal amount must be positive")
		}
	}
	if action == "reduce_price" && amount > price {
		return nil, errors.New("consumer price reduction exceeds the current price")
	}
	if (action == "reverse_payment" || action == "refund") && amount > net {
		return nil, errors.New("consumer cash reduction exceeds retained funds")
	}
	if action == "refund" && released && amount > max(Money(0), net-price) {
		return nil, errors.New("consumer refund exceeds refundable funds for an active released sale")
	}
	postings := []Posting{}
	pair := func(debit, credit string, n Money) {
		if n > 0 {
			postings = append(postings, Posting{Account: debit, Debit: n}, Posting{Account: credit, Credit: n})
		}
	}
	switch action {
	case "payment":
		if released {
			covered := min(amount, max(Money(0), price-net))
			pair("CONSUMER_PAYMENT_CONTROL", "CONSUMER_RECEIVABLE", covered)
			pair("CONSUMER_PAYMENT_CONTROL", "CONSUMER_CUSTOMER_FUNDS", amount-covered)
		} else {
			pair("CONSUMER_PAYMENT_CONTROL", "CONSUMER_CUSTOMER_FUNDS", amount)
		}
	case "reverse_payment":
		if released {
			excess := min(amount, max(Money(0), net-price))
			pair("CONSUMER_CUSTOMER_FUNDS", "CONSUMER_PAYMENT_CONTROL", excess)
			pair("CONSUMER_RECEIVABLE", "CONSUMER_PAYMENT_CONTROL", amount-excess)
		} else {
			pair("CONSUMER_CUSTOMER_FUNDS", "CONSUMER_PAYMENT_CONTROL", amount)
		}
	case "release":
		pair("CONSUMER_RECEIVABLE", "CONSUMER_SALES_CONTROL", price)
		pair("CONSUMER_CUSTOMER_FUNDS", "CONSUMER_RECEIVABLE", min(net, price))
	case "approve_return":
		if released {
			pair("CONSUMER_SALES_CONTROL", "CONSUMER_RECEIVABLE", price)
			pair("CONSUMER_RECEIVABLE", "CONSUMER_CUSTOMER_FUNDS", min(net, price))
		}
	case "reduce_price":
		if released {
			pair("CONSUMER_SALES_CONTROL", "CONSUMER_RECEIVABLE", amount)
			overBefore := max(Money(0), net-price)
			overAfter := max(Money(0), net-(price-amount))
			pair("CONSUMER_RECEIVABLE", "CONSUMER_CUSTOMER_FUNDS", overAfter-overBefore)
		}
	case "refund":
		pair("CONSUMER_CUSTOMER_FUNDS", "CONSUMER_PAYMENT_CONTROL", amount)
	}
	return postings, nil
}
