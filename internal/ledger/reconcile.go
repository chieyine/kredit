package ledger

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ReconciliationReport is a safe aggregate result for operator output. It
// contains no customer or provider payloads.
type ReconciliationReport struct {
	TransactionCount int64    `json:"transaction_count"`
	DebitKobo        int64    `json:"debit_kobo"`
	CreditKobo       int64    `json:"credit_kobo"`
	UnbalancedIDs    []string `json:"unbalanced_transaction_ids"`
}

// Reconcile checks every journal transaction, not only the global debit and
// credit totals. A global total can balance while two individual transactions
// are malformed, so each transaction is grouped independently.
func Reconcile(ctx context.Context, pool *pgxpool.Pool) (ReconciliationReport, error) {
	if pool == nil {
		return ReconciliationReport{}, fmt.Errorf("ledger database is not configured")
	}
	return reconcileQuery(ctx, pool)
}

func reconcileQuery(ctx context.Context, query interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}) (ReconciliationReport, error) {
	var report ReconciliationReport
	// One statement gives totals and exceptions the same database snapshot.
	// Start from journal headers so a header with no postings cannot disappear.
	err := query.QueryRow(ctx, `
		WITH balances AS (
			SELECT t.id, count(p.id) AS posting_count,
			       COALESCE(sum(p.debit_kobo),0) AS debit,
			       COALESCE(sum(p.credit_kobo),0) AS credit
			FROM ledger.transactions t
			LEFT JOIN ledger.postings p ON p.transaction_id=t.id
			GROUP BY t.id
		)
		SELECT count(*), COALESCE(sum(debit),0), COALESCE(sum(credit),0),
		       COALESCE(array_agg(id::text ORDER BY id) FILTER (WHERE debit<>credit OR posting_count<2), ARRAY[]::text[])
		FROM balances`).Scan(&report.TransactionCount, &report.DebitKobo, &report.CreditKobo, &report.UnbalancedIDs)
	if err != nil {
		return ReconciliationReport{}, fmt.Errorf("reconcile ledger: %w", err)
	}
	return report, nil
}
