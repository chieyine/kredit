package payments

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"kredit/internal/db"
	"kredit/internal/ledger"
	"kredit/internal/outbox"
	"kredit/internal/schedules"

	"github.com/jackc/pgx/v5/pgxpool"
)

type paymentFixture struct {
	pool                    *pgxpool.Pool
	ctx                     context.Context
	userID                  string
	organizationID          string
	requestID               string
	agreementID             string
	activationTransactionID string
	obligationID            string
	paymentKeyPrefix        string
}

func newPaymentFixture(t *testing.T, principal int64) *paymentFixture {
	t.Helper()
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" || os.Getenv("KREDIT_INTEGRATION") != "1" {
		t.Skip("KREDIT_INTEGRATION=1 and DATABASE_URL are required")
	}
	base := context.Background()
	pool, err := pgxpool.New(base, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	f := &paymentFixture{pool: pool, paymentKeyPrefix: fmt.Sprintf("phase3-%d", time.Now().UnixNano())}
	email := f.paymentKeyPrefix + "@example.test"
	if err := pool.QueryRow(base, `INSERT INTO app.users(normalized_email) VALUES($1) RETURNING id::text`, email).Scan(&f.userID); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(base, `INSERT INTO app.organizations(legal_name,business_type,business_address,industry) VALUES($1,'limited_company','test','test') RETURNING id::text`, "Phase 3 "+f.paymentKeyPrefix).Scan(&f.organizationID); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(base, `INSERT INTO app.credit_requests(supplier_organization_id,buyer_user_id,buyer_business_id,principal_kobo,goods_description,due_date,collection_at,state,created_by) VALUES($1::uuid,$2::uuid,gen_random_uuid(),$3,'goods',current_date+7,now()+interval '7 days','ACTIVE',$2::uuid) RETURNING id::text`, f.organizationID, f.userID, principal).Scan(&f.requestID); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(base, `INSERT INTO app.agreement_versions(credit_request_id,version,canonical_json,document_hash,terms_version,privacy_version,created_by) VALUES($1::uuid,1,'{}',$2,'v1','v1',$3::uuid) RETURNING id::text`, f.requestID, "phase3-agreement-"+f.requestID, f.userID).Scan(&f.agreementID); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(base, `INSERT INTO ledger.transactions(event_type,reference_type,reference_id,idempotency_key,effective_at) VALUES('test','credit_request',$1,$2,now()) RETURNING id::text`, f.requestID, "phase3-activation-"+f.requestID).Scan(&f.activationTransactionID); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(base, `INSERT INTO app.obligations(credit_request_id,agreement_version_id,supplier_organization_id,buyer_business_id,principal_kobo,currency,lifecycle_status,payment_status,outstanding_kobo,base_fee_kobo,ledger_transaction_id,activated_at) SELECT $1::uuid,$2::uuid,$3::uuid,buyer_business_id,principal_kobo,'NGN','ACTIVE','UNPAID',principal_kobo,50,$4::uuid,now() FROM app.credit_requests WHERE id=$1::uuid RETURNING id::text`, f.requestID, f.agreementID, f.organizationID, f.activationTransactionID).Scan(&f.obligationID); err != nil {
		t.Fatal(err)
	}
	view := map[string]any{"request": map[string]any{"id": f.requestID, "version": 1}, "obligation": map[string]any{"id": f.obligationID, "outstanding_kobo": principal, "payment_status": "UNPAID"}}
	payload, _ := json.Marshal(view)
	if _, err := pool.Exec(base, `INSERT INTO app.credit_aggregate_snapshots(credit_request_id,supplier_organization_id,buyer_user_id,aggregate,version) VALUES($1,$2,$3,$4,1)`, f.requestID, f.organizationID, f.userID, payload); err != nil {
		t.Fatal(err)
	}
	if _, _, err := schedules.NewPostgresStore(pool).CreateDefault(f.obligationID, ledger.Money(principal), time.Now().UTC().AddDate(0, 0, 7).Format("2006-01-02"), time.Now().UTC().AddDate(0, 0, 7), 0); err != nil {
		t.Fatal(err)
	}
	f.ctx = db.WithTenantContext(base, f.userID, f.organizationID)
	t.Cleanup(func() {
		ctx := context.Background()
		_, _ = pool.Exec(ctx, `DELETE FROM app.outbox_events WHERE aggregate_id=$1`, f.obligationID)
		_, _ = pool.Exec(ctx, `DELETE FROM app.fees WHERE obligation_id=$1::uuid`, f.obligationID)
		_, _ = pool.Exec(ctx, `DELETE FROM app.payment_allocations WHERE obligation_id=$1::uuid`, f.obligationID)
		_, _ = pool.Exec(ctx, `DELETE FROM app.payments WHERE obligation_id=$1::uuid`, f.obligationID)
		_, _ = pool.Exec(ctx, `DELETE FROM app.schedule_items WHERE schedule_id IN(SELECT id FROM app.repayment_schedules WHERE obligation_id=$1::uuid)`, f.obligationID)
		_, _ = pool.Exec(ctx, `DELETE FROM app.repayment_schedules WHERE obligation_id=$1::uuid`, f.obligationID)
		_, _ = pool.Exec(ctx, `DELETE FROM app.credit_aggregate_snapshots WHERE credit_request_id=$1`, f.requestID)
		_, _ = pool.Exec(ctx, `DELETE FROM app.obligations WHERE id=$1::uuid`, f.obligationID)
		_, _ = pool.Exec(ctx, `DELETE FROM app.agreement_versions WHERE id=$1::uuid`, f.agreementID)
		_, _ = pool.Exec(ctx, `DELETE FROM app.credit_requests WHERE id=$1::uuid`, f.requestID)
		_, _ = pool.Exec(ctx, `DELETE FROM ledger.postings WHERE transaction_id IN(SELECT id FROM ledger.transactions WHERE reference_id=$1 OR id=$2::uuid)`, f.obligationID, f.activationTransactionID)
		_, _ = pool.Exec(ctx, `DELETE FROM ledger.transactions WHERE reference_id=$1 OR id=$2::uuid`, f.obligationID, f.activationTransactionID)
		_, _ = pool.Exec(ctx, `DELETE FROM app.organizations WHERE id=$1::uuid`, f.organizationID)
		_, _ = pool.Exec(ctx, `DELETE FROM app.users WHERE id=$1::uuid`, f.userID)
		pool.Close()
	})
	return f
}

