package ledger

import (
	"context"
	"fmt"

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
	var report ReconciliationReport
	if err := pool.QueryRow(ctx, `
		SELECT count(DISTINCT transaction_id), COALESCE(sum(debit_kobo),0), COALESCE(sum(credit_kobo),0)
		FROM ledger.postings`).Scan(&report.TransactionCount, &report.DebitKobo, &report.CreditKobo); err != nil {
		return ReconciliationReport{}, fmt.Errorf("load ledger totals: %w", err)
	}
	rows, err := pool.Query(ctx, `
		SELECT transactions.id::text
		FROM ledger.transactions transactions
		JOIN ledger.postings postings ON postings.transaction_id = transactions.id
		GROUP BY transactions.id
		HAVING COALESCE(sum(postings.debit_kobo),0) <> COALESCE(sum(postings.credit_kobo),0)
		ORDER BY transactions.id`)
	if err != nil {
		return ReconciliationReport{}, fmt.Errorf("load unbalanced ledger transactions: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return ReconciliationReport{}, fmt.Errorf("scan unbalanced ledger transaction: %w", err)
		}
		report.UnbalancedIDs = append(report.UnbalancedIDs, id)
	}
	if err := rows.Err(); err != nil {
		return ReconciliationReport{}, fmt.Errorf("read unbalanced ledger transactions: %w", err)
	}
	return report, nil
}
