package payments

import (
	"context"
	"errors"
	"fmt"
	"kredit/internal/ledger"
	"time"

	"github.com/jackc/pgx/v5"
)

// validatePaymentJournalReplay proves that an existing key describes this exact
// accounting intent. A unique-key collision alone is not financial evidence.
// The caller owns the surrounding financial transaction and must roll it back
// on failure; retained journals and postings are never rewritten to match.
func validatePaymentJournalReplay(ctx context.Context, tx pgx.Tx, eventType, referenceID, key string, effectiveAt time.Time, debitAccount, creditAccount string, amount ledger.Money) error {
	var matches bool
	err := tx.QueryRow(ctx, `
		SELECT t.event_type=$2 AND t.reference_type='payment'
		   AND t.reference_id=$3 AND t.effective_at=$4
		   AND (SELECT count(*) FROM ledger.postings p WHERE p.transaction_id=t.id)=2
		   AND EXISTS (
		     SELECT 1 FROM ledger.postings p JOIN ledger.accounts a ON a.id=p.account_id
		     WHERE p.transaction_id=t.id AND a.code=$5 AND p.debit_kobo=$7 AND p.credit_kobo=0
		   )
		   AND EXISTS (
		     SELECT 1 FROM ledger.postings p JOIN ledger.accounts a ON a.id=p.account_id
		     WHERE p.transaction_id=t.id AND a.code=$6 AND p.debit_kobo=0 AND p.credit_kobo=$7
		   )
		FROM ledger.transactions t WHERE t.idempotency_key=$1`,
		key, eventType, referenceID, effectiveAt.UTC().Truncate(time.Microsecond), debitAccount, creditAccount, int64(amount)).Scan(&matches)
	if err != nil {
		return fmt.Errorf("validate existing payment journal: %w", err)
	}
	if !matches {
		return errors.New("payment journal idempotency key conflicts with recorded intent")
	}
	return nil
}
