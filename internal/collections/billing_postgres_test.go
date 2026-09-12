package collections

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"kredit/internal/billing"
	"kredit/internal/ledger"
	"kredit/internal/onboarding"
	"kredit/internal/operations"
	"kredit/internal/outbox"
)

func TestInvoiceBillingReplayPermissionsAndWaiver(t *testing.T) {
	f := financialFixture(t)
	ctx := context.Background()
	if _, err := f.pool.Exec(ctx, `INSERT INTO app.platform_role_assignments(user_id,role,granted_by,reason) VALUES($1::uuid,'platform_owner',$1::uuid,'Synthetic invoice owner')`, f.user); err != nil {
		t.Fatal(err)
	}
	if _, err := f.pool.Exec(ctx, `INSERT INTO app.fees(supplier_organization_id,obligation_id,fee_type,basis_amount_kobo,rate_basis_points,amount_kobo,currency,state,accrued_at) VALUES($1::uuid,$2::uuid,'base_service',50000000,1,50,'NGN','accrued',now()-interval '40 days')`, f.organization, f.id); err != nil {
		t.Fatal(err)
	}
	if _, err := ledger.NewPostgresStore(f.pool).PostActivationWithFee(f.id, 50000000, 50, time.Now(), "invoice-fixture:"+f.id); err != nil {
		t.Fatal(err)
	}
	cfg, err := pgxpool.ParseConfig(f.pool.Config().ConnString())
	if err != nil {
		t.Fatal(err)
	}
	cfg.ConnConfig.RuntimeParams["role"] = "kredit_app"
	app, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer app.Close()
	onboardingStore := onboarding.NewPostgresStore(app)
	p, err := onboardingStore.Ensure(f.organization, f.user, true, true)
	if err != nil {
		t.Fatal(err)
	}
	p, _, err = onboardingStore.UpdateBilling(f.organization, f.user, onboarding.BillingInput{ExpectedVersion: p.Version, Method: "consolidated_invoice", Cycle: "monthly"})
	if err != nil {
		t.Fatal(err)
	}
	if p.BillingState != "pending_verification" {
		t.Fatal("seller preference activated without approval")
	}
	if _, _, err = onboardingStore.ApproveInvoiceBilling(f.organization, f.user, "wrong", p.Version, "Synthetic bank instructions for test only"); err == nil {
		t.Fatal("wrong arrangement approved")
	}
	if _, _, err = onboardingStore.ApproveInvoiceBilling(f.organization, f.user, p.BillingProviderReference, p.Version, "Synthetic bank instructions for test only"); err != nil {
		t.Fatal(err)
	}
	store := billing.NewStore(app)
	issued := time.Now().Add(-time.Minute).Truncate(time.Microsecond)
	var wg sync.WaitGroup
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if e := store.Issue(ctx, f.organization, issued); e != nil {
				t.Error(e)
			}
		}()
	}
	wg.Wait()
	invoices, err := store.List(ctx, f.organization, f.user)
	if err != nil || len(invoices) != 1 {
		t.Fatalf("invoice replay: %d %v", len(invoices), err)
	}
	v := invoices[0]
	if v.Total != 50 || v.Outstanding != 50 || len(v.Lines) != 1 {
		t.Fatalf("incorrect invoice: %+v", v)
	}
	receipt := billing.Receipt{Reference: "synthetic-bank:" + f.id, Amount: 20, ReceivedAt: time.Now().Truncate(time.Microsecond), Evidence: "Verified synthetic bank credit for this invoice"}
	if _, err = store.RecordReceipt(ctx, f.organization, "", v.ID, receipt); err == nil {
		t.Fatal("anonymous receipt accepted")
	}
	for i := 0; i < 2; i++ {
		if _, err = store.RecordReceipt(ctx, f.organization, f.user, v.ID, receipt); err != nil {
			t.Fatal(err)
		}
	}
	changed := receipt
	changed.Amount = 21
	if _, err = store.RecordReceipt(ctx, f.organization, f.user, v.ID, changed); err == nil {
		t.Fatal("changed receipt reused bank reference")
	}
	ops := operations.NewPostgresStore(app, outbox.NewStore(app), nil)
	if _, err = ops.WaiveFeeWithKey(f.user, f.organization, f.id, 51, "Synthetic fee waiver too large", "", "too-large:"+f.id); err == nil {
		t.Fatal("base fee counted twice")
	}
	if _, err = ops.WaiveFeeWithKey(f.user, f.organization, f.id, 10, "Synthetic partial fee waiver", "", "waiver:"+f.id); err != nil {
		t.Fatal(err)
	}
	invoices, err = store.List(ctx, f.organization, f.user)
	if err != nil {
		t.Fatal(err)
	}
	v = invoices[0]
	if v.Credit != 10 || v.Received != 20 || v.Outstanding != 20 {
		t.Fatalf("waiver or replay lost: %+v", v)
	}
	receipt.Reference += "-final"
	receipt.Amount = 21
	if _, err = store.RecordReceipt(ctx, f.organization, f.user, v.ID, receipt); err == nil {
		t.Fatal("invoice overpayment accepted")
	}
	receipt.Amount = 20
	if v, err = store.RecordReceipt(ctx, f.organization, f.user, v.ID, receipt); err != nil || v.Outstanding != 0 || v.Received != 40 {
		t.Fatalf("final receipt: %+v %v", v, err)
	}
	var outstanding, net int64
	if err = f.pool.QueryRow(ctx, `SELECT outstanding_kobo FROM app.obligations WHERE id=$1::uuid`, f.id).Scan(&outstanding); err != nil || outstanding != 50000000 {
		t.Fatalf("fee receipt changed buyer debt: %d %v", outstanding, err)
	}
	if err = f.pool.QueryRow(ctx, `SELECT COALESCE(sum(p.credit_kobo-p.debit_kobo),0) FROM ledger.postings p JOIN ledger.transactions t ON t.id=p.transaction_id JOIN ledger.accounts a ON a.id=p.account_id WHERE t.reference_id IN(SELECT id::text FROM app.fee_invoice_receipts WHERE invoice_id=$1::uuid) AND a.code='SUPPLIER_FEE_RECEIVABLE'`, v.ID).Scan(&net); err != nil || net != 40 {
		t.Fatalf("receipt journal duplicated: %d %v", net, err)
	}
	tx, err := app.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err = tx.Exec(ctx, `SELECT set_config('app.current_organization_id',gen_random_uuid()::text,true)`); err != nil {
		t.Fatal(err)
	}
	var count int
	if err = tx.QueryRow(ctx, `SELECT count(*) FROM app.fee_invoices WHERE id=$1::uuid`, v.ID).Scan(&count); err != nil || count != 0 {
		t.Fatalf("cross-tenant bill visible: %d %v", count, err)
	}
}

