package buyers

import (
	"context"
	"kredit/internal/identity"
	"kredit/internal/purchasing"
	"os"
	"testing"
	"time"

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
	if err = pool.QueryRow(ctx, `SELECT display_name FROM app.users WHERE id=$1::uuid`, owner).Scan(&name); err != nil || name != "Personal identity retained" {
		t.Fatalf("personal details changed: %s %v", name, err)
	}
}

func TestStaffEnrollmentKeepsPersonalIdentityAndRequiresGrant(t *testing.T) {
	if os.Getenv("KREDIT_INTEGRATION") != "1" {
		t.Skip("isolated database required")
	}
	ctx := context.Background()
	admin, err := pgxpool.New(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close()
	owner, staff, profile := uuid.NewString(), uuid.NewString(), uuid.NewString()
	exec := func(q string, args ...any) {
		t.Helper()
		if _, e := admin.Exec(ctx, q, args...); e != nil {
			t.Fatal(e)
		}
	}
	for _, u := range []string{owner, staff} {
		exec(`INSERT INTO app.users(id,normalized_email,display_name) VALUES($1::uuid,$2,'Keep personal name')`, u, u+"@staff.test")
	}
	exec(`INSERT INTO app.businesses(id,owner_user_id,legal_name,business_type,business_address,industry,status) VALUES($1::uuid,$2::uuid,'Staff business','limited_company','Lagos','food','verified')`, profile, owner)
	exec(`INSERT INTO app.memberships(organization_id,user_id,role,status) VALUES($1::uuid,$2::uuid,'sales','active')`, profile, staff)
	cfg, _ := pgxpool.ParseConfig(os.Getenv("DATABASE_URL"))
	cfg.ConnConfig.RuntimeParams["role"] = "kredit_app"
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	s := NewPostgresStore(pool, "staff-test", identity.NewMockProvider())
	input := AcceptInput{FullName: "Acting Staff", ConsentsAccepted: true, TermsVersion: "terms", PrivacyVersion: "privacy", IdentityNoticeVersion: IdentityNoticeVersion}
	if _, err = s.EnrollPurchasingStaff(ctx, staff, profile, input); err == nil {
		t.Fatal("membership implicitly granted purchasing")
	}
	grants := purchasing.Store{Pool: pool}
	if _, err = grants.Save(ctx, owner, profile, purchasing.Grant{UserID: staff, Actions: []string{"read", "accept"}, CeilingKobo: 10000, ExpiresAt: time.Now().Add(time.Hour)}); err != nil {
		t.Fatal(err)
	}
	portal, err := s.EnrollPurchasingStaff(ctx, staff, profile, input)
	if err != nil {
		t.Fatal(err)
	}
	if portal.Person.UserID != staff || portal.Representative.PersonID != portal.Person.ID || portal.Business.OwnerUserID != owner || portal.Representative.AuthorityStatus != "pending" {
		t.Fatalf("incorrect actor or invented verification: %#v", portal)
	}
	input.FullName = "Replacement name"
	replay, err := s.EnrollPurchasingStaff(ctx, staff, profile, input)
	if err != nil || replay.Person.ID != portal.Person.ID || replay.Person.FullName != "Acting Staff" || replay.Representative.ID != portal.Representative.ID {
		t.Fatalf("enrollment replay: %#v %v", replay, err)
	}
	if _, err = s.RefreshBusinessVerification(ctx, staff, profile); err != nil {
		t.Fatal(err)
	}
	profiles, err := s.ListBusinessProfiles(ctx, staff)
	if err != nil || len(profiles) != 1 || profiles[0].ID != profile {
		t.Fatalf("staff profiles: %#v %v", profiles, err)
	}
	exec(`UPDATE app.memberships SET status='removed' WHERE organization_id=$1::uuid AND user_id=$2::uuid`, profile, staff)
	if _, err = s.ReadBusinessPortal(ctx, staff, profile); err == nil {
		t.Fatal("removed staff retained portal")
	}
}
