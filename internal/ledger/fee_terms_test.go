package ledger

import (
	"math"
	"testing"
	"time"
)

func TestRecordedFeesAndZeroFeeJournal(t *testing.T) {
	for _, test := range []struct {
		amount Money
		bps    int64
		want   Money
	}{{100000, 25, 250}, {999, 50, 4}, {Money(math.MaxInt64), 1000, 922337203685477580}, {100000, 0, 0}} {
		got, err := FeeAtRate(test.amount, test.bps)
		if err != nil || got != test.want {
			t.Fatalf("fee %d at %d: %d %v", test.amount, test.bps, got, err)
		}
	}
	terms := &FeeTerms{PolicyRevision: 1, BaseBPS: 25, CollectionBPS: 75}
	copy := terms.Clone()
	terms.BaseBPS = 90
	if base, _ := copy.Base(100000); base != 250 {
		t.Fatal("recorded terms changed")
	}
	store := NewStore()
	journal, err := ActivateWithFee(store, "obligation", 100000, 0, time.Now(), "zero-fee")
	if err != nil || len(journal.Postings) != 2 {
		t.Fatal("zero fee broke activation", err)
	}
}

func TestMinFeeFloorEnforcement(t *testing.T) {
	terms := &FeeTerms{PolicyRevision: 1, BaseBPS: 50, CollectionBPS: 50, MinFeeKobo: 100000} // ₦1,000 floor
	// On ₦50,000 (5,000,000 kobo), 50 BPS = 25,000 kobo (₦250). Floor elevates it to 100,000 kobo (₦1,000).
	base, err := terms.Base(5000000)
	if err != nil || base != 100000 {
		t.Fatalf("expected floor fee 100,000, got %d, err=%v", base, err)
	}
	// On ₦500,000 (50,000,000 kobo), 50 BPS = 250,000 kobo (₦2,500). Higher than floor so 250,000 is used.
	base, err = terms.Base(50000000)
	if err != nil || base != 250000 {
		t.Fatalf("expected 250,000, got %d, err=%v", base, err)
	}
	// On very small principal less than floor (e.g. 50,000 kobo), fee capped at principal
	base, err = terms.Base(50000)
	if err != nil || base != 50000 {
		t.Fatalf("expected capped at principal 50,000, got %d, err=%v", base, err)
	}
}

// TestFeeRoundingAlwaysFavoursTheSupplier pins the rounding direction README
// section 7.3 states. Rounding is a contractual term: the supplier is charged
// the whole kobo below the exact fee, never the one above it, and the error is
// never more than a single kobo.
func TestFeeRoundingAlwaysFavoursTheSupplier(t *testing.T) {
	for _, rate := range []int64{0, 1, 50, 137, 1000} {
		for _, amount := range []Money{0, 1, 99, 100, 999, 1000, 10000, 12345, 999999, 100000000, 1<<62 - 1} {
			fee, err := FeeAtRate(amount, rate)
			if err != nil {
				t.Fatalf("FeeAtRate(%d, %d) failed: %v", amount, rate, err)
			}
			if fee < 0 {
				t.Fatalf("FeeAtRate(%d, %d) returned a negative fee %d", amount, rate, fee)
			}
			// exact = amount * rate / 10000, evaluated without overflow by
			// splitting the multiplication the same way the implementation does.
			whole := (amount / 10000) * Money(rate)
			remainder := (amount % 10000) * Money(rate)
			floor := whole + remainder/10000
			if fee != floor {
				t.Fatalf("FeeAtRate(%d, %d) = %d, expected the floor %d", amount, rate, fee, floor)
			}
			if remainder%10000 != 0 && fee*10000 >= amount*Money(rate) && amount < 1<<40 {
				t.Fatalf("FeeAtRate(%d, %d) rounded up to %d", amount, rate, fee)
			}
			if fee > amount {
				t.Fatalf("FeeAtRate(%d, %d) charged more than the amount: %d", amount, rate, fee)
			}
		}
	}
	if _, err := FeeAtRate(-1, 50); err == nil {
		t.Fatal("a negative amount must be rejected")
	}
	if _, err := FeeAtRate(1000, 1001); err == nil {
		t.Fatal("a rate above 1000 basis points must be rejected")
	}
}
