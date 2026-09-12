package billing

import (
	"testing"
	"time"
)

func TestInvoicePeriodsUseLagosCalendar(t *testing.T) {
	for _, item := range []struct{ now, cycle, want string }{
		{"2026-08-31T22:59:59Z", "monthly", "2026-07-31T23:00:00Z"},
		{"2026-08-31T23:00:00Z", "monthly", "2026-08-31T23:00:00Z"},
		{"2026-09-06T22:59:59Z", "weekly", "2026-08-30T23:00:00Z"},
		{"2026-09-06T23:00:00Z", "weekly", "2026-09-06T23:00:00Z"},
	} {
		now, _ := time.Parse(time.RFC3339, item.now)
		got, err := PeriodEnd(now, item.cycle)
		if err != nil || got.Format(time.RFC3339) != item.want {
			t.Fatalf("%s %s: %s %v", item.now, item.cycle, got, err)
		}
	}
	if _, err := PeriodEnd(time.Now(), "daily"); err == nil {
		t.Fatal("unsupported billing cycle accepted")
	}
}
