package networkops

import (
	"context"
	"errors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"os"
	"testing"
)

func TestNetworkAssignmentsPreserveScopeVersionsAndAuthority(t *testing.T) {
	if os.Getenv("KREDIT_INTEGRATION") != "1" {
		t.Skip("isolated database required")
	}
	ctx := context.Background()
	admin, err := pgxpool.New(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close()
	owner, manager, buyer, org, other, profile := uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString()
	exec := func(q string, args ...any) {
		t.Helper()
		if _, e := admin.Exec(ctx, q, args...); e != nil {
			t.Fatal(e)
		}
	}
	for _, id := range []string{owner, manager, buyer} {
		exec(`INSERT INTO app.users(id,normalized_email,display_name) VALUES($1::uuid,$2,'Network fixture')`, id, id+"@network.test")
	}
	for _, id := range []string{org, other} {
		exec(`INSERT INTO app.organizations(id,legal_name,business_type,business_address,industry) VALUES($1::uuid,'Network fixture','limited_company','Lagos','food')`, id)
	}
	exec(`INSERT INTO app.memberships(organization_id,user_id,role,status) VALUES($1::uuid,$2::uuid,'owner','active'),($1::uuid,$3::uuid,'sales','active')`, org, owner, manager)
	exec(`INSERT INTO app.businesses(id,owner_user_id,legal_name,business_type,business_address,industry,status) VALUES($1::uuid,$2::uuid,'Partner fixture','limited_company','Lagos','food','verified')`, profile, buyer)
	exec(`INSERT INTO app.trade_relationships(supplier_organization_id,buyer_business_id) VALUES($1::uuid,$2::uuid)`, org, profile)
	cfg, err := pgxpool.ParseConfig(os.Getenv("DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	cfg.ConnConfig.RuntimeParams["role"] = "kredit_app"
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	s := Store{Pool: pool}
	branch := Branch{ID: uuid.NewString(), Name: "Lagos " + uuid.NewString(), Territory: "Mainland", Active: true}
	saved, err := s.SaveBranch(ctx, owner, org, branch)
	if err != nil || saved.Version != 1 {
		t.Fatalf("branch: %#v %v", saved, err)
	}
	replay, err := s.SaveBranch(ctx, owner, org, branch)
	if err != nil || replay.Version != 1 {
		t.Fatalf("create replay: %#v %v", replay, err)
	}
	if _, err = s.SaveBranch(ctx, manager, org, branch); !errors.Is(err, ErrAuthority) {
		t.Fatalf("sales changed branch: %v", err)
	}
	assignment := Partner{BusinessID: profile, BranchID: branch.ID, ManagerID: manager}
	savedAssignment, err := s.Assign(ctx, owner, org, assignment)
	if err != nil || savedAssignment.Version != 1 {
		t.Fatalf("assignment: %#v %v", savedAssignment, err)
	}
	replayAssignment, err := s.Assign(ctx, owner, org, assignment)
	if err != nil || replayAssignment.Version != 1 {
		t.Fatalf("assignment replay: %#v %v", replayAssignment, err)
	}
	assignment.ManagerID = ""
	if _, err = s.Assign(ctx, owner, org, assignment); !errors.Is(err, ErrConflict) {
		t.Fatalf("stale assignment accepted: %v", err)
	}
	assignment.Version = 1
	assignment.ManagerID = buyer
	if _, err = s.Assign(ctx, owner, org, assignment); err == nil {
		t.Fatal("non-member manager accepted")
	}
	assignment.ManagerID = manager
	foreign := uuid.NewString()
	exec(`INSERT INTO app.business_branches(id,organization_id,name,updated_by) VALUES($1::uuid,$2::uuid,'Foreign',$3::uuid)`, foreign, other, owner)
	assignment.BranchID = foreign
	if _, err = s.Assign(ctx, owner, org, assignment); err == nil {
		t.Fatal("cross-business branch accepted")
	}
	data, err := s.Read(ctx, owner, org)
	if err != nil || len(data.Branches) != 1 || len(data.Partners) != 1 || len(data.Managers) != 2 {
		t.Fatalf("workspace: %#v %v", data, err)
	}
	if _, err = s.Read(ctx, buyer, org); !errors.Is(err, ErrAuthority) {
		t.Fatalf("customer saw private network: %v", err)
	}
	saved.Active = false
	if _, err = s.SaveBranch(ctx, owner, org, saved); err != nil {
		t.Fatal(err)
	}
	assignment.BranchID = branch.ID
	if _, err = s.Assign(ctx, owner, org, assignment); err == nil {
		t.Fatal("closed branch accepted")
	}
	exec(`UPDATE app.memberships SET status='removed' WHERE organization_id=$1::uuid AND user_id=$2::uuid`, org, manager)
	data, err = s.Read(ctx, owner, org)
	if err != nil || data.Partners[0].ManagerActive {
		t.Fatalf("removed manager still available: %#v %v", data, err)
	}
	assignment.BranchID = ""
	assignment.ManagerID = ""
	if _, err = s.Assign(ctx, owner, org, assignment); err != nil {
		t.Fatal(err)
	}
	// Database policies deny a suspended user even with a forged workspace context.
	exec(`UPDATE app.users SET status='suspended' WHERE id=$1::uuid`, manager)
	tx, e := pool.Begin(ctx)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = tx.Exec(ctx, `SELECT set_config('app.current_user_id',$1,true),set_config('app.current_organization_id',$2,true)`, manager, org); e != nil {
		t.Fatal(e)
	}
	var visible int
	if e = tx.QueryRow(ctx, `SELECT count(*) FROM app.business_branches WHERE organization_id=$1::uuid`, org).Scan(&visible); e != nil || visible != 0 {
		t.Fatalf("revoked workspace visible: %d %v", visible, e)
	}
	tx.Rollback(ctx)
	var count int
	if err = admin.QueryRow(ctx, `SELECT count(*) FROM app.network_operation_history WHERE organization_id=$1::uuid`, org).Scan(&count); err != nil || count != 4 {
		t.Fatalf("history: %d %v", count, err)
	}
}
