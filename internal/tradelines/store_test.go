package tradelines

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestConcurrentReservationsCannotExceedLimit(t *testing.T) {
	store := NewStore()
	now := time.Now().UTC()
	line, err := store.CreateLine(CreateLineInput{SupplierOrganizationID: "org", BuyerUserID: "buyer", BuyerBusinessID: "biz", ApprovedLimitKobo: 1000, Cadence: "friday", StartAt: now.Add(-time.Hour), EndAt: now.Add(24 * time.Hour), MandateID: "mandate", MandateActive: true, MandateVerified: true})
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	var mu sync.Mutex
	success := 0
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, _, _, err := store.ReserveDrawdown(CreateDrawdownInput{LineID: line.ID, PrincipalKobo: 200, GoodsDescription: "goods", IdempotencyKey: fmt.Sprintf("key-%d", i)})
			if err == nil {
				mu.Lock()
				success++
				mu.Unlock()
			}
		}(i)
	}
	wg.Wait()
	if success != 5 {
		t.Fatalf("successful reservations=%d", success)
	}
	got, _ := store.Get(line.ID)
	if got.ReservedPendingKobo != 1000 || got.AvailableLimitKobo != 0 {
		t.Fatalf("line=%+v", got)
	}
}

func TestDrawdownConfirmationActivationAndSuspension(t *testing.T) {
	store := NewStore()
	store.SetActivationHandler(func(input ActivationInput) (string, error) {
		if input.Drawdown.ReceiptState != "no_issue" || input.Drawdown.ReleasedAt.IsZero() {
			return "", fmt.Errorf("missing receipt or release evidence")
		}
		return "obligation-1", nil
	})
	now := time.Now().UTC()
	line, _ := store.CreateLine(CreateLineInput{SupplierOrganizationID: "org", BuyerUserID: "buyer", BuyerBusinessID: "biz", ApprovedLimitKobo: 5000, Cadence: "friday", StartAt: now.Add(-time.Hour), EndAt: now.Add(time.Hour), MandateID: "mandate", MandateActive: true, MandateVerified: true})
	drawdown, reserved, updated, err := store.ReserveDrawdown(CreateDrawdownInput{LineID: line.ID, PrincipalKobo: 1200, GoodsDescription: "stock", IdempotencyKey: "one"})
	if err != nil {
		t.Fatal(err)
	}
	if reserved.State != ReservationPending || updated.AvailableLimitKobo != 3800 {
		t.Fatal("reservation not reflected")
	}
	if _, _, err = store.ConfirmDrawdown(drawdown.ID, "buyer", drawdown.AgreementHash); err != nil {
		t.Fatal(err)
	}
	if _, _, err = store.ReleaseDrawdown(ReleaseInput{DrawdownID: drawdown.ID, SupplierOrganizationID: "org", ActorID: "supplier", DeliveryMethod: "delivery"}); err == nil {
		t.Fatal("release without evidence reference must fail")
	}
	if _, _, err = store.ReleaseDrawdown(ReleaseInput{DrawdownID: drawdown.ID, SupplierOrganizationID: "org", ActorID: "supplier", DeliveryMethod: "delivery", EvidenceReference: "proof-1"}); err != nil {
		t.Fatal(err)
	}
	active, updated, err := store.RecordDrawdownReceipt(ReceiptInput{DrawdownID: drawdown.ID, BuyerUserID: "buyer", State: "no_issue"})
	if err != nil {
		t.Fatal(err)
	}
	if active.State != DrawdownActivated || updated.CurrentExposureKobo != 1200 {
		t.Fatal("activation not reflected")
	}
	if _, err = store.UpdateOutstanding(drawdown.ID, 700); err != nil {
		t.Fatal(err)
	}
	if _, err = store.UpdateOutstanding(drawdown.ID, 700); err != nil {
		t.Fatal(err)
	}
	updated, _ = store.Get(line.ID)
	if updated.CurrentExposureKobo != 700 || updated.AvailableLimitKobo != 4300 {
		t.Fatal("exposure update incorrect")
	}
	if _, err = store.Suspend(line.ID, "buyer overdue"); err != nil {
		t.Fatal(err)
	}
	if _, _, _, err = store.ReserveDrawdown(CreateDrawdownInput{LineID: line.ID, PrincipalKobo: 100, GoodsDescription: "blocked", IdempotencyKey: "blocked"}); err == nil {
		t.Fatal("suspended line accepted drawdown")
	}
	if _, err = store.Resume(line.ID); err != nil {
		t.Fatal(err)
	}
}

