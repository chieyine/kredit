package orders

import (
	"context"
	"testing"
	"time"

	"kredit/internal/db"
	"kredit/internal/erp"
	"kredit/internal/ledger"
	"kredit/internal/payments"
)

func TestAuditERPIncludesPaymentsReversalsAndCreditNotes(t *testing.T) {
	f := newAuditFixture(t)
	now := time.Now().UTC().Truncate(time.Microsecond)
	journal := ledger.NewPostgresStore(f.admin)
	if _, err := journal.PostActivationWithFee(f.obligation[0], 10000, 0, now, "erp-origin:"+f.obligation[0]); err != nil {
		t.Fatal(err)
	}
	paymentStore := payments.NewPostgresStore(f.runtime.Raw(), nil, nil)
	p, _, err := paymentStore.RecordContext(f.as(f.finance), payments.RecordInput{ObligationID: f.obligation[0], SourceType: payments.SourceSupplierTransfer, AmountKobo: 2500, RecordedBy: f.finance, IdempotencyKey: "erp-payment:" + f.obligation[0]})
	if err != nil {
		t.Fatal(err)
	}
	note, err := f.store().CreateCreditNote(f.as(f.sales), CreditNoteInput{OrderID: f.order[0], SupplierOrganizationID: f.org, IssuedBy: f.sales, AmountKobo: 1000, Reason: "verified synthetic shortage"})
	if err != nil {
		t.Fatal(err)
	}
	if err = f.store().ApproveCreditNote(f.as(f.finance), note.ID, f.finance); err != nil {
		t.Fatal(err)
	}
	check := func(count int, total ledger.Money) {
		t.Helper()
		records, err := erp.LoadLedgerMovements(f.as(f.finance), f.runtime.Raw(), f.org, now.Add(-time.Hour), time.Now().UTC().Add(time.Hour))
		if err != nil {
			t.Fatal(err)
		}
		var sum ledger.Money
		external := []erp.ExternalRecord{}
		for _, r := range records {
			sum += r.AmountKobo
			if r.Reference != r.TransactionID || r.Reference == f.obligation[0] {
				t.Fatal("not transaction-grained")
			}
			external = append(external, erp.ExternalRecord{Reference: r.Reference, AmountKobo: r.AmountKobo, Date: r.Date})
		}
		if len(records) != count || sum != total {
			t.Fatalf("lost signed movements: count=%d total=%d records=%+v", len(records), sum, records)
		}
		report, err := erp.ReconcileExternal(f.org, now.Add(-time.Hour), time.Now().UTC().Add(time.Hour), records, external)
		if err != nil || !report.Balanced {
			t.Fatalf("cannot reconcile: %+v %v", report, err)
		}
	}
	check(3, 6500)
	if _, err = paymentStore.ReverseContext(f.as(f.finance), p.ID, f.finance, "synthetic confirmed reversal"); err != nil {
		t.Fatal(err)
	}
	check(4, 9000)
	for _, ctx := range []context.Context{f.ctx, db.WithTenantContext(f.ctx, f.finance, f.buyerOrg), f.as(f.other), f.as(f.sales)} {
		if _, err = erp.LoadLedgerMovements(ctx, f.runtime.Raw(), f.org, now.Add(-time.Hour), now.Add(time.Hour)); err == nil {
			t.Fatal("unauthorized ledger read accepted")
		}
	}
	records, err := erp.LoadLedgerMovements(f.as(f.finance), f.runtime.Raw(), f.org, now.Add(-48*time.Hour), now.Add(-24*time.Hour))
	if err != nil || len(records) != 0 {
		t.Fatalf("period leaked: %+v %v", records, err)
	}
}
