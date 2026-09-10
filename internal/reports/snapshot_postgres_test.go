package reports

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestReportRejectsMissingObligationProjection(t *testing.T) {
	if os.Getenv("KREDIT_INTEGRATION") != "1" || os.Getenv("DATABASE_URL") == "" {
		t.Skip("isolated integration database required")
	}
	ctx := t.Context()
	pool, err := pgxpool.New(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	const org = "00000000-0000-7000-8000-000000000010"
	store := NewPostgresStore(pool, Source{})
	if _, err := store.financialSnapshotTx(ctx, tx, org, ""); err != nil {
		t.Fatalf("baseline report: %v", err)
	}
	var requestID, buyerID string
	if err := tx.QueryRow(ctx, `SELECT o.credit_request_id::text,c.buyer_user_id::text FROM app.obligations o JOIN app.credit_requests c ON c.id=o.credit_request_id WHERE o.supplier_organization_id=$1::uuid ORDER BY o.activated_at,o.id LIMIT 1`, org).Scan(&requestID, &buyerID); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(ctx, `DELETE FROM app.credit_aggregate_snapshots WHERE credit_request_id=$1`, requestID); err != nil {
		t.Fatal(err)
	}
	for _, scope := range []struct{ org, buyer string }{{org, ""}, {"", buyerID}, {org, buyerID}} {
		if _, err := store.financialSnapshotTx(ctx, tx, scope.org, scope.buyer); err == nil || !strings.Contains(err.Error(), "projection is missing") {
			t.Fatalf("missing obligation was not reported for %+v: %v", scope, err)
		}
	}
	// The surrounding transaction always rolls back the synthetic deletion.
}
