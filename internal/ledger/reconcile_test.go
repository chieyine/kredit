package ledger

import (
	"context"
	"os"
	"slices"
	"testing"

	"github.com/jackc/pgx/v5"
)

func TestReconcileRequiresDatabase(t *testing.T) {
	if _, err := Reconcile(context.Background(), nil); err == nil {
		t.Fatal("expected reconciliation to require a database")
	}
}

func TestReconcileFindsJournalWithoutPostings(t *testing.T) {
	if os.Getenv("KREDIT_INTEGRATION") != "1" || os.Getenv("DATABASE_URL") == "" {
		t.Skip("integration database required")
	}
	conn, err := pgx.Connect(t.Context(), os.Getenv("DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = conn.Close(context.Background()) }()
	tx, err := conn.Begin(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	var id string
	err = tx.QueryRow(t.Context(), `INSERT INTO ledger.transactions(event_type,reference_type,reference_id,idempotency_key,effective_at) VALUES('test','test',uuidv7(),uuidv7()::text,now()) RETURNING id::text`).Scan(&id)
	if err != nil {
		t.Fatal(err)
	}
	report, err := reconcileQuery(t.Context(), tx)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(report.UnbalancedIDs, id) {
		t.Fatal("empty journal escaped reconciliation")
	}
}
