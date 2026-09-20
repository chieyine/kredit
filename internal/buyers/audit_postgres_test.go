package buyers

import (
	"context"
	"fmt"
	"kredit/internal/identity"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPostgresInvitationHashAndSecondSupplierReuse(t *testing.T) {
	if os.Getenv("KREDIT_INTEGRATION") != "1" {
		t.Skip("integration database required")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	var buyer, owner string
	for n, target := range []*string{&buyer, &owner} {
		if err = pool.QueryRow(ctx, `INSERT INTO app.users(normalized_email) VALUES($1) RETURNING id::text`, fmt.Sprintf("audit-invite-%d-%d@example.test", time.Now().UnixNano(), n)).Scan(target); err != nil {
			t.Fatal(err)
		}
	}
	runtimeCfg, err := pgxpool.ParseConfig(os.Getenv("DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	runtimeCfg.ConnConfig.RuntimeParams["role"] = "kredit_app"
	runtimePool, err := pgxpool.NewWithConfig(ctx, runtimeCfg)
	if err != nil {
		t.Fatal(err)
	}
	defer runtimePool.Close()
	s := NewPostgresStore(runtimePool, "audit-invitation", identity.NewMockProvider())
	var count int64
	if err = pool.QueryRow(ctx, `SELECT app.business_count()`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	s.SetBusinessLimit(count + 1)
	var first Portal
	for i := 0; i < 2; i++ {
		var org string
		if err = pool.QueryRow(ctx, `INSERT INTO app.organizations(legal_name,business_type,business_address,industry) VALUES('Audit supplier','limited_company','Lagos','retail') RETURNING id::text`).Scan(&org); err != nil {
			t.Fatal(err)
		}
		invitation, err := s.CreateInvitation(owner, org, CreateInvitationInput{Target: "buyer@example.test", TargetType: "email", LegalName: "Buyer Ltd", BusinessType: "limited_company", BusinessAddress: "Lagos", Industry: "retail"})
		if err != nil {
			t.Fatal(err)
		}
		if _, err = s.Preview(invitation.RawToken); err != nil {
			t.Fatal(err)
		}
		portal, err := s.Accept(ctx, invitation.RawToken, buyer, AcceptInput{FullName: "Buyer Name", ConsentsAccepted: true, TermsVersion: "terms-v1", PrivacyVersion: "privacy-v1", IdentityNoticeVersion: IdentityNoticeVersion})
		if err != nil {
			t.Fatal(err)
		}
		if portal.Business.WorkspaceID == "" {
			t.Fatal("distributor has no selling workspace")
		}
		var owners int
		if err := pool.QueryRow(ctx, `SELECT count(*) FROM app.memberships WHERE organization_id=$1::uuid AND user_id=$2::uuid AND role='owner' AND status='active'`, portal.Business.WorkspaceID, buyer).Scan(&owners); err != nil || owners != 1 {
			t.Fatalf("workspace owner authority missing: %v", err)
		}
		profiles, err := s.ListBusinessProfiles(ctx, buyer)
		if err != nil || len(profiles) != 1 || profiles[0].WorkspaceID != portal.Business.WorkspaceID {
			t.Fatalf("purchasing workspace mismatch: %#v %v", profiles, err)
		}
		unrelated, err := s.ListBusinessProfiles(ctx, owner)
		if err != nil || len(unrelated) != 0 {
			t.Fatalf("supplier accessed buyer workspace list: %#v %v", unrelated, err)
		}
		if i == 0 {
			first = portal
		} else if portal.Person.ID != first.Person.ID || portal.Business.ID != first.Business.ID || portal.Representative.ID != first.Representative.ID {
			t.Fatal("duplicate identity on second invitation")
		}
		replay, err := s.Accept(ctx, invitation.RawToken, buyer, AcceptInput{FullName: "Buyer Name", ConsentsAccepted: true, TermsVersion: "terms-v1", PrivacyVersion: "privacy-v1", IdentityNoticeVersion: IdentityNoticeVersion})
		if err != nil || replay.Business.ID != portal.Business.ID {
			t.Fatalf("same-buyer acceptance recovery failed: %v", err)
		}
		if _, err = s.Accept(ctx, invitation.RawToken, owner, AcceptInput{FullName: "Other person"}); err == nil {
			t.Fatal("another user replayed accepted invitation")
		}
		if i == 1 {
			next, err := s.CreateInvitation(owner, org, CreateInvitationInput{Target: "buyer@example.test", TargetType: "email", LegalName: "Another Buyer Ltd", BusinessType: "limited_company", BusinessAddress: "Lagos", Industry: "retail"})
			if err != nil {
				t.Fatal(err)
			}
			if _, err = s.Accept(ctx, next.RawToken, buyer, AcceptInput{FullName: "Buyer Name", ConsentsAccepted: true, TermsVersion: "terms-v1", PrivacyVersion: "privacy-v1", IdentityNoticeVersion: IdentityNoticeVersion}); err == nil {
				t.Fatal("new business exceeded the persisted limit")
			}
		}
	}
}

func TestPostgresImportReplayRetainsOneInvitation(t *testing.T) {
	if os.Getenv("KREDIT_INTEGRATION") != "1" {
		t.Skip("integration database required")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	var user, org string
	if err = pool.QueryRow(ctx, `INSERT INTO app.users(normalized_email) VALUES($1) RETURNING id::text`, fmt.Sprintf("import-%d@example.test", time.Now().UnixNano())).Scan(&user); err != nil {
		t.Fatal(err)
	}
	if err = pool.QueryRow(ctx, `INSERT INTO app.organizations(legal_name,business_type,business_address,industry) VALUES('Import manufacturer','limited_company','Lagos','retail') RETURNING id::text`).Scan(&org); err != nil {
		t.Fatal(err)
	}
	cfg, err := pgxpool.ParseConfig(os.Getenv("DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	cfg.ConnConfig.RuntimeParams["role"] = "kredit_app"
	runtime, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer runtime.Close()
	store := NewPostgresStore(runtime, "import-key", identity.NewMockProvider())
	input := CreateInvitationInput{SourceReference: "roster:test-1", Target: "distributor@example.test", TargetType: "email", LegalName: "Distributor", BusinessType: "limited_company", BusinessAddress: "Lagos", Industry: "retail"}
	first, err := store.CreateInvitation(user, org, input)
	if err != nil {
		t.Fatal(err)
	}
	replay, err := store.CreateInvitation(user, org, input)
	if err != nil {
		t.Fatal(err)
	}
	progress, err := store.InvitationPipeline(ctx, user, org)
	if err != nil || len(progress) != 1 || progress[0].SourceReference != input.SourceReference || progress[0].State != "pending" {
		t.Fatalf("import pipeline mismatch: %#v %v", progress, err)
	}
	if !replay.Replayed || first.Invitation.ID != replay.Invitation.ID || first.RawToken != replay.RawToken {
		t.Fatal("import replay changed invitation or token")
	}
	changed := input
	changed.Target = "different@example.test"
	if _, err = store.CreateInvitation(user, org, changed); err == nil {
		t.Fatal("changed import payload was accepted")
	}
	var count int
	if err = pool.QueryRow(ctx, `SELECT count(*) FROM app.buyer_invitations WHERE organization_id=$1::uuid AND source_reference=$2`, org, input.SourceReference).Scan(&count); err != nil || count != 1 {
		t.Fatalf("duplicate import: %d %v", count, err)
	}
}

func TestBusinessWorkspaceRequiresCurrentOwner(t *testing.T) {
	if os.Getenv("KREDIT_INTEGRATION") != "1" {
		t.Skip("integration database required")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	var owner, outsider, profile string
	for n, id := range []*string{&owner, &outsider} {
		if err := pool.QueryRow(ctx, `INSERT INTO app.users(normalized_email) VALUES($1) RETURNING id::text`, fmt.Sprintf("workspace-%d-%d@example.test", time.Now().UnixNano(), n)).Scan(id); err != nil {
			t.Fatal(err)
		}
	}
	if err := pool.QueryRow(ctx, `INSERT INTO app.businesses(owner_user_id,legal_name,business_type,business_address,industry) VALUES($1::uuid,'Distributor','limited_company','Lagos','retail') RETURNING id::text`, owner).Scan(&profile); err != nil {
		t.Fatal(err)
	}
	ensure := func(actor string) (string, error) {
		tx, err := pool.Begin(ctx)
		if err != nil {
			return "", err
		}
		defer func() { _ = tx.Rollback(ctx) }()
		if _, err := tx.Exec(ctx, `SET LOCAL ROLE kredit_app`); err != nil {
			return "", err
		}
		if _, err := tx.Exec(ctx, `SELECT set_config('app.current_user_id',$1,true),set_config('app.current_organization_id','',true)`, actor); err != nil {
			return "", err
		}
		var id string
		if err := tx.QueryRow(ctx, `SELECT app.ensure_business_workspace($1::uuid)::text`, profile).Scan(&id); err != nil {
			return "", err
		}
		return id, tx.Commit(ctx)
	}
	if _, err := ensure(outsider); err == nil {
		t.Fatal("outsider claimed a business workspace")
	}
	workspace, err := ensure(owner)
	if err != nil {
		t.Fatal(err)
	}
	again, err := ensure(owner)
	if err != nil || again != workspace {
		t.Fatalf("workspace replay created another identity: %v", err)
	}
	if _, err := pool.Exec(ctx, `UPDATE app.memberships SET status='removed' WHERE organization_id=$1::uuid AND user_id=$2::uuid`, workspace, owner); err != nil {
		t.Fatal(err)
	}
	if _, err := ensure(owner); err == nil {
		t.Fatal("revoked ownership was silently reinstated")
	}
}

func TestInvitationJoinsOnlyOwnedWorkspaceAndRetainsIdentity(t *testing.T) {
	if os.Getenv("KREDIT_INTEGRATION") != "1" {
		t.Skip("integration database required")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	var buyer, outsider, workspace, supplier string
	for n, id := range []*string{&buyer, &outsider} {
		if err := pool.QueryRow(ctx, `INSERT INTO app.users(normalized_email) VALUES($1) RETURNING id::text`, fmt.Sprintf("join-workspace-%d-%d@example.test", time.Now().UnixNano(), n)).Scan(id); err != nil {
			t.Fatal(err)
		}
	}
	for _, id := range []*string{&workspace, &supplier} {
		if err := pool.QueryRow(ctx, `INSERT INTO app.organizations(legal_name,business_type,business_address,industry) VALUES('Existing business','limited_company','Lagos','retail') RETURNING id::text`).Scan(id); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := pool.Exec(ctx, `INSERT INTO app.memberships(organization_id,user_id,role,status) VALUES($1::uuid,$2::uuid,'owner','active')`, workspace, buyer); err != nil {
		t.Fatal(err)
	}
	cfg, err := pgxpool.ParseConfig(os.Getenv("DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	cfg.ConnConfig.RuntimeParams["role"] = "kredit_app"
	runtime, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer runtime.Close()
	store := NewPostgresStore(runtime, "workspace-join-key", identity.NewMockProvider())
	invite := func(org string) string {
		t.Helper()
		result, err := store.CreateInvitation(outsider, org, CreateInvitationInput{Target: "buyer@example.test", TargetType: "email", LegalName: "Proposed name", BusinessType: "limited_company", BusinessAddress: "Abuja", Industry: "retail"})
		if err != nil {
			t.Fatal(err)
		}
		return result.RawToken
	}
	input := AcceptInput{FullName: "Business Owner", WorkspaceID: workspace, ConsentsAccepted: true, TermsVersion: "terms-v1", PrivacyVersion: "privacy-v1", IdentityNoticeVersion: IdentityNoticeVersion}
	token := invite(supplier)
	if _, err := store.Accept(ctx, token, outsider, input); err == nil {
		t.Fatal("non-owner joined an existing workspace")
	}
	if _, err := store.Accept(ctx, invite(workspace), buyer, input); err == nil {
		t.Fatal("workspace traded with itself")
	}
	first, err := store.Accept(ctx, token, buyer, input)
	if err != nil {
		t.Fatal(err)
	}
	if first.Business.WorkspaceID != workspace || first.Business.LegalName != "Existing business" {
		t.Fatalf("wrong workspace identity: %#v", first.Business)
	}
	if _, err := pool.Exec(ctx, `UPDATE app.organizations SET legal_name='Renamed business',business_address='Ibadan' WHERE id=$1::uuid`, workspace); err != nil {
		t.Fatal(err)
	}
	second, err := store.Accept(ctx, invite(supplier), buyer, input)
	if err != nil {
		t.Fatal(err)
	}
	if second.Business.ID != first.Business.ID {
		t.Fatal("renaming a workspace created a second purchasing identity")
	}
	if second.Business.LegalName != "Renamed business" || second.Business.BusinessAddress != "Ibadan" {
		t.Fatal("accepted workspace details did not replace proposed purchasing details")
	}
	if _, err := pool.Exec(ctx, `UPDATE app.memberships SET status='removed' WHERE organization_id=$1::uuid AND user_id=$2::uuid`, workspace, buyer); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Accept(ctx, invite(supplier), buyer, input); err == nil {
		t.Fatal("removed owner joined the workspace")
	}
}
