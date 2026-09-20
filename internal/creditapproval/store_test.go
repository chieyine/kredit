package creditapproval

import (
	"context"
	"errors"
	"kredit/internal/credit"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestIndependentApprovalCannotBeBypassedOrReusedAfterRevision(t *testing.T) {
	if os.Getenv("KREDIT_INTEGRATION") != "1" {
		t.Skip("isolated database required")
	}
	ctx := context.Background()
	admin, err := pgxpool.New(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close()
	owner, reviewer, buyer, org, request := uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString()
	exec := func(sql string, args ...any) {
		t.Helper()
		if _, e := admin.Exec(ctx, sql, args...); e != nil {
			t.Fatal(e)
		}
	}
	for _, id := range []string{owner, reviewer, buyer} {
		exec(`INSERT INTO app.users(id,normalized_email) VALUES($1::uuid,$2)`, id, id+"@approval.test")
	}
	exec(`INSERT INTO app.organizations(id,legal_name,business_type,business_address,industry) VALUES($1::uuid,'Approval fixture','limited_company','Lagos','food')`, org)
	exec(`INSERT INTO app.memberships(organization_id,user_id,role,status) VALUES($1::uuid,$2::uuid,'owner','active'),($1::uuid,$3::uuid,'finance','active')`, org, owner, reviewer)
	exec(`INSERT INTO app.credit_requests(id,supplier_organization_id,buyer_user_id,buyer_business_id,principal_kobo,goods_description,due_date,collection_at,state,created_by) VALUES($1::uuid,$2::uuid,$3::uuid,gen_random_uuid(),10000,'Goods',current_date+7,now()+interval '7 days','DRAFT',$4::uuid)`, request, org, buyer, owner)
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
	if _, err = s.SetControls(ctx, reviewer, org, Controls{Enabled: true}); !errors.Is(err, ErrAuthority) {
		t.Fatalf("non-owner changed policy: %v", err)
	}
	controls, err := s.SetControls(ctx, owner, org, Controls{Enabled: true, Threshold: 5000})
	if err != nil || controls.Version != 1 {
		t.Fatalf("policy: %#v %v", controls, err)
	}
	if _, err = s.SetControls(ctx, owner, org, Controls{Enabled: false, Version: 0}); !errors.Is(err, ErrConflict) {
		t.Fatalf("stale policy overwrite: %v", err)
	}
	send := func(actor string) error {
		tx, e := pool.Begin(ctx)
		if e != nil {
			return e
		}
		defer tx.Rollback(ctx)
		if _, e = tx.Exec(ctx, `SELECT set_config('app.current_user_id',$1,true),set_config('app.current_organization_id',$2,true)`, actor, org); e != nil {
			return e
		}
		if _, e = tx.Exec(ctx, `UPDATE app.credit_requests SET state='SENT',version=version+1 WHERE id=$1::uuid`, request); e != nil {
			return e
		}
		return tx.Commit(ctx)
	}
	denied := func(e error) {
		t.Helper()
		var pg *pgconn.PgError
		if !errors.As(e, &pg) || pg.Code != "42501" {
			t.Fatalf("expected approval enforcement, got %v", e)
		}
	}
	denied(send(owner))
	denied(send("")) // A missing actor cannot hide an enabled policy from its trigger.
	id, err := s.Request(ctx, owner, org, request)
	if err != nil {
		t.Fatal(err)
	}
	// Runtime SQL cannot rewrite the captured proposal while deciding it.
	for _, assignment := range []string{"proposal='{}'::jsonb", "request_version=request_version+1", "organization_id=gen_random_uuid()"} {
		tx, e := pool.Begin(ctx)
		if e != nil {
			t.Fatal(e)
		}
		_, e = tx.Exec(ctx, `SELECT set_config('app.current_user_id',$1,true),set_config('app.current_organization_id',$2,true)`, reviewer, org)
		if e != nil {
			t.Fatal(e)
		}
		_, e = tx.Exec(ctx, `UPDATE app.credit_offer_approvals SET state='approved',decided_by=$2::uuid,reason='Reviewed',decided_at=now(),`+assignment+` WHERE id=$1::uuid`, id, reviewer)
		_ = tx.Rollback(ctx)
		var pg *pgconn.PgError
		if !errors.As(e, &pg) || pg.Code != "23514" {
			t.Fatalf("evidence rewrite %s: %v", assignment, e)
		}
	}
	same, err := s.Request(ctx, owner, org, request)
	if err != nil || same != id {
		t.Fatalf("duplicate request: %s %v", same, err)
	}
	if err = s.Decide(ctx, owner, org, id, "approved", "Self approval"); !errors.Is(err, ErrAuthority) {
		t.Fatalf("creator approved own offer: %v", err)
	}
	if _, err = s.SetLimit(ctx, reviewer, org, reviewer, 5000, 0); !errors.Is(err, ErrAuthority) {
		t.Fatalf("reviewer changed own ceiling: %v", err)
	}
	limit, err := s.SetLimit(ctx, owner, org, reviewer, 5000, 0)
	if err != nil || limit.Version != 1 {
		t.Fatalf("set ceiling: %#v %v", limit, err)
	}
	denied(s.Decide(ctx, reviewer, org, id, "approved", "Reviewed the order and terms"))
	if _, err = s.SetLimit(ctx, owner, org, reviewer, 50000, 0); !errors.Is(err, ErrConflict) {
		t.Fatalf("stale ceiling overwrite: %v", err)
	}
	if _, err = s.SetLimit(ctx, owner, org, reviewer, 50000, 1); err != nil {
		t.Fatal(err)
	}
	if err = s.Decide(ctx, reviewer, org, id, "approved", "Reviewed the order and terms"); err != nil {
		t.Fatal(err)
	}
	if err = s.Decide(ctx, reviewer, org, id, "approved", "Reviewed the order and terms"); err != nil {
		t.Fatalf("decision replay failed: %v", err)
	}
	// A changed draft must obtain another independent approval.
	exec(`UPDATE app.credit_requests SET goods_description='Different goods',version=version+1 WHERE id=$1::uuid`, request)
	denied(send(owner))
	next, err := s.Request(ctx, owner, org, request)
	if err != nil || next == id {
		t.Fatalf("revision not recorded independently: %s %v", next, err)
	}
	if err = s.Decide(ctx, reviewer, org, next, "approved", "Reviewed the revised order"); err != nil {
		t.Fatal(err)
	}
	// Reducing a ceiling invalidates an existing approval for a future send.
	if _, err = s.SetLimit(ctx, owner, org, reviewer, 5000, 2); err != nil {
		t.Fatal(err)
	}
	denied(send(owner))
	if _, err = s.SetLimit(ctx, owner, org, reviewer, 50000, 3); err != nil {
		t.Fatal(err)
	}
	var history int
	if err = admin.QueryRow(ctx, `SELECT count(*) FROM app.credit_reviewer_limit_history WHERE organization_id=$1::uuid AND user_id=$2::uuid`, org, reviewer).Scan(&history); err != nil || history != 4 {
		t.Fatalf("ceiling history: %d %v", history, err)
	}
	// Removing a reviewer invalidates their authority to release a new offer.
	exec(`UPDATE app.memberships SET status='suspended' WHERE organization_id=$1::uuid AND user_id=$2::uuid`, org, reviewer)
	denied(send(owner))
	exec(`UPDATE app.memberships SET status='active' WHERE organization_id=$1::uuid AND user_id=$2::uuid`, org, reviewer)
	if err = send(owner); err != nil {
		t.Fatalf("approved unchanged offer blocked: %v", err)
	}
	if err = s.Decide(ctx, reviewer, org, next, "approved", "Reviewed the revised order"); err != nil {
		t.Fatalf("durable approval replay after send failed: %v", err)
	}

	repo := credit.NewPostgresStore(pool, nil)
	offer, err := repo.Create(credit.CreateInput{SupplierOrganizationID: org, SupplierLegalName: "Supplier", BuyerUserID: buyer, BuyerBusinessID: uuid.NewString(), BuyerLegalName: "Distributor", PrincipalKobo: 20000, GoodsDescription: "Reviewed stock", DueDate: time.Now().AddDate(0, 0, 7).Format("2006-01-02"), CollectionAt: time.Now().Add(8 * 24 * time.Hour), CreatedBy: owner})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = repo.Send(offer.ID, owner); err == nil {
		t.Fatal("repository bypassed independent approval")
	}
	approval, err := s.Request(ctx, owner, org, offer.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err = s.Decide(ctx, reviewer, org, approval, "approved", "Checked the actual saved offer"); err != nil {
		t.Fatal(err)
	}
	if _, err = repo.GetForSupplier(offer.ID, org); err != nil {
		t.Fatal(err)
	}
	if _, err = repo.Send(offer.ID, owner); err != nil {
		t.Fatalf("approved repository send: %v", err)
	}
	if _, err = repo.Review(offer.ID, buyer); err != nil {
		t.Fatalf("approval blocked subsequent buyer review: %v", err)
	}
	inbox, err := s.Read(ctx, owner, org)
	if err != nil || len(inbox.Approvals) != 3 {
		t.Fatalf("approval history missing: %#v %v", inbox, err)
	}
	if _, err = s.Read(ctx, buyer, org); !errors.Is(err, ErrAuthority) {
		t.Fatalf("outsider read approvals: %v", err)
	}
}
