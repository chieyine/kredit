package ledger

import (
	"context"
	"errors"
	"math"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

const (
	accountDSAAcquisitionExpense = "DSA_ACQUISITION_EXPENSE"
	accountDSACommissionPayable  = "DSA_COMMISSION_PAYABLE"
	// #nosec G101 -- Public chart-of-accounts identifier, not an authentication credential.
	accountDSAPayoutCash = "DSA_PAYOUT_CASH"
)

func (s *PostgresStore) PostDSATx(ctx context.Context, tx pgx.Tx, id, kind string, amount int64, at time.Time) error {
	transaction, err := dsaTransaction(id, kind, amount, at)
	if err != nil {
		return err
	}
	if tx == nil {
		return errors.New("referral posting requires a transaction")
	}
	_, err = s.postTx(ctx, tx, transaction)
	return err
}

// dsaTransaction preserves the event identity for retries. Negative earnings
// reverse previously accrued rewards; a payout records actual money sent and
// must be positive. Unknown event kinds must never default to an earning.
func dsaTransaction(id, kind string, amount int64, at time.Time) (Transaction, error) {
	if id == "" || strings.TrimSpace(id) != id {
		return Transaction{}, errors.New("referral reference is required without surrounding whitespace")
	}
	if amount == 0 || amount == math.MinInt64 {
		return Transaction{}, errors.New("referral amount must have a nonzero representable magnitude")
	}
	debit, credit := accountDSAAcquisitionExpense, accountDSACommissionPayable
	switch kind {
	case "earning":
		if amount < 0 {
			amount = -amount
			debit, credit = credit, debit
		}
	case "payout":
		if amount < 0 {
			return Transaction{}, errors.New("referral payout amount must be positive")
		}
		debit, credit = accountDSACommissionPayable, accountDSAPayoutCash
	default:
		return Transaction{}, errors.New("unsupported referral journal event")
	}
	return Transaction{
		EventType:      "dsa_" + kind,
		ReferenceType:  "dsa_" + kind,
		ReferenceID:    id,
		IdempotencyKey: "dsa:" + kind + ":" + id,
		EffectiveAt:    at,
		Postings:      []Posting{{Account: debit, Debit: Money(amount)}, {Account: credit, Credit: Money(amount)}},
	}, nil
}
