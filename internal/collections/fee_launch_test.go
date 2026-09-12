package collections

import (
	"context"
	"crypto/sha256"
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"kredit/internal/billing"
	"kredit/internal/ledger"
	"kredit/internal/onboarding"
	"kredit/internal/settlement"
)

func feeApp(t *testing.T, f collectionFixture) *pgxpool.Pool {
	t.Helper()
	ctx := context.Background()
	if _, err := f.pool.Exec(ctx, `INSERT INTO app.platform_role_assignments(user_id,role,granted_by,reason) VALUES($1::uuid,'platform_owner',$1::uuid,'Synthetic fee review');`, f.user); err != nil {
		t.Fatal(err)
	}
	if _, err := f.pool.Exec(ctx, `INSERT INTO app.memberships(organization_id,user_id,role,status) VALUES($1::uuid,$2::uuid,'owner','active')`, f.organization, f.user); err != nil {
		t.Fatal(err)
	}
	cfg, err := pgxpool.ParseConfig(f.pool.Config().ConnString())
	if err != nil {
		t.Fatal(err)
	}
	cfg.ConnConfig.RuntimeParams["role"] = "kredit_app"
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	return pool
}
func TestSplitFeeAllocationReplayAndReversal(t *testing.T) {
	f := financialFixture(t)
	ctx := context.Background()
	app := feeApp(t, f)
	p := NewMockProvider("split-launch")
	engine := f.engine(p)
	engine.RequireSettlementRoute()
	if _, err := onboarding.NewPostgresStore(app).Ensure(f.organization, f.user, true, true); err != nil {
		t.Fatal(err)
	}
	if _, err := f.pool.Exec(ctx, `UPDATE app.supplier_onboarding_profiles SET settlement_state='verified',settlement_provider=$2,settlement_provider_reference='split-destination',settlement_account_name='Synthetic Seller',settlement_account_last4='7890',settlement_bank_name='Synthetic Bank',billing_state='configured',billing_method='split_settlement',billing_provider_reference='split-billing' WHERE organization_id=$1::uuid`, f.organization, p.Name()); err != nil {
		t.Fatal(err)
	}
	if _, err := f.pool.Exec(ctx, `INSERT INTO app.settlement_registrations(id,organization_id,provider,connection_identity,bank_code,account_last4,state,result) VALUES($1,$2::uuid,$3,$1,'000013','7890','REGISTERED','{"provider_reference":"split-destination"}')`, fmt.Sprintf("%x", sha256.Sum256([]byte(f.id))), f.organization, p.Name()); err != nil {
		t.Fatal(err)
	}
	if _, err := f.pool.Exec(ctx, `INSERT INTO app.fees(supplier_organization_id,obligation_id,fee_type,basis_amount_kobo,rate_basis_points,amount_kobo,currency,state,accrued_at) VALUES($1::uuid,$2::uuid,'base_service',50000000,1,100,'NGN','accrued',now())`, f.organization, f.id); err != nil {
		t.Fatal(err)
	}
	a, err := engine.Start(ctx, f.id, "split:"+f.id, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if a.SettlementRoute.FeeAmountKobo != 250100 || a.SettlementRoute.NetAmountKobo != 49749900 {
		t.Fatalf("fee allocation %+v", a.SettlementRoute)
	}
	if _, err = engine.Start(ctx, f.id, "split:"+f.id, time.Now()); err != nil {
		t.Fatal(err)
	}
	var allocated int64
	if err = f.pool.QueryRow(ctx, `SELECT sum(amount_kobo) FROM app.split_fee_allocations WHERE supplier_organization_id=$1::uuid`, f.organization).Scan(&allocated); err != nil || allocated != 250100 {
		t.Fatal("allocation replay", allocated, err)
	}
	payouts, err := settlement.NewReceiptStore(app).List(ctx, f.user, f.organization)
	if err != nil || len(payouts) != 1 || payouts[0].Outstanding != 49749900 {
		t.Fatal("net seller payout", payouts, err)
	}
	list, err := f.payments.List(f.id)
	if err != nil || len(list) != 1 {
		t.Fatal(err)
	}
	if _, err = f.payments.Reverse(list[0].ID, f.user, "Verified synthetic provider reversal"); err != nil {
		t.Fatal(err)
	}
	if err = f.pool.QueryRow(ctx, `SELECT sum(collected_kobo) FROM app.fees WHERE obligation_id=$1::uuid`, f.id).Scan(&allocated); err != nil || allocated != 0 {
		t.Fatal("fee reversal", allocated, err)
	}
}

type feeProviderFixture struct {
	auth  billing.FeeAuthorization
	count int
	state string
}

func (p *feeProviderFixture) Name() string { return "fee-original" }
func (p *feeProviderFixture) CreateFeeCustomer(context.Context, billing.FeeCustomer) (string, error) {
	return "fee-customer", nil
}
func (p *feeProviderFixture) FeeCustomerIdentity(context.Context, string) (string, error) {
	return "12345678901", nil
}
func (p *feeProviderFixture) CreateFeeAuthorization(_ context.Context, a billing.FeeAuthorization) (billing.FeeAuthorization, error) {
	a.ID = "fee-mandate-" + a.Reference
	a.URL = "https://example.test/authorize"
	a.Ready = true
	p.auth = a
	return a, nil
}
func (p *feeProviderFixture) ReadFeeAuthorization(context.Context, string) (billing.FeeAuthorization, error) {
	return p.auth, nil
}
func (p *feeProviderFixture) SubmitFeeDebit(context.Context, billing.FeeDebitRequest) (billing.FeeDebitResult, error) {
	p.count++
	return billing.FeeDebitResult{State: "pending"}, nil
}
func (p *feeProviderFixture) ReadFeeDebit(_ context.Context, r billing.FeeDebitRequest) (billing.FeeDebitResult, error) {
	v := billing.FeeDebitResult{State: p.state}
	if p.state == "succeeded" {
		v.Amount = r.Amount
	}
	return v, nil
}
func TestAuthorizedFeesOriginalAccountAndMoneyGuards(t *testing.T) {
	f := financialFixture(t)
	ctx := context.Background()
	app := feeApp(t, f)
	p := &feeProviderFixture{state: "pending"}
	svc := billing.NewFeeService(app, p.Name(), map[string]billing.FeeProvider{p.Name(): p}, func(s string) string { return fmt.Sprintf("%x", sha256.Sum256([]byte(s))) })
	os := onboarding.NewPostgresStore(app)
	profile, err := os.Ensure(f.organization, f.user, true, true)
	if err != nil {
		t.Fatal(err)
	}
	in := billing.FeeCustomer{Email: "synthetic@example.test", Phone: "08012345678", Address: "Synthetic address", BVN: "12345678901"}
	if _, err = svc.Start(ctx, f.organization, "", in, 100000, "seller-fees-v1"); err == nil {
		t.Fatal("anonymous fee authorization")
	}
	a, err := svc.Start(ctx, f.organization, f.user, in, 100000, "seller-fees-v1")
	if err != nil {
		t.Fatal(err)
	}
	a, err = svc.Authorize(ctx, f.organization, f.user, a.ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = svc.Authorize(ctx, f.organization, f.user, a.ID); err == nil {
		t.Fatal("authorization submitted twice")
	}
	if _, _, err = os.UpdateBilling(f.organization, f.user, onboarding.BillingInput{ExpectedVersion: profile.Version, Method: "authorized_debit", ProviderReference: a.ID, Cycle: "monthly"}); err != nil {
		t.Fatal(err)
	}
	if _, err = svc.Review(ctx, f.organization, f.user, a.ID, "approve", "", "Synthetic verified receiving bank and permission evidence"); err != nil {
		t.Fatal(err)
	}
	if _, err = f.pool.Exec(ctx, `INSERT INTO app.fees(supplier_organization_id,obligation_id,fee_type,basis_amount_kobo,rate_basis_points,amount_kobo,currency,state,accrued_at) VALUES($1::uuid,$2::uuid,'base_service',50000000,1,30000,'NGN','accrued',now()-interval '40 days')`, f.organization, f.id); err != nil {
		t.Fatal(err)
	}
	store := billing.NewStore(app)
	if err = store.Issue(ctx, f.organization, time.Now().AddDate(0, 0, -8)); err != nil {
		t.Fatal(err)
	}
	bills, err := store.List(ctx, f.organization, f.user)
	if err != nil || len(bills) != 1 {
		t.Fatal("bill", bills, err)
	}
	workerCfg, e := pgxpool.ParseConfig(f.pool.Config().ConnString())
	if e != nil {
		t.Fatal(e)
	}
	workerCfg.ConnConfig.RuntimeParams["role"] = "kredit_worker"
	worker, e := pgxpool.NewWithConfig(ctx, workerCfg)
	if e != nil {
		t.Fatal(e)
	}
	defer worker.Close()
	workerService := billing.NewFeeService(worker, p.Name(), svc.Providers, svc.Fingerprint)
	if err = workerService.Run(ctx, f.organization); err != nil {
		t.Fatal(err)
	}
	if p.count != 1 {
		t.Fatal("no fee debit")
	}
	receipt := billing.Receipt{Reference: "bank:" + a.ID, Direction: "received", Amount: 30000, ReceivedAt: time.Now(), Evidence: "Verified synthetic receiving bank statement"}
	if _, err = store.RecordReceipt(ctx, f.organization, f.user, bills[0].ID, receipt); err == nil {
		t.Fatal("double receipt while fee debit pending")
	}
	svc.Active = "replacement"
	p.state = "succeeded"
	if err = svc.Run(ctx, f.organization); err != nil {
		t.Fatal(err)
	}
	if p.count != 1 {
		t.Fatal("original fee request resubmitted")
	}
	bills, err = store.List(ctx, f.organization, f.user)
	if err != nil || bills[0].Outstanding != 0 || bills[0].ProviderReceived != 30000 || bills[0].Received != 0 {
		t.Fatal("provider debit incorrectly recognized", bills, err)
	}
	if err = svc.BankReceipt(ctx, f.organization, "", p.Name(), receipt); err == nil {
		t.Fatal("anonymous bank reconciliation")
	}
	for i := 0; i < 2; i++ {
		if err = svc.BankReceipt(ctx, f.organization, f.user, p.Name(), receipt); err != nil {
			t.Fatal(err)
		}
	}
	banks, debits, err := svc.Operations(ctx, f.organization, f.user)
	if err != nil || len(banks) != 1 || banks[0].Outstanding != 0 || len(debits) != 1 {
		t.Fatal("bank reconciliation", banks, err)
	}
	handled, noticeErr := svc.Notice(ctx, p.Name(), "fee-debit-"+debits[0].ID, p.auth.ID, "synthetic-review:"+a.ID, true)
	if noticeErr != nil || !handled {
		t.Fatal("fee webhook routing", handled, noticeErr)
	}
	_, flagged, noticeErr := svc.Operations(ctx, f.organization, f.user)
	if noticeErr != nil || !flagged[0].ReviewRequired {
		t.Fatal("bank signal omitted review", noticeErr)
	}
	if err = svc.ReviewDebit(ctx, f.organization, f.user, debits[0].ID, "clear_review", "Verified original provider and bank evidence agree"); err != nil {
		t.Fatal(err)
	}
	if err = svc.ReviewDebit(ctx, f.organization, f.user, debits[0].ID, "reversed", "Verified synthetic completed bank fee reversal"); err != nil {
		t.Fatal(err)
	}
	bills, err = store.List(ctx, f.organization, f.user)
	if err != nil || bills[0].Outstanding != 30000 {
		t.Fatal("reversal did not reopen fee bill", bills, err)
	}
	var outstanding ledger.Money
	if err = f.pool.QueryRow(ctx, `SELECT outstanding_kobo FROM app.obligations WHERE id=$1::uuid`, f.id).Scan(&outstanding); err != nil || outstanding != 50000000 {
		t.Fatal("fee debit changed buyer debt", outstanding, err)
	}
}

func TestPartialSplitFeeInvoiceAndReversal(t *testing.T) {
	f := financialFixture(t)
	ctx := context.Background()
	app := feeApp(t, f)
	p := NewMockProvider("partial-fees")
	p.SetNextResponse(Response{State: ProviderPartial, ProviderCollectionID: "partial-fee-" + f.id, SucceededAmountKobo: 50000})
	engine := f.engine(p)
	engine.RequireSettlementRoute()
	if _, err := onboarding.NewPostgresStore(app).Ensure(f.organization, f.user, true, true); err != nil {
		t.Fatal(err)
	}
	if _, err := f.pool.Exec(ctx, `UPDATE app.supplier_onboarding_profiles SET settlement_state='verified',settlement_provider=$2,settlement_provider_reference='split-destination',settlement_account_name='Synthetic Seller',settlement_account_last4='7890',settlement_bank_name='Synthetic Bank',billing_state='configured',billing_method='split_settlement',billing_cycle='per_settlement',billing_provider_reference='split-billing' WHERE organization_id=$1::uuid`, f.organization, p.Name()); err != nil {
		t.Fatal(err)
	}
	if _, err := f.pool.Exec(ctx, `INSERT INTO app.invoice_billing_approvals(organization_id,billing_reference,payment_instructions,approved_by) VALUES($1::uuid,'split-billing','Synthetic verified bank instructions for unpaid fees',$2::uuid)`, f.organization, f.user); err != nil {
		t.Fatal(err)
	}
	if _, err := f.pool.Exec(ctx, `INSERT INTO app.settlement_registrations(id,organization_id,provider,connection_identity,bank_code,account_last4,state,result) VALUES($1,$2::uuid,$3,$1,'000013','7890','REGISTERED','{"provider_reference":"split-destination"}')`, fmt.Sprintf("%x", sha256.Sum256([]byte(f.id))), f.organization, p.Name()); err != nil {
		t.Fatal(err)
	}
	if _, err := f.pool.Exec(ctx, `INSERT INTO app.fees(supplier_organization_id,obligation_id,fee_type,basis_amount_kobo,rate_basis_points,amount_kobo,currency,state,accrued_at) VALUES($1::uuid,$2::uuid,'base_service',50000000,20,100000,'NGN','accrued',now()-interval '40 days')`, f.organization, f.id); err != nil {
		t.Fatal(err)
	}
	a, err := engine.Start(ctx, f.id, "partial-fees:"+f.id, time.Now())
	if err != nil || a.State != AttemptPartial {
		t.Fatal("partial collection", a.State, err)
	}
	store := billing.NewStore(app)
	for i := 0; i < 2; i++ {
		if err = store.Issue(ctx, f.organization, time.Now()); err != nil {
			t.Fatal(err)
		}
	}
	bills, err := store.List(ctx, f.organization, f.user)
	if err != nil || len(bills) != 1 || bills[0].Total != 100000 || bills[0].Deducted != 49750 || bills[0].Outstanding != 50250 {
		t.Fatal("partial fees billed twice or omitted", bills, err)
	}
	payments, err := f.payments.List(f.id)
	if err != nil || len(payments) != 1 {
		t.Fatal(err)
	}
	if _, err = f.payments.Reverse(payments[0].ID, f.user, "Verified synthetic reversal after fee billing"); err != nil {
		t.Fatal(err)
	}
	bills, err = store.List(ctx, f.organization, f.user)
	if err != nil || bills[0].Total != 100000 || bills[0].Deducted != 0 || bills[0].Outstanding != 100000 {
		t.Fatal("reversal lost previously deducted base fee", bills, err)
	}
}
