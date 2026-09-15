package ledger

import (
	"context"
	"github.com/jackc/pgx/v5"
	"time"
)

// PostConsumerEventTx keeps retailer receipts and consumer deposits separate
// from business trade credit, platform fees and wholesaler settlements.
func (s *PostgresStore) PostConsumerEventTx(ctx context.Context, tx pgx.Tx, id, action string, amount Money, released bool, price, net Money, at time.Time) error {
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
	if len(postings) == 0 {
		return nil
	}
	_, err := s.postTx(ctx, tx, Transaction{EventType: "consumer_" + action, ReferenceType: "consumer_event", ReferenceID: id, IdempotencyKey: "consumer:" + id, EffectiveAt: at, Postings: postings})
	return err
}
