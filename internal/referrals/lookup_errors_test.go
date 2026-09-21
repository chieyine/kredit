package referrals

import (
	"context"
	"errors"
	"testing"
)

func TestMissingReferralPersistenceIsNotAnInvalidCode(t *testing.T) {
	for _, store := range []*Store{nil, {}} {
		value, err := store.Lookup(context.Background(), "DSA-SYNTHETIC")
		if !errors.Is(err, ErrUnavailable) || errors.Is(err, ErrCodeUnavailable) || value != nil {
			t.Fatalf("missing persistence became a code result: %s, %v", value, err)
		}
		if _, err := store.begin(context.Background(), "actor", "organization"); !errors.Is(err, ErrUnavailable) {
			t.Fatalf("transaction did not fail closed: %v", err)
		}
	}
}
