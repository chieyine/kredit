package credit

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"kredit/internal/ledger"
	"kredit/internal/mandates"
)

func TestTradeLineDrawdownActivationCreatesOneObligationAndLedgerEntry(t *testing.T) {
	ledgerStore := ledger.NewStore()
	store := NewStore(mandates.NewMockProvider(), ledgerStore)
	now := time.Now().UTC()
	input := TradeLineActivationInput{DrawdownID: "drawdown-1", TradeLineID: "line-1", SupplierOrganizationID: "org-1", BuyerUserID: "buyer-1", BuyerBusinessID: "business-1", MandateID: "mandate-1", PrincipalKobo: 125000, GoodsDescription: "shop inventory", DueDate: "2026-09-30", GraceHours: 24, CollectionAt: now.Add(24 * time.Hour), TermsVersion: "terms-v1", DrawdownAgreementHash: "accepted-drawdown-hash", BuyerConfirmedAt: now.Add(-2 * time.Hour), ReleaseActorID: "supplier-user", DeliveryMethod: "courier", ReleasedAt: now.Add(-time.Hour), ReceiptActorID: "buyer-1", ReceiptAt: now}
	view, transaction, err := store.ActivateTradeLineDrawdown(input)
	if err != nil || transaction == nil || view.Obligation == nil || view.Request.ID != input.DrawdownID || view.Obligation.PrincipalKobo != input.PrincipalKobo {
		t.Fatalf("activation incomplete: view=%+v transaction=%+v err=%v", view, transaction, err)
	}
	replayed, secondTransaction, err := store.ActivateTradeLineDrawdown(input)
	if err != nil || secondTransaction != nil || replayed.Obligation == nil || replayed.Obligation.ID != view.Obligation.ID {
		t.Fatalf("activation replay duplicated money: view=%+v transaction=%+v err=%v", replayed, secondTransaction, err)
	}
}

func TestInstalmentTermsArePartOfImmutableAgreement(t *testing.T) {
	store := NewStore(mandates.NewMockProvider(), ledger.NewStore())
	created, err := store.Create(CreateInput{SupplierOrganizationID: "org", SupplierLegalName: "Supplier", BuyerUserID: "buyer", BuyerBusinessID: "biz", BuyerLegalName: "Buyer", PrincipalKobo: 120000, GoodsDescription: "goods", DueDate: "2026-09-30", CollectionAt: time.Date(2026, 9, 30, 9, 0, 0, 0, time.UTC), ScheduleType: "equal", ScheduleCount: 3, ScheduleCadence: "monthly", MonthEndPolicy: "last_day", CreatedBy: "creator"})
	if err != nil {
		t.Fatal(err)
	}
	view, err := store.Send(created.ID, "creator")
	if err != nil {
		t.Fatal(err)
	}
	var canonical map[string]any
	if err := json.Unmarshal(view.Agreement.CanonicalJSON, &canonical); err != nil {
		t.Fatal(err)
	}
	if canonical["schedule_type"] != "equal" || canonical["schedule_cadence"] != "monthly" || canonical["schedule_count"] != float64(3) || canonical["month_end_policy"] != "last_day" {
		t.Fatalf("schedule terms missing from canonical agreement: %#v", canonical)
	}
}

func TestCustomInstalmentTermsMustEqualPrincipal(t *testing.T) {
	store := NewStore(mandates.NewMockProvider(), ledger.NewStore())
	_, err := store.Create(CreateInput{SupplierOrganizationID: "org", SupplierLegalName: "Supplier", BuyerUserID: "buyer", BuyerBusinessID: "biz", BuyerLegalName: "Buyer", PrincipalKobo: 100000, GoodsDescription: "goods", DueDate: "2026-09-30", CollectionAt: time.Now().UTC().Add(time.Hour), ScheduleType: "custom", ScheduleCadence: "custom", CustomScheduleItems: []ScheduleTerm{{AmountKobo: 40000, DueDate: "2026-09-30"}, {AmountKobo: 50000, DueDate: "2026-10-30"}}, CreatedBy: "creator"})
	if err == nil {
		t.Fatal("expected custom schedule total validation")
	}
}

