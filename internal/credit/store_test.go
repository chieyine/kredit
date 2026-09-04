package credit

import (
	"context"
	"encoding/json"
	"errors"
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

// releaseSaleForTest drives a request from draft to RECEIPT_CONFIRMATION_PENDING
// so deemed-acceptance behaviour can be exercised without repeating the whole
// lifecycle in every test.
func releaseSaleForTest(t *testing.T, store *Store, buyerBusinessID string) CreditRequest {
	t.Helper()
	created, err := store.Create(CreateInput{SupplierOrganizationID: "org", SupplierLegalName: "Supplier", BuyerUserID: "buyer", BuyerBusinessID: buyerBusinessID, BuyerLegalName: "Buyer", PrincipalKobo: 1000, GoodsDescription: "goods", DueDate: "2026-09-01", CollectionAt: store.now().Add(time.Hour), CreatedBy: "creator"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = store.Send(created.ID, "creator"); err != nil {
		t.Fatal(err)
	}
	if _, err = store.Review(created.ID, "buyer"); err != nil {
		t.Fatal(err)
	}
	view, err := store.AuthorizeMandate(context.Background(), created.ID, "buyer")
	if err != nil {
		t.Fatal(err)
	}
	agreement := view.Agreement
	if _, err = store.Accept(created.ID, "buyer", agreement.ID, agreement.DocumentHash, view.Mandate.ProviderID, "AAL1", true, true); err != nil {
		t.Fatal(err)
	}
	if _, err = store.Release(created.ID, "org", "supplier", "pickup", "Waybill #1042"); err != nil {
		t.Fatal(err)
	}
	return created
}

// A buyer who has never answered one of our notices has not been shown to
// receive them at all, so their first sale must never be activated by silence.
func TestFirstTradeCreditIsNeverActivatedBySilence(t *testing.T) {
	now := time.Now().UTC()
	store := NewStore(mandates.NewMockProvider(), ledger.NewStore())
	store.now = func() time.Time { return now }
	created := releaseSaleForTest(t, store, "first-time-buyer")

	activated, err := store.AutoActivateMatured(context.Background(), now.Add(30*24*time.Hour))
	if err != nil || len(activated) != 0 {
		t.Fatalf("a first trade credit must not activate on silence: activated=%v err=%v", activated, err)
	}
	view, err := store.GetForSupplier(created.ID, "org")
	if err != nil {
		t.Fatal(err)
	}
	if view.Request.State != ReceiptConfirmationPending {
		t.Fatalf("expected the sale to stay pending an explicit answer, got %s", view.Request.State)
	}
}

func TestDeemedAcceptanceWaitsTheFullWindowForAKnownBuyer(t *testing.T) {
	now := time.Now().UTC()
	store := NewStore(mandates.NewMockProvider(), ledger.NewStore())
	store.now = func() time.Time { return now }

	// The buyer answers their first sale themselves, which is what establishes
	// that our notices reach them.
	first := releaseSaleForTest(t, store, "biz")
	if _, _, err := store.RecordReceipt(first.ID, "buyer", "confirmed", ""); err != nil {
		t.Fatal(err)
	}

	second := releaseSaleForTest(t, store, "biz")
	// One hour short of the window the sale is still the buyer's to answer.
	activated, err := store.AutoActivateMatured(context.Background(), now.Add(deemedAcceptanceWindow-time.Hour))
	if err != nil || len(activated) != 0 {
		t.Fatalf("expected no activation before the window elapses, got %v err=%v", activated, err)
	}
	// A day short of the old twenty-four hour window would already have swept.
	activated, err = store.AutoActivateMatured(context.Background(), now.Add(25*time.Hour))
	if err != nil || len(activated) != 0 {
		t.Fatalf("expected no activation twenty-five hours after release, got %v err=%v", activated, err)
	}
	activated, err = store.AutoActivateMatured(context.Background(), now.Add(deemedAcceptanceWindow+time.Hour))
	if err != nil || len(activated) != 1 || activated[0] != second.ID {
		t.Fatalf("expected the second sale to activate after the window, got %v err=%v", activated, err)
	}
	view, err := store.GetForSupplier(second.ID, "org")
	if err != nil {
		t.Fatal(err)
	}
	if view.Request.State != Active {
		t.Fatalf("expected Active, got %s", view.Request.State)
	}
}

// The gate is the notification-delivery evidence check. A gate that cannot
// prove the buyer was told must leave the sale alone rather than activate it.
func TestDeemedAcceptanceRefusesWhenTheEvidenceGateFails(t *testing.T) {
	now := time.Now().UTC()
	store := NewStore(mandates.NewMockProvider(), ledger.NewStore())
	store.now = func() time.Time { return now }
	first := releaseSaleForTest(t, store, "biz")
	if _, _, err := store.RecordReceipt(first.ID, "buyer", "confirmed", ""); err != nil {
		t.Fatal(err)
	}
	second := releaseSaleForTest(t, store, "biz")

	calls := 0
	store.SetDeemedAcceptanceGate(func(context.Context, string) error {
		calls++
		return errors.New("delivery receipt unavailable")
	})
	activated, err := store.AutoActivateMatured(context.Background(), now.Add(deemedAcceptanceWindow+time.Hour))
	if err != nil || len(activated) != 0 {
		t.Fatalf("a refused gate must block activation, got %v err=%v", activated, err)
	}
	if calls == 0 {
		t.Fatal("the gate was never consulted")
	}
	view, err := store.GetForSupplier(second.ID, "org")
	if err != nil {
		t.Fatal(err)
	}
	if view.Request.State != ReceiptConfirmationPending {
		t.Fatalf("expected the sale to stay pending, got %s", view.Request.State)
	}

	// Once the evidence exists, the same sale activates.
	store.SetDeemedAcceptanceGate(func(context.Context, string) error { return nil })
	activated, err = store.AutoActivateMatured(context.Background(), now.Add(deemedAcceptanceWindow+time.Hour))
	if err != nil || len(activated) != 1 || activated[0] != second.ID {
		t.Fatalf("expected activation once the gate allows it, got %v err=%v", activated, err)
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

func TestAcceptanceIsIndependentOfBankAuthorization(t *testing.T) {
	s := NewStore(mandates.NewMockProvider(), ledger.NewStore())
	r, err := s.Create(CreateInput{SupplierOrganizationID: "supplier", SupplierLegalName: "Supplier", BuyerUserID: "buyer", BuyerBusinessID: "business", BuyerLegalName: "Buyer", PrincipalKobo: 50000000, GoodsDescription: "Trade goods", DueDate: "2026-09-30", CollectionAt: time.Now().Add(time.Hour), CreatedBy: "supplier"})
	if err != nil {
		t.Fatal(err)
	}
	v, err := s.Send(r.ID, "supplier")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.Review(r.ID, "buyer"); err != nil {
		t.Fatal(err)
	}
	accepted, err := s.Accept(r.ID, "buyer", v.Agreement.ID, v.Agreement.DocumentHash, "", "AAL2", true, true)
	if err != nil || accepted.Acceptance == nil || accepted.Request.State != BuyerAccepted {
		t.Fatalf("acceptance=%+v err=%v", accepted, err)
	}
	if _, err = s.Release(r.ID, "supplier", "supplier", "pickup", ""); err == nil {
		t.Fatal("goods released before authorization")
	}
	authorized, err := s.AuthorizeMandate(context.Background(), r.ID, "buyer", mandates.AuthorizationOptions{AmountCeiling: 100000000})
	if err != nil || authorized.Request.State != ReadyToRelease || authorized.Acceptance.ID != accepted.Acceptance.ID {
		t.Fatalf("authorization replaced acceptance: %+v err=%v", authorized, err)
	}
	if authorized.Mandate.AmountCeiling != 100000000 {
		t.Fatal("buyer ceiling was lost")
	}
}

func TestOfferFeeRatesAreCopiedAndBoundToAgreement(t *testing.T) {
	store := NewStore(mandates.NewMockProvider(), ledger.NewStore())
	terms := &ledger.FeeTerms{PolicyRevision: 7, BaseBPS: 25, CollectionBPS: 75}
	created, err := store.Create(CreateInput{FeeTerms: terms, SupplierOrganizationID: "supplier", SupplierLegalName: "Supplier", BuyerUserID: "buyer", BuyerBusinessID: "business", BuyerLegalName: "Buyer", PrincipalKobo: 100000, GoodsDescription: "goods", DueDate: "2026-09-30", CollectionAt: time.Now().Add(time.Hour), CreatedBy: "creator"})
	if err != nil {
		t.Fatal(err)
	}
	terms.BaseBPS = 99
	created.FeeTerms.BaseBPS = 88
	view, err := store.Send(created.ID, "creator")
	if err != nil {
		t.Fatal(err)
	}
	var canonical struct {
		FeeTerms *ledger.FeeTerms `json:"fee_terms"`
	}
	if err = json.Unmarshal(view.Agreement.CanonicalJSON, &canonical); err != nil {
		t.Fatal(err)
	}
	if canonical.FeeTerms == nil || canonical.FeeTerms.BaseBPS != 25 || canonical.FeeTerms.CollectionBPS != 75 {
		t.Fatalf("fee terms not bound to offer: %+v", canonical)
	}
}

// TestActivatedObligationIsReachableByItsOwnIdentifier locks in the map key used
// for activated obligations. The aggregate was previously indexed by credit
// request id while every financial reader (payments, collections, disputes and
// the durable projection) looks the obligation up by its own id, so an
// activated obligation disappeared from the view and no payment could be
// recorded against it.
func TestActivatedObligationIsReachableByItsOwnIdentifier(t *testing.T) {
	provider := mandates.NewMockProvider()
	store := NewStore(provider, ledger.NewStore())
	now := time.Now().UTC()
	created, err := store.Create(CreateInput{SupplierOrganizationID: "org-1", SupplierLegalName: "Supplier Ltd", BuyerUserID: "buyer-1", BuyerBusinessID: "biz-1", BuyerLegalName: "Buyer Ltd", PrincipalKobo: 100000, GoodsDescription: "Cement", DueDate: "2026-09-01", GraceHours: 24, CollectionAt: now.Add(24 * time.Hour), CreatedBy: "supplier-1"})
	if err != nil {
		t.Fatal(err)
	}
	view, err := store.Send(created.ID, "supplier-1")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = store.Review(created.ID, "buyer-1"); err != nil {
		t.Fatal(err)
	}
	mandated, err := store.AuthorizeMandate(context.Background(), created.ID, "buyer-1")
	if err != nil {
		t.Fatal(err)
	}
	accepted, err := store.Accept(created.ID, "buyer-1", view.Agreement.ID, view.Agreement.DocumentHash, mandated.Mandate.ProviderID, "AAL2", true, true)
	if err != nil {
		t.Fatal(err)
	}
	if accepted.Request.State != ReadyToRelease {
		t.Fatalf("state=%s", accepted.Request.State)
	}
	if _, err = store.Release(created.ID, "org-1", "supplier-1", "courier", "left the depot at noon"); err != nil {
		t.Fatal(err)
	}
	activated, _, err := store.RecordReceipt(created.ID, "buyer-1", "confirmed", "")
	if err != nil {
		t.Fatal(err)
	}
	if activated.Obligation == nil {
		t.Fatal("activation did not attach the obligation to the view")
	}
	obligationID := activated.Obligation.ID
	if obligationID == created.ID {
		t.Fatal("obligation id must be distinct from the credit request id for this test to mean anything")
	}

	snapshot, err := store.PaymentSnapshot(obligationID)
	if err != nil || snapshot.ID != obligationID || snapshot.OutstandingKobo != 100000 {
		t.Fatalf("payment snapshot unavailable by obligation id: snapshot=%+v err=%v", snapshot, err)
	}
	state, err := store.CollectionState(obligationID)
	if err != nil || state.ID != obligationID || state.OutstandingKobo != 100000 {
		t.Fatalf("collection state unavailable by obligation id: state=%+v err=%v", state, err)
	}
	if !store.ObligationBelongsToOrganization(obligationID, "org-1") {
		t.Fatal("obligation ownership lookup failed by obligation id")
	}
	if err := store.ApplyPayment(obligationID, -40000); err != nil {
		t.Fatalf("payment could not be applied by obligation id: %v", err)
	}

	// Re-reading the aggregate must still carry the obligation and the new balance.
	reread, err := store.GetForSupplier(created.ID, "org-1")
	if err != nil || reread.Obligation == nil {
		t.Fatalf("supplier view lost the obligation after activation: view=%+v err=%v", reread, err)
	}
	if reread.Obligation.OutstandingKobo != 60000 || reread.Obligation.PaymentStatus != "PARTIALLY_PAID" {
		t.Fatalf("outstanding balance not reflected in the view: %+v", reread.Obligation)
	}
}

// TestGoodsReleaseDoesNotTreatNotesAsAWaybillNumber guards the release evidence
// trail: free-text notes are not delivery documentation.
func TestGoodsReleaseDoesNotTreatNotesAsAWaybillNumber(t *testing.T) {
	provider := mandates.NewMockProvider()
	store := NewStore(provider, ledger.NewStore())
	now := time.Now().UTC()
	created, err := store.Create(CreateInput{SupplierOrganizationID: "org-1", SupplierLegalName: "Supplier Ltd", BuyerUserID: "buyer-1", BuyerBusinessID: "biz-1", BuyerLegalName: "Buyer Ltd", PrincipalKobo: 50000, GoodsDescription: "Cement", DueDate: "2026-09-01", CollectionAt: now.Add(24 * time.Hour), CreatedBy: "supplier-1"})
	if err != nil {
		t.Fatal(err)
	}
	view, err := store.Send(created.ID, "supplier-1")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = store.Review(created.ID, "buyer-1"); err != nil {
		t.Fatal(err)
	}
	mandated, err := store.AuthorizeMandate(context.Background(), created.ID, "buyer-1")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = store.Accept(created.ID, "buyer-1", view.Agreement.ID, view.Agreement.DocumentHash, mandated.Mandate.ProviderID, "AAL2", true, true); err != nil {
		t.Fatal(err)
	}
	released, err := store.Release(created.ID, "org-1", "supplier-1", "courier", "driver said the gate was closed")
	if err != nil {
		t.Fatal(err)
	}
	if released.Release == nil {
		t.Fatal("expected a goods release record")
	}
	if released.Release.WaybillNumber != "" {
		t.Fatalf("release notes were recorded as a waybill number: %q", released.Release.WaybillNumber)
	}
	if released.Release.Notes != "driver said the gate was closed" {
		t.Fatalf("release notes were not preserved: %q", released.Release.Notes)
	}
}
