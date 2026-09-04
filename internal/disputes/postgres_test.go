package disputes

import (
	"context"
	"encoding/json"
	"fmt"
	"kredit/internal/schedules"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPostgresDisputeDecisionIsAtomicAndRestartSafe(t *testing.T) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" || os.Getenv("KREDIT_INTEGRATION") != "1" {
		t.Skip("KREDIT_INTEGRATION=1 and DATABASE_URL are required")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	var openerID, reviewerID, organizationID, requestID, agreementID, activationID, obligationID string
	for index, target := range []*string{&openerID, &reviewerID} {
		if err := pool.QueryRow(ctx, `INSERT INTO app.users(normalized_email) VALUES($1) RETURNING id::text`, fmt.Sprintf("dispute-%d-%d@example.test", time.Now().UnixNano(), index)).Scan(target); err != nil {
			t.Fatal(err)
		}
	}
	if err := pool.QueryRow(ctx, `INSERT INTO app.organizations(legal_name,business_type,business_address,industry) VALUES('Dispute Test','limited_company','test','test') RETURNING id::text`).Scan(&organizationID); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO app.credit_requests(supplier_organization_id,buyer_user_id,buyer_business_id,principal_kobo,goods_description,due_date,collection_at,state,created_by) VALUES($1::uuid,$2::uuid,gen_random_uuid(),10000,'goods',current_date+7,now()+interval '7 days','ACTIVE',$2::uuid) RETURNING id::text`, organizationID, openerID).Scan(&requestID); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO app.agreement_versions(credit_request_id,version,canonical_json,document_hash,terms_version,privacy_version,created_by) VALUES($1::uuid,1,'{}',$2,'v1','v1',$3::uuid) RETURNING id::text`, requestID, "dispute-"+requestID, openerID).Scan(&agreementID); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO ledger.transactions(event_type,reference_type,reference_id,idempotency_key,effective_at) VALUES('test','credit_request',$1,$2,now()) RETURNING id::text`, requestID, "dispute-activation-"+requestID).Scan(&activationID); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO app.obligations(credit_request_id,agreement_version_id,supplier_organization_id,buyer_business_id,principal_kobo,currency,lifecycle_status,payment_status,outstanding_kobo,base_fee_kobo,ledger_transaction_id,activated_at) SELECT $1::uuid,$2::uuid,$3::uuid,buyer_business_id,principal_kobo,'NGN','ACTIVE','UNPAID',principal_kobo,50,$4::uuid,now() FROM app.credit_requests WHERE id=$1::uuid RETURNING id::text`, requestID, agreementID, organizationID, activationID).Scan(&obligationID); err != nil {
		t.Fatal(err)
	}
	payload, _ := json.Marshal(map[string]any{"request": map[string]any{"id": requestID, "version": 1}, "obligation": map[string]any{"id": obligationID, "outstanding_kobo": 10000, "payment_status": "UNPAID"}})
	if _, err := pool.Exec(ctx, `INSERT INTO app.credit_aggregate_snapshots(credit_request_id,supplier_organization_id,buyer_user_id,aggregate,version) VALUES($1,$2,$3,$4,1)`, requestID, organizationID, openerID, payload); err != nil {
		t.Fatal(err)
	}
	scheduleStore := schedules.NewPostgresStore(pool)
	if _, _, err = scheduleStore.CreateDefault(obligationID, 10000, time.Now().AddDate(0, 0, 7).Format("2006-01-02"), time.Now().AddDate(0, 0, 7), 0); err != nil {
		t.Fatal(err)
	}
	var disputeID string
	defer func() {
		_, _ = pool.Exec(ctx, `DELETE FROM app.dispute_decisions WHERE dispute_id=$1::uuid`, disputeID)
		_, _ = pool.Exec(ctx, `DELETE FROM app.dispute_evidence WHERE dispute_id=$1::uuid`, disputeID)
		_, _ = pool.Exec(ctx, `DELETE FROM app.disputes WHERE id=$1::uuid`, disputeID)
		_, _ = pool.Exec(ctx, `DELETE FROM ledger.postings WHERE transaction_id IN(SELECT id FROM ledger.transactions WHERE reference_id=$1 OR id=$2::uuid)`, disputeID, activationID)
		_, _ = pool.Exec(ctx, `DELETE FROM ledger.transactions WHERE reference_id=$1 OR id=$2::uuid`, disputeID, activationID)
		_, _ = pool.Exec(ctx, `DELETE FROM app.credit_aggregate_snapshots WHERE credit_request_id=$1`, requestID)
		_, _ = pool.Exec(ctx, `DELETE FROM app.obligations WHERE id=$1::uuid`, obligationID)
		_, _ = pool.Exec(ctx, `DELETE FROM app.agreement_versions WHERE id=$1::uuid`, agreementID)
		_, _ = pool.Exec(ctx, `DELETE FROM app.credit_requests WHERE id=$1::uuid`, requestID)
		_, _ = pool.Exec(ctx, `DELETE FROM app.organizations WHERE id=$1::uuid`, organizationID)
		_, _ = pool.Exec(ctx, `DELETE FROM app.users WHERE id=$1::uuid OR id=$2::uuid`, openerID, reviewerID)
	}()
	store := NewPostgresStore(pool, nil)
	opened, err := store.Open(OpenInput{ObligationID: obligationID, OpenedBy: openerID, DisputedAmountKobo: 2500, Reason: "Quantity mismatch"})
	if err != nil {
		t.Fatal(err)
	}
	disputeID = opened.ID
	if _, err := store.AddEvidence(disputeID, openerID, "", "Delivery was short"); err != nil {
		t.Fatal(err)
	}
	updated, decision, err := store.Decide(DecideInput{DisputeID: disputeID, ReviewerID: reviewerID, Outcome: "buyer_supported", AdjustmentKobo: 2500, RemainingDisputedKobo: 0, Reason: "Evidence confirmed"})
	if err != nil {
		t.Fatal(err)
	}
	if updated.State != StateResolved || decision.AdjustmentKobo != 2500 {
		t.Fatalf("unexpected decision: %+v %+v", updated, decision)
	}
	var normalized, snapshot int64
	if err := pool.QueryRow(ctx, `SELECT outstanding_kobo FROM app.obligations WHERE id=$1::uuid`, obligationID).Scan(&normalized); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT (aggregate#>>'{obligation,outstanding_kobo}')::bigint FROM app.credit_aggregate_snapshots WHERE credit_request_id=$1`, requestID).Scan(&snapshot); err != nil {
		t.Fatal(err)
	}
	_, items, scheduleErr := scheduleStore.GetForObligation(obligationID)
	if scheduleErr != nil || len(items) != 1 || items[0].PrincipalDueKobo-items[0].AllocatedKobo != 7500 {
		t.Fatalf("adjustment left collectible excess: %+v %v", items, scheduleErr)
	}
	if normalized != 7500 || snapshot != 7500 {
		t.Fatalf("adjustment not atomic: normalized=%d snapshot=%d", normalized, snapshot)
	}
	loaded, evidence, decisions, err := NewPostgresStore(pool, nil).Get(disputeID)
	if err != nil || loaded.State != StateResolved || len(evidence) != 1 || len(decisions) != 1 {
		t.Fatalf("restart load failed: %v %+v %d %d", err, loaded, len(evidence), len(decisions))
	}
}
