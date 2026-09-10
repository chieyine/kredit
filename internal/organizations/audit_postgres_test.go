//go:build integration

package organizations

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"kredit/internal/access"

	"github.com/jackc/pgx/v5/pgxpool"
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
	var owner, staff, target, org string
	for i, target := range []*string{&owner, &staff, &target} {
		if err = pool.QueryRow(ctx, `INSERT INTO app.users(normalized_email) VALUES($1) RETURNING id::text`, fmt.Sprintf("org-audit-%d-%d@example.test", time.Now().UnixNano(), i)).Scan(target); err != nil {
			t.Fatal(err)
		}
	}
	if err = pool.QueryRow(ctx, `INSERT INTO app.organizations(legal_name,business_type,business_address,industry) VALUES('Org audit','limited_company','Lagos','retail') RETURNING id::text`).Scan(&org); err != nil {
		t.Fatal(err)
	}
	// Keep this isolated fixture: committed audit history intentionally retains its actors and organization.

	if _, err = pool.Exec(ctx, `INSERT INTO app.memberships(organization_id,user_id,role,status) VALUES($1::uuid,$2::uuid,'owner','active'),($1::uuid,$3::uuid,'finance','suspended'),($1::uuid,$4::uuid,'viewer','active')`, org, owner, staff, target); err != nil {
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
	if _, err = s.ChangeRole(org, staff, target, access.RoleAdministrator); err == nil {
		t.Fatal("inactive staff changed another user's role")
	}
	if _, err = pool.Exec(ctx, `UPDATE app.memberships SET role='administrator',status='active',accepted_at=now() WHERE organization_id=$1::uuid AND user_id=$2::uuid`, org, staff); err != nil {
		t.Fatal(err)
	}
	if _, err = s.ChangeRole(org, staff, target, access.RoleFinance); err != nil {
		t.Fatal(err)
	}
	if _, err = s.ChangeStatus(org, owner, staff, "removed"); err != nil {
		t.Fatal(err)
	}
	if _, err = s.ChangeRole(org, staff, target, access.RoleSales); err == nil {
		t.Fatal("revoked administrator changed membership after revocation")
	}

}
