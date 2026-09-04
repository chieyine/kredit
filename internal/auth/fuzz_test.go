package auth

import (
	"strings"
	"testing"
)

// FuzzPhoneNormalisation covers the "phone normalisation" target README
// section 40.3 requires. Normalisation decides account identity, so it has to
// be idempotent: if normalising twice differed from normalising once, the same
// person could be stored under two identifiers and their trade history would
// split.
func FuzzPhoneNormalisation(f *testing.F) {
	for _, seed := range []string{
		"+234 801 234 5678", "08012345678", "8012345678", "2348012345678",
		"+2348012345678", "(0801) 234-5678", "  Owner@Example.TEST ", "", "+", "++234", "0",
	} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, raw string) {
		once := NormalizeIdentifier(raw)
		twice := NormalizeIdentifier(once)
		if once != twice {
			t.Fatalf("normalisation is not idempotent: %q -> %q -> %q", raw, once, twice)
		}
		if strings.Contains(once, "@") {
			if once != strings.ToLower(once) {
				t.Fatalf("email identifier was not lowercased: %q", once)
			}
			return
		}
		for _, separator := range []string{" ", "-", "(", ")", "\t"} {
			if strings.Contains(once, separator) {
				t.Fatalf("normalised phone %q still contains %q", once, separator)
			}
		}
		if plus := strings.Count(once, "+"); plus > 1 || (plus == 1 && !strings.HasPrefix(once, "+")) {
			t.Fatalf("normalised phone %q has a misplaced country-code marker", once)
		}
	})
}
