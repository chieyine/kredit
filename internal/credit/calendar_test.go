package credit

import (
	"testing"
	"time"
)

func TestCollectionInstantIsLagosAndIndependentOfProcessTimezone(t *testing.T) {
	original := time.Local
	t.Cleanup(func() { time.Local = original })
	for _, zone := range []string{"Africa/Lagos", "UTC", "America/New_York", "Asia/Dubai"} {
		location, err := time.LoadLocation(zone)
		if err != nil {
			t.Fatal(err)
		}
		time.Local = location
		instant, err := CollectionInstant("2026-09-18", 24)
		if err != nil {
			t.Fatal(err)
		}
		if got := instant.Format(time.RFC3339); got != "2026-09-19T22:59:00Z" {
			t.Fatalf("%s produced %s", zone, got)
		}
	}
}
func TestCollectionInstantRejectsInvalidTerms(t *testing.T) {
	for _, date := range []string{"", "2026-02-31", "2026-9-1", "September 18", "2026-09-18T00:00:00Z"} {
		if _, err := CollectionInstant(date, 24); err == nil {
			t.Errorf("accepted invalid date %q", date)
		}
	}
	for _, grace := range []int{-1, 721} {
		if _, err := CollectionInstant("2026-09-18", grace); err == nil {
			t.Errorf("accepted grace %d", grace)
		}
	}
}
func TestCollectionInstantMonthAndLeapBoundaries(t *testing.T) {
	for _, row := range []struct {
		date  string
		grace int
		want  string
	}{
		{"2026-12-31", 24, "2027-01-01T22:59:00Z"},
		{"2028-02-29", 24, "2028-03-01T22:59:00Z"},
		{"2026-09-18", 0, "2026-09-18T22:59:00Z"},
	} {
		got, err := CollectionInstant(row.date, row.grace)
		if err != nil || got.Format(time.RFC3339) != row.want {
			t.Errorf("%+v: got %s, %v", row, got, err)
		}
	}
}
