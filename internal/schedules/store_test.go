package schedules

import (
	"math"
	"testing"
	"time"

	"kredit/internal/ledger"
)

func TestEqualMonthlyScheduleAllocatesEarlyPaymentInDueOrder(t *testing.T) {
	start := time.Date(2026, 1, 31, 0, 0, 0, 0, time.UTC)
	store := NewStore()
	schedule, items, err := store.Create(CreateInput{ObligationID: "obl-1", PrincipalKobo: 3000000, ScheduleType: TypeEqual, Count: 6, StartDate: start, DueHour: 9, Timezone: "Africa/Lagos", GraceHours: 48, Cadence: CadenceMonthly, MonthEndPolicy: PolicyCap})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 6 || items[0].PrincipalDueKobo != 500000 || items[5].PrincipalDueKobo != 500000 {
		t.Fatalf("schedule=%+v items=%+v", schedule, items)
	}
	if items[1].DueAt.Day() != 28 {
		t.Fatalf("expected February month-end cap, got %s", items[1].DueAt)
	}
	targets, err := store.Allocate("obl-1", 500000)
	if err != nil {
		t.Fatal(err)
	}
	if len(targets) != 1 || targets[0].ScheduleItemID != items[0].ID {
		t.Fatal("payment did not allocate to earliest item")
	}
	updated := store.ListForObligation("obl-1")
	if updated[0].State != ItemPaid {
		t.Fatalf("state=%s", updated[0].State)
	}
	if err := store.ReverseAllocations(targets); err != nil {
		t.Fatal(err)
	}
	if store.ListForObligation("obl-1")[0].State != ItemOpen {
		t.Fatal("reversal did not reopen item")
	}
}

func TestCustomScheduleValidatesSumAndGraceState(t *testing.T) {
	store := NewStore()
	start := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	if _, _, err := store.Create(CreateInput{ObligationID: "bad", PrincipalKobo: 100, ScheduleType: TypeCustom, StartDate: start, CustomItems: []CustomItem{{AmountKobo: 99, DueDate: start}}, Timezone: "UTC", Cadence: CadenceCustom}); err == nil {
		t.Fatal("expected sum validation")
	}
	_, items, err := store.Create(CreateInput{ObligationID: "obl-2", PrincipalKobo: 100, ScheduleType: TypeCustom, StartDate: start, CustomItems: []CustomItem{{AmountKobo: 100, DueDate: start}}, Timezone: "UTC", Cadence: CadenceCustom, GraceHours: 24})
	if err != nil {
		t.Fatal(err)
	}
	grace := items[0].DueAt.Add(time.Hour)
	got := store.Evaluate(grace)
	if len(got) != 1 || got[0].State != ItemInGrace {
		t.Fatalf("state=%v", got)
	}
	overdue := store.Evaluate(items[0].CollectionAt.Add(time.Minute))
	if len(overdue) != 1 || overdue[0].State != ItemOverdue {
		t.Fatalf("state=%v", overdue)
	}
	_ = ledger.Money(0)
}

func TestCustomScheduleRejectsOverflowingTotal(t *testing.T) {
	start := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	_, _, err := NewStore().Create(CreateInput{ObligationID: "overflow", PrincipalKobo: ledger.Money(math.MaxInt64), ScheduleType: TypeCustom, StartDate: start, CustomItems: []CustomItem{{AmountKobo: ledger.Money(math.MaxInt64), DueDate: start}, {AmountKobo: 1, DueDate: start.AddDate(0, 1, 0)}}, Timezone: "UTC", Cadence: CadenceCustom})
	if err == nil {
		t.Fatal("expected overflowing schedule total to be rejected")
	}
}

func TestRejectedScheduleMutationsLeaveAllAmountsUnchanged(t *testing.T) {
	store := NewStore()
	_, items, err := store.Create(CreateInput{ObligationID: "atomic", PrincipalKobo: 1000, ScheduleType: TypeEqual, Count: 2, StartDate: time.Now(), Timezone: "Africa/Lagos", Cadence: CadenceMonthly, MonthEndPolicy: PolicyCap})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = store.Allocate("atomic", 1000); err != nil {
		t.Fatal(err)
	}
	for _, targets := range [][]AllocationTarget{{{ScheduleItemID: items[0].ID, AmountKobo: 100}, {ScheduleItemID: items[1].ID, AmountKobo: 600}}, {{ScheduleItemID: items[0].ID, AmountKobo: -1}}} {
		if err = store.ReverseAllocations(targets); err == nil {
			t.Fatal("invalid reversal accepted")
		}
		got := store.ListForObligation("atomic")
		if got[0].AllocatedKobo != 500 || got[1].AllocatedKobo != 500 {
			t.Fatal("failed reversal mutated schedule")
		}
	}
	if err = store.MarkCollected("atomic", 1001); err == nil {
		t.Fatal("excess collection accepted")
	}
	for _, item := range store.ListForObligation("atomic") {
		if item.CollectedKobo != 0 {
			t.Fatal("failed collection mutated schedule")
		}
	}
}

// TestMonthEndPoliciesAreDistinct pins the two documented monthly policies
// (README section 26.2). "last_day" previously behaved exactly like "cap": it
// only moved a date when the start day did not exist in the target month, so a
// schedule agreed as "last day of month" from the 30th silently ran on the 30th.
func TestMonthEndPoliciesAreDistinct(t *testing.T) {
	start := time.Date(2026, 1, 30, 0, 0, 0, 0, time.UTC)

	lastDay := NewStore()
	_, lastDayItems, err := lastDay.Create(CreateInput{ObligationID: "obl-last-day", PrincipalKobo: 300000, ScheduleType: TypeEqual, Count: 3, StartDate: start, DueHour: 9, Timezone: "Africa/Lagos", Cadence: CadenceMonthly, MonthEndPolicy: PolicyLastDay})
	if err != nil {
		t.Fatal(err)
	}
	for index, expected := range []int{31, 28, 31} {
		if lastDayItems[index].DueAt.Day() != expected {
			t.Fatalf("last_day instalment %d fell on day %d, expected the month end (%d)", index+1, lastDayItems[index].DueAt.Day(), expected)
		}
	}

	capped := NewStore()
	_, cappedItems, err := capped.Create(CreateInput{ObligationID: "obl-cap", PrincipalKobo: 300000, ScheduleType: TypeEqual, Count: 3, StartDate: start, DueHour: 9, Timezone: "Africa/Lagos", Cadence: CadenceMonthly, MonthEndPolicy: PolicyCap})
	if err != nil {
		t.Fatal(err)
	}
	for index, expected := range []int{30, 28, 30} {
		if cappedItems[index].DueAt.Day() != expected {
			t.Fatalf("cap instalment %d fell on day %d, expected %d", index+1, cappedItems[index].DueAt.Day(), expected)
		}
	}

	// Both policies must still produce strictly increasing, ordered dates.
	for _, items := range [][]Item{lastDayItems, cappedItems} {
		for index := 1; index < len(items); index++ {
			if !items[index].DueAt.After(items[index-1].DueAt) {
				t.Fatalf("schedule dates are not strictly ordered: %s then %s", items[index-1].DueAt, items[index].DueAt)
			}
		}
	}
}
