package credit

import (
	"testing"
	"time"
)

func TestAgreementScheduleValidationRejectsEarlyCollectionAndShiftedMonthEnd(t *testing.T) {
	r := CreditRequest{PrincipalKobo: 10000, DueDate: "2026-09-30", CollectionAt: time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC)}
	if err := ValidateRepaymentSchedule(r); err != nil {
		t.Fatal(err)
	}
	r.CollectionAt = r.CollectionAt.AddDate(0, 0, -1)
	if err := ValidateRepaymentSchedule(r); err == nil {
		t.Fatal("collection before payment day accepted")
	}
	r.CollectionAt = r.CollectionAt.AddDate(0, 0, 1)
	r.ScheduleType, r.ScheduleCount, r.ScheduleCadence, r.MonthEndPolicy = "equal", 3, "monthly", "last_day"
	if err := ValidateRepaymentSchedule(r); err != nil {
		t.Fatal(err)
	}
	r.DueDate = "2026-09-29"
	if err := ValidateRepaymentSchedule(r); err == nil {
		t.Fatal("agreement date silently shifted to month end")
	}
}