func TestDrawdownRejectsChangedTermsAndMissingActivationHandler(t *testing.T) {
	store := NewStore()
	now := time.Now().UTC()
	line, err := store.CreateLine(CreateLineInput{SupplierOrganizationID: "org", BuyerUserID: "buyer", BuyerBusinessID: "biz", ApprovedLimitKobo: 1000, StartAt: now.Add(-time.Hour), EndAt: now.Add(time.Hour), MandateID: "mandate", MandateActive: true, MandateVerified: true})
	if err != nil {
		t.Fatal(err)
	}
	drawdown, _, _, err := store.ReserveDrawdown(CreateDrawdownInput{LineID: line.ID, PrincipalKobo: 100, GoodsDescription: "goods", IdempotencyKey: "validator"})
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := store.ConfirmDrawdown(drawdown.ID, "buyer", "changed-hash"); err == nil {
		t.Fatal("expected changed terms to be rejected")
	}
	if _, _, err := store.ConfirmDrawdown(drawdown.ID, "buyer", drawdown.AgreementHash); err != nil {
		t.Fatal(err)
	}
	if _, _, err := store.ReleaseDrawdown(ReleaseInput{DrawdownID: drawdown.ID, SupplierOrganizationID: "org", ActorID: "supplier", DeliveryMethod: "pickup", EvidenceReference: "pickup-note"}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := store.RecordDrawdownReceipt(ReceiptInput{DrawdownID: drawdown.ID, BuyerUserID: "buyer", State: "no_issue"}); err == nil {
		t.Fatal("expected unavailable internal activation to be rejected")
	}
}

func TestReceiptIssueDoesNotActivateAndCancellationReleasesLimit(t *testing.T) {
	store := NewStore()
	now := time.Now().UTC()
	line, _ := store.CreateLine(CreateLineInput{SupplierOrganizationID: "org", BuyerUserID: "buyer", BuyerBusinessID: "biz", ApprovedLimitKobo: 1000, StartAt: now.Add(-time.Hour), EndAt: now.Add(time.Hour), MandateID: "mandate", MandateActive: true, MandateVerified: true})
	issue, _, _, _ := store.ReserveDrawdown(CreateDrawdownInput{LineID: line.ID, PrincipalKobo: 300, GoodsDescription: "goods", IdempotencyKey: "issue"})
	_, _, _ = store.ConfirmDrawdown(issue.ID, "buyer", issue.AgreementHash)
	_, _, _ = store.ReleaseDrawdown(ReleaseInput{DrawdownID: issue.ID, SupplierOrganizationID: "org", ActorID: "supplier", DeliveryMethod: "delivery", EvidenceReference: "dispatch-note"})
	got, updated, err := store.RecordDrawdownReceipt(ReceiptInput{DrawdownID: issue.ID, BuyerUserID: "buyer", State: "issue_reported", IssueReason: "wrong quantity"})
	if err != nil || got.State != DrawdownReceiptIssue || got.ReceiptDisputeID == "" || got.ObligationID != "" || updated.CurrentExposureKobo != 0 {
		t.Fatalf("issue receipt activated money: drawdown=%+v line=%+v err=%v", got, updated, err)
	}
	if replay, _, replayErr := store.RecordDrawdownReceipt(ReceiptInput{DrawdownID: issue.ID, BuyerUserID: "buyer", State: "issue_reported", IssueReason: "wrong quantity"}); replayErr != nil || replay.ReceiptDisputeID != got.ReceiptDisputeID {
		t.Fatalf("issue replay was not safe: drawdown=%+v err=%v", replay, replayErr)
	}
	cancelled, _, _, _ := store.ReserveDrawdown(CreateDrawdownInput{LineID: line.ID, PrincipalKobo: 200, GoodsDescription: "other", IdempotencyKey: "cancel"})
	for _, wrongLine := range []string{"", "another-line"} {
		if _, _, err = store.CancelDrawdown(wrongLine, cancelled.ID, "supplier"); err == nil {
			t.Fatal("cross-line cancellation accepted")
		}
	}
	if _, _, err = store.CancelDrawdown(cancelled.TradeLineID, cancelled.ID, ""); err == nil {
		t.Fatal("anonymous cancellation accepted")
	}
	if _, updated, err = store.CancelDrawdown(cancelled.TradeLineID, cancelled.ID, "supplier"); err != nil || updated.AvailableLimitKobo != 700 {
		t.Fatalf("cancel did not release only its reservation: line=%+v err=%v", updated, err)
	}
}

func TestDrawdownLifecycleCommandsAreReplaySafe(t *testing.T) {
	store := NewStore()
	activations := 0
	store.SetActivationHandler(func(ActivationInput) (string, error) { activations++; return "obligation-replay", nil })
	now := time.Now().UTC()
	line, _ := store.CreateLine(CreateLineInput{SupplierOrganizationID: "org", BuyerUserID: "buyer", BuyerBusinessID: "biz", ApprovedLimitKobo: 1000, StartAt: now.Add(-time.Hour), EndAt: now.Add(time.Hour), MandateID: "mandate", MandateActive: true, MandateVerified: true})
	drawdown, _, _, _ := store.ReserveDrawdown(CreateDrawdownInput{LineID: line.ID, PrincipalKobo: 400, GoodsDescription: "goods", IdempotencyKey: "replay"})
	for range 2 {
		if _, _, err := store.ConfirmDrawdown(drawdown.ID, "buyer", drawdown.AgreementHash); err != nil {
			t.Fatal(err)
		}
	}
	release := ReleaseInput{DrawdownID: drawdown.ID, SupplierOrganizationID: "org", ActorID: "supplier", DeliveryMethod: "courier", EvidenceReference: "tracking-1"}
	for range 2 {
		if _, _, err := store.ReleaseDrawdown(release); err != nil {
			t.Fatal(err)
		}
	}
	for range 2 {
		if _, _, err := store.RecordDrawdownReceipt(ReceiptInput{DrawdownID: drawdown.ID, BuyerUserID: "buyer", State: "no_issue"}); err != nil {
			t.Fatal(err)
		}
	}
	updated, _ := store.Get(line.ID)
	if activations != 1 || updated.CurrentExposureKobo != 400 || updated.ReservedPendingKobo != 0 {
		t.Fatalf("replay duplicated activation: activations=%d line=%+v", activations, updated)
	}
}

func TestSupplierMayOnlyReduceUnusedLimitWithCurrentVersion(t *testing.T) {
	store := NewStore()
	now := time.Now().UTC()
	line, err := store.CreateLine(CreateLineInput{SupplierOrganizationID: "org", BuyerUserID: "buyer", BuyerBusinessID: "biz", ApprovedLimitKobo: 5000, StartAt: now.Add(-time.Hour), EndAt: now.Add(time.Hour), MandateID: "mandate", MandateActive: true, MandateVerified: true})
	if err != nil {
		t.Fatal(err)
	}
	updated, err := store.ReduceLimit(line.ID, 4000, line.Version)
	if err != nil || updated.ApprovedLimitKobo != 4000 || updated.AvailableLimitKobo != 4000 {
		t.Fatalf("updated=%+v err=%v", updated, err)
	}
	if _, err = store.ReduceLimit(line.ID, 4500, updated.Version); err == nil {
		t.Fatal("supplier limit increase must require fresh buyer acceptance")
	}
	if _, err = store.ReduceLimit(line.ID, 3000, line.Version); err == nil {
		t.Fatal("stale version must fail")
	}
}

func TestTradeLineRejectsClientAssertedMandateState(t *testing.T) {
	store := NewStore()
	_, err := store.CreateLine(CreateLineInput{SupplierOrganizationID: "org", BuyerUserID: "buyer", BuyerBusinessID: "biz", ApprovedLimitKobo: 1000, StartAt: time.Now().UTC(), EndAt: time.Now().UTC().Add(time.Hour), MandateID: "unverified", MandateActive: true})
	if err == nil {
		t.Fatal("expected unverified active mandate to be rejected")
	}
}

func TestFeeDisclosurePreservesAcceptedLegacyHash(t *testing.T) {
	d := Drawdown{ID: "drawdown", PrincipalKobo: 10000}
	l := TradeLine{ID: "line"}
	d.AgreementHash = drawdownHashWithFee(d, l, legacyFeeDisclosure)
	if !VerifyAgreementHash(d, l) || DrawdownFeeDisclosure(d, l) != legacyFeeDisclosure {
		t.Fatal("legacy evidence was rewritten")
	}
	d.AgreementHash = drawdownHash(d, l)
	if !VerifyAgreementHash(d, l) || DrawdownFeeDisclosure(d, l) != currentFeeDisclosure {
		t.Fatal("new agreement uses incorrect fee")
	}
	d.PrincipalKobo++
	if VerifyAgreementHash(d, l) {
		t.Fatal("changed terms verified")
	}
}
