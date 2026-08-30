package onboarding

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPostgresOnboardingPersistsVersionsMasksSettlementAndReconcilesExpiry(t *testing.T) {
	if os.Getenv("KREDIT_INTEGRATION") != "1" || os.Getenv("DATABASE_URL") == "" {
		t.Skip("KREDIT_INTEGRATION=1 and DATABASE_URL are required")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	userID, orgID := uuid.NewString(), uuid.NewString()
	_, err = pool.Exec(ctx, `INSERT INTO app.users(id,normalized_email,status) VALUES($1,$2,'active')`, userID, "onboarding-"+userID+"@example.test")
	if err == nil {
		_, err = pool.Exec(ctx, `INSERT INTO app.organizations(id,legal_name,business_type,business_address,industry,status) VALUES($1,'Onboarding Test Ltd','limited_company','Lagos','testing','onboarding')`, orgID)
	}
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_, _ = pool.Exec(ctx, `DELETE FROM app.organizations WHERE id=$1`, orgID)
		_, _ = pool.Exec(ctx, `DELETE FROM app.users WHERE id=$1`, userID)
	}()
	store := NewPostgresStore(pool)
	p, err := store.Ensure(orgID, userID, true, true)
	if err != nil {
		t.Fatal(err)
	}
	p, _, err = store.UpdateRepresentative(orgID, userID, RepresentativeInput{ExpectedVersion: p.Version, Name: "Ada", Title: "Director"})
	if err != nil {
		t.Fatal(err)
	}
	p, _, err = store.UpdateSettlement(orgID, userID, SettlementInput{ExpectedVersion: p.Version, Provider: "provider", ProviderReference: "opaque-provider-reference", BankName: "Demo Bank", AccountName: "Onboarding Test Ltd", AccountLast4: "9876"})
	if err != nil {
		t.Fatal(err)
	}
	p, _, err = store.RecordSettlementDecision(orgID, userID, "verified", "")
	if err != nil {
		t.Fatal(err)
	}
	restarted := NewPostgresStore(pool)
	loaded, _, err := restarted.Get(orgID)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.SettlementAccountLast4 != "9876" || loaded.SettlementProviderReference != "opaque-provider-reference" || loaded.Version != p.Version {
		t.Fatalf("persistence mismatch: %#v", loaded)
	}
	var snapshot string
	err = pool.QueryRow(ctx, `SELECT snapshot::text FROM app.supplier_onboarding_revisions WHERE organization_id=$1 ORDER BY profile_version DESC LIMIT 1`, orgID).Scan(&snapshot)
	if err != nil {
		t.Fatal(err)
	}
	if containsText(snapshot, "1234567890") {
		t.Fatal("full account number leaked into onboarding history")
	}
	p, _, err = store.SubmitKYB(orgID, userID, "kyb-reference", p.Version)
	if err != nil {
		t.Fatal(err)
	}
	expires := time.Now().UTC().Add(-time.Minute)
	_, _, err = store.RecordKYBDecision(orgID, userID, "approved", "", expires)
	if err != nil {
		t.Fatal(err)
	}
	changed := restarted.Reconcile(time.Now().UTC())
	found := false
	for _, item := range changed {
		if item.OrganizationID == orgID && item.KYBState == "expired" && item.ReadinessState == "expired" {
			found = true
		}
	}
	if !found {
		t.Fatal("expected reconciliation to durably expire provider evidence")
	}
	var outboxCount int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM app.outbox_events WHERE aggregate_type='supplier_onboarding' AND aggregate_id=$1 AND event_type='SupplierOnboardingRequirementExpired'`, orgID).Scan(&outboxCount); err != nil || outboxCount != 1 {
		t.Fatalf("expected one atomic expiry outbox event, count=%d err=%v", outboxCount, err)
	}
}

func containsText(value, part string) bool {
	for i := 0; i+len(part) <= len(value); i++ {
		if value[i:i+len(part)] == part {
			return true
		}
	}
	return false
}
