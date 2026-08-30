package support

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPostgresStoreRoundTrip(t *testing.T) {
	url := os.Getenv("DATABASE_URL")
	if url == "" || os.Getenv("KREDIT_INTEGRATION") != "1" {
		t.Skip("KREDIT_INTEGRATION=1 and DATABASE_URL are required")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	var userID, organizationID string
	if err := pool.QueryRow(ctx, `INSERT INTO app.users (normalized_email) VALUES ($1) RETURNING id::text`, fmt.Sprintf("support-test-%d@example.test", time.Now().UnixNano())).Scan(&userID); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO app.organizations (legal_name, business_type, business_address, industry) VALUES ('Support Test', 'limited_company', 'test', 'test') RETURNING id::text`).Scan(&organizationID); err != nil {
		t.Fatal(err)
	}
	defer func() {
		_, _ = pool.Exec(ctx, `DELETE FROM app.support_case_events WHERE actor_id = $1::uuid`, userID)
		_, _ = pool.Exec(ctx, `DELETE FROM app.support_cases WHERE opened_by = $1::uuid`, userID)
		_, _ = pool.Exec(ctx, `DELETE FROM app.organizations WHERE id = $1::uuid`, organizationID)
		_, _ = pool.Exec(ctx, `DELETE FROM app.users WHERE id = $1::uuid`, userID)
	}()

	store := NewPostgresStore(pool)
	item, err := store.Open("obligation", "subject-1", userID, organizationID, false)
	if err != nil {
		t.Fatal(err)
	}
	if _, event, err := store.Transition(item.ID, userID, InProgress, "assigned"); err != nil || event.CaseID != item.ID {
		t.Fatalf("transition: %v %+v", err, event)
	}
	if got, ok := store.Get(item.ID); !ok || got.State != InProgress {
		t.Fatalf("unexpected durable case: %+v %v", got, ok)
	}
	if timeline := store.Timeline(item.ID); len(timeline) != 2 {
		t.Fatalf("expected open and transition events, got %d", len(timeline))
	}
}
