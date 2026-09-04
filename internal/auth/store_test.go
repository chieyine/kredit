package auth

import (
	"errors"
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
	code = totp(method.Secret, now.Unix()/30)
	updatedSession, rotatedToken, err := store.StepUpSession(token, code)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err = store.SessionFromToken(token); err == nil {
		t.Fatal("pre-MFA session remained usable")
	}
	updated, _, err := store.SessionFromToken(rotatedToken)
	if err != nil {
		t.Fatal(err)
	}
	if updated.AuthenticationLevel != AAL2 || updatedSession.AuthenticationLevel != AAL2 || session.AuthenticationLevel != AAL1 {
		t.Fatalf("unexpected assurance levels: updated=%s initial=%s", updated.AuthenticationLevel, session.AuthenticationLevel)
	}
	if _, _, err = store.StepUpSession(rotatedToken, code); err == nil {
		t.Fatal("replayed TOTP code was accepted")
	}
}

func TestOTPCooldownCannotBeBypassedByChangingPurpose(t *testing.T) {
	s := NewStore("secret")
	if _, _, err := s.RequestOTP("cooldown@example.test", "email", "login"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.RequestOTP("cooldown@example.test", "email", "different-purpose"); err == nil {
		t.Fatal("purpose bypassed cooldown")
	}
}

func TestEnrollmentCannotReplaceExistingMFA(t *testing.T) {
	s := NewStore("secret")
	user, err := s.FindOrCreateUser("mfa@example.test", "email")
	if err != nil {
		t.Fatal(err)
	}
	method, err := s.BeginTOTPEnrollment(user.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.VerifyTOTP(user.ID, TOTPCode(method.Secret, s.now())); err != nil {
		t.Fatal(err)
	}
	if _, err := s.BeginTOTPEnrollment(user.ID); err == nil {
		t.Fatal("existing MFA replaced without recovery")
	}
}

// TestTOTPVerificationRejectsMalformedCodes covers the authentication bypass
// that an empty submitted code produced: totp() returns "" when the stored
// secret cannot be decoded, and the old plain "==" comparison then matched.
func TestTOTPVerificationRejectsMalformedCodes(t *testing.T) {
	now := time.Date(2026, 8, 16, 12, 0, 0, 0, time.UTC)
	valid := "JBSWY3DPEHPK3PXP"
	for _, testCase := range []struct {
		name   string
		secret string
		code   string
	}{
		{"empty code against an undecodable secret", "not-valid-base32!!", ""},
		{"empty code against a valid secret", valid, ""},
		{"whitespace only", valid, "   "},
		{"too short", valid, "12345"},
		{"too long", valid, "1234567"},
		{"non-numeric", valid, "12a456"},
	} {
		if validTOTP(testCase.secret, testCase.code, now) {
			t.Fatalf("%s: malformed MFA code was accepted", testCase.name)
		}
	}
	current := TOTPCode(valid, now)
	if !validTOTP(valid, current, now) {
		t.Fatal("the current TOTP code must still verify")
	}
	if !validTOTP(valid, " "+current+" ", now) {
		t.Fatal("a padded but otherwise correct code must still verify")
	}
	if !validTOTP(valid, TOTPCode(valid, now.Add(-30*time.Second)), now) || !validTOTP(valid, TOTPCode(valid, now.Add(30*time.Second)), now) {
		t.Fatal("the one-step clock-skew window must be preserved")
	}
}

// TestGeneratedOTPCodesAreWellFormedAndUnbiased guards the rejection sampling
// that replaced a biased "uint32 % 1e6" reduction.
func TestGeneratedOTPCodesAreWellFormedAndUnbiased(t *testing.T) {
	buckets := make([]int, 10)
	const draws = 20000
	for i := 0; i < draws; i++ {
		code, err := randomOTP()
		if err != nil {
			t.Fatal(err)
		}
		if len(code) != 6 {
			t.Fatalf("otp %q is not six digits", code)
		}
		for _, character := range code {
			if character < '0' || character > '9' {
				t.Fatalf("otp %q is not numeric", code)
			}
		}
		buckets[code[0]-'0']++
	}
	expected := float64(draws) / 10
	for digit, count := range buckets {
		deviation := float64(count)/expected - 1
		if deviation < -0.2 || deviation > 0.2 {
			t.Fatalf("leading digit %d appeared %d times, expected roughly %.0f", digit, count, expected)
		}
	}
}

// TestMFAVerificationLocksAfterRepeatedFailures covers the per-account throttle.
// A six-digit code with a three-step acceptance window is brute-forceable behind
// a shared per-IP budget alone, so verification has to count attempts per user.
func TestMFAVerificationLocksAfterRepeatedFailures(t *testing.T) {
	store := NewStore("test-key")
	now := time.Date(2026, 8, 16, 12, 0, 0, 0, time.UTC)
	store.now = func() time.Time { return now }
	challenge, code, err := store.RequestOTP("owner@example.test", "email", "login")
	if err != nil {
		t.Fatal(err)
	}
	user, _, _, err := store.VerifyOTP(challenge.ID, code, "device")
	if err != nil {
		t.Fatal(err)
	}
	method, err := store.BeginTOTPEnrollment(user.ID)
	if err != nil {
		t.Fatal(err)
	}

	for attempt := 1; attempt < mfaMaxFailedAttempts; attempt++ {
		if err := store.VerifyTOTP(user.ID, "000000"); err == nil || errors.Is(err, ErrMFALocked) {
			t.Fatalf("attempt %d should be a plain rejection, got %v", attempt, err)
		}
	}
	if err := store.VerifyTOTP(user.ID, "000000"); !errors.Is(err, ErrMFALocked) {
		t.Fatalf("the account should be locked after %d failures, got %v", mfaMaxFailedAttempts, err)
	}
	// A correct code is refused while the lock holds, otherwise the throttle
	// would only slow an attacker who never guesses right.
	if err := store.VerifyTOTP(user.ID, TOTPCode(method.Secret, now)); !errors.Is(err, ErrMFALocked) {
		t.Fatalf("a correct code must not clear an active lock, got %v", err)
	}
	// The lock releases itself so a legitimate user is never shut out for good.
	now = now.Add(mfaLockDuration + time.Minute)
	if err := store.VerifyTOTP(user.ID, TOTPCode(method.Secret, now)); err != nil {
		t.Fatalf("verification should succeed once the lock expires: %v", err)
	}
	// A success resets the counter.
	for attempt := 1; attempt < mfaMaxFailedAttempts; attempt++ {
		if err := store.VerifyTOTP(user.ID, "000000"); errors.Is(err, ErrMFALocked) {
			t.Fatalf("the failure counter did not reset after a success (attempt %d)", attempt)
		}
	}
}

// TestSessionsExpireWhenIdle covers the idle deadline README section 21.2
// requires alongside the absolute lifetime.
func TestSessionsExpireWhenIdle(t *testing.T) {
	if sessionIdleTimeout+time.Minute >= sessionAbsoluteLifetime {
		t.Fatal("the idle window must be shorter than the absolute lifetime for this test to isolate it")
	}
	newSession := func(clock *time.Time) (*Store, string) {
		t.Helper()
		store := NewStore("test-key")
		store.now = func() time.Time { return *clock }
		challenge, code, err := store.RequestOTP("+2348012345678", "phone", "login")
		if err != nil {
			t.Fatal(err)
		}
		_, _, token, err := store.VerifyOTP(challenge.ID, code, "device")
		if err != nil {
			t.Fatal(err)
		}
		return store, token
	}

	// An untouched session is refused once the idle window passes, while its
	// absolute lifetime still has two weeks to run.
	idleClock := time.Date(2026, 8, 16, 12, 0, 0, 0, time.UTC)
	idleStore, idleToken := newSession(&idleClock)
	idleClock = idleClock.Add(sessionIdleTimeout + time.Minute)
	if _, _, err := idleStore.SessionFromToken(idleToken); err == nil {
		t.Fatal("an idle session was still accepted")
	}

	// Regular use keeps a session alive across several idle windows.
	activeClock := time.Date(2026, 8, 16, 12, 0, 0, 0, time.UTC)
	activeStore, activeToken := newSession(&activeClock)
	for step := 1; step <= 3; step++ {
		activeClock = activeClock.Add(sessionIdleTimeout / 2)
		if _, _, err := activeStore.SessionFromToken(activeToken); err != nil {
			t.Fatalf("an actively used session expired at step %d: %v", step, err)
		}
	}
}

func TestAbandonedEnrollmentCanRestartAndRecoveryRevokesOldFactor(t *testing.T) {
	s := NewStore("secret")
	c, code, err := s.RequestOTP("restart@example.test", "email", "login")
	if err != nil {
		t.Fatal(err)
	}
	u, _, token, err := s.VerifyOTP(c.ID, code, "device")
	if err != nil {
		t.Fatal(err)
	}
	old, err := s.BeginTOTPEnrollment(u.ID)
	if err != nil {
		t.Fatal(err)
	}
	next, err := s.BeginTOTPEnrollment(u.ID)
	if err != nil || next.Secret == old.Secret {
		t.Fatalf("restart: %v", err)
	}
	if err = s.VerifyTOTP(u.ID, TOTPCode(next.Secret, s.now())); err != nil {
		t.Fatal(err)
	}
	if err = s.ResetAfterRecovery(u.ID); err != nil {
		t.Fatal(err)
	}
	if _, _, err = s.SessionFromToken(token); err == nil {
		t.Fatal("old session survived recovery")
	}
	if err = s.VerifyTOTP(u.ID, TOTPCode(next.Secret, s.now())); err == nil {
		t.Fatal("lost factor survived recovery")
	}
	replacement, err := s.BeginTOTPEnrollment(u.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err = s.VerifyTOTP(u.ID, TOTPCode(replacement.Secret, s.now())); err != nil {
		t.Fatal(err)
	}
}