func TestCreditLifecycleActivatesObligationAndPostsBalancedLedger(t *testing.T) {
	provider := mandates.NewMockProvider()
	ledgerStore := ledger.NewStore()
	store := NewStore(provider, ledgerStore)
	created, err := store.Create(CreateInput{SupplierOrganizationID: "org-1", SupplierLegalName: "Supplier Ltd", BuyerUserID: "buyer-1", BuyerBusinessID: "biz-1", BuyerLegalName: "Buyer Ltd", PrincipalKobo: 100000, Currency: "NGN", GoodsDescription: "Cement", InvoiceReference: "INV-1", DueDate: "2026-09-01", GraceHours: 24, CollectionAt: time.Now().UTC().Add(24 * time.Hour), CreatedBy: "supplier-1"})
	if err != nil {
		t.Fatal(err)
	}
	sent, err := store.Send(created.ID, "supplier-1")
	if err != nil {
		t.Fatal(err)
	}
	if sent.Agreement.DocumentHash == "" {
		t.Fatal("expected agreement hash")
	}
	reviewed, err := store.Review(created.ID, "buyer-1")
	if err != nil {
		t.Fatal(err)
	}
	mandated, err := store.AuthorizeMandate(context.Background(), created.ID, "buyer-1")
	if err != nil {
		t.Fatal(err)
	}
	accepted, err := store.Accept(created.ID, "buyer-1", reviewed.Agreement.ID, reviewed.Agreement.DocumentHash, mandated.Mandate.ProviderID, "AAL2", true, true)
	if err != nil {
		t.Fatal(err)
	}
	if accepted.Request.State != ReadyToRelease {
		t.Fatalf("state=%s", accepted.Request.State)
	}
	if _, err := store.Release(created.ID, "org-1", "supplier-1", "courier", "dispatch note"); err != nil {
		t.Fatal(err)
	}
	active, tx, err := store.RecordReceipt(created.ID, "buyer-1", "confirmed", "")
	if err != nil {
		t.Fatal(err)
	}
	if active.Request.State != Active || active.Obligation == nil || tx == nil {
		t.Fatalf("activation incomplete: %#v", active)
	}
	if active.Obligation.BaseFeeKobo != 500 {
		t.Fatalf("base fee=%d", active.Obligation.BaseFeeKobo)
	}
	if len(tx.Postings) != 4 {
		t.Fatalf("postings=%d", len(tx.Postings))
	}
	if tx.Postings[0].Debit != 100000 || tx.Postings[1].Credit != 100000 {
		t.Fatal("principal postings incorrect")
	}
	if tx.Postings[2].Debit != 500 || tx.Postings[3].Credit != 500 {
		t.Fatal("fee postings incorrect")
	}
	if tx2, err := ledgerStore.PostActivation(created.ID, 100000, time.Now(), created.ID+":activation"); err != nil || tx2.ID != tx.ID {
		t.Fatal("activation was not idempotent")
	}
}

