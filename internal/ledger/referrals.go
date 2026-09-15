package ledger

import (
	"context"
	"github.com/jackc/pgx/v5"
	"time"
)

func (s *PostgresStore) PostDSATx(ctx context.Context, tx pgx.Tx, id, kind string, amount int64, at time.Time) error {
	debit, credit := "DSA_ACQUISITION_EXPENSE", "DSA_COMMISSION_PAYABLE"
	if kind == "payout" {
		debit, credit = "DSA_COMMISSION_PAYABLE", "DSA_PAYOUT_CASH"
	}
	if amount < 0 {
		amount = -amount
		debit, credit = credit, debit
	}
	_, e := s.postTx(ctx, tx, Transaction{EventType: "dsa_" + kind, ReferenceType: "dsa_" + kind, ReferenceID: id, IdempotencyKey: "dsa:" + kind + ":" + id, EffectiveAt: at, Postings: []Posting{{Account: debit, Debit: Money(amount)}, {Account: credit, Credit: Money(amount)}}})
	return e
}
