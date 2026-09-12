package collections

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"kredit/internal/onboarding"
	"kredit/internal/settlement"
)

func TestFrozenSettlementAndBankReceiptPermissions(t *testing.T) {
	f := financialFixture(t)
	ctx := context.Background()
	provider := NewMockProvider("settlement-fixture")
	engine := f.engine(provider)
	engine.RequireSettlementRoute()
	if _, err := engine.Start(ctx, f.id, "unconfigured:"+f.id, time.Now()); err == nil {
		t.Fatal("debit without bank destination")
	}
	if _, err := onboarding.NewPostgresStore(f.pool).Ensure(f.organization, f.user, true, true); err != nil {
		t.Fatal(err)
	}
	if _, err := f.pool.Exec(ctx, `UPDATE app.supplier_onboarding_profiles SET settlement_state='verified',settlement_provider=$2,settlement_provider_reference='original-destination',settlement_account_name='Original Seller',settlement_account_last4='7890',settlement_bank_name='Synthetic Bank',billing_state='configured',billing_method='consolidated_invoice',billing_provider_reference='synthetic-invoice' WHERE organization_id=$1::uuid`, f.organization, provider.Name()); err != nil {
		t.Fatal(err)
	}
	if _, err := f.pool.Exec(ctx, `INSERT INTO app.settlement_registrations(id,organization_id,provider,connection_identity,bank_code,account_last4,state,result) VALUES($1,$2::uuid,$3,'0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef','000013','7890','REGISTERED','{"provider_reference":"original-destination"}')`, fmt.Sprintf("%x", sha256.Sum256([]byte(f.id))), f.organization, provider.Name()); err != nil {
		t.Fatal(err)
	}
	a, err := engine.Start(ctx, f.id, "with-route:"+f.id, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if a.SettlementRoute == nil || a.SettlementRoute.Destination != "original-destination" {
		t.Fatal("destination was not frozen")
	}
	if _, err = f.pool.Exec(ctx, `UPDATE app.supplier_onboarding_profiles SET settlement_provider_reference='changed-destination' WHERE organization_id=$1::uuid`, f.organization); err != nil {
		t.Fatal(err)
	}
	saved, ok := f.engine(provider).GetAttempt(a.ID)
	if !ok || saved.SettlementRoute == nil || saved.SettlementRoute.Destination != "original-destination" {
		t.Fatal("bank change redirected saved collection")
	}
	if _, err = f.pool.Exec(ctx, `INSERT INTO app.platform_role_assignments(user_id,role,granted_by,reason) VALUES($1::uuid,'platform_owner',$1::uuid,'Synthetic settlement owner')`, f.user); err != nil {
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
	store := settlement.NewReceiptStore(app)
	items, err := store.List(ctx, f.user, "")
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, item := range items {
		if item.AttemptID == a.ID {
			found = true
			if item.Outstanding != a.SucceededAmountKobo {
				t.Fatal("debit was incorrectly treated as seller payout")
			}
		}
	}
	if !found {
		t.Fatal("settlement absent from owner queue")
	}
	in := settlement.BankReceipt{Reference: "synthetic-bank:" + a.ID, Amount: 10000000, Direction: "paid", OccurredAt: time.Now().Truncate(time.Microsecond), Evidence: "Verified synthetic bank credit to original destination"}
	if _, err = store.Record(ctx, "", f.organization, a.ID, in); err == nil {
		t.Fatal("anonymous settlement accepted")
	}
	for i := 0; i < 2; i++ {
		v, err := store.Record(ctx, f.user, f.organization, a.ID, in)
		if err != nil || v.Paid != in.Amount {
			t.Fatal("receipt replay", err)
		}
	}
	excess := in
	excess.Reference += "-excess"
	excess.Amount = a.SucceededAmountKobo
	if _, err = store.Record(ctx, f.user, f.organization, a.ID, excess); err == nil {
		t.Fatal("overpayment accepted")
	}
	returned := in
	returned.Reference += "-return"
	returned.Direction = "returned"
	returned.Amount = 5000000
	v, err := store.Record(ctx, f.user, f.organization, a.ID, returned)
	if err != nil || v.Returned != returned.Amount {
		t.Fatal("returned bank transfer", err)
	}
	var outstanding int64
	if err = f.pool.QueryRow(ctx, `SELECT outstanding_kobo FROM app.obligations WHERE id=$1::uuid`, f.id).Scan(&outstanding); err != nil || outstanding != 0 {
		t.Fatal("settlement changed customer debt", err)
	}
	raw, _ := json.Marshal(map[string]string{"destination": "altered"})
	if _, err = app.Exec(ctx, `UPDATE app.collection_settlement_routes SET route=$2::jsonb WHERE attempt_id=$1::uuid`, a.ID, raw); err == nil {
		t.Fatal("frozen route was mutable")
	}
}
