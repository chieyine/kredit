package publictoken

import (
	"testing"
	"time"
)

func TestPurposeAndExpiryAreBound(t *testing.T) {
	now := time.Date(2026, 8, 21, 0, 0, 0, 0, time.UTC)
	token, err := Issue("development-test-key", "receipt", "payment-1", now.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if id, err := Parse("development-test-key", token, "receipt", now); err != nil || id != "payment-1" {
		t.Fatalf("id=%q err=%v", id, err)
	}
	if _, err := Parse("development-test-key", token, "payment", now); err == nil {
		t.Fatal("purpose replay must fail")
	}
	if _, err := Parse("development-test-key", token, "receipt", now.Add(time.Hour)); err == nil {
		t.Fatal("expired token must fail")
	}
}
