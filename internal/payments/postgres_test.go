package payments

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	"kredit/internal/outbox"
	"kredit/internal/schedules"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPostgresPaymentIsAtomicAndRestartSafe(t *testing.T) {
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
	var userID, organizationID, requestID, agreementID, activationTransactionID, obligationID string
	email := fmt.Sprintf("payment-test-%d@example.test", time.Now().UnixNano())
	if err := pool.QueryRow(ctx, `INSERT INTO app.users(normalized_email) VALUES($1) RETURNING id::text`, email).Scan(&userID); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO app.organizations(legal_name,business_type,business_address,industry) VALUES('Payment Test','limited_company','test','test') RETURNING id::text`).Scan(&organizationID); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO app.credit_requests(supplier_organization_id,buyer_user_id,buyer_business_id,principal_kobo,goods_description,due_date,collection_at,state,created_by) VALUES($1::uuid,$2::uuid,gen_random_uuid(),10000,'goods',current_date+7,now()+interval '7 days','ACTIVE',$2::uuid) RETURNING id::text`, organizationID, userID).Scan(&requestID); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO app.agreement_versions(credit_request_id,version,canonical_json,document_hash,terms_version,privacy_version,created_by) VALUES($1::uuid,1,'{}',$2,'v1','v1',$3::uuid) RETURNING id::text`, requestID, "payment-test-"+requestID, userID).Scan(&agreementID); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO ledger.transactions(event_type,reference_type,reference_id,idempotency_key,effective_at) VALUES('test','credit_request',$1,$2,now()) RETURNING id::text`, requestID, "payment-activation-"+requestID).Scan(&activationTransactionID); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO app.obligations(credit_request_id,agreement_version_id,supplier_organization_id,buyer_business_id,principal_kobo,currency,lifecycle_status,payment_status,outstanding_kobo,base_fee_kobo,ledger_transaction_id,activated_at) SELECT $1::uuid,$2::uuid,$3::uuid,buyer_business_id,principal_kobo,'NGN','ACTIVE','UNPAID',principal_kobo,50,$4::uuid,now() FROM app.credit_requests WHERE id=$1::uuid RETURNING id::text`, requestID, agreementID, organizationID, activationTransactionID).Scan(&obligationID); err != nil {
		t.Fatal(err)
	}
	view := map[string]any{"request": map[string]any{"id": requestID, "version": 1}, "obligation": map[string]any{"id": obligationID, "outstanding_kobo": 10000, "payment_status": "UNPAID"}}
	payload, _ := json.Marshal(view)
	if _, err := pool.Exec(ctx, `INSERT INTO app.credit_aggregate_snapshots(credit_request_id,supplier_organization_id,buyer_user_id,aggregate,version) VALUES($1,$2,$3,$4,1)`, requestID, organizationID, userID, payload); err != nil {
		t.Fatal(err)
	}
	paymentKey := "payment-test-" + requestID
	defer func() {
		_, _ = pool.Exec(ctx, `DELETE FROM app.outbox_events WHERE idempotency_key=$1`, `payment:`+paymentKey)
		_, _ = pool.Exec(ctx, `DELETE FROM app.fees WHERE obligation_id=$1::uuid`, obligationID)
		_, _ = pool.Exec(ctx, `DELETE FROM app.payment_allocations WHERE obligation_id=$1::uuid`, obligationID)
		_, _ = pool.Exec(ctx, `DELETE FROM app.payments WHERE obligation_id=$1::uuid`, obligationID)
		_, _ = pool.Exec(ctx, `DELETE FROM app.schedule_items WHERE schedule_id IN(SELECT id FROM app.repayment_schedules WHERE obligation_id=$1::uuid)`, obligationID)
		_, _ = pool.Exec(ctx, `DELETE FROM app.repayment_schedules WHERE obligation_id=$1::uuid`, obligationID)
		_, _ = pool.Exec(ctx, `DELETE FROM app.credit_aggregate_snapshots WHERE credit_request_id=$1`, requestID)
		_, _ = pool.Exec(ctx, `DELETE FROM app.obligations WHERE id=$1::uuid`, obligationID)
		_, _ = pool.Exec(ctx, `DELETE FROM app.agreement_versions WHERE id=$1::uuid`, agreementID)
		_, _ = pool.Exec(ctx, `DELETE FROM app.credit_requests WHERE id=$1::uuid`, requestID)
		_, _ = pool.Exec(ctx, `DELETE FROM ledger.postings WHERE transaction_id IN(SELECT id FROM ledger.transactions WHERE idempotency_key=$1 OR id=$2::uuid)`, `payment:`+paymentKey, activationTransactionID)
		_, _ = pool.Exec(ctx, `DELETE FROM ledger.transactions WHERE idempotency_key=$1 OR id=$2::uuid`, `payment:`+paymentKey, activationTransactionID)
		_, _ = pool.Exec(ctx, `DELETE FROM app.organizations WHERE id=$1::uuid`, organizationID)
		_, _ = pool.Exec(ctx, `DELETE FROM app.users WHERE id=$1::uuid`, userID)
	}()
	scheduleStore := schedules.NewPostgresStore(pool)
	if _, _, err := scheduleStore.CreateDefault(obligationID, 10000, time.Now().UTC().AddDate(0, 0, 7).Format("2006-01-02"), time.Now().UTC().AddDate(0, 0, 7), 0); err != nil {
		t.Fatal(err)
	}
	store := NewPostgresStore(pool, outbox.NewStore(pool), nil)
	payment, allocation, err := store.Record(RecordInput{ObligationID: obligationID, SourceType: SourceVoluntary, AmountKobo: 2500, RecordedBy: userID, IdempotencyKey: paymentKey})
	if err != nil {
		t.Fatal(err)
	}
	if allocation.AmountKobo != 2500 || payment.State != StateRecognized {
		t.Fatalf("unexpected payment: %+v %+v", payment, allocation)
	}
	duplicate, _, err := store.Record(RecordInput{ObligationID: obligationID, SourceType: SourceVoluntary, AmountKobo: 2500, RecordedBy: userID, IdempotencyKey: paymentKey})
	if err != nil || duplicate.ID != payment.ID {
		t.Fatalf("idempotent replay failed: %v %+v", err, duplicate)
	}
	var outstanding int64
	var snapshotOutstanding int64
	if err := pool.QueryRow(ctx, `SELECT outstanding_kobo FROM app.obligations WHERE id=$1::uuid`, obligationID).Scan(&outstanding); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT (aggregate#>>'{obligation,outstanding_kobo}')::bigint FROM app.credit_aggregate_snapshots WHERE credit_request_id=$1`, requestID).Scan(&snapshotOutstanding); err != nil {
		t.Fatal(err)
	}
	if outstanding != 7500 || snapshotOutstanding != 7500 {
		t.Fatalf("balances were not committed atomically: normalized=%d snapshot=%d", outstanding, snapshotOutstanding)
	}
	loaded, err := NewPostgresStore(pool, outbox.NewStore(pool), nil).Get(payment.ID)
	if err != nil || loaded.ID != payment.ID {
		t.Fatalf("restart-safe load failed: %v %+v", err, loaded)
	}
}
