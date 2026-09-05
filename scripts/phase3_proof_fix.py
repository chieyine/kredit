from pathlib import Path


def one(text: str, old: str, new: str, label: str) -> str:
    count = text.count(old)
    if count != 1:
        raise SystemExit(f"{label}: expected one match, found {count}")
    return text.replace(old, new, 1)

# Fix new payment proof types and cleanup query.
p = Path('internal/payments/postgres_test.go')
text = p.read_text()
if '"kredit/internal/ledger"' not in text:
    text = text.replace('"kredit/internal/db"\n', '"kredit/internal/db"\n\t"kredit/internal/ledger"\n', 1)
text = text.replace('CreateDefault(f.obligationID, principal,', 'CreateDefault(f.obligationID, ledger.Money(principal),', 1)
text = text.replace('duplicateAllocation.ID != allocation.ID', 'duplicateAllocation.PaymentID != allocation.PaymentID || duplicateAllocation.AmountKobo != allocation.AmountKobo', 1)
text = text.replace('AmountKobo: Money(amount)', 'AmountKobo: ledger.Money(amount)', 1)
text = text.replace('DELETE FROM app.credit_aggregate_snapshots WHERE credit_request_id=$1::uuid', 'DELETE FROM app.credit_aggregate_snapshots WHERE credit_request_id=$1', 1)
p.write_text(text)

# Make the shared collection test fixture tenant-aware without changing production APIs.
p = Path('internal/collections/sweep_postgres_test.go')
text = p.read_text()
if '"kredit/internal/db"' not in text:
    text = text.replace('"kredit/internal/credit"\n', '"kredit/internal/credit"\n\t"kredit/internal/db"\n', 1)

old = '''type collectionFixture struct {
\tpool                            *pgxpool.Pool
\tid, user, request, organization string
\tpayments                        *payments.PostgresStore
\tsnapshot                        SnapshotFunc
}
'''
new = '''type tenantPaymentStore struct {
\t*payments.PostgresStore
\tctx context.Context
}

func (s *tenantPaymentStore) Record(input payments.RecordInput) (payments.Payment, payments.Allocation, error) {
\treturn s.PostgresStore.RecordContext(s.ctx, input)
}
func (s *tenantPaymentStore) Reverse(paymentID, actor, reason string) (payments.Payment, error) {
\treturn s.PostgresStore.ReverseContext(s.ctx, paymentID, actor, reason)
}
func (s *tenantPaymentStore) List(obligationID string) ([]payments.Payment, error) {
\treturn s.PostgresStore.ReadContext(s.ctx, obligationID)
}
func (s *tenantPaymentStore) Get(paymentID string) (payments.Payment, error) {
\treturn s.PostgresStore.GetContext(s.ctx, paymentID)
}
func (s *tenantPaymentStore) Rebuild(obligationID string) (ledger.Money, error) {
\treturn s.PostgresStore.RebuildContext(s.ctx, obligationID)
}
func (s *tenantPaymentStore) RecordTx(ctx context.Context, tx pgx.Tx, input payments.RecordInput) (payments.Payment, payments.Allocation, error) {
\tidentity, _ := db.TenantFromContext(s.ctx)
\treturn s.PostgresStore.RecordTx(db.WithTenantContext(ctx, identity.UserID, identity.OrganizationID), tx, input)
}

type fixtureEngine struct {
\t*PostgresEngine
\tctx context.Context
}
func (e *fixtureEngine) Start(_ context.Context, id, key string, now time.Time) (Attempt, error) {
\treturn e.PostgresEngine.Start(e.ctx, id, key, now)
}
func (e *fixtureEngine) ProcessWebhook(_ context.Context, event Webhook) (Attempt, error) {
\treturn e.PostgresEngine.ProcessWebhook(e.ctx, event)
}
func (e *fixtureEngine) SignalWebhook(_ context.Context, event Webhook) (Attempt, error) {
\treturn e.PostgresEngine.SignalWebhook(e.ctx, event)
}
func (e *fixtureEngine) Reconcile(_ context.Context, attemptID string) (Attempt, error) {
\treturn e.PostgresEngine.Reconcile(e.ctx, attemptID)
}
func (e *fixtureEngine) Retry(_ context.Context, attemptID string, now time.Time) (Attempt, error) {
\treturn e.PostgresEngine.Retry(e.ctx, attemptID, now)
}
func (e *fixtureEngine) Cancel(_ context.Context, attemptID string) (Attempt, error) {
\treturn e.PostgresEngine.Cancel(e.ctx, attemptID)
}
func (e *fixtureEngine) GetAttempt(attemptID string) (Attempt, bool) {
\treturn e.PostgresEngine.GetAttemptContext(e.ctx, attemptID)
}

type collectionFixture struct {
\tpool                            *pgxpool.Pool
\tid, user, request, organization string
\tctx                             context.Context
\tpayments                        *tenantPaymentStore
\tsnapshot                        SnapshotFunc
}
'''
text = one(text, old, new, 'collection fixture types')