func TestInvoiceRefundAfterWaivedCollectionReversal(t *testing.T) {
	f := financialFixture(t)
	ctx := context.Background()
	if _, err := f.pool.Exec(ctx, `INSERT INTO app.platform_role_assignments(user_id,role,granted_by,reason) VALUES($1::uuid,'platform_owner',$1::uuid,'Synthetic refund owner')`, f.user); err != nil {
		t.Fatal(err)
	}
	provider := NewMockProvider("synthetic-refund")
	engine := f.engine(provider)
	attempt, err := engine.Start(ctx, f.id, "refund-collection:"+f.id, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	ops := operations.NewPostgresStore(f.pool, outbox.NewStore(f.pool), nil)
	if _, err = ops.WaiveFeeWithKey(f.user, f.organization, f.id, 10, "Synthetic partial collection fee waiver", "", "refund-waiver:"+f.id); err != nil {
		t.Fatal(err)
	}
	if _, err = f.pool.Exec(ctx, `UPDATE app.fees SET accrued_at=now()-interval '40 days' WHERE obligation_id=$1::uuid`, f.id); err != nil {
		t.Fatal(err)
	}
	onboardingStore := onboarding.NewPostgresStore(f.pool)
	p, err := onboardingStore.Ensure(f.organization, f.user, true, true)
	if err != nil {
		t.Fatal(err)
	}
	p, _, err = onboardingStore.UpdateBilling(f.organization, f.user, onboarding.BillingInput{ExpectedVersion: p.Version, Method: "consolidated_invoice", Cycle: "monthly"})
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err = onboardingStore.ApproveInvoiceBilling(f.organization, f.user, p.BillingProviderReference, p.Version, "Synthetic bank instructions only"); err != nil {
		t.Fatal(err)
	}
	store := billing.NewStore(f.pool)
	if err = store.Issue(ctx, f.organization, time.Now().Add(-time.Minute)); err != nil {
		t.Fatal(err)
	}
	invoices, err := store.List(ctx, f.organization, f.user)
	if err != nil || len(invoices) != 1 {
		t.Fatalf("invoice: %v %v", invoices, err)
	}
	v := invoices[0]
	receipt := billing.Receipt{Reference: "refund-receipt:" + f.id, Amount: v.Outstanding, ReceivedAt: time.Now().Truncate(time.Microsecond), Evidence: "Synthetic bank credit independently checked"}
	if _, err = store.RecordReceipt(ctx, f.organization, f.user, v.ID, receipt); err != nil {
		t.Fatal(err)
	}
	provider.SetProviderResponse(attempt.ProviderCollectionID, Response{State: ProviderReversed, ProviderCollectionID: attempt.ProviderCollectionID})
	if _, err = engine.Reconcile(ctx, attempt.ID); err != nil {
		t.Fatal(err)
	}
	// A provider reversal opens review; explicit recognition follows checked bank evidence.
	var paymentID string
	if err = f.pool.QueryRow(ctx, `SELECT id::text FROM app.payments WHERE idempotency_key=$1`, "collection-attempt:"+attempt.ID).Scan(&paymentID); err != nil {
		t.Fatal(err)
	}
	if _, err = f.payments.Reverse(paymentID, f.user, "Synthetic verified returned-funds evidence"); err != nil {
		t.Fatal(err)
	}
	invoices, err = store.List(ctx, f.organization, f.user)
	if err != nil {
		t.Fatal(err)
	}
	v = invoices[0]
	if v.Outstanding != 0 || v.CreditBalance != receipt.Amount {
		t.Fatalf("reversal lost fee credit: %+v", v)
	}
	receipt.Direction = "refunded"
	receipt.Reference = "refund-bank-debit:" + f.id
	oversized := receipt
	oversized.Amount++
	if _, err = store.RecordReceipt(ctx, f.organization, f.user, v.ID, oversized); err == nil {
		t.Fatal("refund exceeded credit balance")
	}
	for i := 0; i < 2; i++ {
		v, err = store.RecordReceipt(ctx, f.organization, f.user, v.ID, receipt)
		if err != nil {
			t.Fatal(err)
		}
	}
	if v.CreditBalance != 0 || v.Refunded != receipt.Amount || v.Outstanding != 0 {
		t.Fatalf("refund replay: %+v", v)
	}
	var remainingRevenue int64
	err = f.pool.QueryRow(ctx, `SELECT COALESCE(sum(p.credit_kobo-p.debit_kobo),0) FROM ledger.postings p JOIN ledger.accounts a ON a.id=p.account_id JOIN ledger.transactions t ON t.id=p.transaction_id WHERE a.code='PLATFORM_COLLECTION_REVENUE' AND (t.reference_id=$1 OR t.reference_id IN(SELECT id::text FROM app.payments WHERE obligation_id=$1::uuid))`, f.id).Scan(&remainingRevenue)
	if err != nil || remainingRevenue != 0 {
		t.Fatalf("waiver plus reversal overstated fee reversal: %d %v", remainingRevenue, err)
	}
}
