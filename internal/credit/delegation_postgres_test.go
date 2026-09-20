package credit

import (
	"context"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"kredit/internal/ledger"
	"kredit/internal/mandates"
	"kredit/internal/purchasing"
	"os"
	"testing"
	"time"
)

func TestStaffPurchasingPreservesActorLimitsAndReceipt(t *testing.T) {
	if os.Getenv("KREDIT_INTEGRATION") != "1" {
		t.Skip("isolated database required")
	}
	ctx := context.Background()
	admin, err := pgxpool.New(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close()
	buyer, staff, seller, org, profile := uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString()
	exec := func(q string, args ...any) {
		t.Helper()
		if _, e := admin.Exec(ctx, q, args...); e != nil {
			t.Fatal(e)
		}
	}
	for _, u := range []string{buyer, staff, seller} {
		exec(`INSERT INTO app.users(id,normalized_email) VALUES($1::uuid,$2)`, u, u+"@staff-credit.test")
	}
	exec(`INSERT INTO app.persons(id,user_id,full_name,status) VALUES($1::uuid,$2::uuid,'Staff Person','verified')`, uuid.NewString(), staff)
	exec(`INSERT INTO app.organizations(id,legal_name,business_type,business_address,industry) VALUES($1::uuid,'Supplier','limited_company','Lagos','food')`, org)
	exec(`INSERT INTO app.memberships(organization_id,user_id,role,status) VALUES($1::uuid,$2::uuid,'owner','active')`, org, seller)
	exec(`INSERT INTO app.businesses(id,owner_user_id,legal_name,business_type,business_address,industry,status) VALUES($1::uuid,$2::uuid,'Buyer','limited_company','Lagos','food','verified')`, profile, buyer)
	exec(`INSERT INTO app.memberships(organization_id,user_id,role,status) VALUES($1::uuid,$2::uuid,'sales','active')`, profile, staff)
	cfg, _ := pgxpool.ParseConfig(os.Getenv("DATABASE_URL"))
	cfg.ConnConfig.RuntimeParams["role"] = "kredit_app"
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	repo := NewPostgresStore(pool, NewStore(mandates.NewMockProvider(), ledger.NewPostgresStore(pool)))
	c, err := repo.Create(CreateInput{SupplierOrganizationID: org, SupplierLegalName: "Supplier", BuyerUserID: buyer, BuyerBusinessID: profile, BuyerLegalName: "Buyer", PrincipalKobo: 10000, GoodsDescription: "Stock", DueDate: time.Now().AddDate(0, 0, 7).Format("2006-01-02"), CollectionAt: time.Now().Add(8 * 24 * time.Hour), CreatedBy: seller})
	if err != nil {
		t.Fatal(err)
	}
	sent, err := repo.Send(c.ID, seller)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = repo.GetForBuyer(c.ID, staff); err == nil {
		t.Fatal("supplier role gave purchasing access")
	}
	grants := purchasing.Store{Pool: pool}
	g, err := grants.Save(ctx, buyer, profile, purchasing.Grant{UserID: staff, Actions: []string{"read", "review", "accept", "receive"}, CeilingKobo: 9999, ExpiresAt: time.Now().Add(time.Hour)})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = repo.Review(c.ID, staff); err != nil {
		t.Fatal(err)
	}
	if _, err = repo.Accept(c.ID, staff, sent.Agreement.ID, sent.Agreement.DocumentHash, "", "AAL2", true, true); err == nil {
		t.Fatal("staff exceeded acceptance ceiling")
	}
	if _, err = repo.AuthorizeMandate(ctx, c.ID, staff); err == nil {
		t.Fatal("staff took over owner's bank authorization")
	}
	mandated, err := repo.SetMandate(c.ID, buyer, mandates.Mandate{ID: uuid.NewString(), ProviderID: "staff-test-bank-" + uuid.NewString(), Provider: "fixture", UserID: buyer, BusinessID: profile, SupplierOrganizationID: org, AmountCeiling: 10000, Status: mandates.Active, AuthorizationURL: "https://provider.invalid/private-session"})
	if err != nil {
		t.Fatal(err)
	}
	g.CeilingKobo = 10000
	g, err = grants.Save(ctx, buyer, profile, g)
	if err != nil {
		t.Fatal(err)
	}
	accepted, err := repo.Accept(c.ID, staff, sent.Agreement.ID, sent.Agreement.DocumentHash, mandated.Mandate.ProviderID, "AAL2", true, true)
	if err != nil {
		t.Fatal(err)
	}
	if accepted.Acceptance.AcceptingUserID != staff || accepted.Acceptance.PersonID == staff || accepted.Request.BuyerUserID != buyer || accepted.Mandate.AuthorizationURL != "" {
		t.Fatal("actor attribution or private provider session leaked")
	}
	if _, err = repo.GetForSupplier(c.ID, org); err != nil {
		t.Fatal(err)
	}
	if _, err = repo.Release(c.ID, org, seller, "courier", "Recorded shipment"); err != nil {
		t.Fatal(err)
	}
	active, journal, err := repo.RecordReceipt(c.ID, staff, "confirmed", "")
	if err != nil {
		t.Fatal(err)
	}
	if active.Obligation == nil || journal == nil || active.Mandate.AuthorizationURL != "" || len(active.Receipts) != 1 || active.Receipts[0].BuyerUserID != staff {
		t.Fatalf("staff receipt attribution: %#v", active)
	}
	g.Actions = []string{}
	if _, err = grants.Save(ctx, buyer, profile, g); err != nil {
		t.Fatal(err)
	}
	if _, err = repo.GetForBuyer(c.ID, staff); err == nil {
		t.Fatal("revoked staff read cached purchase")
	}
	if _, err = repo.GetForBuyer(c.ID, buyer); err != nil {
		t.Fatal("revocation hid owner's purchase")
	}
}
