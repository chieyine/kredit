package publictoken

import (
	"testing"
	"time"
	"unicode/utf8"
)

// FuzzTokenReferenceParsing covers the "reference parsing" target README
// section 40.3 requires. A public token is the only reference a stranger can
// present, so forging one, reusing it under another purpose, or verifying it
// with the wrong key must all fail.
func FuzzTokenReferenceParsing(f *testing.F) {
	f.Add("receipt", "obligation-1", "forged.token")
	f.Add("payment-intent", "019-abc", "")
	f.Add("p", "i", "....")
	f.Fuzz(func(t *testing.T, purpose, id, candidate string) {
		now := time.Unix(1_800_000_000, 0).UTC()

		// Parse must never panic on arbitrary input.
		_, _ = Parse("token-key", candidate, "receipt", now)

		if purpose == "" || id == "" || !utf8.ValidString(purpose) || !utf8.ValidString(id) {
			return
		}
		issued, err := Issue("token-key", purpose, id, now.Add(time.Hour))
		if err != nil {
			t.Fatalf("issuing a token with a valid purpose and id failed: %v", err)
		}
		resolved, err := Parse("token-key", issued, purpose, now)
		if err != nil || resolved != id {
			t.Fatalf("issued token did not round trip: got %q err=%v", resolved, err)
		}
		if _, err := Parse("other-key", issued, purpose, now); err == nil {
			t.Fatal("a token verified under the wrong signing key")
		}
		if _, err := Parse("token-key", issued, purpose+"-other", now); err == nil {
			t.Fatal("a token verified under a different purpose")
		}
		if _, err := Parse("token-key", issued, purpose, now.Add(2*time.Hour)); err == nil {
			t.Fatal("an expired token verified")
		}
		if candidate != issued {
			if _, err := Parse("token-key", candidate, purpose, now); err == nil {
				t.Fatalf("a token that was never issued verified: %q", candidate)
			}
		}
	})
}