func (f *paymentFixture) store() *PostgresStore {
	return NewPostgresStore(f.pool, outbox.NewStore(f.pool), nil)
}

func TestPostgresPaymentIsAtomicIdempotentAndRestartSafe(t *testing.T) {
	f := newPaymentFixture(t, 10_000)
	store := f.store()
	key := f.paymentKeyPrefix + "-partial"
	input := RecordInput{ObligationID: f.obligationID, SourceType: SourceVoluntary, AmountKobo: 2_500, RecordedBy: f.userID, IdempotencyKey: key}
	payment, allocation, err := store.RecordContext(f.ctx, input)
	if err != nil {
		t.Fatal(err)
	}
	if allocation.AmountKobo != 2_500 || payment.State != StateRecognized {
		t.Fatalf("unexpected payment: %+v %+v", payment, allocation)
	}
	duplicate, duplicateAllocation, err := store.RecordContext(f.ctx, input)
	if err != nil || duplicate.ID != payment.ID || duplicateAllocation.PaymentID != allocation.PaymentID || duplicateAllocation.AmountKobo != allocation.AmountKobo {
		t.Fatalf("idempotent replay failed: %v %+v %+v", err, duplicate, duplicateAllocation)
	}
	if _, _, err = store.RecordContext(f.ctx, RecordInput{ObligationID: f.obligationID, SourceType: SourceVoluntary, AmountKobo: 2_501, RecordedBy: f.userID, IdempotencyKey: key}); err == nil {
		t.Fatal("idempotency key reuse with a different amount was accepted")
	}
	var outstanding, snapshotOutstanding int64
	if err := f.pool.QueryRow(f.ctx, `SELECT outstanding_kobo FROM app.obligations WHERE id=$1::uuid`, f.obligationID).Scan(&outstanding); err != nil {
		t.Fatal(err)
	}
	if err := f.pool.QueryRow(f.ctx, `SELECT (aggregate#>>'{obligation,outstanding_kobo}')::bigint FROM app.credit_aggregate_snapshots WHERE credit_request_id=$1`, f.requestID).Scan(&snapshotOutstanding); err != nil {
		t.Fatal(err)
	}
	if outstanding != 7_500 || snapshotOutstanding != 7_500 {
		t.Fatalf("balances were not committed atomically: normalized=%d snapshot=%d", outstanding, snapshotOutstanding)
	}
	loaded, err := f.store().GetContext(f.ctx, payment.ID)
	if err != nil || loaded.ID != payment.ID {
		t.Fatalf("restart-safe load failed: %v %+v", err, loaded)
	}
}

