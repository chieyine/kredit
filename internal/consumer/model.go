// Package consumer manages retailer-to-person sales independently of trade debt.
package consumer

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"kredit/internal/ledger"
	"kredit/internal/schedules"
	"strings"
	"time"
)

type Due struct {
	Date   string `json:"date"`
	Amount int64  `json:"amount_kobo"`
	Paid   int64  `json:"paid_kobo"`
}
type Terms struct {
	Item           string `json:"item"`
	Quantity       int    `json:"quantity"`
	Total          int64  `json:"total_kobo"`
	Deposit        int64  `json:"deposit_kobo"`
	DepositDate    string `json:"deposit_date"`
	FirstDate      string `json:"first_date"`
	Count          int    `json:"count"`
	Cadence        string `json:"cadence"`
	Fulfillment    string `json:"fulfillment"`
	Threshold      int    `json:"threshold_percent"`
	DeliveryDays   int    `json:"delivery_days"`
	StockReference string `json:"stock_reference"`
	StockReserved  bool   `json:"stock_reserved"`
	Returns        string `json:"returns_policy"`
	BankName       string `json:"bank_name"`
	AccountName    string `json:"account_name"`
	AccountNumber  string `json:"account_number"`
	SellerName     string `json:"seller_name"`
	SellerAddress  string `json:"seller_address"`
	TermsVersion   string `json:"terms_version"`
	Schedule       []Due  `json:"schedule"`
}
type Sale struct {
	Schedule       []Due      `json:"schedule_progress"`
	ID             string     `json:"id"`
	OrganizationID string     `json:"organization_id"`
	BuyerID        string     `json:"buyer_user_id"`
	TargetType     string     `json:"target_type"`
	Target         string     `json:"target"`
	Terms          Terms      `json:"terms"`
	Hash           string     `json:"agreement_hash"`
	State          string     `json:"state"`
	CustomerName   string     `json:"customer_name"`
	Address        string     `json:"delivery_address"`
	AcceptedAt     *time.Time `json:"accepted_at"`
	ReleasedAt     *time.Time `json:"released_at"`
	ReceivedAt     *time.Time `json:"received_at"`
	CaseState      string     `json:"case_state"`
	Version        int64      `json:"version"`
	CreatedAt      time.Time  `json:"created_at"`
	Events         []Event    `json:"events"`
	Paid           int64      `json:"paid_kobo"`
	Refunded       int64      `json:"refunded_kobo"`
	Reduction      int64      `json:"reduction_kobo"`
	Outstanding    int64      `json:"outstanding_kobo"`
	RefundDue      int64      `json:"refund_due_kobo"`
	Eligible       bool       `json:"release_eligible"`
	DeliveryDue    *time.Time `json:"delivery_due_at"`
	Role           string     `json:"role"`
}
type Event struct {
	ID         string    `json:"id"`
	Action     string    `json:"action"`
	Amount     int64     `json:"amount_kobo"`
	Reference  string    `json:"reference"`
	RelatedID  string    `json:"related_id"`
	Note       string    `json:"note"`
	At         time.Time `json:"occurred_at"`
	RecordedAt time.Time `json:"recorded_at"`
}
type Input struct {
	TargetType string `json:"target_type"`
	Target     string `json:"target"`
	Terms      Terms  `json:"terms"`
}
type Action struct {
	Action    string    `json:"action"`
	Version   int64     `json:"version"`
	Hash      string    `json:"agreement_hash"`
	Name      string    `json:"full_name"`
	Address   string    `json:"delivery_address"`
	Consent   bool      `json:"consent"`
	Amount    int64     `json:"amount_kobo"`
	Reference string    `json:"reference"`
	RelatedID string    `json:"related_id"`
	Note      string    `json:"note"`
	At        time.Time `json:"occurred_at"`
}

var ErrUnavailable = errors.New("this purchase is unavailable for your account")

