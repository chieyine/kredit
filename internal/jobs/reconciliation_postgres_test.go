package jobs

import (
	"os"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestReconciliationDetectsEmptyAndUnbalancedJournals(t *testing.T) {
	if os.Getenv("KREDIT_INTEGRATION") != "1" || os.Getenv("DATABASE_URL") == "" {
		t.Skip("isolated integration database required")
	}
	pool, err := pgxpool.New(t.Context(), os.Getenv("DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	query := `WITH transactions(id) AS (VALUES ('balanced'),('empty'),('unbalanced')),
 postings(id,transaction_id,debit_kobo,credit_kobo) AS (VALUES
 (1,'balanced',100::bigint,0::bigint),(2,'balanced',0,100),(3,'unbalanced',100,0)) ` +
		strings.NewReplacer("ledger.transactions", "transactions", "ledger.postings", "postings").Replace(reconciliationSQL)
	rows, err := pool.Query(t.Context(), query)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var found []string
	for rows.Next() {
		var id string
		var debit, credit int64
		if err := rows.Scan(&id, &debit, &credit); err != nil {
			t.Fatal(err)
		}
		found = append(found, id)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	if strings.Join(found, ",") != "empty,unbalanced" {
		t.Fatalf("reconciliation exceptions = %v", found)
	}
}
