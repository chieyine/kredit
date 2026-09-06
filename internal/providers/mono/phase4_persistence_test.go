//go:build integration

package mono_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"

	"kredit/internal/collections"
	"kredit/internal/db"
	"kredit/internal/jobs"
	"kredit/internal/ledger"
	"kredit/internal/mandates"
	"kredit/internal/outbox"
	"kredit/internal/payments"
	"kredit/internal/providers/mono"
	"kredit/internal/schedules"
	"kredit/internal/web"
)

const phase4Principal int64 = 50000000

// A real Mono adapter talks to a local HTTPS fixture. The real PostgreSQL
// repositories and production inbox worker handle all financial effects.
// This is NOT Mono sandbox certification or a hosted buyer-consent test.
type phase4Provider struct {
	mu        sync.Mutex
	mandate   string
	reference string
	status    string
	collected int64
	pending   int64
	wrongID   bool
	loseReply bool
	posts     int
	gets      int
	onSubmit  func(string)
}

func (p *phase4Provider) serve(t *testing.T, w http.ResponseWriter, r *http.Request) {
	t.Helper()
	p.mu.Lock()
	defer p.mu.Unlock()
	if r.Header.Get("mono-sec-key") != "test_sk_fixture" {
		t.Error("fixture received an unexpected credential")
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	data := map[string]any{"mandate": p.mandate, "reference": p.reference, "currency": "NGN", "live_mode": false, "amount": phase4Principal, "collected_amount": p.collected, "pending_amount": p.pending, "status": p.status}
	switch {
	case r.Method == http.MethodPost && r.URL.Path == "/v3/payments/mandates/"+p.mandate+"/debit":
		p.posts++
		var in struct {
			Reference string `json:"reference"`
			Amount    int64  `json:"amount"`
		}
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil || in.Amount != phase4Principal || in.Reference == "" {
			t.Error("invalid fixture debit intent")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		p.reference = in.Reference
		if p.onSubmit != nil {
			p.onSubmit(in.Reference)
		}
		if p.loseReply {
			connection, _, err := w.(http.Hijacker).Hijack()
			if err != nil {
				t.Error(err)
				return
			}
			_ = connection.Close()
			return
		}
		data["reference"], data["status"] = in.Reference, "processing"
		data["collected_amount"], data["pending_amount"] = int64(0), phase4Principal
	case r.Method == http.MethodGet && r.URL.Path == "/v3/payments/mandates/"+p.mandate+"/debit/"+p.reference:
		p.gets++
		if p.wrongID {
			data["mandate"] = "mmc_other_tenant"
		}
	case r.Method == http.MethodGet && r.URL.Path == "/v3/payments/mandates/"+p.mandate:
		data = map[string]any{"id": p.mandate, "status": "approved", "approved": true, "ready_to_debit": true, "mandate_type": "sweep", "debit_type": "variable", "amount": phase4Principal, "start_date": time.Now().Add(-24 * time.Hour).UTC().Format(time.RFC3339), "end_date": time.Now().Add(30 * 24 * time.Hour).UTC().Format(time.RFC3339)}
	default:
		t.Errorf("unexpected fixture request: %s %s", r.Method, r.URL.Path)
		w.WriteHeader(http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]any{"status": "successful", "data": data}); err != nil {
		t.Error(err)
	}
}

func (p *phase4Provider) result(status string, collected, pending int64) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.status, p.collected, p.pending = status, collected, pending
}

// The legacy service interface has no context argument. Pin the fixture's
// already-authorized tenant only at that interface; database writes still use
// the production scoped PostgresStore implementation and worker login.
type phase4Payments struct {
	*payments.PostgresStore
	ctx context.Context
}

func (p *phase4Payments) Record(in payments.RecordInput) (payments.Payment, payments.Allocation, error) {
	return p.RecordContext(p.ctx, in)
}

type phase4Fixture struct {
	admin                                *pgxpool.Pool
	worker                               *db.Pool
	workerURL                            string
	ctx                                  context.Context
	user, organization, obligation       string
	mandateID                            string
	client                               *mono.Client
	provider                             *phase4Provider
	engine                               *collections.PostgresEngine
	runtime                              *web.Runtime
}

func phase4NewFixture(t *testing.T) *phase4Fixture {
	t.Helper()
	adminURL, workerURL := os.Getenv("DATABASE_URL"), os.Getenv("RIVER_DATABASE_URL")
	if adminURL == "" || workerURL == "" {
		t.Fatal("integration requires isolated DATABASE_URL and restricted RIVER_DATABASE_URL; tests must not silently skip")
	}
	ctx := context.Background()
	admin, err := pgxpool.New(ctx, adminURL)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(admin.Close)
	f := &phase4Fixture{admin: admin, workerURL: workerURL}
	var requestID, agreementID, businessID, activationID string
	if err := admin.QueryRow(ctx, `INSERT INTO app.users(normalized_email) VALUES($1) RETURNING id::text`, "phase4-"+uuid.NewString()+"@example.test").Scan(&f.user); err != nil {
		t.Fatal(err)
	}
	if err := admin.QueryRow(ctx, `INSERT INTO app.organizations(legal_name,business_type,business_address,industry) VALUES('Phase 4 fixture','limited_company','test','test') RETURNING id::text`).Scan(&f.organization); err != nil {
		t.Fatal(err)
	}
	businessID = uuid.NewString()
	if err := admin.QueryRow(ctx, `INSERT INTO app.credit_requests(supplier_organization_id,buyer_user_id,buyer_business_id,principal_kobo,goods_description,due_date,collection_at,state,created_by) VALUES($1::uuid,$2::uuid,$3::uuid,$4,'test goods',current_date,now()-interval '1 day','ACTIVE',$2::uuid) RETURNING id::text`, f.organization, f.user, businessID, phase4Principal).Scan(&requestID); err != nil {
		t.Fatal(err)
	}
	if err := admin.QueryRow(ctx, `INSERT INTO app.agreement_versions(credit_request_id,version,canonical_json,document_hash,terms_version,privacy_version,created_by) VALUES($1::uuid,1,'{}',$2,'test-v1','test-v1',$3::uuid) RETURNING id::text`, requestID, "phase4-"+requestID, f.user).Scan(&agreementID); err != nil {
		t.Fatal(err)
	}
	if err := admin.QueryRow(ctx, `INSERT INTO ledger.transactions(event_type,reference_type,reference_id,idempotency_key,effective_at) VALUES('test','credit_request',$1,$2,now()) RETURNING id::text`, requestID, "phase4-activation-"+requestID).Scan(&activationID); err != nil {
		t.Fatal(err)
	}
	if err := admin.QueryRow(ctx, `INSERT INTO app.obligations(credit_request_id,agreement_version_id,supplier_organization_id,buyer_business_id,principal_kobo,currency,lifecycle_status,payment_status,outstanding_kobo,base_fee_kobo,ledger_transaction_id,activated_at) VALUES($1::uuid,$2::uuid,$3::uuid,$4::uuid,$5,'NGN','ACTIVE','UNPAID',$5,50,$6::uuid,now()) RETURNING id::text`, requestID, agreementID, f.organization, businessID, phase4Principal, activationID).Scan(&f.obligation); err != nil {
		t.Fatal(err)
	}
	payload, err := json.Marshal(map[string]any{"request": map[string]any{"id": requestID, "buyer_user_id": f.user, "supplier_organization_id": f.organization, "version": 1}, "obligation": map[string]any{"id": f.obligation, "credit_request_id": requestID, "supplier_organization_id": f.organization, "currency": "NGN", "outstanding_kobo": phase4Principal, "payment_status": "UNPAID"}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := admin.Exec(ctx, `INSERT INTO app.credit_aggregate_snapshots(credit_request_id,supplier_organization_id,buyer_user_id,aggregate,version) VALUES($1,$2,$3,$4,1)`, requestID, f.organization, f.user, payload); err != nil {
		t.Fatal(err)
	}
	if _, _, err := schedules.NewPostgresStore(admin).CreateDefault(f.obligation, ledger.Money(phase4Principal), time.Now().UTC().Format("2006-01-02"), time.Now().Add(-time.Hour), 0); err != nil {
		t.Fatal(err)
	}
	f.provider = &phase4Provider{mandate: "mmc_phase4_" + uuid.NewString(), status: "processing", pending: phase4Principal}
	f.client = mono.NewPhase4FixtureClient(t, func(w http.ResponseWriter, r *http.Request) { f.provider.serve(t, w, r) })
	f.ctx = db.WithTenantContext(ctx, f.user, f.organization)
	f.provider.onSubmit = func(reference string) {
		var count int
		if err := admin.QueryRow(ctx, `SELECT count(*) FROM app.collection_attempt_index WHERE external_reference=$1 AND obligation_id=$2::uuid`, reference, f.obligation).Scan(&count); err != nil || count != 1 {
			t.Errorf("provider contacted before committed reservation/reference: count=%d err=%v", count, err)
		}
	}
	// This synthetic active mandate is a fixture, not hosted consent evidence.
	if err := admin.QueryRow(ctx, `INSERT INTO app.payment_mandates(buyer_subject_type,buyer_subject_id,provider,provider_mandate_id,mandate_type,amount_ceiling_kobo,state,accepted_disclosure_version,supplier_organization_id) VALUES('business',$1::uuid,'mono-sweep',$2,'variable',$3,'active','test-v1',$4::uuid) RETURNING id::text`, businessID, f.provider.mandate, phase4Principal, f.organization).Scan(&f.mandateID); err != nil {
		t.Fatal(err)
	}
	metadata, err := json.Marshal(mandates.Mandate{ID: f.mandateID, Provider: "mono-sweep", ProviderID: f.provider.mandate, UserID: f.user, BusinessID: businessID, SupplierOrganizationID: f.organization, AmountCeiling: phase4Principal, Status: mandates.Active, Variable: true, MultiAccount: true, PartialRecovery: true})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := admin.Exec(ctx, `UPDATE app.payment_mandates SET metadata=$2::jsonb WHERE id=$1::uuid`, f.mandateID, metadata); err != nil {
		t.Fatal(err)
	}
	// Deliberately test callback/payment persistence independently of the hosted
	// authorization and prior-notice gates, which remain separate acceptance items.
	f.restart(t)
	t.Cleanup(func() { f.worker.Close() })
	return f
}

func (f *phase4Fixture) restart(t *testing.T) {
	t.Helper()
	if f.worker != nil {
		f.worker.Close()
	}
	var err error
	f.worker, err = db.OpenAsRole(context.Background(), f.workerURL, "kredit_worker")
	if err != nil {
		t.Fatal(err)
	}
	store := &phase4Payments{PostgresStore: payments.NewPostgresStore(f.worker.Raw(), outbox.NewStore(f.worker.Raw()), nil), ctx: f.ctx}
	snapshot := func(id string) (collections.ObligationSnapshot, error) {
		var outstanding int64
		err := f.worker.WithTenantTx(context.Background(), f.user, f.organization, func(tx pgx.Tx) error {
			return tx.QueryRow(context.Background(), `SELECT outstanding_kobo FROM app.obligations WHERE id=$1::uuid`, id).Scan(&outstanding)
		})
		return collections.ObligationSnapshot{ID: id, BuyerUserID: f.user, Currency: "NGN", Active: true, OutstandingKobo: ledger.Money(outstanding), MandateReference: f.provider.mandate, MandateActive: true, MandateRemainingKobo: ledger.Money(phase4Principal), CollectionEnabled: true, ProviderSupported: true, Version: 1}, err
	}
	f.engine = collections.NewPostgresEngine(f.worker.Raw(), collections.NewEngine(f.client, store, snapshot, func(string, time.Time) (ledger.Money, error) { return ledger.Money(phase4Principal), nil }))
	f.runtime = &web.Runtime{Database: f.worker, Mono: f.client, Collections: f.engine}
}

func (f *phase4Fixture) start(t *testing.T) collections.Attempt {
	t.Helper()
	a, err := f.engine.Start(f.ctx, f.obligation, "phase4-start:"+f.obligation, time.Now().UTC())
	if err != nil || (a.State != collections.AttemptSubmitted && a.State != collections.AttemptUnknown) {
		t.Fatalf("start=%+v err=%v", a, err)
	}
	return a
}

func (f *phase4Fixture) notice(t *testing.T, eventType string) jobs.ProviderWebhookArgs {
	t.Helper()
	f.provider.mu.Lock()
	reference := f.provider.reference
	f.provider.mu.Unlock()
	raw, err := json.Marshal(map[string]any{"event_id": "phase4-" + uuid.NewString(), "event": eventType, "data": map[string]any{"mandate": f.provider.mandate, "reference_number": reference, "live_mode": false, "amount": 1, "collected_amount": 1, "bvn": "private-bvn-fixture", "account_number": "private-account-fixture"}})
	if err != nil {
		t.Fatal(err)
	}
	notice, err := f.client.ParseWebhook("fixture-webhook-secret", raw)
	if err != nil {
		t.Fatal(err)
	}
	safe, err := json.Marshal(notice)
	if err != nil {
		t.Fatal(err)
	}
	return jobs.ProviderWebhookArgs{Provider: "mono-sweep", EventID: notice.EventID, EventType: notice.Type, Payload: safe, SignatureValid: true}
}

func (f *phase4Fixture) deliver(args jobs.ProviderWebhookArgs) error {
	worker := &jobs.ProviderWebhookWorker{Pool: f.worker.Raw(), Handler: f.runtime.HandleProviderNotice}
	return worker.Work(context.Background(), &river.Job[jobs.ProviderWebhookArgs]{Args: args})
}

func (f *phase4Fixture) assertMoney(t *testing.T, collected int64, expectedPayments int) {
	t.Helper()
	var outstanding, total, allocations int64
	var count, unbalanced int
	ctx := context.Background()
	if err := f.admin.QueryRow(ctx, `SELECT outstanding_kobo FROM app.obligations WHERE id=$1::uuid`, f.obligation).Scan(&outstanding); err != nil {
		t.Fatal(err)
	}
	if err := f.admin.QueryRow(ctx, `SELECT count(*),COALESCE(sum(amount_kobo),0) FROM app.payments WHERE obligation_id=$1::uuid`, f.obligation).Scan(&count, &total); err != nil {
		t.Fatal(err)
	}
	if err := f.admin.QueryRow(ctx, `SELECT COALESCE(sum(amount_kobo),0) FROM app.payment_allocations WHERE obligation_id=$1::uuid`, f.obligation).Scan(&allocations); err != nil {
		t.Fatal(err)
	}
	if err := f.admin.QueryRow(ctx, `SELECT count(*) FROM (SELECT transaction_id FROM ledger.postings GROUP BY transaction_id HAVING sum(debit_kobo)<>sum(credit_kobo)) bad`).Scan(&unbalanced); err != nil {
		t.Fatal(err)
	}
	if outstanding != phase4Principal-collected || total != collected || allocations != collected || count != expectedPayments || unbalanced != 0 {
		t.Fatalf("financial invariant failed: outstanding=%d total=%d allocations=%d payments=%d unbalanced=%d", outstanding, total, allocations, count, unbalanced)
	}
}

func TestPhase4PersistedDuplicateAndOutOfOrderNotices(t *testing.T) {
	f := phase4NewFixture(t)
	a := f.start(t)
	interim := f.notice(t, "events.mandates.debit_attempt.successful")
	if err := f.deliver(interim); err != nil {
		t.Fatal(err)
	}
	f.assertMoney(t, 0, 0)
	f.provider.result("successful", phase4Principal, 0)
	final := f.notice(t, "events.mandates.debit.successful")
	for i := 0; i < 5; i++ {
		if err := f.deliver(final); err != nil {
			t.Fatal(err)
		}
	}
	f.restart(t)
	for _, name := range []string{"events.mandates.debit.processing", "events.mandates.debit.failed", "events.mandates.debit.successful"} {
		if err := f.deliver(f.notice(t, name)); err != nil {
			t.Fatal(err)
		}
	}
	f.assertMoney(t, phase4Principal, 1)
	loaded, ok := f.engine.GetAttemptContext(f.ctx, a.ID)
	if !ok || loaded.State != collections.AttemptSucceeded {
		t.Fatalf("terminal payment regressed: %+v", loaded)
	}
	var safe, state string
	var duplicates int
	if err := f.admin.QueryRow(context.Background(), `SELECT payload::text,state,duplicate_count FROM app.provider_webhook_inbox WHERE provider='mono-sweep' AND event_id=$1`, final.EventID).Scan(&safe, &state, &duplicates); err != nil {
		t.Fatal(err)
	}
	if state != "processed" || duplicates != 4 || strings.Contains(safe, "private-") || strings.Contains(safe, "collected_amount") {
		t.Fatalf("unsafe or non-idempotent inbox: state=%s duplicates=%d", state, duplicates)
	}
	f.provider.mu.Lock()
	defer f.provider.mu.Unlock()
	if f.provider.posts != 1 {
		t.Fatalf("replay resubmitted %d debits", f.provider.posts)
	}
}

func TestPhase4PersistedPartialFailedAndUnknownOutcomes(t *testing.T) {
	for _, tc := range []struct {
		name, status string
		amount       int64
		payments     int
	}{
		{"partial", "partial-debit-successful", 30000000, 1},
		{"failed", "failed", 0, 0},
		{"unknown", "provider-added-state", 0, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := phase4NewFixture(t)
			a := f.start(t)
			f.provider.result(tc.status, tc.amount, phase4Principal-tc.amount)
			args := f.notice(t, "events.mandates.debit.successful")
			for range 3 {
				if err := f.deliver(args); err != nil {
					t.Fatal(err)
				}
			}
			f.restart(t)
			if _, err := f.engine.Reconcile(f.ctx, a.ID); err != nil {
				t.Fatal(err)
			}
			f.assertMoney(t, tc.amount, tc.payments)
			if tc.name == "unknown" {
				var held int64
				if err := f.admin.QueryRow(context.Background(), `SELECT COALESCE(sum(reserved_amount_kobo),0) FROM app.collection_reservations WHERE obligation_id=$1::uuid AND state IN ('PROCESSING','COMPLETED')`, f.obligation).Scan(&held); err != nil || held != phase4Principal {
					t.Fatalf("uncertain debit lost its reservation: held=%d err=%v", held, err)
				}
			}
		})
	}
}

func TestPhase4PersistedLostReplyAndIdentityMismatchRecovery(t *testing.T) {
	f := phase4NewFixture(t)
	f.provider.loseReply = true
	a := f.start(t)
	if a.State != collections.AttemptUnknown {
		t.Fatalf("lost response did not remain unknown: %+v", a)
	}
	f.restart(t)
	f.provider.result("successful", phase4Principal, 0)
	f.provider.mu.Lock()
	f.provider.wrongID = true
	f.provider.mu.Unlock()
	args := f.notice(t, "events.mandates.debit.successful")
	if err := f.deliver(args); err == nil {
		t.Fatal("foreign mandate identity was accepted")
	}
	f.assertMoney(t, 0, 0)
	f.provider.mu.Lock()
	f.provider.wrongID = false
	f.provider.mu.Unlock()
	if err := f.deliver(args); err != nil {
		t.Fatal(err)
	}
	f.assertMoney(t, phase4Principal, 1)
	f.provider.mu.Lock()
	defer f.provider.mu.Unlock()
	if f.provider.posts != 1 || f.provider.reference != a.ExternalReference {
		t.Fatal("unknown-outcome recovery resubmitted or changed the original debit")
	}
}

func TestPhase4PersistedCancelledMandateCannotBeReactivatedByStaleLookup(t *testing.T) {
	for _, status := range []mandates.Status{mandates.Cancelled, mandates.Expired} {
		t.Run(string(status), func(t *testing.T) {
			f := phase4NewFixture(t)
			p := mandates.NewPostgresProviderWithRemote(f.worker.Raw(), f.client)
			if _, err := p.BlockMandate(context.Background(), f.provider.mandate, status, "phase4-block-"+uuid.NewString()); err != nil {
				t.Fatal(err)
			}
			f.restart(t)
			p = mandates.NewPostgresProviderWithRemote(f.worker.Raw(), f.client)
			m, err := p.GetMandate(context.Background(), f.provider.mandate)
			if err != nil || m.Status != status {
				t.Fatalf("stale approved provider result reactivated terminal mandate: %+v err=%v", m, err)
			}
			var persisted string
			if err := f.admin.QueryRow(context.Background(), `SELECT state FROM app.payment_mandates WHERE id=$1::uuid`, f.mandateID).Scan(&persisted); err != nil || persisted != strings.ToLower(string(status)) {
				t.Fatalf("terminal mandate state lost: %s %v", persisted, err)
			}
		})
	}
}

func TestPhase4PersistedUnauthenticatedJobCannotPostMoney(t *testing.T) {
	f := phase4NewFixture(t)
	f.start(t)
	args := f.notice(t, "events.mandates.debit.successful")
	args.SignatureValid = false
	if err := f.deliver(args); err == nil {
		t.Fatal("unauthenticated job was accepted")
	}
	f.assertMoney(t, 0, 0)
	var count int
	if err := f.admin.QueryRow(context.Background(), `SELECT count(*) FROM app.provider_webhook_inbox WHERE event_id=$1`, args.EventID).Scan(&count); err != nil || count != 0 {
		t.Fatal(fmt.Sprintf("unauthenticated notice persisted: %d %v", count, err))
	}
}
