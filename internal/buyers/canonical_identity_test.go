package buyers

import (
	"context"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestCanonicalIdentityProjectsWithoutRewritingAuthority(t *testing.T) {
	if os.Getenv("KREDIT_INTEGRATION") != "1" {
		t.Skip("isolated database required")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	owner, profile, foreignOwner, foreign := uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString()
	exec := func(q string, args ...any) {
		t.Helper()
		if _, e := pool.Exec(ctx, q, args...); e != nil {
			t.Fatal(e)
		}
	}
	for _, u := range []string{owner, foreignOwner} {
		exec(`INSERT INTO app.users(id,normalized_email,display_name) VALUES($1::uuid,$2,'Personal identity retained')`, u, u+"@canonical.test")
	}
	exec(`INSERT INTO app.businesses(id,owner_user_id,legal_name,business_type,business_address,industry) VALUES($1::uuid,$2::uuid,'Original company','limited_company','Lagos','food')`, profile, owner)
	var workspace, name, status string
	if err = pool.QueryRow(ctx, `SELECT organization_id::text,legal_name,status FROM app.businesses WHERE id=$1::uuid`, profile).Scan(&workspace, &name, &status); err != nil || workspace != profile || status != "pending_verification" {
		t.Fatalf("canonical binding: %s %s %v", workspace, status, err)
	}
	var count int
	if err = pool.QueryRow(ctx, `SELECT count(*) FROM app.supplier_onboarding_profiles WHERE organization_id=$1::uuid AND kyb_state='not_started' AND owner_email_verified_at IS NULL`, workspace).Scan(&count); err != nil || count != 1 {
		t.Fatalf("unverified onboarding: %d %v", count, err)
	}
	exec(`UPDATE app.organizations SET legal_name='Canonical company',business_address='Abuja' WHERE id=$1::uuid`, workspace)
	if err = pool.QueryRow(ctx, `SELECT legal_name FROM app.businesses WHERE id=$1::uuid`, profile).Scan(&name); err != nil || name != "Canonical company" {
		t.Fatalf("identity projection: %s %v", name, err)
	}
	exec(`UPDATE app.businesses SET legal_name='Conflicting copy' WHERE id=$1::uuid`, profile)
	if err = pool.QueryRow(ctx, `SELECT legal_name FROM app.businesses WHERE id=$1::uuid`, profile).Scan(&name); err != nil || name != "Canonical company" {
		t.Fatalf("second editable identity: %s %v", name, err)
	}
	exec(`INSERT INTO app.organizations(id,legal_name,business_type,business_address,industry) VALUES($1::uuid,'Unrelated company','limited_company','Kano','food')`, foreign)
	exec(`INSERT INTO app.memberships(organization_id,user_id,role,status) VALUES($1::uuid,$2::uuid,'owner','active')`, foreign, foreignOwner)
	if _, err = pool.Exec(ctx, `UPDATE app.businesses SET organization_id=$2::uuid WHERE id=$1::uuid`, profile, foreign); err == nil {
		t.Fatal("capability moved to unrelated entity")
	}
	if _, err = pool.Exec(ctx, `INSERT INTO app.businesses(organization_id,owner_user_id,legal_name,business_type,business_address,industry) VALUES($1::uuid,$2::uuid,'Forged','limited_company','Kano','food')`, foreign, owner); err == nil {
		t.Fatal("unrelated owner bound competitor business")
	}
}
