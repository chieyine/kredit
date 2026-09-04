//go:build integration

package organizations

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5/pgxpool"
	"kredit/internal/access"
	"os"
	"testing"
	"time"
)

func TestPostgresInactiveMembershipAndOwnerRoleAreProtected(t *testing.T) {
	if os.Getenv("KREDIT_INTEGRATION") != "1" {
		t.Skip("integration database required")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	var owner, staff, org string
	for i, target := range []*string{&owner, &staff} {
		if err = pool.QueryRow(ctx, `INSERT INTO app.users(normalized_email) VALUES($1) RETURNING id::text`, fmt.Sprintf("org-audit-%d-%d@example.test", time.Now().UnixNano(), i)).Scan(target); err != nil {
			t.Fatal(err)
		}
	}
	if err = pool.QueryRow(ctx, `INSERT INTO app.organizations(legal_name,business_type,business_address,industry) VALUES('Org audit','limited_company','Lagos','retail') RETURNING id::text`).Scan(&org); err != nil {
		t.Fatal(err)
	}
	defer pool.Exec(ctx, `DELETE FROM app.memberships WHERE organization_id=$1::uuid;DELETE FROM app.organizations WHERE id=$1::uuid;DELETE FROM app.users WHERE id IN($2::uuid,$3::uuid)`, org, owner, staff)
	if _, err = pool.Exec(ctx, `INSERT INTO app.memberships(organization_id,user_id,role,status) VALUES($1::uuid,$2::uuid,'owner','active'),($1::uuid,$3::uuid,'finance','suspended')`, org, owner, staff); err != nil {
		t.Fatal(err)
	}
	s := NewPostgresStore(pool, "audit")
	if _, ok := s.Membership(org, staff); ok {
		t.Fatal("suspended membership authorized")
	}
	if _, err = s.ChangeRole(org, staff, owner, access.RoleSales); err == nil {
		t.Fatal("owner was demoted")
	}
	if _, err = s.ChangeStatus(org, staff, staff, "active"); err == nil {
		t.Fatal("self-reactivation accepted")
	}
}
