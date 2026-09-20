package reports

import (
	"encoding/csv"
	"math"
	"strings"
	"testing"
	"time"

	"kredit/internal/ledger"
	"kredit/internal/schedules"
)

func TestAuditEnterpriseAgeingUsesEachUnpaidInstalment(t *testing.T) {
	now := time.Date(2026, 9, 20, 12, 0, 0, 0, time.UTC)
	items := []schedules.Item{}
	for _, days := range []int{-2, 30, 31, 60, 61, 90, 91} {
		items = append(items, schedules.Item{PrincipalDueKobo: 100, AllocatedKobo: 25, CollectionAt: now.Add(-time.Duration(days) * 24 * time.Hour)})
	}
	buckets, err := enterpriseAgeing(525, items, 0, now)
	if err != nil {
		t.Fatal(err)
	}
	for key, want := range map[string]ledger.Money{"0-30": 150, "31-60": 150, "61-90": 150, "90+": 75} {
		if buckets[key] != want {
			t.Fatalf("bucket %s=%d, want %d", key, buckets[key], want)
		}
	}
	items[0].AllocatedKobo = 101
	if _, err = enterpriseAgeing(525, items, 0, now); err == nil {
		t.Fatal("over-allocated schedule accepted")
	}
	if _, err = enterpriseAgeing(1, nil, 0, now); err == nil {
		t.Fatal("unexplained balance accepted")
	}
	if _, err = enterpriseAgeing(1, []schedules.Item{{PrincipalDueKobo: 1, State: schedules.ItemPaid}}, 0, now); err == nil {
		t.Fatal("paid item with balance accepted")
	}
}

func TestAuditPortfolioRatiosDoNotOverflow(t *testing.T) {
	for _, test := range []struct {
		n, d ledger.Money
		want int64
	}{{math.MaxInt64, math.MaxInt64, 10000}, {math.MaxInt64 / 2, math.MaxInt64, 4999}, {0, 0, 0}} {
		got, err := portfolioBPS(test.n, test.d)
		if err != nil || got != test.want {
			t.Fatalf("ratio %d/%d = %d %v", test.n, test.d, got, err)
		}
	}
	for _, test := range [][2]ledger.Money{{-1, 1}, {2, 1}, {1, 0}} {
		if _, err := portfolioBPS(test[0], test[1]); err == nil {
			t.Fatal("invalid ratio accepted")
		}
	}
}

func TestAuditEnterpriseHashCSVAndLateCollection(t *testing.T) {
	at := time.Date(2026, 9, 20, 12, 0, 0, 0, time.UTC)
	receivables := Receivables{GeneratedAt: at, Summary: Summary{PrincipalKobo: 200, OutstandingKobo: 100}, Rows: []ObligationRow{
		{ObligationID: "late", BuyerBusinessID: "buyer", PrincipalKobo: 100, VoluntaryPaidKobo: 100, LatePayment: true},
		{ObligationID: "unpaid", BuyerBusinessID: "buyer", PrincipalKobo: 100, OutstandingKobo: 100},
	}}
	branches := []BranchExposure{newBranchExposure("branch", "=1+1", "+territory")}
	assignments := map[string]string{"buyer": "branch"}
	buckets := map[string]map[string]ledger.Money{"unpaid": {"0-30": 100}}
	r, err := buildEnterpriseReport("org", receivables, branches, assignments, buckets)
	if err != nil {
		t.Fatal(err)
	}
	if r.PortfolioHealth.OnTimeCollectionBPS != 0 || r.PortfolioHealth.TopConcentrationBPS != 10000 {
		t.Fatalf("incorrect health: %+v", r)
	}
	if r.BranchExposures[0].BranchID != "branch" || r.BranchExposures[0].CustomerCount != 1 || r.BranchExposures[0].AgeingBuckets["0-30"] != 100 {
		t.Fatal("branch totals not preserved")
	}
	raw, _, err := enterpriseCSV(r)
	if err != nil {
		t.Fatal(err)
	}
	reader := csv.NewReader(strings.NewReader(string(raw)))
	reader.FieldsPerRecord = -1
	rows, err := reader.ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	if rows[3][1] != "'=1+1" || rows[3][2] != "'+territory" {
		t.Fatalf("untrusted spreadsheet cells: %+v", rows)
	}
	branches[0].Territory = "changed"
	changed, err := buildEnterpriseReport("org", receivables, branches, assignments, buckets)
	if err != nil || changed.IntegrityHash == r.IntegrityHash {
		t.Fatal("branch evidence omitted from hash")
	}
	branches[0].Territory = "+territory"
	receivables.GeneratedAt = at.Add(time.Hour)
	repeat, err := buildEnterpriseReport("org", receivables, branches, assignments, buckets)
	if err != nil || repeat.IntegrityHash != r.IntegrityHash {
		t.Fatal("generation time changed evidence hash")
	}
	assignments["buyer"] = "unknown"
	if _, err = buildEnterpriseReport("org", receivables, branches, assignments, buckets); err == nil {
		t.Fatal("missing branch silently moved to headquarters")
	}
	empty, err := buildEnterpriseReport("org", Receivables{GeneratedAt: at}, nil, nil, nil)
	if err != nil || empty.PortfolioHealth.HealthRating != "NO_DATA" {
		t.Fatal("empty portfolio classified as excellent")
	}
}
