package orders

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"kredit/internal/billing"
	"kredit/internal/db"
	"kredit/internal/ledger"
	"kredit/internal/payments"
)

func TestAuditSplitReversalRetainsEvidenceAndIsReplaySafe(t *testing.T) {
	f := newAuditFixture(t)
	store := payments.NewPostgresStore(f.runtime.Raw(), nil, nil)
	p, _, err := store.RecordContext(f.as(f.finance), payments.RecordInput{ObligationID: f.obligation[0], SourceType: payments.SourceSupplierTransfer, AmountKobo: 2500, RecordedBy: f.finance, IdempotencyKey: "audit-split:" + f.obligation[0]})
	if err != nil {
		t.Fatal(err)
	}
	// Install synthetic historical split evidence with the fixture owner. The
	// operation under test (including all undo writes) uses the restricted login.
	fee := uuid.NewString()
	tx, err := f.admin.Begin(f.ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(f.ctx) }()
	if _, err = tx.Exec(f.ctx, `INSERT INTO app.fees(id,supplier_organization_id,obligation_id,fee_type,basis_amount_kobo,rate_basis_points,amount_kobo,collected_kobo,currency,state) VALUES($1::uuid,$2::uuid,$3::uuid,'base_service',10000,500,500,500,'NGN','accrued')`, fee, f.org, f.obligation[0]); err != nil {
		t.Fatal(err)
	}
	if _, err = tx.Exec(f.ctx, `INSERT INTO app.split_fee_allocations(payment_id,fee_id,supplier_organization_id,amount_kobo) VALUES($1::uuid,$2::uuid,$3::uuid,500)`, p.ID, fee, f.org); err != nil {
		t.Fatal(err)
	}
	if _, err = ledger.NewPostgresStore(nil).PostSplitFeeTx(f.ctx, tx, p.ID, 500, false, time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	if err = tx.Commit(f.ctx); err != nil {
		t.Fatal(err)
	}
	if _, err = store.ReverseContext(f.as(f.finance), p.ID, f.finance, "synthetic confirmed reversal"); err != nil {
		t.Fatal(err)
	}
	if n := f.scalar(t, `SELECT amount_kobo FROM app.split_fee_allocations WHERE payment_id=$1::uuid`, p.ID); n != 500 {
		t.Fatal("original allocation evidence changed")
	}
	if n := f.scalar(t, `SELECT collected_kobo FROM app.fees WHERE id=$1::uuid`, fee); n != 0 {
		t.Fatalf("fee undo=%d", n)
	}
	if _, err = store.ReverseContext(f.as(f.finance), p.ID, f.finance, "repeat confirmed reversal"); err != nil {
		t.Fatal(err)
	}
	// The composing helper itself must be replay-safe, not just the outer method.
	again, err := f.runtime.Raw().Begin(f.as(f.finance))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = again.Rollback(f.ctx) }()
	if err = db.SetTenantContext(f.as(f.finance), again); err != nil {
		t.Fatal(err)
	}
	if err = billing.ReverseSplitTx(f.as(f.finance), again, p.ID, time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	if err = again.Commit(f.ctx); err != nil {
		t.Fatal(err)
	}
	if n := f.scalar(t, `SELECT collected_kobo FROM app.fees WHERE id=$1::uuid`, fee); n != 0 {
		t.Fatal("replay subtracted fee twice")
	}
	if n := f.scalar(t, `SELECT count(*) FROM ledger.transactions WHERE idempotency_key=$1`, "split_fee_reversed:"+p.ID); n != 1 {
		t.Fatal("duplicate undo journal")
	}
	if n := f.scalar(t, `SELECT outstanding_kobo FROM app.obligations WHERE id=$1::uuid`, f.obligation[0]); n != 10000 {
		t.Fatal("incorrect restored buyer balance")
	}
}
