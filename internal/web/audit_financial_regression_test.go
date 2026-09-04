package web

import (
	"kredit/internal/payments"
	"kredit/internal/tradelines"
	"testing"
	"time"
)

func verifyRevolvingRepayments(t *testing.T, runtime *Runtime, lineID, obligationID, actor string) {
	t.Helper()
	line, ok := runtime.TradeLines.Get(lineID)
	if !ok {
		t.Fatal("trade line not found")
	}
	check := func(exposure int64) {
		t.Helper()
		got, found := runtime.TradeLines.Get(lineID)
		wantAvailable := max(0, int64(line.ApprovedLimitKobo)-exposure)
		if !found || int64(got.CurrentExposureKobo) != exposure || int64(got.AvailableLimitKobo) != wantAvailable {
			t.Fatalf("capacity mismatch: %+v err=%v want exposure=%d available=%d", got, found, exposure, wantAvailable)
		}
	}
	input := payments.RecordInput{ObligationID: obligationID, AmountKobo: 25000, SourceType: payments.SourceVoluntary, RecordedBy: actor, IdempotencyKey: "audit-partial-" + obligationID}
	partial, _, err := runtime.Payments.Record(input)
	if err != nil {
		t.Fatal(err)
	}
	check(100000)
	if _, _, err = runtime.Payments.Record(input); err != nil {
		t.Fatal(err)
	}
	check(100000)
	if _, err = runtime.Payments.Reverse(partial.ID, actor, "correct duplicate receipt"); err != nil {
		t.Fatal(err)
	}
	check(125000)
	full, _, err := runtime.Payments.Record(payments.RecordInput{ObligationID: obligationID, AmountKobo: 125000, SourceType: payments.SourceVoluntary, RecordedBy: actor, IdempotencyKey: "audit-full-" + obligationID})
	if err != nil {
		t.Fatal(err)
	}
	check(0)
	// Reuse the released capacity, then reverse the old payment. Debt must remain
	// truthful even above the line limit, with no new borrowing available.
	draw, _, _, err := runtime.TradeLines.ReserveDrawdown(tradelines.CreateDrawdownInput{LineID: line.ID, PrincipalKobo: line.ApprovedLimitKobo, GoodsDescription: "replacement inventory", DueDate: "2026-10-15", CollectionAt: time.Date(2026, 10, 16, 9, 0, 0, 0, time.UTC), IdempotencyKey: "audit-reuse-" + line.ID})
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err = runtime.TradeLines.ConfirmDrawdown(draw.ID, line.BuyerUserID, draw.AgreementHash); err != nil {
		t.Fatal(err)
	}
	if _, _, err = runtime.TradeLines.ReleaseDrawdown(tradelines.ReleaseInput{DrawdownID: draw.ID, SupplierOrganizationID: line.SupplierOrganizationID, ActorID: actor, DeliveryMethod: "courier", EvidenceReference: "audit-reuse"}); err != nil {
		t.Fatal(err)
	}
	if _, _, err = runtime.TradeLines.RecordDrawdownReceipt(tradelines.ReceiptInput{DrawdownID: draw.ID, BuyerUserID: line.BuyerUserID, State: "no_issue"}); err != nil {
		t.Fatal(err)
	}
	if _, err = runtime.Payments.Reverse(full.ID, actor, "bank reversed the transfer"); err != nil {
		t.Fatal(err)
	}
	check(int64(line.ApprovedLimitKobo) + 125000)
	if _, _, _, err = runtime.TradeLines.ReserveDrawdown(tradelines.CreateDrawdownInput{LineID: line.ID, PrincipalKobo: 1, GoodsDescription: "blocked", IdempotencyKey: "audit-over-limit-" + line.ID}); err == nil {
		t.Fatal("over-limit line accepted more exposure")
	}
}
