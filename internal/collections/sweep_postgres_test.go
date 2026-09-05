package collections

import (
	"context"
	"encoding/json"
	"fmt"
	"kredit/internal/credit"
	"kredit/internal/db"
	"kredit/internal/ledger"
	"kredit/internal/outbox"
	"kredit/internal/payments"
	"kredit/internal/schedules"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type tenantPaymentStore struct {
	*payments.PostgresStore
	ctx context.Context
}

func (s *tenantPaymentStore) Record(input payments.RecordInput) (payments.Payment, payments.Allocation, error) {
	return s.RecordContext(s.ctx, input)
}
func (s *tenantPaymentStore) Reverse(paymentID, actor, reason string) (payments.Payment, error) {
	return s.ReverseContext(s.ctx, paymentID, actor, reason)
}
func (s *tenantPaymentStore) List(obligationID string) ([]payments.Payment, error) {
	return s.ReadContext(s.ctx, obligationID)
}
func (s *tenantPaymentStore) Get(paymentID string) (payments.Payment, error) {
	return s.PostgresStore.GetContext(s.ctx, paymentID)
}
func (s *tenantPaymentStore) Rebuild(obligationID string) (ledger.Money, error) {
	return s.PostgresStore.RebuildContext(s.ctx, obligationID)
}
func (s *tenantPaymentStore) RecordTx(ctx context.Context, tx pgx.Tx, input payments.RecordInput) (payments.Payment, payments.Allocation, error) {
	identity, _ := db.TenantFromContext(s.ctx)
	return s.PostgresStore.RecordTx(db.WithTenantContext(ctx, identity.UserID, identity.OrganizationID), tx, input)
}

type fixtureEngine struct {
	*PostgresEngine
	ctx context.Context
}

func (e *fixtureEngine) Start(_ context.Context, id, key string, now time.Time) (Attempt, error) {
	return e.PostgresEngine.Start(e.ctx, id, key, now)
}
func (e *fixtureEngine) ProcessWebhook(_ context.Context, event Webhook) (Attempt, error) {
	return e.PostgresEngine.ProcessWebhook(e.ctx, event)
}
func (e *fixtureEngine) SignalWebhook(_ context.Context, event Webhook) (Attempt, error) {
	return e.PostgresEngine.SignalWebhook(e.ctx, event)
}
func (e *fixtureEngine) Reconcile(_ context.Context, attemptID string) (Attempt, error) {
	return e.PostgresEngine.Reconcile(e.ctx, attemptID)
}
func (e *fixtureEngine) Retry(_ context.Context, attemptID string, now time.Time) (Attempt, error) {
	return e.PostgresEngine.Retry(e.ctx, attemptID, now)
}
func (e *fixtureEngine) Cancel(_ context.Context, attemptID string) (Attempt, error) {
	return e.PostgresEngine.Cancel(e.ctx, attemptID)
}
func (e *fixtureEngine) GetAttempt(attemptID string) (Attempt, bool) {
	return e.GetAttemptContext(e.ctx, attemptID)
}

type collectionFixture struct {
	pool                            *pgxpool.Pool
	id, user, request, organization string
	ctx                             context.Context
	payments                        *tenantPaymentStore
	snapshot                        SnapshotFunc
}

func financialFixture(t *testing.T, terms ...*ledger.FeeTerms) collectionFixture {
	var feeTerms *ledger.FeeTerms
	if len(terms) > 0 {
		feeTerms = terms[0]
	}
	feeJSON, _ := json.Marshal(feeTerms)
	t.Helper()
	if os.Getenv("KREDIT_INTEGRATION") != "1" || os.Getenv("DATABASE_URL") == "" {
		t.Skip("integration database required")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	var userID, organizationID, requestID, agreementID, activationTransactionID, obligationID string
	email := fmt.Sprintf("collection-test-%d@example.test", time.Now().UnixNano())
	if err := pool.QueryRow(ctx, `INSERT INTO app.users(normalized_email) VALUES($1) RETURNING id::text`, email).Scan(&userID); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO app.organizations(legal_name,business_type,business_address,industry) VALUES('Collection Test','limited_company','test','test') RETURNING id::text`).Scan(&organizationID); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO app.credit_requests(supplier_organization_id,buyer_user_id,buyer_business_id,principal_kobo,goods_description,due_date,collection_at,state,created_by,fee_terms) VALUES($1::uuid,$2::uuid,gen_random_uuid(),50000000,'goods',current_date,now()-interval '1 day','ACTIVE',$2::uuid,$3::jsonb) RETURNING id::text`, organizationID, userID, feeJSON).Scan(&requestID); err != nil {
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
	view := map[string]any{"request": map[string]any{"fee_terms": feeTerms, "id": requestID, "buyer_user_id": userID, "supplier_organization_id": organizationID, "version": 1}, "obligation": map[string]any{"id": obligationID, "credit_request_id": requestID, "supplier_organization_id": organizationID, "currency": "NGN", "outstanding_kobo": 50000000, "payment_status": "UNPAID"}}
	payload, _ := json.Marshal(view)
	if _, err := pool.Exec(ctx, `INSERT INTO app.credit_aggregate_snapshots(credit_request_id,supplier_organization_id,buyer_user_id,aggregate,version) VALUES($1,$2,$3,$4,1)`, requestID, organizationID, userID, payload); err != nil {
		t.Fatal(err)
	}

	if _, _, err := schedules.NewPostgresStore(pool).CreateDefault(obligationID, 50000000, time.Now().UTC().Format("2006-01-02"), time.Now().UTC().Add(-time.Hour), 0); err != nil {
		t.Fatal(err)
	}
	snapshot := func(id string) (ObligationSnapshot, error) {
		var outstanding int64
		err := pool.QueryRow(ctx, `SELECT outstanding_kobo FROM app.obligations WHERE id=$1::uuid`, id).Scan(&outstanding)
		return ObligationSnapshot{ID: id, BuyerUserID: userID, Currency: "NGN", Active: true, OutstandingKobo: ledger.Money(outstanding), MandateActive: true, MandateRemainingKobo: 50000000, CollectionEnabled: true, ProviderSupported: true, Version: 1}, err
	}
	tenantCtx := db.WithTenantContext(context.Background(), userID, organizationID)
	paymentStore := &tenantPaymentStore{PostgresStore: payments.NewPostgresStore(pool, outbox.NewStore(pool), nil), ctx: tenantCtx}
	return collectionFixture{pool: pool, id: obligationID, user: userID, request: requestID, organization: organizationID, ctx: tenantCtx, payments: paymentStore, snapshot: snapshot}
}
func (f collectionFixture) engine(p Provider) *fixtureEngine {
	base := NewPostgresEngine(f.pool, NewEngine(p, f.payments, f.snapshot, func(string, time.Time) (ledger.Money, error) { return 50000000, nil }))
	return &fixtureEngine{PostgresEngine: base, ctx: f.ctx}
}

type observedProvider struct {
	*MockProvider
	count    atomic.Int32
	onSubmit func(Request)
}

func (p *observedProvider) Submit(_ context.Context, r Request) (Response, error) {
	p.count.Add(1)
	if p.onSubmit != nil {
		p.onSubmit(r)
	}
	return Response{State: ProviderPending, ProviderCollectionID: "pending:" + r.ExternalReference}, nil
}

func TestPostgresReservationCommitsBeforeProviderAndDuplicateJobs(t *testing.T) {
	f := financialFixture(t)
	ctx := context.Background()
	p := &observedProvider{MockProvider: NewMockProvider("secret")}
	p.onSubmit = func(r Request) {
		var n int
		if err := f.pool.QueryRow(ctx, `SELECT count(*) FROM app.collection_attempt_index WHERE external_reference=$1`, r.ExternalReference).Scan(&n); err != nil || n != 1 {
			t.Errorf("provider contacted before durable reference: n=%d err=%v", n, err)
		}
	}
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			if _, err := f.engine(p).Start(ctx, f.id, fmt.Sprintf("duplicate-%d", i), time.Now()); err != nil {
				t.Logf("start: %v", err)
			}
		}(i)
	}
	wg.Wait()
	if p.count.Load() != 1 {
		t.Fatalf("submitted %d duplicate debits", p.count.Load())
	}
	if _, _, err := f.payments.Record(payments.RecordInput{ObligationID: f.id, AmountKobo: 10000000, SourceType: payments.SourceSupplierTransfer, RecordedBy: f.user, IdempotencyKey: "manual-racing:" + f.id}); err == nil {
		t.Fatal("manual repayment consumed an unresolved debit reservation")
	}
}
func TestPostgresPaymentBeforeDebitAndReplayAfterPaymentCommit(t *testing.T) {
	f := financialFixture(t)
	ctx := context.Background()
	p := &observedProvider{MockProvider: NewMockProvider("secret")}
	if _, _, err := f.payments.Record(payments.RecordInput{ObligationID: f.id, AmountKobo: 30000000, SourceType: payments.SourceSupplierTransfer, RecordedBy: f.user, IdempotencyKey: "manual-first:" + f.id}); err != nil {
		t.Fatal(err)
	}
	e := f.engine(p)
	a, err := e.Start(ctx, f.id, "remaining:"+f.id, time.Now())
	if err != nil || a.RequestedAmountKobo != 20000000 {
		t.Fatalf("attempt=%+v err=%v", a, err)
	}
	// Model a crash after the payment committed but before the attempt snapshot.
	in := payments.RecordInput{ObligationID: f.id, AmountKobo: 10000000, SourceType: payments.SourceCollected, RecordedBy: "collection-worker", Provider: p.Name(), ProviderReference: a.ProviderCollectionID, IdempotencyKey: "collection-attempt:" + a.ID}
	if _, _, err = f.payments.Record(in); err != nil {
		t.Fatal(err)
	}
	event := Webhook{EventID: "partial-final", ExternalReference: a.ExternalReference, ProviderCollectionID: a.ProviderCollectionID, State: ProviderPartial, SucceededAmountKobo: 10000000}
	event.Signature = p.Sign(event)
	for i := 0; i < 5; i++ {
		if _, err = f.engine(p).ProcessWebhook(ctx, event); err != nil {
			t.Fatal(err)
		}
	}
	var outstanding, count int64
	if err := f.pool.QueryRow(ctx, `SELECT outstanding_kobo FROM app.obligations WHERE id=$1::uuid`, f.id).Scan(&outstanding); err != nil {
		t.Fatal(err)
	}
	if err := f.pool.QueryRow(ctx, `SELECT count(*) FROM app.payments WHERE obligation_id=$1::uuid`, f.id).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if outstanding != 10000000 || count != 2 {
		t.Fatalf("duplicate financial effect: outstanding=%d payments=%d", outstanding, count)
	}
	loaded, _ := f.engine(p).GetAttempt(a.ID)
	if loaded.State != AttemptPartial {
		t.Fatalf("partial became %s", loaded.State)
	}
	if _, err = f.pool.Exec(ctx, `UPDATE app.collection_events SET amount_kobo=0 WHERE attempt_id=$1::uuid`, a.ID); err == nil {
		t.Fatal("financial audit was mutable")
	}
}
func TestPostgresSharedMandateCannotOverReserve(t *testing.T) {
	f, g := financialFixture(t), financialFixture(t)
	ctx := context.Background()
	var mid, buyer string
	if err := f.pool.QueryRow(ctx, `SELECT buyer_business_id::text FROM app.credit_requests WHERE id=$1::uuid`, f.request).Scan(&buyer); err != nil {
		t.Fatal(err)
	}
	if err := f.pool.QueryRow(ctx, `INSERT INTO app.payment_mandates(buyer_subject_type,buyer_subject_id,provider,provider_mandate_id,mandate_type,amount_ceiling_kobo,state,accepted_disclosure_version) VALUES('business',$2::uuid,'mock-collection',$1,'variable',50000000,'active','v1') RETURNING id::text`, f.id, buyer).Scan(&mid); err != nil {
		t.Fatal(err)
	}
	for _, fixture := range []collectionFixture{f, g} {
		if _, err := f.pool.Exec(ctx, `UPDATE app.credit_requests SET mandate_id=$2::uuid,buyer_business_id=$3::uuid WHERE id=$1::uuid`, fixture.request, mid, buyer); err != nil {
			t.Fatal(err)
		}
	}
	for _, fixture := range []*collectionFixture{&f, &g} {
		lookup := fixture.snapshot
		fixture.snapshot = func(id string) (ObligationSnapshot, error) {
			v, err := lookup(id)
			v.MandateReference = f.id
			return v, err
		}
	}
	p := &observedProvider{MockProvider: NewMockProvider("secret")}
	var wg sync.WaitGroup
	for _, fixture := range []collectionFixture{f, g} {
		wg.Add(1)
		go func(f collectionFixture) {
			defer wg.Done()
			if _, err := f.engine(p).Start(ctx, f.id, "shared:"+f.id, time.Now()); err != nil {
				t.Logf("shared start: %v", err)
			}
		}(fixture)
	}
	wg.Wait()
	if p.count.Load() != 1 {
		t.Fatalf("shared mandate produced %d debits over its ceiling", p.count.Load())
	}
}

