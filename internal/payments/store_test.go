package payments

import (
	"testing"
	"time"

	"kredit/internal/ledger"
	"kredit/internal/schedules"
)

func TestRecordAllocationReversalAndCollectionFee(t *testing.T) {
	book := ObligationSnapshot{ID: "obl-1", BuyerUserID: "buyer", SupplierOrganizationID: "org", PrincipalKobo: 100000, OutstandingKobo: 100000, CollectionAt: time.Now().UTC().Add(-time.Hour), Currency: "NGN"}
	apply := func(_ string, delta ledger.Money) error { book.OutstandingKobo += delta; return nil }
	store := NewStore(ledger.NewStore(), func(_ string) (ObligationSnapshot, error) { return book, nil }, apply)
	voluntary, allocation, err := store.Record(RecordInput{ObligationID: "obl-1", SourceType: SourceVoluntary, AmountKobo: 15000, RecordedBy: "supplier", IdempotencyKey: "vol-1", PaidAt: time.Now().UTC().Add(-2 * time.Hour)})
	if err != nil {
		t.Fatal(err)
	}
	if allocation.AmountKobo != 15000 || book.OutstandingKobo != 85000 {
		t.Fatalf("allocation=%+v outstanding=%d", allocation, book.OutstandingKobo)
	}
	duplicate, _, err := store.Record(RecordInput{ObligationID: "obl-1", SourceType: SourceVoluntary, AmountKobo: 15000, RecordedBy: "supplier", IdempotencyKey: "vol-1"})
	if err != nil || duplicate.ID != voluntary.ID || book.OutstandingKobo != 85000 {
		t.Fatal("duplicate payment was not idempotent")
	}
	collected, _, err := store.Record(RecordInput{ObligationID: "obl-1", SourceType: SourceCollected, AmountKobo: 50000, RecordedBy: CollectionRecorder, IdempotencyKey: CollectionKeyPrefix + "attempt-1", Provider: "mock", ProviderReference: "provider-1", PaidAt: time.Now().UTC()})
	if err != nil {
		t.Fatal(err)
	}
	if collected.CollectionFeeKobo != 250 {
		t.Fatalf("collection fee=%d", collected.CollectionFeeKobo)
	}
	if book.OutstandingKobo != 35000 {
		t.Fatalf("outstanding=%d", book.OutstandingKobo)
	}
	if _, err := store.Reverse(voluntary.ID, "finance", "bank return"); err != nil {
		t.Fatal(err)
	}
	if book.OutstandingKobo != 50000 {
		t.Fatalf("reversed outstanding=%d", book.OutstandingKobo)
	}
	if _, err := store.Reverse(voluntary.ID, "finance", "duplicate"); err != nil {
		t.Fatal(err)
	}
	if expected, err := store.Rebuild("obl-1"); err != nil || expected != 50000 {
		t.Fatalf("rebuild expected=%d err=%v", expected, err)
	}
}

func TestCollectedPaymentRequiresWorkerAttemptProvenance(t *testing.T) {
	store := NewStore(ledger.NewStore(), func(string) (ObligationSnapshot, error) {
		return ObligationSnapshot{ID: "obl-1", BuyerUserID: "buyer", SupplierOrganizationID: "org", PrincipalKobo: 1000, OutstandingKobo: 1000, Currency: "NGN"}, nil
	}, func(string, ledger.Money) error { return nil })
	_, _, err := store.Record(RecordInput{ObligationID: "obl-1", SourceType: SourceCollected, AmountKobo: 100, RecordedBy: "supplier-user", IdempotencyKey: CollectionKeyPrefix + "attempt", Provider: "provider", ProviderReference: "ref"})
	if err == nil {
		t.Fatal("supplier-originated collected payment was accepted")
	}
}

