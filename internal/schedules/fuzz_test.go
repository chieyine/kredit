package schedules

import (
	"testing"
	"time"

	"kredit/internal/ledger"
)

func fuzzPick(index int, values []string) string {
	if len(values) == 0 {
		return ""
	}
	position := index % len(values)
	if position < 0 {
		position += len(values)
	}
	return values[position]
}

// FuzzScheduleGeneration covers the "schedule generation" target README
// section 40.3 requires. Whatever the inputs, a schedule that is accepted must
// still sum to the principal and run in strict date order: an instalment plan
// whose parts do not add up to the agreed amount is a wrong debit waiting to
// happen.
func FuzzScheduleGeneration(f *testing.F) {
	f.Add(int64(300000), 3, 30, 2, 0, 24)
	f.Add(int64(1), 1, 31, 0, 1, 0)
	f.Add(int64(999999999999), 60, 29, 2, 1, 720)
	f.Fuzz(func(t *testing.T, principal int64, count, day, cadenceIndex, policyIndex, graceHours int) {
		if principal <= 0 || principal > 1<<50 || count < 1 || count > 60 || day < 1 || day > 31 {
			return
		}
		if graceHours < 0 || graceHours > 720 {
			return
		}
		store := NewStore()
		_, items, err := store.Create(CreateInput{
			ObligationID:   "fuzz-obligation",
			PrincipalKobo:  ledger.Money(principal),
			ScheduleType:   TypeEqual,
			Count:          count,
			StartDate:      time.Date(2026, 1, day, 0, 0, 0, 0, time.UTC),
			DueHour:        9,
			Timezone:       "Africa/Lagos",
			GraceHours:     graceHours,
			Cadence:        fuzzPick(cadenceIndex, []string{CadenceWeekly, CadenceFortnightly, CadenceMonthly, CadenceCustom}),
			MonthEndPolicy: fuzzPick(policyIndex, []string{PolicyCap, PolicyLastDay}),
		})
		if err != nil {
			return
		}
		if len(items) != count {
			t.Fatalf("asked for %d instalments, got %d", count, len(items))
		}
		var total ledger.Money
		for index, item := range items {
			if item.PrincipalDueKobo <= 0 {
				t.Fatalf("instalment %d is not positive: %d", index+1, item.PrincipalDueKobo)
			}
			total, err = ledger.CheckedAdd(total, item.PrincipalDueKobo)
			if err != nil {
				t.Fatalf("instalment total overflowed: %v", err)
			}
			if index > 0 && !item.DueAt.After(items[index-1].DueAt) {
				t.Fatalf("instalment %d (%s) does not fall after instalment %d (%s)", index+1, item.DueAt, index, items[index-1].DueAt)
			}
			if item.CollectionAt.Before(item.DueAt) {
				t.Fatalf("instalment %d may be collected before it is due", index+1)
			}
		}
		if total != ledger.Money(principal) {
			t.Fatalf("instalments sum to %d, principal is %d", total, principal)
		}
	})
}

// FuzzPaymentAllocation covers the "allocation" target README section 40.3
// requires. An accepted allocation must place exactly the payment amount and
// must never allocate an instalment beyond what it is owed.
func FuzzPaymentAllocation(f *testing.F) {
	f.Add(int64(300000), int64(100000), 3)
	f.Add(int64(500), int64(500), 1)
	f.Add(int64(1000000), int64(999999), 7)
	f.Fuzz(func(t *testing.T, principal, payment int64, count int) {
		if principal <= 0 || principal > 1<<40 || count < 1 || count > 24 || principal < int64(count) {
			return
		}
		store := NewStore()
		_, _, err := store.Create(CreateInput{
			ObligationID:   "fuzz-obligation",
			PrincipalKobo:  ledger.Money(principal),
			ScheduleType:   TypeEqual,
			Count:          count,
			StartDate:      time.Date(2026, 1, 5, 0, 0, 0, 0, time.UTC),
			DueHour:        9,
			Timezone:       "Africa/Lagos",
			Cadence:        CadenceMonthly,
			MonthEndPolicy: PolicyCap,
		})
		if err != nil {
			return
		}
		targets, err := store.Allocate("fuzz-obligation", ledger.Money(payment))
		if err != nil {
			return
		}
		var placed ledger.Money
		for _, target := range targets {
			if target.AmountKobo <= 0 {
				t.Fatalf("allocation placed a non-positive amount: %d", target.AmountKobo)
			}
			placed, err = ledger.CheckedAdd(placed, target.AmountKobo)
			if err != nil {
				t.Fatalf("allocation total overflowed: %v", err)
			}
		}
		if placed != ledger.Money(payment) {
			t.Fatalf("allocated %d of a %d payment", placed, payment)
		}
		for _, item := range store.ListForObligation("fuzz-obligation") {
			if item.AllocatedKobo > item.PrincipalDueKobo {
				t.Fatalf("instalment %s allocated %d against %d due", item.ID, item.AllocatedKobo, item.PrincipalDueKobo)
			}
		}
	})
}