func TestPostgresPaymentPartialFullReversalAndOverpaymentInvariants(t *testing.T) {
	f := newPaymentFixture(t, 10_000)
	store := f.store()
	partial, _, err := store.RecordContext(f.ctx, RecordInput{ObligationID: f.obligationID, SourceType: SourceVoluntary, AmountKobo: 4_000, RecordedBy: f.userID, IdempotencyKey: f.paymentKeyPrefix + "-p1"})
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err = store.RecordContext(f.ctx, RecordInput{ObligationID: f.obligationID, SourceType: SourceVoluntary, AmountKobo: 6_001, RecordedBy: f.userID, IdempotencyKey: f.paymentKeyPrefix + "-over"}); err == nil {
		t.Fatal("overpayment was accepted")
	}
	full, _, err := store.RecordContext(f.ctx, RecordInput{ObligationID: f.obligationID, SourceType: SourceVoluntary, AmountKobo: 6_000, RecordedBy: f.userID, IdempotencyKey: f.paymentKeyPrefix + "-p2"})
	if err != nil {
		t.Fatal(err)
	}
	var outstanding int64
	var status string
	if err := f.pool.QueryRow(f.ctx, `SELECT outstanding_kobo,payment_status FROM app.obligations WHERE id=$1::uuid`, f.obligationID).Scan(&outstanding, &status); err != nil {
		t.Fatal(err)
	}
	if outstanding != 0 || status != "PAID" {
		t.Fatalf("full payment invariant failed: outstanding=%d status=%s", outstanding, status)
	}
	if _, err := store.ReverseContext(f.ctx, full.ID, f.userID, "bank reversal"); err != nil {
		t.Fatal(err)
	}
	if err := f.pool.QueryRow(f.ctx, `SELECT outstanding_kobo,payment_status FROM app.obligations WHERE id=$1::uuid`, f.obligationID).Scan(&outstanding, &status); err != nil {
		t.Fatal(err)
	}
	if outstanding != 6_000 || status != "PARTIALLY_PAID" {
		t.Fatalf("reversal did not restore debt: outstanding=%d status=%s", outstanding, status)
	}
	if _, err := store.ReverseContext(f.ctx, partial.ID, f.userID, "receipt correction"); err != nil {
		t.Fatal(err)
	}
	if err := f.pool.QueryRow(f.ctx, `SELECT outstanding_kobo,payment_status FROM app.obligations WHERE id=$1::uuid`, f.obligationID).Scan(&outstanding, &status); err != nil {
		t.Fatal(err)
	}
	if outstanding != 10_000 || status != "UNPAID" {
		t.Fatalf("all reversals did not restore original debt: outstanding=%d status=%s", outstanding, status)
	}
	var paymentSum, allocationSum int64
	if err := f.pool.QueryRow(f.ctx, `SELECT COALESCE(SUM(amount_kobo) FILTER (WHERE state='recognized'),0) FROM app.payments WHERE obligation_id=$1::uuid`, f.obligationID).Scan(&paymentSum); err != nil {
		t.Fatal(err)
	}
	if err := f.pool.QueryRow(f.ctx, `SELECT COALESCE(SUM(pa.amount_kobo),0) FROM app.payment_allocations pa JOIN app.payments p ON p.id=pa.payment_id WHERE pa.obligation_id=$1::uuid AND p.state='recognized'`, f.obligationID).Scan(&allocationSum); err != nil {
		t.Fatal(err)
	}
	if paymentSum != allocationSum {
		t.Fatalf("recognized payment/allocation sums diverged: payments=%d allocations=%d", paymentSum, allocationSum)
	}
}

func TestPostgresConcurrentPaymentsNeverOverAllocate(t *testing.T) {
	f := newPaymentFixture(t, 10_000)
	store := f.store()
	amounts := []int64{7_000, 7_000}
	results := make(chan error, len(amounts))
	var wg sync.WaitGroup
	for i, amount := range amounts {
		wg.Add(1)
		go func(i int, amount int64) {
			defer wg.Done()
			_, _, err := store.RecordContext(f.ctx, RecordInput{ObligationID: f.obligationID, SourceType: SourceVoluntary, AmountKobo: ledger.Money(amount), RecordedBy: f.userID, IdempotencyKey: fmt.Sprintf("%s-race-%d", f.paymentKeyPrefix, i)})
			results <- err
		}(i, amount)
	}
	wg.Wait()
	close(results)
	succeeded := 0
	for err := range results {
		if err == nil {
			succeeded++
		}
	}
	if succeeded != 1 {
		t.Fatalf("expected exactly one concurrent payment to succeed, got %d", succeeded)
	}
	var outstanding, recognized, allocated int64
	if err := f.pool.QueryRow(f.ctx, `SELECT outstanding_kobo FROM app.obligations WHERE id=$1::uuid`, f.obligationID).Scan(&outstanding); err != nil {
		t.Fatal(err)
	}
	if err := f.pool.QueryRow(f.ctx, `SELECT COALESCE(SUM(amount_kobo),0) FROM app.payments WHERE obligation_id=$1::uuid AND state='recognized'`, f.obligationID).Scan(&recognized); err != nil {
		t.Fatal(err)
	}
	if err := f.pool.QueryRow(f.ctx, `SELECT COALESCE(SUM(amount_kobo),0) FROM app.payment_allocations WHERE obligation_id=$1::uuid`, f.obligationID).Scan(&allocated); err != nil {
		t.Fatal(err)
	}
	if outstanding < 0 || recognized > 10_000 || allocated > 10_000 || recognized != allocated || outstanding+recognized != 10_000 {
		t.Fatalf("concurrency invariant failed: outstanding=%d recognized=%d allocated=%d", outstanding, recognized, allocated)
	}
}
