package credit

import (
	"context"
	"encoding/json"
	"os"
	"sync"
	"testing"
	"time"

	"kredit/internal/db"
	"kredit/internal/identifier"
	"kredit/internal/ledger"
	"kredit/internal/mandates"
)

func TestSystemAcceptanceUsesSeparateEvidenceOnColdWorkers(t *testing.T) {
	if os.Getenv("KREDIT_INTEGRATION") != "1" {
		t.Skip("isolated database required")
	}
	ctx := context.Background()
	root, err := db.Open(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	pool := root.Raw()
	if _, err = pool.Exec(ctx, `UPDATE app.platform_settings SET value='true' WHERE key='features.system_acceptance'`); err != nil {
		t.Fatal(err)
	}
	defer func() {
		_, _ = pool.Exec(ctx, `UPDATE app.platform_settings SET value='false' WHERE key='features.system_acceptance'`)
	}()
	worker, err := db.OpenAsRole(ctx, os.Getenv("RIVER_DATABASE_URL"), "kredit_worker")
	if err != nil {
		t.Fatal(err)
	}
	defer worker.Close()
	buyer, org, business := identifier.New(), identifier.New(), identifier.New()
	if _, err = pool.Exec(ctx, `INSERT INTO app.users(id,normalized_email) VALUES($1,$2)`, buyer, "system-"+buyer+"@example.test"); err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, `INSERT INTO app.organizations(id,legal_name,business_type,business_address,industry) VALUES($1,'System evidence fixture','limited_company','Lagos','retail')`, org); err != nil {
		t.Fatal(err)
	}
	memory := NewStore(mandates.NewMockProvider(), ledger.NewStore())
	memory.now = func() time.Time { return time.Now().UTC().Add(-6 * 24 * time.Hour) }
	fixtureIDs := map[string]bool{}
	create := func() View {
		t.Helper()
		r, err := memory.Create(CreateInput{SupplierOrganizationID: org, SupplierLegalName: "Supplier", BuyerUserID: buyer, BuyerBusinessID: business, BuyerLegalName: "Buyer", PrincipalKobo: 1000, GoodsDescription: "goods", DueDate: time.Now().AddDate(0, 0, 7).Format("2006-01-02"), CollectionAt: time.Now().Add(8 * 24 * time.Hour), CreatedBy: buyer})
		if err != nil {
			t.Fatal(err)
		}
		v, err := memory.Send(r.ID, buyer)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = memory.Review(r.ID, buyer); err != nil {
			t.Fatal(err)
		}
		v, err = memory.AuthorizeMandate(ctx, r.ID, buyer)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = memory.Accept(r.ID, buyer, v.Agreement.ID, v.Agreement.DocumentHash, v.Mandate.ProviderID, "AAL2", true, true); err != nil {
			t.Fatal(err)
		}
		v, err = memory.Release(r.ID, org, buyer, "pickup", "fixture")
		if err != nil {
			t.Fatal(err)
		}
		v.Request.MandateID = identifier.New()
		v.Mandate.ID = v.Request.MandateID
		tx, err := pool.Begin(ctx)
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = tx.Rollback(ctx) }()
		if err = syncNormalizedCredit(ctx, tx, v); err != nil {
			t.Fatal(err)
		}
		encoded, _ := json.Marshal(v)
		if _, err = tx.Exec(ctx, `INSERT INTO app.credit_aggregate_snapshots(credit_request_id,supplier_organization_id,buyer_user_id,aggregate,version) VALUES($1,$2,$3,$4,$5)`, r.ID, org, buyer, encoded, v.Request.Version); err != nil {
			t.Fatal(err)
		}
		if err = tx.Commit(ctx); err != nil {
			t.Fatal(err)
		}
		fixtureIDs[v.Request.ID] = true
		return v
	}
	eligible := create()
	newWorker := func() *PostgresStore {
		return NewPostgresStore(worker.Raw(), NewStore(nil, ledger.NewPostgresStore(worker.Raw())))
	}
	run := func() []string {
		t.Helper()
		ids, err := newWorker().AutoActivateMatured(ctx, time.Now().Add(1000*time.Hour))
		if err != nil {
			t.Fatal(err)
		}
		filtered := []string{}
		for _, id := range ids {
			if fixtureIDs[id] {
				filtered = append(filtered, id)
			}
		}
		return filtered
	}
	if len(run()) != 0 {
		t.Fatal("first sale without delivery was activated")
	}
	prior := create()
	if _, err = pool.Exec(ctx, `INSERT INTO app.receipt_confirmations(credit_request_id,buyer_user_id,state) VALUES($1,$2,'confirmed')`, prior.Request.ID, buyer); err != nil {
		t.Fatal(err)
	}
	if len(run()) != 0 {
		t.Fatal("missing delivery activated a sale")
	}
	notify := func(v View, old bool) {
		t.Helper()
		n := identifier.New()
		age := time.Now().UTC()
		if old {
			age = age.Add(-4 * 24 * time.Hour)
		}
		if _, err = pool.Exec(ctx, `INSERT INTO app.notifications(id,recipient_id,channel,template,template_version,event_reference,state,body) SELECT $1,$2,'email','GoodsReleased','v1','outbox:'||id::text,'delivered','fixture' FROM app.outbox_events WHERE idempotency_key=$3`, n, buyer, "goods-release-notification:"+v.Release.ID); err != nil {
			t.Fatal(err)
		}
		if _, err = pool.Exec(ctx, `INSERT INTO app.notification_delivery_receipts(channel,event_id,payload_hash,notification_id,received_at) VALUES('email',$1,'fixture',$2,$3)`, n, n, age); err != nil {
			t.Fatal(err)
		}
	}
	recent := create()
	notify(recent, false)
	issue := create()
	notify(issue, true)
	if _, err = pool.Exec(ctx, `INSERT INTO app.receipt_confirmations(credit_request_id,buyer_user_id,state,issue_reason) VALUES($1,$2,'issue_raised','missing goods')`, issue.Request.ID, buyer); err != nil {
		t.Fatal(err)
	}
	notify(eligible, true)
	var wg sync.WaitGroup
	for range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := newWorker().AutoActivateMatured(ctx, time.Now())
			if err != nil {
				t.Errorf("cold worker: %v", err)
			}
		}()
	}
	wg.Wait()
	var evidence, obligations, receipts, schedules, notices int
	if err = pool.QueryRow(ctx, `SELECT (SELECT count(*) FROM app.system_acceptances WHERE credit_request_id=$1),(SELECT count(*) FROM app.obligations WHERE credit_request_id=$1),(SELECT count(*) FROM app.receipt_confirmations WHERE credit_request_id=$1),(SELECT count(*) FROM app.repayment_schedules WHERE obligation_id IN(SELECT id FROM app.obligations WHERE credit_request_id=$1)),(SELECT count(*) FROM app.outbox_events WHERE aggregate_id=$1::text AND payload->>'event'='SYSTEM_ACCEPTANCE')`, eligible.Request.ID).Scan(&evidence, &obligations, &receipts, &schedules, &notices); err != nil {
		t.Fatal(err)
	}
	if evidence != 1 || obligations != 1 || receipts != 0 || schedules != 1 || notices != 1 {
		t.Fatalf("incomplete or fabricated activation: %d %d %d %d %d", evidence, obligations, receipts, schedules, notices)
	}
	if len(run()) != 0 {
		t.Fatal("blocked sale or duplicate activated")
	}
	var mayWrite bool
	if err = pool.QueryRow(ctx, `SELECT has_table_privilege('kredit_worker','app.receipt_confirmations','INSERT')`).Scan(&mayWrite); err != nil || mayWrite {
		t.Fatalf("buyer evidence protection changed: %v %v", mayWrite, err)
	}
}
