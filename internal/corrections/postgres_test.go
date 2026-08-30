package corrections

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"kredit/internal/identifier"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPostgresCorrectionIsRestartSafeAndEnforcesSeparateReviewer(t *testing.T) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" || os.Getenv("KREDIT_INTEGRATION") != "1" {
		t.Skip("KREDIT_INTEGRATION=1 and DATABASE_URL are required")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	var organizationID, requesterID, reviewerID string
	if err := pool.QueryRow(ctx, `INSERT INTO app.organizations(legal_name,business_type,business_address,industry) VALUES('Correction Test','limited_company','test','test') RETURNING id::text`).Scan(&organizationID); err != nil {
		t.Fatal(err)
	}
	for index, target := range []*string{&requesterID, &reviewerID} {
		if err := pool.QueryRow(ctx, `INSERT INTO app.users(normalized_email) VALUES($1) RETURNING id::text`, fmt.Sprintf("correction-%d-%d@example.test", time.Now().UnixNano(), index)).Scan(target); err != nil {
			t.Fatal(err)
		}
	}
	defer func() {
		_, _ = pool.Exec(ctx, `DELETE FROM app.correction_decisions WHERE request_id IN(SELECT id FROM app.correction_requests WHERE organization_id=$1::uuid)`, organizationID)
		_, _ = pool.Exec(ctx, `DELETE FROM app.correction_requests WHERE organization_id=$1::uuid`, organizationID)
		_, _ = pool.Exec(ctx, `DELETE FROM app.organizations WHERE id=$1::uuid`, organizationID)
		_, _ = pool.Exec(ctx, `DELETE FROM app.users WHERE id=$1::uuid OR id=$2::uuid`, requesterID, reviewerID)
	}()
	store := NewPostgresStore(pool)
	opened, err := store.Open(organizationID, "obligation", identifier.New(), "source-1", requesterID, "Incorrect payment date", []string{"document-reference"})
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := store.Decide(opened.ID, requesterID, StateApproved, "self approval"); err == nil {
		t.Fatal("requester was allowed to approve their own correction")
	}
	restarted := NewPostgresStore(pool)
	loaded, _, err := restarted.Get(opened.ID)
	if err != nil || loaded.Reason != opened.Reason || len(loaded.Evidence) != 1 {
		t.Fatalf("restart-safe load failed: %v %+v", err, loaded)
	}
	if _, err := restarted.StartReview(opened.ID, reviewerID); err != nil {
		t.Fatal(err)
	}
	decided, decision, err := restarted.Decide(opened.ID, reviewerID, StateApproved, "Evidence confirms the correction")
	if err != nil {
		t.Fatal(err)
	}
	if decided.State != StateApproved || decision.CorrectionID == "" {
		t.Fatalf("unexpected decision: %+v %+v", decided, decision)
	}
}
