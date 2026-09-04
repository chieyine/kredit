package web

import (
	"context"
	"testing"
	"time"

	"kredit/internal/config"
	"kredit/internal/credit"
	"kredit/internal/tradelines"
)

func TestAcceptedInstalmentTermsCreateTheActivatedSchedule(t *testing.T) {
	runtime := NewRuntime(config.Config{Environment: "development", TokenHashKey: "development-test-key", Timezone: "Africa/Lagos", Currency: "NGN"})
	created, err := runtime.Credit.Create(credit.CreateInput{SupplierOrganizationID: "org-1", SupplierLegalName: "Supplier Ltd", BuyerUserID: "buyer-1", BuyerBusinessID: "business-1", BuyerLegalName: "Buyer Ltd", PrincipalKobo: 120000, GoodsDescription: "Inventory", DueDate: "2026-09-30", GraceHours: 24, CollectionAt: time.Date(2026, 9, 30, 8, 0, 0, 0, time.UTC), ScheduleType: "equal", ScheduleCount: 3, ScheduleCadence: "monthly", MonthEndPolicy: "last_day", CreatedBy: "supplier-1"})
	if err != nil {
		t.Fatal(err)
	}
	view, err := runtime.Credit.Send(created.ID, "supplier-1")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = runtime.Credit.Review(created.ID, "buyer-1"); err != nil {
		t.Fatal(err)
	}
	view, err = runtime.Credit.AuthorizeMandate(context.Background(), created.ID, "buyer-1")
	if err != nil {
		t.Fatal(err)
	}
	view, err = runtime.Credit.Accept(created.ID, "buyer-1", view.Agreement.ID, view.Agreement.DocumentHash, view.Mandate.ProviderID, "AAL2", true, true)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = runtime.Credit.Release(created.ID, "org-1", "supplier-1", "supplier_delivery", ""); err != nil {
		t.Fatal(err)
	}
	view, _, err = runtime.Credit.RecordReceipt(created.ID, "buyer-1", "confirmed", "")
	if err != nil {
		t.Fatal(err)
	}
	if view.Obligation == nil {
		t.Fatal("expected activated obligation")
	}
	_, items, err := runtime.Schedules.GetForObligation(view.Obligation.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 3 || items[0].PrincipalDueKobo+items[1].PrincipalDueKobo+items[2].PrincipalDueKobo != 120000 {
		t.Fatalf("unexpected accepted schedule: %#v", items)
	}
}

func TestTradeLineReceiptCreatesItsOwnObligationScheduleAndBalancedLedger(t *testing.T) {
	runtime := NewRuntime(config.Config{Environment: "development", TokenHashKey: "development-test-key", Timezone: "Africa/Lagos", Currency: "NGN"})
	now := time.Now().UTC()
	line, err := runtime.TradeLines.CreateLine(tradelines.CreateLineInput{SupplierOrganizationID: "org-1", BuyerUserID: "buyer-1", BuyerBusinessID: "business-1", ApprovedLimitKobo: 500000, Cadence: "monthly", StartAt: now.Add(-time.Hour), EndAt: now.AddDate(1, 0, 0), MandateID: "mandate-1", MandateActive: true, MandateVerified: true})
	if err != nil {
		t.Fatal(err)
	}
	drawdown, _, _, err := runtime.TradeLines.ReserveDrawdown(tradelines.CreateDrawdownInput{LineID: line.ID, PrincipalKobo: 125000, GoodsDescription: "repeat inventory", DueDate: "2026-09-30", CollectionAt: time.Date(2026, 10, 1, 9, 0, 0, 0, time.UTC), IdempotencyKey: "runtime-drawdown"})
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err = runtime.TradeLines.ConfirmDrawdown(drawdown.ID, "buyer-1", drawdown.AgreementHash); err != nil {
		t.Fatal(err)
	}
	if _, _, err = runtime.TradeLines.ReleaseDrawdown(tradelines.ReleaseInput{DrawdownID: drawdown.ID, SupplierOrganizationID: "org-1", ActorID: "supplier-1", DeliveryMethod: "courier", EvidenceReference: "TRACK-1"}); err != nil {
		t.Fatal(err)
	}
	activated, activatedLine, err := runtime.TradeLines.RecordDrawdownReceipt(tradelines.ReceiptInput{DrawdownID: drawdown.ID, BuyerUserID: "buyer-1", State: "no_issue"})
	if err != nil || activated.ObligationID == "" || activatedLine.CurrentExposureKobo != 125000 || activatedLine.ReservedPendingKobo != 0 {
		t.Fatalf("activation failed: drawdown=%+v line=%+v err=%v", activated, activatedLine, err)
	}
	view, err := runtime.Credit.GetForSupplier(drawdown.ID, "org-1")
	if err != nil || view.Obligation == nil || view.Obligation.ID != activated.ObligationID {
		t.Fatalf("internal obligation missing: view=%+v err=%v", view, err)
	}
	_, items, err := runtime.Schedules.GetForObligation(activated.ObligationID)
	if err != nil || len(items) != 1 || items[0].PrincipalDueKobo != 125000 {
		t.Fatalf("drawdown schedule mismatch: items=%+v err=%v", items, err)
	}
	if !items[0].CollectionAt.Equal(drawdown.CollectionAt) {
		t.Fatalf("accepted debit date changed: %s != %s", items[0].CollectionAt, drawdown.CollectionAt)
	}
	verifyRevolvingRepayments(t, runtime, line.ID, activated.ObligationID, "supplier-1")
	transactions, err := runtime.Ledger.GetByReference(activated.ObligationID)
	if err != nil || len(transactions) != 1 {
		t.Fatalf("activation ledger missing: transactions=%+v err=%v", transactions, err)
	}
	var debits, credits int64
	for _, posting := range transactions[0].Postings {
		debits += int64(posting.Debit)
		credits += int64(posting.Credit)
	}
	if debits != credits || debits != 125625 {
		t.Fatalf("activation ledger is not balanced: debits=%d credits=%d", debits, credits)
	}
}
