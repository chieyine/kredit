package collections

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	"kredit/internal/db"
	"kredit/internal/identifier"
	"kredit/internal/ledger"
	"kredit/internal/outbox"
	"kredit/internal/payments"
	"kredit/internal/schedules"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPostgresCollectionIsRestartSafeAndFinanciallyIdempotent(t *testing.T) {
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
	email := fmt.Sprintf("collection-test-%d@example.test", time.Now().UnixNano())
	if err := pool.QueryRow(ctx, `INSERT INTO app.users(normalized_email) VALUES($1) RETURNING id::text`, email).Scan(&userID); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO app.organizations(legal_name,business_type,business_address,industry) VALUES('Collection Test','limited_company','test','test') RETURNING id::text`).Scan(&organizationID); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO app.credit_requests(supplier_organization_id,buyer_user_id,buyer_business_id,principal_kobo,goods_description,due_date,collection_at,state,created_by) VALUES($1::uuid,$2::uuid,gen_random_uuid(),10000,'goods',current_date,now()-interval '1 day','ACTIVE',$2::uuid) RETURNING id::text`, organizationID, userID).Scan(&requestID); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO app.agreement_versions(credit_request_id,version,canonical_json,document_hash,terms_version,privacy_version,created_by) VALUES($1::uuid,1,'{}',$2,'v1','v1',$3::uuid) RETURNING id::text`, requestID, "collection-test-"+requestID, userID).Scan(&agreementID); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO ledger.transactions(event_type,reference_type,reference_id,idempotency_key,effective_at) VALUES('test','credit_request',$1,$2,now()) RETURNING id::text`, requestID, "collection-activation-"+requestID).Scan(&activationTransactionID); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO app.obligations(credit_request_id,agreement_version_id,supplier_organization_id,buyer_business_id,principal_kobo,currency,lifecycle_status,payment_status,outstanding_kobo,base_fee_kobo,ledger_transaction_id,activated_at) SELECT $1::uuid,$2::uuid,$3::uuid,buyer_business_id,principal_kobo,'NGN','ACTIVE','UNPAID',principal_kobo,50,$4::uuid,now() FROM app.credit_requests WHERE id=$1::uuid RETURNING id::text`, requestID, agreementID, organizationID, activationTransactionID).Scan(&obligationID); err != nil {
		t.Fatal(err)
	}
	ctx = db.WithTenantContext(ctx, userID, organizationID)
	view := map[string]any{"request": map[string]any{"id": requestID, "version": 1}, "obligation": map[string]any{"id": obligationID, "outstanding_kobo": 10000, "payment_status": "UNPAID"}}
	payload, _ := json.Marshal(view)
	if _, err := pool.Exec(ctx, `INSERT INTO app.credit_aggregate_snapshots(credit_request_id,supplier_organization_id,buyer_user_id,aggregate,version) VALUES($1,$2,$3,$4,1)`, requestID, organizationID, userID, payload); err != nil {
		t.Fatal(err)
	}
	key := "collection-test-" + requestID
	defer func() {
		_, _ = pool.Exec(ctx, `DELETE FROM app.outbox_events WHERE idempotency_key=$1`, "payment:collection-event:provider-event-"+identifier.FromKey("collection-attempt:"+obligationID, key))
		_, _ = pool.Exec(ctx, `DELETE FROM app.fees WHERE obligation_id=$1::uuid`, obligationID)
		_, _ = pool.Exec(ctx, `DELETE FROM app.payment_allocations WHERE obligation_id=$1::uuid`, obligationID)
		_, _ = pool.Exec(ctx, `DELETE FROM app.payments WHERE obligation_id=$1::uuid`, obligationID)
		_, _ = pool.Exec(ctx, `DELETE FROM app.collection_attempt_index WHERE obligation_id=$1::uuid`, obligationID)
		_, _ = pool.Exec(ctx, `DELETE FROM app.collection_attempts WHERE obligation_id=$1::uuid`, obligationID)
		_, _ = pool.Exec(ctx, `DELETE FROM app.collection_reservations WHERE obligation_id=$1::uuid`, obligationID)
		_, _ = pool.Exec(ctx, `DELETE FROM app.collection_aggregate_snapshots WHERE obligation_id=$1::uuid`, obligationID)
		_, _ = pool.Exec(ctx, `DELETE FROM app.schedule_items WHERE schedule_id IN(SELECT id FROM app.repayment_schedules WHERE obligation_id=$1::uuid)`, obligationID)
		_, _ = pool.Exec(ctx, `DELETE FROM app.repayment_schedules WHERE obligation_id=$1::uuid`, obligationID)
		_, _ = pool.Exec(ctx, `DELETE FROM app.credit_aggregate_snapshots WHERE credit_request_id=$1`, requestID)
		_, _ = pool.Exec(ctx, `DELETE FROM app.obligations WHERE id=$1::uuid`, obligationID)
		_, _ = pool.Exec(ctx, `DELETE FROM app.agreement_versions WHERE id=$1::uuid`, agreementID)
		_, _ = pool.Exec(ctx, `DELETE FROM app.credit_requests WHERE id=$1::uuid`, requestID)
		_, _ = pool.Exec(ctx, `DELETE FROM ledger.postings WHERE transaction_id IN(SELECT id FROM ledger.transactions WHERE reference_id=$1 OR id=$2::uuid)`, obligationID, activationTransactionID)
		_, _ = pool.Exec(ctx, `DELETE FROM ledger.transactions WHERE reference_id=$1 OR id=$2::uuid`, obligationID, activationTransactionID)
		_, _ = pool.Exec(ctx, `DELETE FROM app.organizations WHERE id=$1::uuid`, organizationID)
		_, _ = pool.Exec(ctx, `DELETE FROM app.users WHERE id=$1::uuid`, userID)
	}()

	if _, _, err := schedules.NewPostgresStore(pool).CreateDefault(obligationID, 10000, time.Now().UTC().Format("2006-01-02"), time.Now().UTC().Add(-time.Hour), 0); err != nil {
		t.Fatal(err)
	}
	provider := NewMockProvider("collection-secret")
	paymentStore := &tenantPaymentStore{PostgresStore: payments.NewPostgresStore(pool, outbox.NewStore(pool), nil), ctx: ctx}
	snapshot := func(string) (ObligationSnapshot, error) {
		var outstanding int64
		if err := pool.QueryRow(ctx, `SELECT outstanding_kobo FROM app.obligations WHERE id=$1::uuid`, obligationID).Scan(&outstanding); err != nil {
			return ObligationSnapshot{}, err
		}
		return ObligationSnapshot{ID: obligationID, BuyerUserID: userID, Currency: "NGN", Active: true, OutstandingKobo: ledger.Money(outstanding), MandateActive: true, MandateRemainingKobo: 10000, CollectionEnabled: true, ProviderSupported: true, Version: 1}, nil
	}
	base := NewEngine(provider, paymentStore, snapshot, func(string, time.Time) (ledger.Money, error) { return 10000, nil })
	engine := NewPostgresEngine(pool, base)
	attempt, err := engine.Start(ctx, obligationID, key, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	if attempt.State != AttemptSucceeded || attempt.SucceededAmountKobo != 10000 {
		t.Fatalf("unexpected collection attempt: %+v", attempt)
	}

	restarted := NewPostgresEngine(pool, NewEngine(provider, paymentStore, snapshot, func(string, time.Time) (ledger.Money, error) { return 10000, nil }))
	loaded, ok := restarted.GetAttemptContext(ctx, attempt.ID)
	if !ok || loaded.ID != attempt.ID || loaded.State != AttemptSucceeded {
		t.Fatalf("restart-safe load failed: ok=%t attempt=%+v", ok, loaded)
	}
	duplicate, err := restarted.Start(ctx, obligationID, key, time.Now().UTC())
	if err != nil || duplicate.ID != attempt.ID {
		t.Fatalf("idempotent replay failed: %v %+v", err, duplicate)
	}
	var outstanding, paymentCount int64
	if err := pool.QueryRow(ctx, `SELECT outstanding_kobo FROM app.obligations WHERE id=$1::uuid`, obligationID).Scan(&outstanding); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM app.payments WHERE obligation_id=$1::uuid`, obligationID).Scan(&paymentCount); err != nil {
		t.Fatal(err)
	}
	if outstanding != 0 || paymentCount != 1 {
		t.Fatalf("collection financial effect was not exactly once: outstanding=%d payments=%d", outstanding, paymentCount)
	}
}
