package auth

import (
	"testing"
	"time"
)

func TestOTPLoginStoresOnlyOpaqueSessionLookup(t *testing.T) {
	store := NewStore("test-key")
	challenge, code, err := store.RequestOTP("+234 800 000 0000", "phone", "login")
	if err != nil {
		t.Fatal(err)
	}
	user, session, token, err := store.VerifyOTP(challenge.ID, code, "test device")
	if err != nil {
		t.Fatal(err)
	}
	if user.Phone != "+2348000000000" || session.AuthenticationLevel != AAL1 || token == "" {
		t.Fatalf("unexpected login result: user=%+v session=%+v token=%q", user, session, token)
	}
	if _, _, err := store.SessionFromToken(token); err != nil {
		t.Fatal(err)
	}
	if err := store.RevokeSession(token); err != nil {
		t.Fatal(err)
	}
	if _, _, err := store.SessionFromToken(token); err == nil {
		t.Fatal("revoked session should not authenticate")
	}
}

func TestTOTPEnrollmentElevatesSession(t *testing.T) {
	store := NewStore("test-key")
	now := time.Date(2026, 8, 16, 12, 0, 0, 0, time.UTC)
	store.now = func() time.Time { return now }
	challenge, code, err := store.RequestOTP("owner@example.test", "email", "login")
	if err != nil {
		t.Fatal(err)
	}
	user, session, token, err := store.VerifyOTP(challenge.ID, code, "test device")
	if err != nil {
		t.Fatal(err)
	}
	method, err := store.BeginTOTPEnrollment(user.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.VerifyTOTP(user.ID, totp(method.Secret, now.Unix()/30)); err != nil {
		t.Fatal(err)
	}
	if err := store.ElevateSession(token); err != nil {
		t.Fatal(err)
	}
	updated, _, err := store.SessionFromToken(token)
	if err != nil {
		t.Fatal(err)
	}
	if updated.AuthenticationLevel != AAL2 || session.AuthenticationLevel != AAL1 {
		t.Fatalf("unexpected assurance levels: updated=%s initial=%s", updated.AuthenticationLevel, session.AuthenticationLevel)
	}
}
