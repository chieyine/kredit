package erp

import (
	"errors"
	"math"
	"testing"
	"time"

	"kredit/internal/ledger"
)

func TestAuditReconciliationRejectsDuplicateAndInvalidEvidence(t *testing.T) {
	at := time.Date(2026, 9, 20, 12, 0, 0, 0, time.UTC)
	start, end := at.Add(-time.Hour), at.Add(time.Hour)
	in := InternalRecord{Reference: "tx-1", AmountKobo: 100, Date: at}
	ex := ExternalRecord{Reference: "tx-1", AmountKobo: 100, Date: at}
	tests := []struct {
		name string
		ins  []InternalRecord
		exts []ExternalRecord
	}{
		{"duplicate internal", []InternalRecord{in, in}, []ExternalRecord{ex}},
		{"duplicate external", []InternalRecord{in}, []ExternalRecord{ex, ex}},
		{"blank reference", []InternalRecord{{Reference: " ", Date: at}}, nil},
		{"zero date", nil, []ExternalRecord{{Reference: "tx-1"}}},
		{"out of period", nil, []ExternalRecord{{Reference: "tx-1", Date: end.Add(time.Second)}}},
		{"internal total overflow", []InternalRecord{{Reference: "a", AmountKobo: math.MaxInt64, Date: at}, {Reference: "b", AmountKobo: 1, Date: at}}, nil},
		{"external total overflow", nil, []ExternalRecord{{Reference: "a", AmountKobo: math.MaxInt64, Date: at}, {Reference: "b", AmountKobo: 1, Date: at}}},
		{"difference overflow", []InternalRecord{{Reference: "a", AmountKobo: math.MaxInt64, Date: at}}, []ExternalRecord{{Reference: "a", AmountKobo: -1, Date: at}}},
		{"negation overflow", nil, []ExternalRecord{{Reference: "a", AmountKobo: math.MinInt64, Date: at}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := ReconcileExternal("org", start, end, tt.ins, tt.exts); !errors.Is(err, ErrInvalidRecords) {
				t.Fatalf("untrusted evidence accepted: %v", err)
			}
		})
	}
	for _, period := range [][2]time.Time{{{}, end}, {start, {}}, {end, start}} {
		if _, err := ReconcileExternal("org", period[0], period[1], nil, nil); !errors.Is(err, ErrInvalidRecords) {
			t.Fatal("invalid period accepted")
		}
	}
	if _, err := ReconcileExternal("org", start, end, nil, make([]ExternalRecord, MaxReconciliationRecords+1)); err == nil {
		t.Fatal("unbounded input accepted")
	}
}

func TestAuditReconciliationHashBindsEvidenceAndIsOrderIndependent(t *testing.T) {
	at := time.Date(2026, 9, 20, 12, 0, 0, 0, time.UTC)
	ins := []InternalRecord{{Reference: "b", AmountKobo: -40, Date: at}, {Reference: "a", AmountKobo: 100, Date: at}}
	exts := []ExternalRecord{{Reference: "a", AmountKobo: 100, Date: at}, {Reference: "b", AmountKobo: -40, Date: at}}
	run := func(in []InternalRecord, ex []ExternalRecord) ReconciliationResult {
		t.Helper()
		r, e := ReconcileExternal("org", at.Add(-time.Hour), at.Add(time.Hour), in, ex)
		if e != nil {
			t.Fatal(e)
		}
		return r
	}
	original := run(ins, exts)
	if !original.Balanced || original.TotalInternalKobo != ledger.Money(60) || original.MatchedRecordCount != 2 {
		t.Fatalf("signed movements lost: %+v", original)
	}
	if ins[0].Reference != "b" {
		t.Fatal("caller input mutated")
	}
	reordered := run([]InternalRecord{ins[1], ins[0]}, []ExternalRecord{exts[1], exts[0]})
	if original.ReportHash != reordered.ReportHash {
		t.Fatal("same evidence has unstable hash")
	}
	exts[0].Note = "different source evidence"
	if run(ins, exts).ReportHash == original.ReportHash {
		t.Fatal("source evidence not bound to hash")
	}
	exts[0].Note = ""
	ins[1].Reference = "c"
	exts[0].Reference = "c"
	if run(ins, exts).ReportHash == original.ReportHash {
		t.Fatal("identities not bound to hash")
	}
}
