//go:build integration

package audit

import (
	"context"
	"github.com/jackc/pgx/v5/pgxpool"
	"os"
	"testing"
)

func TestPostgresActivityAllowsMissingRequestID(t *testing.T) {
	if os.Getenv("KREDIT_INTEGRATION") != "1" {
		t.Skip("integration database required")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	var org string
	if err = pool.QueryRow(ctx, `INSERT INTO app.organizations(legal_name,business_type,business_address,industry) VALUES('Audit event org','limited_company','Lagos','retail') RETURNING id::text`).Scan(&org); err != nil {
		t.Fatal(err)
	}
	defer pool.Exec(ctx, `DELETE FROM app.audit_events WHERE organization_id=$1::uuid;DELETE FROM app.organizations WHERE id=$1::uuid`, org)
	s := NewPostgresStore(pool)
	e := s.Append(Event{OrganizationID: org, Action: "system.reconciled", ResourceType: "system"})
	if e.ID == "" {
		t.Fatal("event with no request id was dropped")
	}
	items := s.ListForOrganization(org)
	if len(items) != 1 || items[0].RequestID != "" {
		t.Fatalf("events=%+v", items)
	}
}
