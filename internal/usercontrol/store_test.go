package usercontrol

import (
	"context"
	"encoding/hex"
	"testing"
	"time"
)

func TestRecoveryRequiresIndependentEvidenceReviewAndCoolingOff(t *testing.T) {
	s := NewStore("secret")
	now := time.Date(2026, 8, 29, 10, 0, 0, 0, time.UTC)
	s.now = func() time.Time { return now }
	s.BindUser("owner", "owner@example.test", "+2348000000001")
	codes, err := s.GenerateRecoveryCodes(context.Background(), "owner")
	if err != nil || len(codes) != 10 {
		t.Fatalf("codes=%d err=%v", len(codes), err)
	}
	id, err := s.RequestRecovery(context.Background(), "owner@example.test", "email", "device-a")
	if err != nil || id == "" {
		t.Fatalf("id=%q err=%v", id, err)
	}
	r, err := s.AddRecoveryEvidence(context.Background(), id, "verified_phone", "otp-proof")
	if err != nil || r.State != RecoveryPendingVerification {
		t.Fatalf("phone-only=%+v err=%v", r, err)
	}
	r, err = s.AddRecoveryEvidence(context.Background(), id, "recovery_code", codes[0])
	if err != nil || r.State != RecoveryPendingReview {
		t.Fatalf("two factors=%+v err=%v", r, err)
	}
	if _, err = s.AddRecoveryEvidence(context.Background(), id, "recovery_code", codes[0]); err == nil {
		t.Fatal("replayed recovery code accepted")
	}
	if _, _, err = s.ReviewRecovery(context.Background(), id, "owner", "approve", "identity evidence matched", r.Version); err == nil {
		t.Fatal("self approval accepted")
	}
	r, token, err := s.ReviewRecovery(context.Background(), id, "reviewer", "approve", "identity evidence matched", r.Version)
	if err != nil || token == "" || r.State != RecoveryCoolingOff {
		t.Fatalf("review=%+v token=%q err=%v", r, token, err)
	}
	if _, err = s.CompleteRecovery(context.Background(), id, token); err == nil {
		t.Fatal("cooling off bypassed")
	}
	now = now.Add(25 * time.Hour)
	if userID, err := s.CompleteRecovery(context.Background(), id, token); err != nil || userID != "owner" {
		t.Fatalf("complete user=%q err=%v", userID, err)
	}
}

func TestPrivacyNeedsIndependentCompletionAndAppliesRestriction(t *testing.T) {
	s := NewStore("secret")
	r, err := s.CreatePrivacyRequest(context.Background(), "subject", "", "DELETION", "Remove optional profile information")
	if err != nil || r.State != "IN_REVIEW" {
		t.Fatalf("create=%+v err=%v", r, err)
	}
	if _, err = s.DecidePrivacy(context.Background(), r.ID, "subject", "APPROVED", "request is identity bound", r.Version); err == nil {
		t.Fatal("self decision accepted")
	}
	r, err = s.DecidePrivacy(context.Background(), r.ID, "reviewer-a", "APPROVED", "request is identity bound", r.Version)
	if err != nil || !s.restrictions["subject"] {
		t.Fatalf("decision=%+v restriction=%v err=%v", r, s.restrictions["subject"], err)
	}
	if _, err = s.CompletePrivacy(context.Background(), r.ID, "reviewer-a", "reviewer-a", r.Version); err == nil {
		t.Fatal("single-person destructive completion accepted")
	}
	r, err = s.CompletePrivacy(context.Background(), r.ID, "reviewer-a", "reviewer-b", r.Version)
	if err != nil || r.State != "COMPLETED" {
		t.Fatalf("complete=%+v err=%v", r, err)
	}
}

func TestUnknownRecoveryIsEnumerationSafe(t *testing.T) {
	s := NewStore("secret")
	id, err := s.RequestRecovery(context.Background(), "unknown@example.test", "email", "device")
	if err != nil || id != "" {
		t.Fatalf("id=%q err=%v", id, err)
	}
}

func TestRecoveryRateLimitCountsUnknownIdentifiers(t *testing.T) {
	s := NewStore("secret")
	for i := 0; i < 7; i++ {
		id, err := s.RequestRecovery(context.Background(), "unknown@example.test", "email", "shared-device")
		if err != nil || id != "" {
			t.Fatalf("attempt %d id=%q err=%v", i, id, err)
		}
	}
	if len(s.rate[hex.EncodeToString(s.digest("shared-device"))]) != 7 {
		t.Fatal("unknown-account attempts did not share the rate limiter")
	}
}
