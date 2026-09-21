package orders

import (
	"testing"
	"time"

	"kredit/internal/db"
	"kredit/internal/reports"

	"github.com/google/uuid"
)

func TestAuditEnterpriseUsesVisibleBranchTerritoryAndExactBalances(t *testing.T) {
	f := newAuditFixture(t)
	branch := uuid.NewString()
	exec := func(sql string, args ...any) {
		t.Helper()
		if _, err := f.admin.Exec(f.ctx, sql, args...); err != nil {
			t.Fatal(err)
		}
	}
	exec(`INSERT INTO app.business_branches(id,organization_id,name,territory,updated_by) VALUES($1::uuid,$2::uuid,'Synthetic branch','North East',$3::uuid)`, branch, f.org, f.owner)
	exec(`INSERT INTO app.partner_assignments(organization_id,buyer_business_id,branch_id,updated_by) VALUES($1::uuid,$2::uuid,$3::uuid,$4::uuid)`, f.org, f.profile, branch, f.owner)
	// A closed branch still owns its historic exposure until explicitly reassigned.
	exec(`UPDATE app.business_branches SET active=false,version=version+1 WHERE id=$1::uuid`, branch)
	now := time.Now().UTC()
	store := reports.NewPostgresStore(f.runtime.Raw(), reports.Source{Now: func() time.Time { return now }})
	report, err := store.EnterpriseReport(f.as(f.finance), f.org)
	if err != nil {
		t.Fatal(err)
	}
	if report.Summary.OutstandingKobo != 20000 {
		t.Fatalf("wrong total: %+v", report)
	}
	found := false
	for _, b := range report.BranchExposures {
		if b.BranchID == branch {
			found = true
			if b.Territory != "North East" || b.TotalOutstandingKobo != 20000 || b.AgeingBuckets["0-30"] != 20000 || b.CustomerCount != 1 {
				t.Fatalf("branch report lost facts: %+v", b)
			}
		} else if b.TotalOutstandingKobo != 0 {
			t.Fatal("exposure moved to incorrect branch")
		}
	}
	if !found {
		t.Fatal("existing inactive branch disappeared")
	}
	if _, err = store.EnterpriseReport(db.WithTenantContext(f.ctx, f.finance, f.buyerOrg), f.org); err == nil {
		t.Fatal("tenant override accepted")
	}
	if _, err = store.EnterpriseReport(f.as(f.other), f.org); err == nil {
		t.Fatal("nonmember read financial report")
	}
}
