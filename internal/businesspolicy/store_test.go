package businesspolicy

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"kredit/internal/config"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPostgresIndependentApprovalHistoryAndEffectivePolicy(t *testing.T) {
	if os.Getenv("KREDIT_INTEGRATION") != "1" || os.Getenv("DATABASE_URL") == "" {
		t.Skip("integration database required")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	// Nested service transactions keep this test's global policy changes isolated.
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var a, b, c string
	for _, target := range []*string{&a, &b, &c} {
		if err = tx.QueryRow(ctx, `INSERT INTO app.users(normalized_email) VALUES($1) RETURNING id::text`, uuid.NewString()+"@policy.test").Scan(target); err != nil {
			t.Fatal(err)
		}
	}
	for i, actor := range []string{a, b} {
		role := []string{"policy_manager", "approver"}[i]
		if _, err = tx.Exec(ctx, `INSERT INTO app.platform_role_assignments(user_id,role,granted_by,reason) VALUES($1::uuid,$2,$1::uuid,'Policy workflow test')`, actor, role); err != nil {
			t.Fatal(err)
		}
	}
	if _, err = tx.Exec(ctx, `SET LOCAL ROLE kredit_app`); err != nil {
		t.Fatal(err)
	}
	s := NewStore(tx, config.Config{})
	current, err := s.Read(ctx)
	if err != nil {
		t.Fatal(err)
	}
	in := Proposal{ID: uuid.NewString(), BaseRevision: current.Revision, Values: current.Values, Reason: "Change the operating notice period", EffectiveAt: time.Now().UTC().Add(time.Hour).Truncate(time.Microsecond)}
	in.Values.NoticeHours = 48
	if _, err = s.Propose(ctx, b, in); err == nil {
		t.Fatal("approver proposed policy")
	}
	if _, err = s.Propose(ctx, c, in); err == nil {
		t.Fatal("non-admin proposed settings")
	}
	if _, err = s.Propose(ctx, a, in); err != nil {
		t.Fatal(err)
	}
	if _, err = s.Propose(ctx, a, in); err != nil {
		t.Fatal("exact replay", err)
	}
	bad := in
	bad.Values.NoticeHours = 72
	if _, err = s.Propose(ctx, a, bad); err == nil {
		t.Fatal("conflicting replay accepted")
	}
	if err = s.Decide(ctx, in.ID, a, "approve", "Approve this policy change"); err == nil {
		t.Fatal("self-approval accepted")
	}
	if err = s.Decide(ctx, in.ID, b, "approve", "Independent review completed"); err != nil {
		t.Fatal(err)
	}
	now, err := s.Read(ctx)
	if err != nil || now.Revision != current.Revision {
		t.Fatal("future policy applied early", err)
	}
	other := in
	other.ID = uuid.NewString()
	if _, err = s.Propose(ctx, a, other); err == nil {
		t.Fatal("overlapping schedule accepted")
	}
	if err = s.Decide(ctx, in.ID, b, "cancel", "Cancel before the effective date"); err != nil {
		t.Fatal(err)
	}
	changes, events, err := s.History(ctx)
	if err != nil || len(changes) < 1 || len(events) < 3 {
		t.Fatal("missing immutable history", err)
	}
	// A completed historical revision is selected instead of environment defaults.
	bytes, _ := json.Marshal(in.Values)
	var revision int64
	if err = tx.QueryRow(ctx, `INSERT INTO app.business_policy_changes(id,base_revision,values,proposed_by,reason,effective_at,state,decided_by,decided_at) VALUES($1::uuid,$2,$3::jsonb,$4::uuid,'Historical approved policy',now()-interval '1 minute','approved',$5::uuid,now()-interval '2 minutes') RETURNING revision`, uuid.NewString(), current.Revision, bytes, a, b).Scan(&revision); err != nil {
		t.Fatal(err)
	}
	now, err = ReadTx(ctx, tx)
	if err != nil || now.Revision != revision || now.Values.NoticeHours != 48 {
		t.Fatal("effective policy not selected", err)
	}
	in.ID = uuid.NewString()
	if _, err = s.Propose(ctx, a, in); err == nil {
		t.Fatal("stale proposal accepted")
	}
	// Every mutation attempt gets its own savepoint so rejected SQL cannot abort the fixture.
	sp, _ := tx.Begin(ctx)
	_, err = sp.Exec(ctx, `UPDATE app.business_policy_changes SET reason='Rewrite the history' WHERE id=$1::uuid`, changes[0].ID)
	_ = sp.Rollback(ctx)
	if err == nil {
		t.Fatal("history can be rewritten")
	}
	// Read-only runtime role can see effective values but cannot change decisions.
	sp, _ = tx.Begin(ctx)
	if _, err = sp.Exec(ctx, `SET LOCAL ROLE kredit_worker`); err == nil {
		if _, err = ReadTx(ctx, sp); err != nil {
			t.Fatal(err)
		}
		_, err = sp.Exec(ctx, `UPDATE app.business_policy_changes SET state='cancelled' WHERE revision=$1`, revision)
		if err == nil {
			t.Fatal("worker changed business settings")
		}
	}
	_ = sp.Rollback(ctx)
}

// This separate test proves the database singleton lock serializes competing
// policy writers without creating committed business settings in the test DB.
func TestPolicyLockSerializesWriters(t *testing.T) {
	if os.Getenv("KREDIT_INTEGRATION") != "1" || os.Getenv("DATABASE_URL") == "" {
		t.Skip("integration database required")
	}
	ctx := context.Background()
	p, err := pgxpool.New(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()
	first, err := p.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = first.Rollback(ctx) }()
	_, err = first.Exec(ctx, `SELECT pg_advisory_xact_lock(616161)`)
	if err != nil {
		t.Fatal(err)
	}
	// Exercise the real defaults row if initialized, without introducing seed state.
	var present bool
	if err = first.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM app.business_policy_defaults)`).Scan(&present); err != nil {
		t.Fatal(err)
	}
	if !present {
		t.Skip("defaults are initialized by a database-backed runtime")
	}
	if _, err = first.Exec(ctx, `SELECT singleton FROM app.business_policy_defaults FOR UPDATE`); err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	wg.Add(1)
	var blockedErr error
	go func() {
		defer wg.Done()
		timed, cancel := context.WithTimeout(ctx, 150*time.Millisecond)
		defer cancel()
		_, blockedErr = p.Exec(timed, `SELECT singleton FROM app.business_policy_defaults FOR UPDATE`)
	}()
	wg.Wait()
	if blockedErr == nil || !strings.Contains(blockedErr.Error(), "context deadline") {
		t.Fatalf("writer did not block: %v", blockedErr)
	}
}

func TestPolicyCapsApplyAtDatabaseWriteBoundary(t *testing.T) {
	if os.Getenv("KREDIT_INTEGRATION") != "1" || os.Getenv("DATABASE_URL") == "" {
		t.Skip("integration database required")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var a, b string
	for _, v := range []*string{&a, &b} {
		if err = tx.QueryRow(ctx, `INSERT INTO app.users(normalized_email) VALUES($1) RETURNING id::text`, uuid.NewString()+"@policy-limit.test").Scan(v); err != nil {
			t.Fatal(err)
		}
	}
	values := Defaults(config.Config{})
	values.AllowedIndustries = "retail"
	values.MaxPrincipal = 50000
	values.MaxExposure = 50000
	if err = tx.QueryRow(ctx, `SELECT count(*)+1 FROM app.organizations`).Scan(&values.MaxSuppliers); err != nil {
		t.Fatal(err)
	}
	data, _ := json.Marshal(values)
	if _, err = tx.Exec(ctx, `INSERT INTO app.business_policy_changes(id,base_revision,values,proposed_by,reason,effective_at,state,decided_by,decided_at) VALUES($1::uuid,0,$2::jsonb,$3::uuid,'Approved test boundary settings',now()-interval '1 minute','approved',$4::uuid,now()-interval '2 minutes')`, uuid.NewString(), data, a, b); err != nil {
		t.Fatal(err)
	}
	// This historical policy fixture is rolled back; it never changes another test's settings.
	var org string
	if err = tx.QueryRow(ctx, `INSERT INTO app.organizations(legal_name,business_type,business_address,industry) VALUES('Policy limit fixture','limited_company','test','retail') RETURNING id::text`).Scan(&org); err != nil {
		t.Fatal(err)
	}
	blocked := func(query, want string, args ...any) {
		t.Helper()
		sp, err := tx.Begin(ctx)
		if err != nil {
			t.Fatal(err)
		}
		_, err = sp.Exec(ctx, query, args...)
		_ = sp.Rollback(ctx)
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Fatalf("wanted %q, got %v", want, err)
		}
	}
	blocked(`INSERT INTO app.organizations(legal_name,business_type,business_address,industry) VALUES('Over count','limited_company','test','retail')`, "business count limit")
	blocked(`INSERT INTO app.organizations(legal_name,business_type,business_address,industry) VALUES('Wrong industry','limited_company','test','other')`, "industry is outside")
	blocked(`INSERT INTO app.credit_requests(supplier_organization_id,buyer_user_id,buyer_business_id,principal_kobo,goods_description,due_date,collection_at,state,created_by) VALUES($1::uuid,$2::uuid,$3::uuid,50001,'goods',current_date+1,now()+interval '2 days','DRAFT',$2::uuid)`, "principal exceeds", org, a, uuid.NewString())
	buyer := uuid.NewString()
	for i, amount := range []int64{40000, 20000} {
		var request, agreement, journal string
		if err = tx.QueryRow(ctx, `INSERT INTO app.credit_requests(supplier_organization_id,buyer_user_id,buyer_business_id,principal_kobo,goods_description,due_date,collection_at,state,created_by) VALUES($1::uuid,$2::uuid,$3::uuid,$4,'goods',current_date+1,now()+interval '2 days','ACTIVE',$2::uuid) RETURNING id::text`, org, a, buyer, amount).Scan(&request); err != nil {
			t.Fatal(err)
		}
		if err = tx.QueryRow(ctx, `INSERT INTO app.agreement_versions(credit_request_id,version,canonical_json,document_hash,terms_version,privacy_version,created_by) VALUES($1::uuid,1,'{}',$2,'v1','v1',$3::uuid) RETURNING id::text`, request, uuid.NewString(), a).Scan(&agreement); err != nil {
			t.Fatal(err)
		}
		if err = tx.QueryRow(ctx, `INSERT INTO ledger.transactions(event_type,reference_type,reference_id,idempotency_key,effective_at) VALUES('test','credit_request',$1,$2,now()) RETURNING id::text`, request, uuid.NewString()).Scan(&journal); err != nil {
			t.Fatal(err)
		}
		query := `INSERT INTO app.obligations(credit_request_id,agreement_version_id,supplier_organization_id,buyer_business_id,principal_kobo,currency,lifecycle_status,payment_status,outstanding_kobo,base_fee_kobo,ledger_transaction_id,activated_at) VALUES($1::uuid,$2::uuid,$3::uuid,$4::uuid,$5,'NGN','ACTIVE','UNPAID',$5,0,$6::uuid,now())`
		if i == 0 {
			if _, err = tx.Exec(ctx, query, request, agreement, org, buyer, amount, journal); err != nil {
				t.Fatal(err)
			}
		} else {
			blocked(query, "buyer exposure exceeds", request, agreement, org, buyer, amount, journal)
		}
	}
	// A previously released sale must still be recognized after limits tighten.
	committedBuyer := uuid.NewString()
	var request, agreement, journal string
	if err = tx.QueryRow(ctx, `INSERT INTO app.credit_requests(supplier_organization_id,buyer_user_id,buyer_business_id,principal_kobo,goods_description,due_date,collection_at,state,created_by) VALUES($1::uuid,$2::uuid,$3::uuid,40000,'released goods',current_date+1,now()+interval '2 days','RECEIPT_CONFIRMATION_PENDING',$2::uuid) RETURNING id::text`, org, a, committedBuyer).Scan(&request); err != nil {
		t.Fatal(err)
	}
	if err = tx.QueryRow(ctx, `INSERT INTO app.agreement_versions(credit_request_id,version,canonical_json,document_hash,terms_version,privacy_version,created_by) VALUES($1::uuid,1,'{}',$2,'v1','v1',$3::uuid) RETURNING id::text`, request, uuid.NewString(), a).Scan(&agreement); err != nil {
		t.Fatal(err)
	}
	if err = tx.QueryRow(ctx, `INSERT INTO ledger.transactions(event_type,reference_type,reference_id,idempotency_key,effective_at) VALUES('test','credit_request',$1,$2,now()) RETURNING id::text`, request, uuid.NewString()).Scan(&journal); err != nil {
		t.Fatal(err)
	}
	if _, err = tx.Exec(ctx, `INSERT INTO app.goods_releases(credit_request_id,supplier_actor_id,delivery_method) VALUES($1::uuid,$2::uuid,'courier')`, request, a); err != nil {
		t.Fatal(err)
	}
	var second string
	if err = tx.QueryRow(ctx, `INSERT INTO app.credit_requests(supplier_organization_id,buyer_user_id,buyer_business_id,principal_kobo,goods_description,due_date,collection_at,state,created_by) VALUES($1::uuid,$2::uuid,$3::uuid,20000,'new goods',current_date+1,now()+interval '2 days','READY_TO_RELEASE',$2::uuid) RETURNING id::text`, org, a, committedBuyer).Scan(&second); err != nil {
		t.Fatal(err)
	}
	blocked(`INSERT INTO app.goods_releases(credit_request_id,supplier_actor_id,delivery_method) VALUES($1::uuid,$2::uuid,'courier')`, "buyer exposure exceeds", second, a)

	var legacyLine, legacyDrawdown string
	if err = tx.QueryRow(ctx, `INSERT INTO app.trade_lines(supplier_organization_id,buyer_user_id,buyer_business_id,approved_limit_kobo,available_limit_kobo,cadence,default_grace_hours,start_at,end_at,state,terms_version) VALUES($1::uuid,$2::uuid,$3::uuid,50000,50000,'monthly',0,now(),now()+interval '30 days','PROPOSED','v1') RETURNING id::text`, org, a, committedBuyer).Scan(&legacyLine); err != nil {
		t.Fatal(err)
	}
	if err = tx.QueryRow(ctx, `INSERT INTO app.drawdowns(trade_line_id,principal_kobo,goods_description,state,due_date,collection_at,grace_hours,terms_version,agreement_hash) VALUES($1::uuid,40000,'legacy offer','PENDING_BUYER_CONFIRMATION',current_date+1,now()+interval '2 days',0,'v1',repeat('a',64)) RETURNING id::text`, legacyLine).Scan(&legacyDrawdown); err != nil {
		t.Fatal(err)
	}
	values.MaxExposure = 10000
	values.MaxPrincipal = 10000
	data, _ = json.Marshal(values)
	if _, err = tx.Exec(ctx, `INSERT INTO app.business_policy_changes(id,base_revision,values,proposed_by,reason,effective_at,state,decided_by,decided_at) VALUES($1::uuid,0,$2::jsonb,$3::uuid,'Tighten future commitment limits',now()-interval '1 minute','approved',$4::uuid,now()-interval '2 minutes')`, uuid.NewString(), data, a, b); err != nil {
		t.Fatal(err)
	}
	if _, err = tx.Exec(ctx, `INSERT INTO app.obligations(credit_request_id,agreement_version_id,supplier_organization_id,buyer_business_id,principal_kobo,currency,lifecycle_status,payment_status,outstanding_kobo,base_fee_kobo,ledger_transaction_id,activated_at) VALUES($1::uuid,$2::uuid,$3::uuid,$4::uuid,40000,'NGN','ACTIVE','UNPAID',40000,0,$5::uuid,now())`, request, agreement, org, committedBuyer, journal); err != nil {
		t.Fatal("tightened policy suppressed an existing commitment", err)
	}

	if _, err = tx.Exec(ctx, `INSERT INTO app.credit_requests(id,supplier_organization_id,buyer_user_id,buyer_business_id,principal_kobo,goods_description,due_date,collection_at,state,created_by,fee_terms) VALUES($1::uuid,$2::uuid,$3::uuid,$4::uuid,40000,'legacy offer',current_date+1,now()+interval '2 days','ACTIVE',$3::uuid,'null'::jsonb)`, legacyDrawdown, org, a, committedBuyer); err != nil {
		t.Fatal("legacy drawdown lost its recorded offer on recognition", err)
	}

}
