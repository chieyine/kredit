package consumer

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"testing"
	"time"
)

func termsFixture(now time.Time) Terms {
	return Terms{Item: "Refrigerator", Quantity: 1, Total: 100000, Deposit: 10000, DepositDate: now.Format("2006-01-02"), FirstDate: now.AddDate(0, 0, 7).Format("2006-01-02"), Count: 3, Cadence: "weekly", Fulfillment: "on_percentage", Threshold: 50, DeliveryDays: 7, StockReserved: true, StockReference: "stock-test", Returns: "Full refund before release; inspect and return defective goods."}
}
func TestConsumerDeliveryAndRefundLifecycle(t *testing.T) {
	now := time.Date(2026, 9, 15, 10, 0, 0, 0, time.UTC)
	terms, hash, e := Prepare(termsFixture(now), now)
	if e != nil {
		t.Fatal(e)
	}
	var total int64
	for _, d := range terms.Schedule {
		total += d.Amount
	}
	if total != terms.Total {
		t.Fatal("schedule changed purchase amount")
	}
	s := Sale{Terms: terms, Hash: hash, State: "offered", Version: 1}
	apply := func(role string, a Action) {
		t.Helper()
		ev, e := s.Apply("customer", role, a, now)
		if e != nil {
			t.Fatal(a.Action, e)
		}
		s.Events[len(s.Events)-1].ID = ev.Action + time.Now().String()
		s.Version++
	}
	apply("buyer", Action{Action: "accept", Hash: hash, Name: "Test Customer", Address: "Synthetic delivery address", Consent: true})
	apply("buyer", Action{Action: "claim", Amount: 50000, Reference: "claim-bank", At: now, Note: "Reported completed bank transfer"})
	if s.Eligible || s.Paid != 0 || s.DueAmount("2026-10-15") != 0 {
		t.Fatal("claim must not release goods, change money, or prompt another payment")
	}
	claim := s.Events[len(s.Events)-1]
	apply("seller", Action{Action: "payment", Amount: 50000, Reference: claim.Reference, RelatedID: claim.ID, At: now, Note: "Verified actual bank credit receipt"})
	raw, _ := json.Marshal(s.Terms)
	sum := sha256.Sum256(raw)
	if hex.EncodeToString(sum[:]) != hash {
		t.Fatal("payment progress changed the accepted agreement hash")
	}
	if !s.Eligible || s.Outstanding != 50000 {
		t.Fatal("threshold or outstanding incorrect")
	}
	if _, e = s.Apply("customer", "buyer", Action{Action: "release", Note: "Customer trying to release goods"}, now); e == nil {
		t.Fatal("buyer released goods")
	}
	apply("seller", Action{Action: "release", Note: "Verified handover to named customer"})
	apply("buyer", Action{Action: "received"})
	apply("buyer", Action{Action: "request_return", Note: "Item damaged on delivery; return requested"})
	apply("seller", Action{Action: "reject_return", Note: "Seller's review and rejection evidence"})
	apply("buyer", Action{Action: "escalate", Note: "Customer asks Kredit to review defect"})
	if _, e = s.Apply("seller", "seller", Action{Action: "approve_return", Note: "Seller attempts to decide escalated case"}, now); e == nil {
		t.Fatal("seller decided escalated case")
	}
	apply("admin", Action{Action: "approve_return", Note: "Return received and full refund approved"})
	if s.Outstanding != 0 || s.RefundDue != 50000 {
		t.Fatal("return must close debt and create refund due")
	}
	if _, e = s.Apply("seller", "seller", Action{Action: "refund", Amount: 50001, Reference: "refund", At: now, Note: "Actual bank refund evidence checked"}, now); e == nil {
		t.Fatal("overrefund allowed")
	}
	apply("seller", Action{Action: "refund", Amount: 50000, Reference: "refund-bank", At: now, Note: "Actual bank refund evidence checked"})
	if s.RefundDue != 0 || s.Outstanding != 0 {
		t.Fatal("refund not completed")
	}
}
func TestConsumerLayawayCancellationAndPriceReduction(t *testing.T) {
	now := time.Date(2026, 9, 15, 10, 0, 0, 0, time.UTC)
	terms := termsFixture(now)
	terms.Fulfillment = "on_full_payment"
	terms, hash, e := Prepare(terms, now)
	if e != nil {
		t.Fatal(e)
	}
	s := Sale{Terms: terms, Hash: hash, State: "offered", Version: 1}
	for _, a := range []Action{{Action: "accept", Hash: hash, Name: "Test Person", Address: "Test delivery address", Consent: true}, {Action: "payment", Amount: 100000, Reference: "bank", At: now, Note: "Confirmed full payment on bank statement"}} {
		role := "seller"
		if a.Action == "accept" {
			role = "buyer"
		}
		if _, e = s.Apply("test", role, a, now); e != nil {
			t.Fatal(e)
		}
	}
	if !s.Eligible {
		t.Fatal("fully paid layaway not eligible")
	}
	if _, e = s.Apply("test", "buyer", Action{Action: "cancel", Note: "Cancelled before retailer released stock"}, now); e != nil {
		t.Fatal(e)
	}
	if s.Eligible || s.Outstanding != 0 || s.RefundDue != 100000 {
		t.Fatal("cancelled layaway balances incorrect")
	}
}
