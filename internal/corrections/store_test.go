package corrections

import "testing"

func TestCorrectionIsAppendOnlyAndAuditable(t *testing.T) {
	s := NewStore()
	r, err := s.Open("org-1", "payment", "payment-1", "ledger-tx-1", "buyer-1", "Payment was duplicated", []string{"document-1"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.StartReview(r.ID, "reviewer-1"); err != nil {
		t.Fatal(err)
	}
	updated, decision, err := s.Decide(r.ID, "reviewer-1", StateApproved, "Ledger and bank evidence agree")
	if err != nil {
		t.Fatal(err)
	}
	if updated.State != StateApproved || decision.CorrectionID == "" {
		t.Fatalf("expected approved correction event: %#v %#v", updated, decision)
	}
	_, decisions, err := s.Get(r.ID)
	if err != nil || len(decisions) != 1 {
		t.Fatalf("expected decision history: %v %#v", err, decisions)
	}
}
