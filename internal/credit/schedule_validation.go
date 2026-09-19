package credit

import (
	"errors"
	"time"

	"kredit/internal/schedules"
)

// repaymentScheduleInput is shared by agreement validation and activation.
func repaymentScheduleInput(r CreditRequest, obligationID string) (schedules.CreateInput, error) {
	loc, err := time.LoadLocation("Africa/Lagos")
	if err != nil {
		return schedules.CreateInput{}, err
	}
	start, err := time.ParseInLocation("2006-01-02", r.DueDate, loc)
	if err != nil {
		return schedules.CreateInput{}, err
	}
	if r.ScheduleType == "equal" && r.ScheduleCadence == "monthly" && r.MonthEndPolicy == "last_day" && start.AddDate(0, 0, 1).Month() == start.Month() {
		return schedules.CreateInput{}, errors.New("last-day monthly payments require a first payment date at month end")
	}
	input := schedules.CreateInput{FirstCollectionAt: r.CollectionAt, ObligationID: obligationID, PrincipalKobo: r.PrincipalKobo, ScheduleType: r.ScheduleType, Count: r.ScheduleCount, StartDate: start, DueHour: r.CollectionAt.In(loc).Hour(), DueMinute: r.CollectionAt.In(loc).Minute(), Timezone: "Africa/Lagos", GraceHours: r.GraceHours, Cadence: r.ScheduleCadence, MonthEndPolicy: r.MonthEndPolicy, AllocationPolicy: "due_date_order"}
	if r.ScheduleType == "" || r.ScheduleType == "one_time" {
		input.ScheduleType = schedules.TypeEqual
		input.Count = 1
		input.Cadence = schedules.CadenceCustom
	}
	for i, term := range r.CustomScheduleItems {
		due, err := time.ParseInLocation("2006-01-02", term.DueDate, loc)
		if err != nil {
			return schedules.CreateInput{}, err
		}
		if i == 0 && r.ScheduleType == "custom" && !due.Equal(start) {
			return schedules.CreateInput{}, errors.New("first instalment must match the agreed first payment date")
		}
		input.CustomItems = append(input.CustomItems, schedules.CustomItem{AmountKobo: term.AmountKobo, DueDate: due})
	}
	return input, nil
}

// ValidateRepaymentSchedule generates the same schedule activation will persist,
// in an isolated memory store, before any agreement can be sent.
func ValidateRepaymentSchedule(r CreditRequest) error {
	if err := validateScheduleTerms(r.PrincipalKobo, r.ScheduleType, r.ScheduleCount, r.ScheduleCadence, r.MonthEndPolicy, r.CustomScheduleItems); err != nil {
		return err
	}
	input, err := repaymentScheduleInput(r, "preview")
	if err != nil {
		return err
	}
	_, _, err = schedules.NewStore().Create(input)
	return err
}
