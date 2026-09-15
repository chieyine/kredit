package consumer

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5/pgxpool"
	"kredit/internal/settlement"
	"os"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestConsumerPostgresPermissionsMoneyAndReplay(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" || os.Getenv("KREDIT_INTEGRATION") != "1" {
		t.Skip("disposable integration database required")
	}
	ctx := context.Background()
	root, e := pgxpool.New(ctx, dsn)
	if e != nil {
		t.Fatal(e)
	}
	defer root.Close()
	user := func(prefix string) string {
		var id string
		e := root.QueryRow(ctx, `INSERT INTO app.users(normalized_email) VALUES($1) RETURNING id::text`, fmt.Sprintf("%s-%d@example.test", prefix, time.Now().UnixNano())).Scan(&id)
		if e != nil {
			t.Fatal(e)
		}
		return id
	}
	owner, buyer, stranger, viewer := user("consumer-owner"), user("consumer-buyer"), user("consumer-stranger"), user("consumer-viewer")
	var org, email string
	if e = root.QueryRow(ctx, `INSERT INTO app.organizations(legal_name,business_type,business_address,industry) VALUES('Synthetic Retailer','limited_company','Test address','retail') RETURNING id::text`).Scan(&org); e != nil {
		t.Fatal(e)
	}
	for _, v := range []struct{ id, role string }{{owner, "owner"}, {viewer, "viewer"}} {
		if _, e = root.Exec(ctx, `INSERT INTO app.memberships(organization_id,user_id,role,status) VALUES($1::uuid,$2::uuid,$3,'active')`, org, v.id, v.role); e != nil {
			t.Fatal(e)
		}
	}
	if _, e = root.Exec(ctx, `INSERT INTO app.supplier_onboarding_profiles(organization_id,readiness_state,kyb_state,settlement_state,settlement_provider,settlement_provider_reference,settlement_account_last4,settlement_bank_name,billing_state) VALUES($1::uuid,'pilot_ready','approved','verified','fixture','fixture','7890','Synthetic Bank','configured')`, org); e != nil {
		t.Fatal(e)
	}
	if e = root.QueryRow(ctx, `SELECT normalized_email FROM app.users WHERE id=$1::uuid`, buyer).Scan(&email); e != nil {
		t.Fatal(e)
	}
	cfg, e := pgxpool.ParseConfig(dsn)
	if e != nil {
		t.Fatal(e)
	}
	cfg.ConnConfig.RuntimeParams["role"] = "kredit_app"
	app, e := pgxpool.NewWithConfig(ctx, cfg)
	if e != nil {
		t.Fatal(e)
	}
	defer app.Close()
	s := &Store{Pool: app}
	key := "synthetic-consumer-account-key-0123456789"
	connection := strings.Repeat("a", 64)
	registration := settlement.RegistrationID(key, org, "fixture", connection, "000013", "1234567890")
	if _, e = root.Exec(ctx, `INSERT INTO app.settlement_registrations(id,organization_id,provider,connection_identity,bank_code,account_last4,state,result) VALUES($1,$2::uuid,'fixture',$3,'000013','7890','REGISTERED','{"provider_reference":"fixture","bank_code":"000013","account_name":"Synthetic Retailer","account_last4":"7890"}')`, registration, org, connection); e != nil {
		t.Fatal(e)
	}
	if e = s.ConnectBank(ctx, buyer, org, key, "1234567890"); e == nil {
		t.Fatal("customer configured retailer bank")
	}
	if e = s.ConnectBank(ctx, owner, org, key, "4234567890"); e == nil {
		t.Fatal("same last four digits accepted a different account")
	}
	if e = s.ConnectBank(ctx, owner, org, key, "1234567890"); e != nil {
		t.Fatal("self-service bank match", e)
	}
	now := time.Now().UTC()
	terms := termsFixture(now)
	sale, e := s.Create(ctx, owner, org, Input{TargetType: "email", Target: email, Terms: terms})
	if e != nil {
		t.Fatal(e)
	}
	if _, e = root.Exec(ctx, `INSERT INTO app.platform_role_assignments(user_id,role,granted_by,reason) VALUES($1::uuid,'platform_owner',$1::uuid,'Consumer synthetic fixture')`, owner); e != nil {
		t.Fatal(e)
	}
	settings := Settings{Enabled: false, Evidence: "Synthetic exception: temporarily restrict new consumer sales"}
	if _, e = s.Settings(ctx, buyer, org, &settings); e == nil {
		t.Fatal("customer changed restrictions")
	}
	if _, e = s.Settings(ctx, owner, org, &settings); e != nil {
		t.Fatal(e)
	}
	if e = s.ConnectBank(ctx, owner, org, key, "1234567890"); e != nil {
		t.Fatal(e)
	}
	if _, e = s.Create(ctx, owner, org, Input{TargetType: "email", Target: email, Terms: terms}); e == nil {
		t.Fatal("bank reconnect bypassed restriction")
	}
	if _, e = s.Act(ctx, buyer, "", sale.ID, false, Action{Action: "accept", Version: sale.Version, Hash: sale.Hash, Name: "Personal Buyer", Address: "Customer delivery address", Consent: true}); e == nil {
		t.Fatal("restricted retailer offer accepted")
	}
	settings.Enabled = true
	if _, e = s.Settings(ctx, owner, org, &settings); e != nil {
		t.Fatal(e)
	}
	if _, e = s.Get(ctx, stranger, "", sale.ID, false); e == nil {
		t.Fatal("stranger saw personal purchase")
	}
	if _, e = s.Get(ctx, buyer, "", sale.ID, false); e != nil {
		t.Fatal("invited customer cannot view", e)
	}
	if _, e = s.Create(ctx, viewer, org, Input{TargetType: "email", Target: email, Terms: terms}); e == nil {
		t.Fatal("viewer created sale")
	}
	act := func(actor, orgID string, a Action) {
		t.Helper()
		a.Version = sale.Version
		next, err := s.Act(ctx, actor, orgID, sale.ID, false, a)
		if err != nil {
			t.Fatal(a.Action, err)
		}
		sale = next
	}
	act(buyer, "", Action{Action: "accept", Hash: sale.Hash, Name: "Personal Buyer", Address: "Customer's own delivery address", Consent: true})
	at := time.Now().UTC().Truncate(time.Microsecond)
	act(buyer, "", Action{Action: "claim", Amount: 50000, Reference: "consumer-transfer-" + sale.ID, Note: "Customer bank transfer reference and date", At: at})
	claim := sale.Events[len(sale.Events)-1]
	if sale.Paid != 0 || sale.Eligible {
		t.Fatal("claim became money")
	}
	a := Action{Action: "payment", Version: sale.Version, Amount: claim.Amount, Reference: claim.Reference, RelatedID: claim.ID, At: claim.At, Note: "Confirmed exact credit in retailer bank"}
	if _, e = s.Act(ctx, buyer, "", sale.ID, false, a); e == nil {
		t.Fatal("buyer confirmed own claim")
	}
	if _, e = s.Act(ctx, viewer, org, sale.ID, false, a); e == nil {
		t.Fatal("viewer recorded money")
	}
	var wg sync.WaitGroup
	results := make(chan error, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); _, err := s.Act(ctx, owner, org, sale.ID, false, a); results <- err }()
	}
	wg.Wait()
	close(results)
	success := 0
	for err := range results {
		if err == nil {
			success++
		}
	}
	if success != 1 {
		t.Fatalf("concurrent receipt successes=%d", success)
	}
	sale, e = s.Get(ctx, owner, org, sale.ID, false)
	if e != nil {
		t.Fatal(e)
	}
	if sale.Paid != 50000 || !sale.Eligible {
		t.Fatal("receipt balance or release threshold incorrect")
	}
	act(owner, org, Action{Action: "release", Note: "Retailer dispatch evidence for named buyer"})
	act(buyer, "", Action{Action: "received"})
	act(owner, org, Action{Action: "reduce_price", Amount: 60000, Note: "Retailer grants documented goodwill reduction"})
	if sale.RefundDue != 10000 || sale.Outstanding != 0 {
		t.Fatal("price reduction refund incorrect")
	}
	act(owner, org, Action{Action: "refund", Amount: 10000, Reference: "goodwill-refund-" + sale.ID, At: time.Now().UTC(), Note: "Completed retailer bank refund to customer"})
	act(buyer, "", Action{Action: "request_return", Note: "Customer reports defect and requests return"})
	act(owner, org, Action{Action: "approve_return", Note: "Returned defective item received by retailer"})
	if sale.RefundDue != 40000 || sale.Outstanding != 0 {
		t.Fatal("return after goodwill refund incorrect")
	}
	act(owner, org, Action{Action: "refund", Amount: 40000, Reference: "return-refund-" + sale.ID, At: time.Now().UTC(), Note: "Completed remaining refund to the customer"})
	if sale.RefundDue != 0 || sale.Paid != 50000 || sale.Refunded != 50000 {
		t.Fatal("final refund balance incorrect")
	}
	var balance, trade int64
	if e = root.QueryRow(ctx, `SELECT COALESCE(sum(p.debit_kobo-p.credit_kobo),0) FROM ledger.postings p JOIN ledger.transactions t ON t.id=p.transaction_id JOIN app.consumer_events e ON e.id::text=t.reference_id WHERE e.sale_id=$1::uuid`, sale.ID).Scan(&balance); e != nil || balance != 0 {
		t.Fatal("consumer journal unbalanced", balance, e)
	}
	if e = root.QueryRow(ctx, `SELECT count(*) FROM ledger.postings p JOIN ledger.transactions t ON t.id=p.transaction_id JOIN ledger.accounts a ON a.id=p.account_id JOIN app.consumer_events e ON e.id::text=t.reference_id WHERE e.sale_id=$1::uuid AND a.code NOT LIKE 'CONSUMER_%'`, sale.ID).Scan(&trade); e != nil || trade != 0 {
		t.Fatal("consumer money touched trade-credit accounts", trade, e)
	}
	// Direct permission checks ensure immutable evidence and customer write restrictions.
	tx, e := s.begin(ctx, buyer, org)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = tx.Exec(ctx, `INSERT INTO app.consumer_events(sale_id,actor_id,action,amount_kobo,reference,note) VALUES($1::uuid,$2::uuid,'payment',1,'forged','forged buyer receipt')`, sale.ID, buyer); e == nil {
		t.Fatal("database accepted a customer-created receipt")
	}
	_ = tx.Rollback(ctx)
	tx, e = s.begin(ctx, owner, org)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = tx.Exec(ctx, `DELETE FROM app.consumer_events WHERE sale_id=$1::uuid`, sale.ID); e == nil {
		t.Fatal("financial history was deletable")
	}
	_ = tx.Rollback(ctx)

	// A second active purchase exercises discovery, RLS and durable due notices.
	active, err := s.Create(ctx, owner, org, Input{TargetType: "email", Target: email, Terms: terms})
	if err != nil {
		t.Fatal(err)
	}
	active, err = s.Act(ctx, buyer, "", active.ID, false, Action{Action: "accept", Version: active.Version, Hash: active.Hash, Name: "Personal Buyer", Address: "Customer delivery address", Consent: true})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.List(ctx, buyer, "", false, ""); err != nil {
		t.Fatal("buyer list", err)
	}
	if _, err = s.List(ctx, owner, org, false, active.ID); err != nil {
		t.Fatal("seller pagination", err)
	}
	if _, err = s.List(ctx, owner, "", true, ""); err != nil {
		t.Fatal("owner list", err)
	}
	if _, err = s.Act(ctx, buyer, "", active.ID, false, Action{Action: "accept", Version: active.Version, Hash: active.Hash, Name: "Changed Buyer", Address: "Different delivery address", Consent: true}); err == nil {
		t.Fatal("accepted identity was replaceable")
	}

	// A reversed deposit restores the scheduled balance without touching delivery.
	active, err = s.Act(ctx, owner, org, active.ID, false, Action{Action: "payment", Version: active.Version, Amount: 20000, Reference: "reversed-deposit-" + active.ID, At: time.Now().UTC(), Note: "Confirmed synthetic bank deposit for reversal check"})
	if err != nil {
		t.Fatal(err)
	}
	receiptID := active.Events[len(active.Events)-1].ID
	active, err = s.Act(ctx, owner, org, active.ID, false, Action{Action: "reverse_payment", Version: active.Version, RelatedID: receiptID, Note: "Bank confirmed the deposit was reversed; evidence retained"})
	if err != nil {
		t.Fatal(err)
	}
	if active.Paid != 0 || active.Outstanding != active.Terms.Total || active.Eligible {
		t.Fatal("deposit reversal failed to restore purchase balance")
	}
	cfg.ConnConfig.RuntimeParams["role"] = "kredit_worker"
	worker, e := pgxpool.NewWithConfig(ctx, cfg)
	if e != nil {
		t.Fatal(e)
	}
	defer worker.Close()
	if e = (&Store{Pool: worker}).EnqueueReminders(ctx); e != nil {
		t.Fatal("worker reminder permissions", e)
	}

	if e = (&Store{Pool: worker}).EnqueueReminders(ctx); e != nil {
		t.Fatal(e)
	}
	var noticeCount int
	if e = root.QueryRow(ctx, `SELECT count(*) FROM app.outbox_events WHERE aggregate_type='consumer_sale' AND aggregate_id=$1 AND idempotency_key LIKE 'consumer-due:%'`, active.ID).Scan(&noticeCount); e != nil || noticeCount != 1 {
		t.Fatal("reminder replay", noticeCount, e)
	}
	var nonzero int
	if e = root.QueryRow(ctx, `SELECT count(*) FROM (SELECT a.code FROM ledger.postings p JOIN ledger.transactions t ON t.id=p.transaction_id JOIN ledger.accounts a ON a.id=p.account_id JOIN app.consumer_events e ON e.id::text=t.reference_id WHERE e.sale_id=$1::uuid GROUP BY a.code HAVING sum(p.debit_kobo-p.credit_kobo)<>0) balances`, sale.ID).Scan(&nonzero); e != nil || nonzero != 0 {
		t.Fatal("fully refunded purchase left accounting balances", nonzero, e)
	}
}
