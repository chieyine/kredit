package publictoken

import (
	"strings"
	"testing"
	"time"
)

func TestAuditIssueRejectsUnverifiableInputs(t *testing.T) {
	for _, tc := range []struct{ name, purpose, id string }{
		{"oversized reference", "receipt", strings.Repeat("x", 8192)},
		{"oversized purpose", strings.Repeat("x", 8192), "reference"},
		{"invalid reference encoding", "receipt", "ref\xff"},
		{"invalid purpose encoding", "receipt\xff", "reference"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			token, err := Issue("synthetic-key", tc.purpose, tc.id, time.Unix(1800003600, 0))
			if err == nil || token != "" {
				t.Fatal("issued a token that cannot preserve the caller's reference")
			}
		})
	}
}

func TestAuditTokenSignatureHasOneEncoding(t *testing.T) {
	now := time.Unix(1800000000, 0)
	token, err := Issue("synthetic-key", "receipt", "reference", now.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	const alphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_"
	last := strings.IndexByte(alphabet, token[len(token)-1])
	alias := token[:len(token)-1] + string(alphabet[last+1])
	for _, altered := range []string{alias, token + "\n", token[:len(token)-3] + "\r\n" + token[len(token)-3:]} {
		if _, err := Parse("synthetic-key", altered, "receipt", now); err == nil {
			t.Fatal("accepted a noncanonical signature encoding")
		}
	}
	if id, err := Parse("synthetic-key", token, "receipt", now); err != nil || id != "reference" {
		t.Fatalf("original token failed: %q %v", id, err)
	}
}

func TestAuditTokenSizeBoundaryRoundTrip(t *testing.T) {
	now := time.Unix(1800000000, 0)
	lo, hi := 1, 8192
	for lo < hi {
		mid := (lo + hi + 1) / 2
		token, err := Issue("synthetic-key", "receipt", strings.Repeat("x", mid), now.Add(time.Hour))
		if err == nil && len(token) <= 8192 {
			lo = mid
		} else {
			hi = mid - 1
		}
	}
	id := strings.Repeat("x", lo)
	token, err := Issue("synthetic-key", "receipt", id, now.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	got, err := Parse("synthetic-key", token, "receipt", now)
	if err != nil || got != id {
		t.Fatalf("boundary token failed: %v", err)
	}
	if token, err := Issue("synthetic-key", "receipt", id+"x", now.Add(time.Hour)); err == nil || token != "" {
		t.Fatal("oversized token was issued")
	}
}
