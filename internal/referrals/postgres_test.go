package referrals

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"kredit/internal/db"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestDSAPermissionsRewardsAndPayouts(t *testing.T) {
	if os.Getenv("KREDIT_INTEGRATION") != "1" || os.Getenv("DATABASE_URL") == "" {
		t.Skip("disposable database required")
	}
	ctx := context.Background()
	root, e := pgxpool.New(ctx, os.Getenv("DATABASE_URL"))
	if e != nil {
		t.Fatal(e)
	}
	defer root.Close()
	exec := func(q string, args ...any) {
		t.Helper()
		if _, e := root.Exec(ctx, q, args...); e != nil {
			t.Fatal(e)
		}
	}
	user := func() string {
		id := uuid.NewString()
		exec(`INSERT INTO app.users(id,normalized_email) VALUES($1::uuid,$2)`, id, id+"@example.test")
		return id
	}
	owner, agent, outsider := user(), user(), user()
	exec(`INSERT INTO app.platform_role_assignments(user_id,role,granted_by,reason) VALUES($1::uuid,'platform_owner',$1::uuid,'Synthetic DSA integration check')`, owner)
	pool := func(role string) *pgxpool.Pool {
		dsn := os.Getenv("APP_DATABASE_URL")
		if role == "kredit_worker" {
			dsn = os.Getenv("RIVER_DATABASE_URL")
		}
		runtime, err := db.OpenAsRole(ctx, dsn, role)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(runtime.Close)
		return runtime.Raw()
	}
	s := &Store{Pool: pool("kredit_app")}
	worker := &Store{Pool: pool("kredit_worker")}
	in := Input{Action: "enrol", Version: 1, Name: "Synthetic Agent", Phone: "+2348012345678", Bank: "Synthetic Bank", AccountName: "Synthetic Agent", Account: "1234567890", Consent: true}
	if _, e = s.Act(ctx, agent, false, in); e != nil {
		t.Fatal(e)
	}
	var code string
	if e = root.QueryRow(ctx, `SELECT code FROM app.dsa_agents WHERE user_id=$1::uuid`, agent).Scan(&code); e != nil {
		t.Fatal(e)
	}
	org := uuid.NewString()
	cac := fmt.Sprintf("RC%d", time.Now().UnixNano())
	exec(`INSERT INTO app.organizations(id,legal_name,business_type,registration_info,business_address,industry) VALUES($1::uuid,'Synthetic DSA Merchant','limited_company',to_jsonb($2::text),'Synthetic address','retail')`, org, cac)
	exec(`INSERT INTO app.memberships(organization_id,user_id,role,status) VALUES($1::uuid,$2::uuid,'owner','active')`, org, owner)
	if e = s.Claim(ctx, outsider, org, code); e == nil {
		t.Fatal("non-owner attributed business")
	}
	if e = s.Claim(ctx, owner, org, code); e != nil {
		t.Fatal(e)
	}
	if e = s.Claim(ctx, owner, org, code); e != nil {
		t.Fatal("claim replay", e)
	}
	refresh := func() {
		t.Helper()
		if e := worker.Refresh(ctx); e != nil {
			t.Fatal(e)
		}
	}
	earned := func() int64 {
		t.Helper()
		var n int64
		if e := root.QueryRow(ctx, `SELECT COALESCE(sum(amount_kobo),0) FROM app.dsa_earnings WHERE agent_id=$1::uuid`, agent).Scan(&n); e != nil {
			t.Fatal(e)
		}
		return n
	}
	refresh()
	if earned() != 0 {
		t.Fatal("unverified registration rewarded")
	}
	exec(`INSERT INTO app.supplier_onboarding_profiles(organization_id,kyb_state,owner_email_verified_at,settlement_state,settlement_provider,settlement_provider_reference,settlement_account_last4) VALUES($1::uuid,'approved',now(),'verified','fixture','synthetic','7890')`, org)
	refresh()
	if earned() != 0 {
		t.Fatal("KYB without CAC evidence rewarded")
	}
	exec(`INSERT INTO app.native_identity_sessions(user_id,provider,subject_id,kind,state,safe_result) VALUES($1::uuid,'fixture',$2::uuid,'business','verified','{"cac_status":"verified"}')`, owner, org)
	refresh()
	refresh()
	if earned() != 100000 {
		t.Fatal("onboarding expected 100000", earned())
	}
	view, e := s.Read(ctx, outsider, false, "", nil)
	if e != nil {
		t.Fatal(e)
	}
	// Assert the public JSON boundary, not its private representation in Go.
	encodedView, err := json.Marshal(view)
	if err != nil {
		t.Fatal(err)
	}
	var publicView struct {
		Referrals []json.RawMessage `json:"referrals"`
	}
	if err := json.Unmarshal(encodedView, &publicView); err != nil {
		t.Fatal(err)
	}
	if publicView.Referrals == nil || len(publicView.Referrals) != 0 {
		t.Fatal("cross-agent disclosure or missing referral array")
	}
	// PostgreSQL bigint versions must survive the complete Read -> JSON path.
	const largeVersion = int64(9007199254740993)
	exec(`UPDATE app.dsa_referrals SET version=$2 WHERE organization_id=$1::uuid`, org, largeVersion)
	agentView, err := s.Read(ctx, agent, false, "", nil)
	if err != nil {
		t.Fatal(err)
	}
	encodedView, err = json.Marshal(agentView)
	if err != nil {
		t.Fatal(err)
	}
	var exactView struct {
		Referrals []struct {
			OrganizationID string `json:"organization_id"`
			Version        int64  `json:"version"`
		} `json:"referrals"`
	}
	if err := json.Unmarshal(encodedView, &exactView); err != nil {
		t.Fatal(err)
	}
	if len(exactView.Referrals) != 1 || exactView.Referrals[0].OrganizationID != org || exactView.Referrals[0].Version != largeVersion {
		t.Fatal("referral identity or exact version changed in the JSON response")
	}
	if _, e = s.Act(ctx, agent, true, Input{Action: "prepare", ID: agent, Reason: "Synthetic unauthorized payout"}); e == nil {
		t.Fatal("agent created own payout")
	}
	// Independently reconciled fees from a real fee invoice. Customer sale alone earns nothing.
	exec(`INSERT INTO app.consumer_sales(organization_id,created_by,buyer_user_id,target_type,target_value,terms,agreement_hash,state,accepted_at) VALUES($1::uuid,$2::uuid,$3::uuid,'email','synthetic@example.test','{}','synthetic','active',now())`, org, owner, outsider)
	refresh()
	if earned() != 100000 {
		t.Fatal("accepted sale generated cash reward")
	}
	credit, agreement, obligation, fee, invoice := uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString()
	exec(`INSERT INTO app.credit_requests(id,supplier_organization_id,buyer_user_id,buyer_business_id,principal_kobo,goods_description,due_date,collection_at,state,created_by) VALUES($1::uuid,$2::uuid,$3::uuid,$3::uuid,10000000,'Synthetic goods',current_date+30,now()+interval '30 days','DRAFT',$4::uuid)`, credit, org, outsider, owner)
	exec(`INSERT INTO app.agreement_versions(id,credit_request_id,version,canonical_json,document_hash,terms_version,privacy_version,created_by) VALUES($1::uuid,$2::uuid,1,'{}',$1,'synthetic','synthetic',$3::uuid)`, agreement, credit, owner)
	exec(`INSERT INTO app.obligations(id,credit_request_id,agreement_version_id,supplier_organization_id,buyer_business_id,principal_kobo,currency,lifecycle_status,payment_status,outstanding_kobo,base_fee_kobo,ledger_transaction_id,activated_at) VALUES($1::uuid,$2::uuid,$3::uuid,$4::uuid,$5::uuid,10000000,'NGN','ACTIVE','UNPAID',10000000,1000000,$1::uuid,now())`, obligation, credit, agreement, org, outsider)
	exec(`INSERT INTO app.fees(id,supplier_organization_id,obligation_id,fee_type,basis_amount_kobo,rate_basis_points,amount_kobo,currency,state) VALUES($1::uuid,$2::uuid,$3::uuid,'base_service',10000000,1000,1000000,'NGN','accrued')`, fee, org, obligation)
	exec(`INSERT INTO app.fee_invoices(id,organization_id,billing_reference,cycle,period_end,due_at,business_name,business_address,payment_instructions) VALUES($1::uuid,$2::uuid,'synthetic','monthly',now(),now()+interval '7 days','Synthetic','Synthetic address','Synthetic bank instructions')`, invoice, org)
	exec(`INSERT INTO app.fee_invoice_lines(invoice_id,fee_id,organization_id,amount_kobo,waived_at_issue_kobo) VALUES($1::uuid,$2::uuid,$3::uuid,1000000,0)`, invoice, fee, org)
	receipt := func(amount int64, direction string) {
		exec(`INSERT INTO app.fee_invoice_receipts(invoice_id,organization_id,bank_reference,amount_kobo,received_at,recorded_by,evidence,direction) VALUES($1::uuid,$2::uuid,$3,$4,now(),$5::uuid,'Synthetic independent bank evidence',$6)`, invoice, org, uuid.NewString(), amount, owner, direction)
	}
	receipt(500000, "received")
	refresh()
	if earned() != 300000 {
		t.Fatal("activation reward", earned())
	}
	receipt(100000, "received")
	refresh()
	if earned() != 310000 {
		t.Fatal("fee share", earned())
	}
	var wg sync.WaitGroup
	errs := make(chan error, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); errs <- worker.Refresh(ctx) }()
	}
	wg.Wait()
	close(errs)
	for e := range errs {
		if e != nil {
			t.Fatal(e)
		}
	}
	if earned() != 310000 {
		t.Fatal("concurrent duplicate rewards")
	}
	if _, e = s.Act(ctx, owner, true, Input{Action: "prepare", ID: agent, Reason: "Synthetic held payout attempt"}); e == nil {
		t.Fatal("seven-day hold bypassed")
	}
	exec(`UPDATE app.dsa_agents SET bank_updated_at=now()-interval '8 days' WHERE user_id=$1::uuid`, agent)
	exec(`UPDATE app.dsa_earnings SET available_at=now()-interval '1 minute' WHERE agent_id=$1::uuid`, agent)
	result, e := s.Act(ctx, owner, true, Input{Action: "prepare", ID: agent, Reason: "Synthetic matured payout reservation"})
	if e != nil {
		t.Fatal(e)
	}
	payout := result.(map[string]string)["id"]
	if _, e = s.Act(ctx, owner, true, Input{Action: "prepare", ID: agent, Reason: "Synthetic duplicate reservation attempt"}); e == nil {
		t.Fatal("duplicate payout")
	}
	in.Action = "bank"
	if _, e = s.Act(ctx, agent, false, in); e == nil {
		t.Fatal("pending payout bank changed")
	}
	if _, e = s.Act(ctx, owner, true, Input{Action: "paid", ID: payout, Version: 1, Reference: uuid.NewString(), Reason: "Synthetic completed bank transfer evidence"}); e != nil {
		t.Fatal(e)
	}
	receipt(200000, "refunded")
	refresh()
	if earned() != 100000 {
		t.Fatal("refund did not reverse activation and share", earned())
	}
	var paid int64
	if e = root.QueryRow(ctx, `SELECT sum(amount_kobo) FROM app.dsa_payouts WHERE agent_id=$1::uuid AND state='paid'`, agent).Scan(&paid); e != nil || paid != 310000 {
		t.Fatal("paid history changed", paid, e)
	}
	if _, e = s.Act(ctx, owner, true, Input{Action: "prepare", ID: agent, Reason: "Synthetic negative balance payout"}); e == nil {
		t.Fatal("negative balance paid")
	}
	tx, e := s.begin(ctx, agent, "")
	if e != nil {
		t.Fatal(e)
	}
	if _, e = tx.Exec(ctx, `UPDATE app.dsa_earnings SET amount_kobo=999999 WHERE agent_id=$1::uuid`, agent); e == nil {
		t.Fatal("immutable reward edited")
	}
	_ = tx.Rollback(ctx) // The deliberately rejected write already aborted this transaction.

	// Provider confirmation is not bank cash; only independently received fees qualify.
	authID := uuid.NewString()
	exec(`INSERT INTO app.fee_authorizations(id,organization_id,provider,state,ceiling_kobo,starts_at,ends_at,identity_fingerprint,consent_version,created_by) VALUES($1::uuid,$2::uuid,'fixture','ready',1000000,now(),now()+interval '1 year','synthetic','synthetic',$3::uuid)`, authID, org, owner)
	exec(`INSERT INTO app.fee_debits(organization_id,authorization_id,invoice_id,amount_kobo,state) VALUES($1::uuid,$2::uuid,$3::uuid,300000,'succeeded')`, org, authID, invoice)
	refresh()
	if earned() != 100000 {
		t.Fatal("provider-held fees paid commission", earned())
	}
	bank := func(direction string) {
		exec(`INSERT INTO app.fee_bank_receipts(organization_id,provider,bank_reference,amount_kobo,direction,occurred_at,recorded_by,evidence) VALUES($1::uuid,'fixture',$2,300000,$3,now(),$4::uuid,'Synthetic fee bank reconciliation')`, org, uuid.NewString(), direction, owner)
	}
	bank("received")
	refresh()
	if earned() != 320000 {
		t.Fatal("reconciled provider fee share", earned())
	}
	exec(`UPDATE app.fee_debits SET review_required=true WHERE invoice_id=$1::uuid`, invoice)
	refresh()
	if earned() != 100000 {
		t.Fatal("disputed provider fee rewarded")
	}
	exec(`UPDATE app.fee_debits SET review_required=false WHERE invoice_id=$1::uuid`, invoice)
	bank("returned")
	refresh()
	if earned() != 100000 {
		t.Fatal("returned provider cash rewarded")
	}
	// Cap and duplicate CAC checks do not rely on the merchant's account ID.
	second := uuid.NewString()
	exec(`INSERT INTO app.organizations(id,legal_name,business_type,registration_info,business_address,industry) VALUES($1::uuid,'Duplicate synthetic merchant','limited_company',to_jsonb($2::text),'Synthetic address','retail')`, second, cac)
	exec(`INSERT INTO app.memberships(organization_id,user_id,role,status) VALUES($1::uuid,$2::uuid,'owner','active')`, second, owner)
	if e = s.Claim(ctx, owner, second, code); e != nil {
		t.Fatal(e)
	}
	exec(`INSERT INTO app.supplier_onboarding_profiles(organization_id,kyb_state,owner_email_verified_at,settlement_state,settlement_provider,settlement_provider_reference,settlement_account_last4) VALUES($1::uuid,'approved',now(),'verified','fixture','synthetic','7890')`, second)
	exec(`INSERT INTO app.native_identity_sessions(user_id,provider,subject_id,kind,state,safe_result) VALUES($1::uuid,'fixture',$2::uuid,'business','verified','{"cac_status":"verified"}')`, owner, second)
	refresh()
	var blocked bool
	if e = root.QueryRow(ctx, `SELECT blocked FROM app.dsa_referrals WHERE organization_id=$1::uuid`, second).Scan(&blocked); e != nil || !blocked {
		t.Fatal("duplicate CAC not blocked", e)
	}
	exec(`UPDATE app.dsa_agents SET onboarding_limit=0 WHERE user_id=$1::uuid`, agent)
	third := uuid.NewString()
	exec(`INSERT INTO app.organizations(id,legal_name,business_type,registration_info,business_address,industry) VALUES($1::uuid,'Capped synthetic merchant','limited_company',to_jsonb($2::text),'Synthetic address','retail')`, third, cac+"9")
	exec(`INSERT INTO app.memberships(organization_id,user_id,role,status) VALUES($1::uuid,$2::uuid,'owner','active')`, third, owner)
	if e = s.Claim(ctx, owner, third, code); e != nil {
		t.Fatal(e)
	}
	exec(`INSERT INTO app.supplier_onboarding_profiles(organization_id,kyb_state,owner_email_verified_at,settlement_state,settlement_provider,settlement_provider_reference,settlement_account_last4) VALUES($1::uuid,'approved',now(),'verified','fixture','synthetic','7890')`, third)
	exec(`INSERT INTO app.native_identity_sessions(user_id,provider,subject_id,kind,state,safe_result) VALUES($1::uuid,'fixture',$2::uuid,'business','verified','{"cac_status":"verified"}')`, owner, third)
	refresh()
	if earned() != 100000 {
		t.Fatal("agent cap bypassed")
	}
	exec(`UPDATE app.dsa_agents SET onboarding_limit=20 WHERE user_id=$1::uuid`, agent)
	refresh()
	if earned() != 200000 {
		t.Fatal("increased cap did not release waiting reward")
	}
	var balance int64
	e = root.QueryRow(ctx, `SELECT COALESCE(sum(p.credit_kobo-p.debit_kobo),0) FROM ledger.postings p JOIN ledger.accounts a ON a.id=p.account_id JOIN ledger.transactions l ON l.id=p.transaction_id WHERE a.code='DSA_COMMISSION_PAYABLE' AND (l.reference_id IN (SELECT id::text FROM app.dsa_earnings WHERE agent_id=$1::uuid) OR l.reference_id IN (SELECT id::text FROM app.dsa_payouts WHERE agent_id=$1::uuid))`, agent).Scan(&balance)
	if e != nil || balance != earned()-paid {
		t.Fatal("commission liability mismatch", balance, e)
	}
}
