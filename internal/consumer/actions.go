package consumer

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

func (s *Sale) Apply(actor, role string, in Action, now time.Time) (Event, error) {
	fail := func(message string) (Event, error) { return Event{}, errors.New(message) }
	buyer := role == "buyer"
	seller := role == "seller" || role == "admin"
	ev := Event{Action: in.Action, Amount: in.Amount, Reference: strings.TrimSpace(in.Reference), RelatedID: in.RelatedID, Note: strings.TrimSpace(in.Note), At: now, RecordedAt: now}
	if len(ev.Note) > 2000 || len(ev.Reference) > 200 {
		return fail("Keep evidence below 2000 characters and references below 200.")
	}
	find := func(id string) *Event {
		for i := range s.Events {
			if s.Events[i].ID == id {
				return &s.Events[i]
			}
		}
		return nil
	}
	decided := func(id string) bool {
		for _, e := range s.Events {
			if e.RelatedID == id && (e.Action == "payment" || e.Action == "reject_claim" || e.Action == "reverse_payment") {
				return true
			}
		}
		return false
	}
	evidence := func() bool { return len(ev.Note) >= 20 }
	monetary := in.Action == "payment" || in.Action == "claim" || in.Action == "refund"
	if monetary {
		if ev.Amount <= 0 || ev.Amount > s.Terms.Total || len(ev.Reference) < 3 || in.At.IsZero() || in.At.After(now.Add(time.Minute)) || s.AcceptedAt == nil || in.At.Before(*s.AcceptedAt) {
			return fail("Enter a positive amount, unique transfer or cash receipt reference, and actual payment time after acceptance.")
		}
		ev.At = in.At
	}
	if !monetary && in.Action != "reduce_price" {
		ev.Amount = 0
	}
	switch in.Action {
	case "accept":
		if !buyer || s.State != "offered" || s.BuyerID != "" || in.Hash != s.Hash || !in.Consent || len(strings.TrimSpace(in.Name)) < 3 || len(in.Name) > 200 || len(strings.TrimSpace(in.Address)) < 10 || len(in.Address) > 1000 {
			return fail("Review the agreement, confirm you are 18 or older, and enter your full name and delivery address.")
		}
		first := s.Terms.FirstDate
		if s.Terms.Deposit > 0 {
			first = s.Terms.DepositDate
		}
		if first < now.In(time.FixedZone("Africa/Lagos", 3600)).Format("2006-01-02") {
			return fail("The payment dates have passed. Ask the retailer for a new offer.")
		}
		s.BuyerID = actor
		s.CustomerName = strings.TrimSpace(in.Name)
		s.Address = strings.TrimSpace(in.Address)
		s.AcceptedAt = &now
		s.State = "active"
		ev.Note = "Accepted " + s.Hash + "; adult and terms consent recorded"
		ev.Reference = s.Hash
	case "decline":
		if !buyer || s.State != "offered" {
			return fail("Only an unaccepted offer can be declined.")
		}
		s.State = "cancelled"
	case "claim":
		if !buyer || s.AcceptedAt == nil || !evidence() {
			return fail("Describe your completed transfer or cash payment. The retailer must verify it.")
		}
		if ev.Amount > s.Terms.Total-s.Paid {
			return fail("The reported amount exceeds the unrecorded purchase amount.")
		}
		for _, e := range s.Events {
			if e.Action == "claim" && e.Reference == ev.Reference {
				return fail("This payment has already been reported.")
			}
		}
	case "payment":
		if !seller || !evidence() || ev.Amount > s.Terms.Total-s.Paid {
			return fail("Only an authorised retailer can confirm a receipt, within the purchase amount, with evidence.")
		}
		if in.RelatedID != "" {
			claim := find(in.RelatedID)
			if claim == nil || claim.Action != "claim" || decided(in.RelatedID) || claim.Amount != ev.Amount || claim.Reference != ev.Reference || !claim.At.Equal(ev.At) {
				return fail("The receipt must match the unresolved customer claim exactly.")
			}
		}
	case "reject_claim":
		claim := find(in.RelatedID)
		if !seller || !evidence() || claim == nil || claim.Action != "claim" || decided(in.RelatedID) {
			return fail("Choose an unresolved claim and explain why the payment was not found.")
		}
		ev.Reference = in.RelatedID
	case "reverse_payment":
		payment := find(in.RelatedID)
		if !seller || !evidence() || payment == nil || payment.Action != "payment" || decided(in.RelatedID) {
			return fail("Choose a recorded payment and explain the confirmed correction or bank reversal.")
		}
		if s.Paid-payment.Amount < s.Refunded {
			return fail("This reversal would exceed funds retained after refunds. Escalate the discrepancy before changing the ledger.")
		}
		ev.Amount = payment.Amount
		ev.Reference = in.RelatedID
		if s.State == "completed" {
			s.State = "active"
		}
	case "release":
		if !seller || !s.Eligible || !evidence() {
			return fail("Delivery requires sufficient confirmed payments, no open case, and dispatch or handover evidence.")
		}
		s.ReleasedAt = &now
		ev.Note = "Stock " + s.Terms.StockReference + ": " + ev.Note
	case "received":
		if !buyer || s.ReleasedAt == nil || s.ReceivedAt != nil || s.State == "cancelled" {
			return fail("Only the customer can confirm receipt after delivery is recorded.")
		}
		s.ReceivedAt = &now
	case "cancel":
		if (!buyer && !seller) || s.ReleasedAt != nil || s.State == "cancelled" || !evidence() {
			return fail("Before goods are released, either party may cancel with a reason. After release, request a return.")
		}
		s.State = "cancelled"
		s.CaseState = "approved"
	case "request_return":
		if !buyer || s.ReleasedAt == nil || s.State == "cancelled" || s.CaseState == "requested" || s.CaseState == "escalated" || !evidence() {
			return fail("Explain the delivery issue or return request.")
		}
		s.CaseState = "requested"
	case "escalate":
		if !buyer || s.CaseState != "rejected" || !evidence() {
			return fail("A rejected return can be escalated to super admin with a reason.")
		}
		s.CaseState = "escalated"
	case "approve_return", "reject_return":
		if !seller || (s.CaseState != "requested" && s.CaseState != "escalated") || !evidence() || (s.CaseState == "escalated" && role != "admin") {
			return fail("An authorised reviewer must decide the open case and record return, non-delivery or rejection evidence.")
		}
		if in.Action == "approve_return" {
			s.State = "cancelled"
			s.CaseState = "approved"
		} else {
			s.CaseState = "rejected"
		}
	case "reduce_price":
		if !seller || s.AcceptedAt == nil || s.State == "cancelled" || ev.Amount <= 0 || ev.Amount > s.Terms.Total-s.Reduction || !evidence() {
			return fail("Enter a price reduction within the remaining sale price and explain it. Any overpayment becomes refundable.")
		}
	case "refund":
		if !seller || !evidence() || ev.Amount > s.RefundDue {
			return fail("Record only a completed refund, within the amount due to the customer, with bank or signed cash-return evidence.")
		}
	default:
		return fail("Unknown purchase action.")
	}
	if ev.Reference == "" {
		ev.Reference = fmt.Sprintf("version-%d", s.Version)
	}
	s.Events = append(s.Events, ev)
	s.Calculate()
	if s.State == "active" && s.ReceivedAt != nil && s.Outstanding == 0 && s.CaseState != "requested" && s.CaseState != "escalated" {
		s.State = "completed"
	}
	return ev, nil
}
