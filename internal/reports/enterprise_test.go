package reports

import (
	"context"
	"testing"
)

func TestEnterpriseReportAndCSV(t *testing.T) {
	ctx := context.Background()
	store := NewStore(Source{})

	orgID := "org-distributor-500"

	report, err := store.EnterpriseReport(ctx, orgID)
	if err != nil {
		t.Fatalf("enterprise report: %v", err)
	}

	if report.OrganizationID != orgID {
		t.Fatalf("expected org %s, got %s", orgID, report.OrganizationID)
	}
	if len(report.IntegrityHash) != 64 {
		t.Fatalf("expected 64-char hex integrity hash, got %s", report.IntegrityHash)
	}
	if len(report.BranchExposures) == 0 {
		t.Fatal("expected at least 1 branch exposure entry")
	}

	csvData, checksum, err := store.ExportEnterpriseCSV(ctx, orgID)
	if err != nil {
		t.Fatalf("export enterprise csv: %v", err)
	}
	if len(csvData) == 0 || len(checksum) != 64 {
		t.Fatalf("invalid csv export (len %d, checksum %s)", len(csvData), checksum)
	}
}
