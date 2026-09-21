package paymentclaims

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"kredit/internal/payments"
)

type reviewReplayRecorder struct {
	payments.Service
	calls int
}

func (r *reviewReplayRecorder) Record(in payments.RecordInput) (payments.Payment, payments.Allocation, error) {
	r.calls++
	return payments.Payment{ID: "recognized-payment", ObligationID: in.ObligationID}, payments.Allocation{}, nil
}

func newReviewReplayClaim(t *testing.T) (*Store, Claim) {
	t.Helper()
	s := NewStore(func(id string) (ObligationSnapshot, error) {
		return ObligationSnapshot{ID: id, BuyerUserID: "buyer", SupplierOrganizationID: "seller", OutstandingKobo: 10000, Currency: "NGN"}, nil
	})
	s.now = func() time.Time { return time.Date(2026, 9, 21, 10, 0, 0, 0, time.UTC) }
	claim, err := s.Create(context.Background(), CreateInput{ObligationID: "obligation", BuyerUserID: "buyer", AmountKobo: 2500, TransferReference: "bank-transfer", IdempotencyKey: "claim-create"})
	if err != nil {
		t.Fatal(err)
	}
	return s, claim
}

func TestConfirmationReplayRetainsReviewerEvidenceAndSinglePayment(t *testing.T) {
	ctx := context.Background()
	s, claim := newReviewReplayClaim(t)
	recorder := &reviewReplayRecorder{}
	first, err := s.Confirm(ctx, claim.ID, "reviewer-a", "  Matched bank evidence  ", recorder)
	if err != nil {
		t.Fatal(err)
	}
	s.now = func() time.Time { return first.ReviewedAt.Add(time.Hour) }
	for _, reason := range []string{"Matched bank evidence", "  Matched bank evidence  "} {
		retry, err := s.Confirm(ctx, claim.ID, "reviewer-a", reason, recorder)
		if err != nil || !reflect.DeepEqual(first, retry) {
			t.Fatalf("exact replay changed the recorded decision: %+v %v", retry, err)
		}
	}
	for _, tc := range []struct{ actor, reason string }{
		{"reviewer-b", "Matched bank evidence"},
		{"reviewer-a", "Different bank evidence"},
	} {
		if _, err := s.Confirm(ctx, claim.ID, tc.actor, tc.reason, recorder); !errors.Is(err, ErrReviewConflict) {
			t.Fatalf("different review acknowledged as the original: %v", err)
		}
	}
	stored, err := s.Get(ctx, claim.ID)
	if err != nil || !reflect.DeepEqual(first, stored) || recorder.calls != 1 {
		t.Fatalf("review replay changed history or recorded another payment: calls=%d error=%v", recorder.calls, err)
	}
}

func TestMemoryDecisionReplayRequiresTheOriginalReviewerReasonAndPayment(t *testing.T) {
	for _, state := range []string{Confirmed, Rejected} {
		t.Run(state, func(t *testing.T) {
			ctx := context.Background()
			s, claim := newReviewReplayClaim(t)
			paymentID := ""
			if state == Confirmed {
				paymentID = "recognized-payment"
			}
			first, err := s.Decide(ctx, claim.ID, "reviewer-a", state, "Reviewed original evidence", paymentID)
			if err != nil {
				t.Fatal(err)
			}
			for _, tc := range []struct{ actor, reason, payment string }{
				{"reviewer-b", "Reviewed original evidence", paymentID},
				{"reviewer-a", "Different evidence", paymentID},
				{"reviewer-a", "Reviewed original evidence", "another-payment"},
			} {
				if _, err := s.Decide(ctx, claim.ID, tc.actor, state, tc.reason, tc.payment); !errors.Is(err, ErrReviewConflict) {
					t.Fatalf("changed review replay was accepted: %v", err)
				}
			}
			retry, err := s.Decide(ctx, claim.ID, "reviewer-a", state, "  Reviewed original evidence  ", paymentID)
			if err != nil || !reflect.DeepEqual(first, retry) {
				t.Fatalf("exact replay changed saved evidence: %+v %v", retry, err)
			}
		})
	}
}