func Prepare(t Terms, now time.Time) (Terms, string, error) {
	fail := func(message string) (Terms, string, error) { return Terms{}, "", errors.New(message) }
	t.Item = strings.TrimSpace(t.Item)
	t.Returns = strings.TrimSpace(t.Returns)
	if len(t.Item) < 3 || len(t.Item) > 500 || t.Quantity < 1 || t.Quantity > 10000 || t.Total < 100 || t.Total > 2500000000 || t.Deposit < 0 || t.Deposit >= t.Total {
		return fail("Enter an item, quantity and total price. The deposit must be less than the price.")
	}
	if t.Count < 1 || t.Count > 60 || t.DeliveryDays < 1 || t.DeliveryDays > 90 || !t.StockReserved || len(strings.TrimSpace(t.StockReference)) < 3 || len(t.StockReference) > 200 {
		return fail("Confirm reserved stock, 1–60 payments and delivery within 1–90 days of eligibility.")
	}
	if len(t.Returns) < 20 || len(t.Returns) > 2000 {
		return fail("Enter clear cancellation, return and refund terms (20–2000 characters). Statutory rights still apply.")
	}
	switch t.Fulfillment {
	case "immediate":
		t.Threshold = 0
	case "on_full_payment":
		t.Threshold = 100
	case "on_percentage":
		if t.Threshold < 1 || t.Threshold > 99 {
			return fail("Choose a delivery threshold between 1% and 99%.")
		}
	default:
		return fail("Choose when goods will be delivered.")
	}
	loc := time.FixedZone("Africa/Lagos", 3600)
	start, err := time.ParseInLocation("2006-01-02", t.FirstDate, loc)
	if err != nil || t.FirstDate < now.In(loc).Format("2006-01-02") {
		return fail("Choose a first payment date today or later.")
	}
	t.Schedule = nil
	if t.Deposit > 0 {
		d, e := time.ParseInLocation("2006-01-02", t.DepositDate, loc)
		if e != nil || t.DepositDate < now.In(loc).Format("2006-01-02") || !d.Before(start) {
			return fail("The deposit date must be today or later and before the first installment.")
		}
		t.Schedule = append(t.Schedule, Due{Date: t.DepositDate, Amount: t.Deposit})
	} else {
		t.DepositDate = ""
	}
	_, items, err := schedules.NewStore().Create(schedules.CreateInput{ObligationID: "consumer-preview", PrincipalKobo: ledger.Money(t.Total - t.Deposit), ScheduleType: schedules.TypeEqual, Count: t.Count, StartDate: start, DueHour: 23, DueMinute: 59, Timezone: "Africa/Lagos", Cadence: t.Cadence, MonthEndPolicy: schedules.PolicyCap})
	if err != nil {
		return fail(err.Error())
	}
	for _, i := range items {
		t.Schedule = append(t.Schedule, Due{Date: i.DueAt.In(loc).Format("2006-01-02"), Amount: int64(i.PrincipalDueKobo)})
	}
	t.TermsVersion = "consumer-sale-v1"
	raw, _ := json.Marshal(t)
	sum := sha256.Sum256(raw)
	return t, hex.EncodeToString(sum[:]), nil
}
func (s *Sale) Calculate() {
	s.Paid = 0
	s.Refunded = 0
	s.Reduction = 0
	s.DeliveryDue = nil
	var eligibleAt *time.Time
	frozen := false
	if s.AcceptedAt != nil && s.Terms.Fulfillment == "immediate" {
		a := *s.AcceptedAt
		eligibleAt = &a
	}
	for _, e := range s.Events {
		switch e.Action {
		case "payment":
			s.Paid += e.Amount
		case "reverse_payment":
			s.Paid -= e.Amount
		case "refund":
			s.Refunded += e.Amount
		case "reduce_price":
			s.Reduction += e.Amount
		}
		if s.AcceptedAt != nil && s.Paid-s.Refunded >= s.releaseAmount() && eligibleAt == nil {
			a := e.RecordedAt
			if a.IsZero() {
				a = e.At
			}
			eligibleAt = &a
		}
		if !frozen && s.Paid-s.Refunded < s.releaseAmount() && s.Terms.Fulfillment != "immediate" {
			eligibleAt = nil
		}
		if e.Action == "release" {
			frozen = true
		}
	}
	net := s.Paid - s.Refunded
	price := s.Terms.Total - s.Reduction
	s.Outstanding = max(int64(0), price-net)
	s.RefundDue = max(int64(0), net-price)
	if s.State == "cancelled" {
		s.Outstanding = 0
		s.RefundDue = max(int64(0), net)
	}
	s.Eligible = s.State == "active" && s.ReleasedAt == nil && s.CaseState != "requested" && s.CaseState != "escalated" && net >= s.releaseAmount()
	if eligibleAt != nil {
		d := eligibleAt.AddDate(0, 0, s.Terms.DeliveryDays)
		s.DeliveryDue = &d
	}
	remaining := net
	s.Schedule = append([]Due(nil), s.Terms.Schedule...)
	for i := range s.Schedule {
		s.Schedule[i].Paid = min(max(remaining, 0), s.Schedule[i].Amount)
		remaining -= s.Schedule[i].Paid
	}
}
func (s *Sale) releaseAmount() int64 {
	return ((s.Terms.Total-s.Reduction)*int64(s.Terms.Threshold) + 99) / 100
}