old = '''\treturn collectionFixture{pool, obligationID, userID, requestID, organizationID, payments.NewPostgresStore(pool, outbox.NewStore(pool), nil), snapshot}
}
func (f collectionFixture) engine(p Provider) *PostgresEngine {
\treturn NewPostgresEngine(f.pool, NewEngine(p, f.payments, f.snapshot, func(string, time.Time) (ledger.Money, error) { return 50000000, nil }))
}
'''
new = '''\ttenantCtx := db.WithTenantContext(context.Background(), userID, organizationID)
\tpaymentStore := &tenantPaymentStore{PostgresStore: payments.NewPostgresStore(pool, outbox.NewStore(pool), nil), ctx: tenantCtx}
\treturn collectionFixture{pool: pool, id: obligationID, user: userID, request: requestID, organization: organizationID, ctx: tenantCtx, payments: paymentStore, snapshot: snapshot}
}
func (f collectionFixture) engine(p Provider) *fixtureEngine {
\tbase := NewPostgresEngine(f.pool, NewEngine(p, f.payments, f.snapshot, func(string, time.Time) (ledger.Money, error) { return 50000000, nil }))
\treturn &fixtureEngine{PostgresEngine: base, ctx: f.ctx}
}
'''
text = one(text, old, new, 'financial fixture return')

# Restricted-worker test swaps pools; preserve the tenant wrapper.
text = text.replace(
    'f.payments = payments.NewPostgresStore(worker, outbox.NewStore(worker), nil)',
    'f.payments = &tenantPaymentStore{PostgresStore: payments.NewPostgresStore(worker, outbox.NewStore(worker), nil), ctx: f.ctx}',
    1,
)
p.write_text(text)

# The standalone Postgres collection test gets an explicit tenant context.
p = Path('internal/collections/postgres_test.go')
text = p.read_text()
if '"kredit/internal/db"' not in text:
    text = text.replace('"kredit/internal/identifier"\n', '"kredit/internal/db"\n\t"kredit/internal/identifier"\n', 1)
anchor = '''\tif err := pool.QueryRow(ctx, `INSERT INTO app.obligations(credit_request_id,agreement_version_id,supplier_organization_id,buyer_business_id,principal_kobo,currency,lifecycle_status,payment_status,outstanding_kobo,base_fee_kobo,ledger_transaction_id,activated_at) SELECT $1::uuid,$2::uuid,$3::uuid,buyer_business_id,principal_kobo,'NGN','ACTIVE','UNPAID',principal_kobo,50,$4::uuid,now() FROM app.credit_requests WHERE id=$1::uuid RETURNING id::text`, requestID, agreementID, organizationID, activationTransactionID).Scan(&obligationID); err != nil {
\t\tt.Fatal(err)
\t}
'''
replacement = anchor + '\tctx = db.WithTenantContext(ctx, userID, organizationID)\n'
text = one(text, anchor, replacement, 'standalone collection tenant context')
text = text.replace('restarted.GetAttempt(attempt.ID)', 'restarted.GetAttemptContext(ctx, attempt.ID)', 1)
text = text.replace('DELETE FROM app.credit_aggregate_snapshots WHERE credit_request_id=$1::uuid', 'DELETE FROM app.credit_aggregate_snapshots WHERE credit_request_id=$1', 1)
p.write_text(text)
