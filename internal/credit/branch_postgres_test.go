package credit

import (
	"context"
	"errors"
	"kredit/internal/db"
	"kredit/internal/ledger"
	"kredit/internal/mandates"
	"kredit/internal/networkops"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestBranchSalesBoundariesAndCurrentAssignments(t *testing.T) {
	if os.Getenv("KREDIT_INTEGRATION") != "1" {
		t.Skip("isolated database required")
	}
	ctx := context.Background()
	admin, err := pgxpool.New(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close()
	exec := func(q string, args ...any) {
		t.Helper()
		if _, e := admin.Exec(ctx, q, args...); e != nil {
			t.Fatal(e)
		}
	}
	owner, staff, buyer, org := uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString()
	for _, u := range []string{owner, staff, buyer} {
		exec(`INSERT INTO app.users(id,normalized_email) VALUES($1::uuid,$2)`, u, u+"@branches.test")
	}
	exec(`INSERT INTO app.organizations(id,legal_name,business_type,business_address,industry) VALUES($1::uuid,'Branch supplier','limited_company','Lagos','food')`, org)
	exec(`INSERT INTO app.memberships(organization_id,user_id,role,status) VALUES($1::uuid,$2::uuid,'owner','active'),($1::uuid,$3::uuid,'sales','active')`, org, owner, staff)
	profiles := []string{uuid.NewString(), uuid.NewString()}
	for _, id := range profiles {
		exec(`INSERT INTO app.businesses(id,owner_user_id,legal_name,business_type,business_address,industry,status) VALUES($1::uuid,$2::uuid,'Customer','limited_company','Lagos','food','verified')`, id, buyer)
		exec(`INSERT INTO app.trade_relationships(supplier_organization_id,buyer_business_id) VALUES($1::uuid,$2::uuid)`, org, id)
	}
	cfg, _ := pgxpool.ParseConfig(os.Getenv("DATABASE_URL"))
	cfg.ConnConfig.RuntimeParams["role"] = "kredit_app"
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	network := networkops.Store{Pool: pool}
	branches := []networkops.Branch{{ID: uuid.NewString(), Name: "Mainland", Active: true}, {ID: uuid.NewString(), Name: "Island", Active: true}}
	assignments := []networkops.Partner{}
	for i, b := range branches {
		saved, e := network.SaveBranch(ctx, owner, org, b)
		if e != nil {
			t.Fatal(e)
		}
		branches[i] = saved
		a, e := network.Assign(ctx, owner, org, networkops.Partner{BusinessID: profiles[i], BranchID: b.ID, ManagerID: staff})
		if e != nil {
			t.Fatal(e)
		}
		assignments = append(assignments, a)
	}
	repo := NewPostgresStore(pool, NewStore(mandates.NewMockProvider(), ledger.NewPostgresStore(pool)))
	requests := []CreditRequest{}
	for _, profile := range profiles {
		c, e := repo.Create(CreateInput{SupplierOrganizationID: org, SupplierLegalName: "Supplier", BuyerUserID: buyer, BuyerBusinessID: profile, BuyerLegalName: "Buyer", PrincipalKobo: 10000, GoodsDescription: "Stock", DueDate: time.Now().AddDate(0, 0, 7).Format("2006-01-02"), CollectionAt: time.Now().Add(8 * 24 * time.Hour), CreatedBy: staff})
		if e != nil {
			t.Fatal(e)
		}
		requests = append(requests, c)
	}
	grant := networkops.Scope{UserID: staff, Mode: "branches", BranchIDs: []string{branches[0].ID}}
	saved, err := network.SaveScope(ctx, owner, org, grant)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = network.SaveScope(ctx, staff, org, grant); !errors.Is(err, networkops.ErrAuthority) {
		t.Fatalf("staff self-expanded scope: %v", err)
	}
	replay, err := network.SaveScope(ctx, owner, org, grant)
	if err != nil || replay.Version != saved.Version {
		t.Fatalf("scope retry: %#v %v", replay, err)
	}
	grant.BranchIDs = []string{branches[1].ID}
	if _, err = network.SaveScope(ctx, owner, org, grant); !errors.Is(err, networkops.ErrConflict) {
		t.Fatalf("stale scope changed: %v", err)
	}
	actorCtx := db.WithTenantContext(ctx, staff, org)
	views, err := repo.ReadForSupplier(actorCtx, org)
	if err != nil || len(views) != 1 || views[0].Request.ID != requests[0].ID {
		t.Fatalf("branch list: %#v %v", views, err)
	}
	if _, err = repo.GetForSupplierContext(actorCtx, requests[1].ID, org); err == nil {
		t.Fatal("direct ID exposed another branch")
	}
	if _, err = repo.Send(requests[1].ID, staff); err == nil {
		t.Fatal("cached aggregate bypassed branch write protection")
	}
	if _, err = repo.GetForSupplierContext(actorCtx, requests[0].ID, org); err != nil {
		t.Fatal(err)
	}
	if _, err = repo.Send(requests[0].ID, staff); err != nil {
		t.Fatal(err)
	}
	if _, err = repo.GetForSupplier(requests[0].ID, org); err == nil {
		t.Fatal("actorless read bypassed configured branch boundaries")
	}
	ownerViews, err := repo.ReadForSupplier(db.WithTenantContext(ctx, owner, org), org)
	if err != nil || len(ownerViews) != 2 {
		t.Fatalf("owner lost network: %d %v", len(ownerViews), err)
	}
	workspace, err := network.Read(ctx, staff, org)
	if err != nil || len(workspace.Partners) != 1 || len(workspace.Branches) != 1 {
		t.Fatalf("private directory leaked: %#v %v", workspace, err)
	}
	// Activate a real sale and verify that its financial children follow the branch.
	exec(`INSERT INTO app.persons(id,user_id,full_name,status) VALUES($1::uuid,$2::uuid,'Customer Owner','verified')`, uuid.NewString(), buyer)
	sent, e := repo.GetForBuyer(requests[0].ID, buyer)
	if e != nil {
		t.Fatal(e)
	}
	mandated, e := repo.SetMandate(requests[0].ID, buyer, mandates.Mandate{ID: uuid.NewString(), ProviderID: "branch-" + uuid.NewString(), Provider: "fixture", UserID: buyer, BusinessID: profiles[0], SupplierOrganizationID: org, AmountCeiling: 10000, Status: mandates.Active})
	if e != nil {
		t.Fatal(e)
	}
	if _, e = repo.Accept(requests[0].ID, buyer, sent.Agreement.ID, sent.Agreement.DocumentHash, mandated.Mandate.ProviderID, "AAL2", true, true); e != nil {
		t.Fatal(e)
	}
	if _, e = repo.Release(requests[0].ID, org, staff, "courier", "Branch shipment"); e != nil {
		t.Fatal(e)
	}
	activated, _, e := repo.RecordReceipt(requests[0].ID, buyer, "confirmed", "")
	if e != nil || activated.Obligation == nil {
		t.Fatalf("branch sale activation: %v", e)
	}
	financialCount := func(want int) {
		t.Helper()
		tx, e := pool.Begin(actorCtx)
		if e != nil {
			t.Fatal(e)
		}
		defer func() { _ = tx.Rollback(ctx) }()
		if _, e = tx.Exec(ctx, `SELECT set_config('app.current_user_id',$1,true),set_config('app.current_organization_id',$2,true)`, staff, org); e != nil {
			t.Fatal(e)
		}
		var count int
		if e = tx.QueryRow(ctx, `SELECT count(*) FROM app.obligations WHERE id=$1::uuid`, activated.Obligation.ID).Scan(&count); e != nil || count != want {
			t.Fatalf("branch financial visibility=%d wanted=%d: %v", count, want, e)
		}
	}
	financialCount(1)
	// Reassignment immediately removes both direct and cached access.
	a := assignments[0]
	a.BranchID = branches[1].ID
	// An in-flight financial write holds the assignment until its transaction ends.
	locked, e := pool.Begin(actorCtx)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = locked.Exec(ctx, `SELECT set_config('app.current_user_id',$1,true),set_config('app.current_organization_id',$2,true)`, staff, org); e != nil {
		_ = locked.Rollback(ctx)
		t.Fatal(e)
	}
	if _, e = locked.Exec(ctx, `UPDATE app.credit_requests SET updated_at=updated_at WHERE id=$1::uuid`, requests[0].ID); e != nil {
		_ = locked.Rollback(ctx)
		t.Fatal(e)
	}
	short, cancel := context.WithTimeout(ctx, 150*time.Millisecond)
	_, blocked := network.Assign(short, owner, org, a)
	cancel()
	if !errors.Is(blocked, context.DeadlineExceeded) {
		_ = locked.Rollback(ctx)
		t.Fatalf("assignment crossed an active financial transaction: %v", blocked)
	}
	if e = locked.Rollback(ctx); e != nil {
		t.Fatal(e)
	}
	if _, err = network.Assign(ctx, owner, org, a); err != nil {
		t.Fatal(err)
	}
	if _, err = repo.GetForSupplierContext(actorCtx, requests[0].ID, org); err == nil {
		t.Fatal("reassigned customer remained visible")
	}
	if _, err = repo.Cancel(requests[0].ID, staff); err == nil {
		t.Fatal("cached mutation survived reassignment")
	}
	financialCount(0)
	// Restoring membership does not silently restore an earlier branch decision.
	exec(`UPDATE app.memberships SET status='removed' WHERE organization_id=$1::uuid AND user_id=$2::uuid`, org, staff)
	exec(`UPDATE app.memberships SET status='active' WHERE organization_id=$1::uuid AND user_id=$2::uuid`, org, staff)
	directory, err := network.ReadScopes(ctx, owner, org)
	if err != nil || len(directory.Scopes) != 1 || directory.Scopes[0].Active {
		t.Fatalf("old scope revived: %#v %v", directory, err)
	}
	saved.BranchIDs = []string{branches[1].ID}
	if _, err = network.SaveScope(ctx, owner, org, saved); err != nil {
		t.Fatal(err)
	}
	views, err = repo.ReadForSupplier(actorCtx, org)
	if err != nil || len(views) != 2 {
		t.Fatalf("renewed scope: %d %v", len(views), err)
	}
	var revisions int
	if e := admin.QueryRow(ctx, `SELECT count(*) FROM app.member_branch_scope_history WHERE organization_id=$1::uuid AND user_id=$2::uuid`, org, staff).Scan(&revisions); e != nil || revisions != 2 {
		t.Fatalf("scope replay duplicated evidence: %d %v", revisions, e)
	}
	closed := branches[1]
	closed.Active = false
	if _, e := network.SaveBranch(ctx, owner, org, closed); e != nil {
		t.Fatal(e)
	}
	views, err = repo.ReadForSupplier(actorCtx, org)
	if err != nil || len(views) != 0 {
		t.Fatalf("closed branch retained staff access: %d %v", len(views), err)
	}
	// Background collections retain their separate worker authority.
	workerCfg, _ := pgxpool.ParseConfig(os.Getenv("DATABASE_URL"))
	workerCfg.ConnConfig.RuntimeParams["role"] = "kredit_worker"
	worker, e := pgxpool.NewWithConfig(ctx, workerCfg)
	if e != nil {
		t.Fatal(e)
	}
	defer worker.Close()
	var allowed bool
	if e = worker.QueryRow(ctx, `SELECT app.branch_customer_access($1::uuid,$2::uuid)`, org, profiles[0]).Scan(&allowed); e != nil || !allowed {
		t.Fatalf("worker lost collection authority: %v", e)
	}
	// A privileged snapshot reader must not revive a historical purchasing owner.
	exec(`UPDATE app.memberships SET status='removed' WHERE organization_id=$1::uuid AND user_id=$2::uuid`, profiles[0], buyer)
	tx, e := pool.Begin(ctx)
	if e != nil {
		t.Fatal(e)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, e = tx.Exec(ctx, `SELECT set_config('app.current_user_id',$1,true),set_config('app.current_organization_id',$2,true)`, buyer, org); e != nil {
		t.Fatal(e)
	}
	var snapshot []byte
	if e = tx.QueryRow(ctx, `SELECT app.credit_snapshot_by_id($1)`, requests[0].ID).Scan(&snapshot); e != nil || len(snapshot) != 0 {
		t.Fatalf("privileged snapshot revived a removed buyer: %v", e)
	}

}
