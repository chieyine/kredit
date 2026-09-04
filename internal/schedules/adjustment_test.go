package schedules

import (
	"errors"
	"testing"
	"time"
)

func TestAcceptedCollectionInstantPreservedAcrossInstalments(t *testing.T) {
	first := time.Date(2026, 9, 7, 9, 37, 0, 0, time.UTC)
	for _, kind := range []string{TypeEqual, TypeCustom} {
		s := NewStore()
		_, items, err := s.Create(CreateInput{ObligationID: "o", PrincipalKobo: 200, ScheduleType: kind, Count: 2, StartDate: time.Date(2026, 9, 3, 0, 0, 0, 0, time.UTC), FirstCollectionAt: first, DueHour: 10, DueMinute: 37, Timezone: "Africa/Lagos", GraceHours: 24, Cadence: CadenceWeekly, CustomItems: []CustomItem{{AmountKobo: 100, DueDate: time.Date(2026, 9, 3, 0, 0, 0, 0, time.UTC)}, {AmountKobo: 100, DueDate: time.Date(2026, 9, 10, 0, 0, 0, 0, time.UTC)}}})
		if err != nil {
			t.Fatal(err)
		}
		if !items[0].CollectionAt.Equal(first) || !items[1].CollectionAt.Equal(first.AddDate(0, 0, 7)) {
			t.Fatalf("%s changed accepted debit dates: %+v", kind, items)
		}
		amount, err := s.CollectionTarget("o", first.Add(-time.Second))
		if err != nil || amount != 0 {
			t.Fatalf("early debit became eligible: %d %v", amount, err)
		}
	}
}

func TestPrincipalAdjustmentIsAtomicAndReducesLastUnpaidItems(t *testing.T) {
	s := NewStore()
	_, _, err := s.Create(CreateInput{ObligationID: "o", PrincipalKobo: 300, ScheduleType: TypeEqual, Count: 3, StartDate: time.Now(), Timezone: "UTC", Cadence: CadenceMonthly})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.Allocate("o", 40); err != nil {
		t.Fatal(err)
	}
	failed := errors.New("balance write failed")
	if err = s.ReducePrincipal("o", 260, 150, true, func() error { return failed }); !errors.Is(err, failed) {
		t.Fatal(err)
	}
	_, items, _ := s.GetForObligation("o")
	if items[2].State == ItemCancelled || items[1].PrincipalDueKobo != 100 {
		t.Fatal("failed adjustment changed schedule")
	}
	if err = s.ReducePrincipal("o", 260, 150, true, func() error { return nil }); err != nil {
		t.Fatal(err)
	}
	_, items, _ = s.GetForObligation("o")
	if items[0].PrincipalDueKobo != 100 || items[0].AllocatedKobo != 40 || items[1].PrincipalDueKobo != 50 || items[2].State != ItemCancelled {
		t.Fatalf("adjustment damaged allocations: %+v", items)
	}
	if _, err = s.Allocate("o", 111); err == nil {
		t.Fatal("forgiven amount remains collectible")
	}
	if _, err = s.Allocate("o", 110); err != nil {
		t.Fatal(err)
	}
}
