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
