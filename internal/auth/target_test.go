package auth

import "testing"

func TestOTPVerificationCanBeBoundToInvitationTarget(t *testing.T) {
	store := NewStore("test-key")
	challenge, code, err := store.RequestOTP("buyer@example.test", "email", "buyer_invitation")
	if err != nil {
		t.Fatal(err)
	}
	if _, _, _, err := store.VerifyOTPForTarget(challenge.ID, code, "device", "email", "other@example.test"); err == nil {
		t.Fatal("mismatched OTP target should be rejected")
	}
	if _, _, _, err := store.VerifyOTPForTarget(challenge.ID, code, "device", "email", "buyer@example.test"); err != nil {
		t.Fatal(err)
	}
}