func TestCollectionWorkerRoleCanPostAndReconcilePayment(t *testing.T) {
	f := financialFixture(t)
	ctx := context.Background()
	var roleExists bool
	if err := f.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM pg_roles WHERE rolname='kredit_worker')`).Scan(&roleExists); err != nil {
		t.Fatal(err)
	}
	if !roleExists {
		t.Skip("apply infra/postgres/roles.sql to test restricted workers")
	}
	cfg, err := pgxpool.ParseConfig(os.Getenv("DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	cfg.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
		_, err := conn.Exec(ctx, `SET ROLE kredit_worker`)
		return err
	}
	worker, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer worker.Close()
	f.pool = worker
	f.payments = &tenantPaymentStore{PostgresStore: payments.NewPostgresStore(worker, outbox.NewStore(worker), nil), ctx: f.ctx}
	p := NewMockProvider("secret")
	a, err := f.engine(p).Start(ctx, f.id, "worker-role:"+f.id, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if a.State != AttemptSucceeded || a.SucceededAmountKobo != 50000000 {
		t.Fatalf("worker could not post payment: %+v", a)
	}
	p.SetProviderResponse(a.ProviderCollectionID, Response{State: ProviderSucceeded, ProviderCollectionID: a.ProviderCollectionID, SucceededAmountKobo: a.SucceededAmountKobo})
	if _, err = f.engine(p).Reconcile(ctx, a.ID); err != nil {
		t.Fatal(err)
	}
	var snapshot []byte
	if err = worker.QueryRow(ctx, `SELECT app.credit_snapshot_by_id($1)`, f.request).Scan(&snapshot); err != nil {
		t.Fatal(err)
	}
	if _, err = worker.Exec(ctx, `SELECT * FROM app.payment_mandate_by_provider('mono-sweep','nonexistent-test-reference')`); err != nil {
		t.Fatal(err)
	}
}

func TestCollectionStateLoadsAfterRestartAndObservesExternalManualPayment(t *testing.T) {
	f := financialFixture(t)
	cold := credit.NewPostgresStore(f.pool, credit.NewStore(nil, ledger.NewStore()))
	first, err := cold.CollectionState(f.id)
	if err != nil || first.OutstandingKobo != 50000000 {
		t.Fatalf("cold worker failed: %+v %v", first, err)
	}
	if _, _, err = f.payments.Record(payments.RecordInput{ObligationID: f.id, AmountKobo: 30000000, SourceType: payments.SourceSupplierTransfer, RecordedBy: f.user, IdempotencyKey: "external-manual:" + f.id}); err != nil {
		t.Fatal(err)
	}
	next, err := cold.CollectionState(f.id)
	if err != nil || next.OutstandingKobo != 20000000 {
		t.Fatalf("worker used stale balance: %+v %v", next, err)
	}
	if _, err = cold.CollectionStateForOrganization(f.id, "other-supplier"); err == nil {
		t.Fatal("collection state crossed supplier boundary")
	}
}