func TestScheduleAwarePaymentAllocatesAndReversesItem(t *testing.T) {
	scheduleStore := schedules.NewStore()
	start := time.Now().UTC().Truncate(time.Second)
	_, items, err := scheduleStore.Create(schedules.CreateInput{ObligationID: "obl-s", PrincipalKobo: 1000, ScheduleType: schedules.TypeEqual, Count: 2, StartDate: start, Timezone: "UTC", Cadence: schedules.CadenceMonthly, MonthEndPolicy: schedules.PolicyCap})
	if err != nil {
		t.Fatal(err)
	}
	book := ObligationSnapshot{ID: "obl-s", PrincipalKobo: 1000, OutstandingKobo: 1000, Currency: "NGN"}
	apply := func(_ string, delta ledger.Money) error { book.OutstandingKobo += delta; return nil }
	allocator := func(id string, amount ledger.Money) ([]AllocationTarget, error) {
		targets, e := scheduleStore.Allocate(id, amount)
		out := make([]AllocationTarget, 0, len(targets))
		for _, target := range targets {
			out = append(out, AllocationTarget{ScheduleItemID: target.ScheduleItemID, AmountKobo: target.AmountKobo})
		}
		return out, e
	}
	reallocator := func(targets []AllocationTarget) error {
		converted := make([]schedules.AllocationTarget, 0, len(targets))
		for _, target := range targets {
			converted = append(converted, schedules.AllocationTarget{ScheduleItemID: target.ScheduleItemID, AmountKobo: target.AmountKobo})
		}
		return scheduleStore.ReverseAllocations(converted)
	}
	store := NewStoreWithAllocator(ledger.NewStore(), func(_ string) (ObligationSnapshot, error) { return book, nil }, apply, allocator, reallocator)
	payment, allocation, err := store.Record(RecordInput{ObligationID: "obl-s", SourceType: SourceVoluntary, AmountKobo: 500, RecordedBy: "supplier", IdempotencyKey: "schedule-payment"})
	if err != nil {
		t.Fatal(err)
	}
	if allocation.ScheduleItemID != items[0].ID {
		t.Fatal("wrong schedule item")
	}
	if scheduleStore.ListForObligation("obl-s")[0].State != schedules.ItemPaid {
		t.Fatal("item not paid")
	}
	if _, err := store.Reverse(payment.ID, "finance", "returned"); err != nil {
		t.Fatal(err)
	}
	if scheduleStore.ListForObligation("obl-s")[0].State != schedules.ItemOpen {
		t.Fatal("item not reopened")
	}
}

func TestPaymentCannotExceedOutstanding(t *testing.T) {
	book := ObligationSnapshot{ID: "obl", PrincipalKobo: 100, OutstandingKobo: 100, Currency: "NGN"}
	store := NewStore(ledger.NewStore(), func(_ string) (ObligationSnapshot, error) { return book, nil }, func(_ string, delta ledger.Money) error { book.OutstandingKobo += delta; return nil })
	if _, _, err := store.Record(RecordInput{ObligationID: "obl", SourceType: SourceVoluntary, AmountKobo: 101, RecordedBy: "user", IdempotencyKey: "too-much"}); err == nil {
		t.Fatal("expected overpayment rejection")
	}
}

func TestPaymentRejectsChangedIntentForIdempotencyKey(t *testing.T) {
	book := ObligationSnapshot{ID: "obl", PrincipalKobo: 100, OutstandingKobo: 100, Currency: "NGN"}
	store := NewStore(ledger.NewStore(), func(_ string) (ObligationSnapshot, error) { return book, nil }, func(_ string, delta ledger.Money) error { book.OutstandingKobo += delta; return nil })
	if _, _, err := store.Record(RecordInput{ObligationID: "obl", SourceType: SourceVoluntary, AmountKobo: 40, RecordedBy: "user", IdempotencyKey: "same-key"}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := store.Record(RecordInput{ObligationID: "obl", SourceType: SourceVoluntary, AmountKobo: 41, RecordedBy: "user", IdempotencyKey: "same-key"}); err == nil {
		t.Fatal("expected changed payment amount to be rejected")
	}
}