func TestSendRequiresTheRequestCreator(t *testing.T) {
	store := NewStore(mandates.NewMockProvider(), ledger.NewStore())
	created, err := store.Create(CreateInput{SupplierOrganizationID: "org", SupplierLegalName: "Supplier", BuyerUserID: "buyer", BuyerBusinessID: "biz", BuyerLegalName: "Buyer", PrincipalKobo: 1000, GoodsDescription: "goods", DueDate: "2026-09-01", CollectionAt: time.Now().UTC().Add(time.Hour), CreatedBy: "creator"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Send(created.ID, "other-user"); err == nil {
		t.Fatal("expected non-creator send to be rejected")
	}
}

func TestIssueReceiptDoesNotActivate(t *testing.T) {
	provider := mandates.NewMockProvider()
	store := NewStore(provider, ledger.NewStore())
	created, err := store.Create(CreateInput{SupplierOrganizationID: "org", SupplierLegalName: "Supplier", BuyerUserID: "buyer", BuyerBusinessID: "biz", BuyerLegalName: "Buyer", PrincipalKobo: 1000, GoodsDescription: "goods", DueDate: "2026-09-01", CollectionAt: time.Now().UTC().Add(time.Hour), CreatedBy: "supplier"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = store.Send(created.ID, "supplier"); err != nil {
		t.Fatal(err)
	}
	if _, err = store.Review(created.ID, "buyer"); err != nil {
		t.Fatal(err)
	}
	v, err := store.AuthorizeMandate(context.Background(), created.ID, "buyer")
	if err != nil {
		t.Fatal(err)
	}
	a := v.Agreement
	if _, err = store.Accept(created.ID, "buyer", a.ID, a.DocumentHash, v.Mandate.ProviderID, "AAL1", true, true); err != nil {
		t.Fatal(err)
	}
	if _, err = store.Release(created.ID, "org", "supplier", "pickup", ""); err != nil {
		t.Fatal(err)
	}
	v, tx, err := store.RecordReceipt(created.ID, "buyer", "issue_raised", "damaged")
	if err != nil || tx != nil {
		t.Fatalf("receipt err=%v tx=%v", err, tx)
	}
	if v.Request.State != ReceiptConfirmationPending {
		t.Fatal(v.Request.State)
	}
}

func TestDraftAmendmentUsesOptimisticVersionAndBecomesImmutableOnSend(t *testing.T) {
	store := NewStore(mandates.NewMockProvider(), ledger.NewStore())
	created, err := store.Create(CreateInput{SupplierOrganizationID: "org", SupplierLegalName: "Supplier", BuyerUserID: "buyer", BuyerBusinessID: "biz", BuyerLegalName: "Buyer", PrincipalKobo: 1000, GoodsDescription: "goods", DueDate: "2026-09-01", CollectionAt: time.Now().UTC().Add(time.Hour), CreatedBy: "creator"})
	if err != nil {
		t.Fatal(err)
	}
	updated, err := store.UpdateDraft(created.ID, "creator", UpdateDraftInput{ExpectedVersion: created.Version, PrincipalKobo: 2500, GoodsDescription: "updated exact goods", DueDate: "2026-09-02", GraceHours: 12, CollectionAt: time.Now().UTC().Add(2 * time.Hour)})
	if err != nil {
		t.Fatal(err)
	}
	if updated.PrincipalKobo != 2500 || updated.Version != created.Version+1 {
		t.Fatalf("unexpected update: %#v", updated)
	}
	if _, err := store.UpdateDraft(created.ID, "creator", UpdateDraftInput{ExpectedVersion: created.Version, PrincipalKobo: 3000, GoodsDescription: "stale", DueDate: "2026-09-03", CollectionAt: time.Now().UTC().Add(3 * time.Hour)}); err == nil {
		t.Fatal("expected stale version conflict")
	}
	if _, err := store.Send(created.ID, "creator"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.UpdateDraft(created.ID, "creator", UpdateDraftInput{ExpectedVersion: updated.Version + 1, PrincipalKobo: 3000, GoodsDescription: "late change", DueDate: "2026-09-03", CollectionAt: time.Now().UTC().Add(3 * time.Hour)}); err == nil {
		t.Fatal("accepted presentation must be immutable after send")
	}
}

func TestCancellationAndDeclinePreserveTerminalHistory(t *testing.T) {
	store := NewStore(mandates.NewMockProvider(), ledger.NewStore())
	create := func(buyer string) CreditRequest {
		request, err := store.Create(CreateInput{SupplierOrganizationID: "org", SupplierLegalName: "Supplier", BuyerUserID: buyer, BuyerBusinessID: "biz-" + buyer, BuyerLegalName: "Buyer", PrincipalKobo: 1000, GoodsDescription: "goods", DueDate: "2026-09-01", CollectionAt: time.Now().UTC().Add(time.Hour), CreatedBy: "creator"})
		if err != nil {
			t.Fatal(err)
		}
		return request
	}
	cancelled := create("buyer-cancel")
	view, err := store.Cancel(cancelled.ID, "creator")
	if err != nil || view.Request.State != Cancelled {
		t.Fatalf("cancel view=%#v err=%v", view, err)
	}
	if _, err := store.Send(cancelled.ID, "creator"); err == nil {
		t.Fatal("cancelled request must not be sendable")
	}
	declined := create("buyer-decline")
	if _, err := store.Send(declined.ID, "creator"); err != nil {
		t.Fatal(err)
	}
	view, err = store.Decline(declined.ID, "buyer-decline")
	if err != nil || view.Request.State != Declined {
		t.Fatalf("decline view=%#v err=%v", view, err)
	}
	if _, err := store.AuthorizeMandate(context.Background(), declined.ID, "buyer-decline"); err == nil {
		t.Fatal("declined request must not continue")
	}
}
