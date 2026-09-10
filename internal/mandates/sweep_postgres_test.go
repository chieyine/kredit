package mandates

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"kredit/internal/db"

	"github.com/jackc/pgx/v5/pgxpool"
)

type reusableRemote struct {
	*MockProvider
	creates   int
	beforeGet func()
	replay    *Mandate
}

func (p *reusableRemote) Name() string { return "sandbox-mandate" }
func (p *reusableRemote) CreateAuthorizationSession(ctx context.Context, in AuthorizationInput) (Mandate, error) {
	p.creates++
	if p.replay != nil {
		return *p.replay, nil
	}
	m, err := p.MockProvider.CreateAuthorizationSession(ctx, in)
	m.Variable = true
	m.StartsAt = time.Now().Add(-time.Hour)
	m.EndsAt = time.Now().AddDate(1, 0, 0)
	return m, err
}
func (p *reusableRemote) GetMandate(ctx context.Context, id string) (Mandate, error) {
	if p.beforeGet != nil {
		p.beforeGet()
	}
	return p.MockProvider.GetMandate(ctx, id)
}

func (p *reusableRemote) RestoreAuthorization(ctx context.Context, id string) (Mandate, error) {
	previous, err := p.MockProvider.GetMandate(ctx, id)
	if err != nil {
		return Mandate{}, err
	}
	return p.CreateAuthorizationSession(ctx, AuthorizationInput{UserID: previous.UserID, BusinessID: previous.BusinessID, SupplierOrganizationID: previous.SupplierOrganizationID, AmountCeiling: previous.AmountCeiling})
}
func TestPostgresVariableMandateReuseAndRevocationAreDurable(t *testing.T) {
	if os.Getenv("KREDIT_INTEGRATION") != "1" {
		t.Skip("integration database required")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	var user, business, org string
	if err = pool.QueryRow(ctx, `INSERT INTO app.users(normalized_email) VALUES($1) RETURNING id::text`, fmt.Sprintf("mandate-%d@example.test", time.Now().UnixNano())).Scan(&user); err != nil {
		t.Fatal(err)
	}
	if err = pool.QueryRow(ctx, `INSERT INTO app.businesses(owner_user_id,legal_name,business_type,business_address,industry) VALUES($1::uuid,'Buyer','limited_company','Test','Test') RETURNING id::text`, user).Scan(&business); err != nil {
		t.Fatal(err)
	}
	if err = pool.QueryRow(ctx, `INSERT INTO app.organizations(legal_name,business_type,business_address,industry) VALUES('Supplier','limited_company','Test','Test') RETURNING id::text`).Scan(&org); err != nil {
		t.Fatal(err)
	}
	remote := &reusableRemote{MockProvider: NewMockProvider()}
	p := NewPostgresProviderWithRemote(pool, remote)
	input := AuthorizationInput{UserID: user, BusinessID: business, SupplierOrganizationID: org, AmountCeiling: 100000000, RequiredUntil: time.Now().Add(time.Hour), Purpose: "first"}
	first, err := p.CreateAuthorizationSession(ctx, input)
	if err != nil {
		t.Fatal(err)
	}
	input.Purpose = "second"
	input.AmountCeiling = 50000000
	reused, err := NewPostgresProviderWithRemote(pool, remote).CreateAuthorizationSession(ctx, input)
	if err != nil || first.ID != reused.ID || remote.creates != 1 {
		t.Fatalf("mandate not reused: %v %+v", err, reused)
	}
	// An older active lookup arriving after a cancellation cannot reactivate it.
	remote.beforeGet = func() {
		remote.beforeGet = nil
		if _, err := p.BlockMandate(ctx, first.ProviderID, Cancelled, "cancelled-event"); err != nil {
			t.Error(err)
		}
	}
	m, err := p.GetMandate(ctx, first.ProviderID)
	if err != nil || m.Status != Cancelled {
		t.Fatalf("stale lookup reactivated: %s %v", m.Status, err)
	}
	if _, err = p.BlockMandate(ctx, first.ProviderID, Paused, "older-pause-event"); err != nil {
		t.Fatal(err)
	}
	m, err = p.GetMandate(ctx, first.ProviderID)
	if err != nil || m.Status != Cancelled {
		t.Fatalf("out of order pause reactivated: %s %v", m.Status, err)
	}
	if m.SupplierOrganizationID != org || !m.Variable || m.EndsAt.IsZero() {
		t.Fatalf("terminal lookup lost mandate details: %+v", m)
	}
	if os.Getenv("APP_DATABASE_URL") == "" {
		t.Fatal("restricted application database required for restoration check")
	}
	app, err := db.OpenAsRole(ctx, os.Getenv("APP_DATABASE_URL"), "kredit_app")
	if err != nil {
		t.Fatal(err)
	}
	defer app.Close()
	appProvider := NewPostgresProviderWithRemote(app.Raw(), remote)
	restored, err := appProvider.RestoreAuthorization(ctx, first.ProviderID)
	if err != nil {
		t.Fatalf("restore under application role: %v", err)
	}
	if restored.ID == first.ID || restored.SupplierOrganizationID != org || restored.UserID != user || restored.BusinessID != business {
		t.Fatalf("restoration lost ownership or reused old mandate: %+v", restored)
	}
	// A provider-neutral read after restart must preserve all accepted details.
	reloaded, err := NewPostgresProvider(app.Raw(), remote.Name()).GetMandate(ctx, restored.ProviderID)
	if err != nil || reloaded.SupplierOrganizationID != org || !reloaded.Variable || !reloaded.EndsAt.Equal(restored.EndsAt.Truncate(time.Microsecond)) {
		t.Fatalf("restored mandate details were not durable: %+v %v", reloaded, err)
	}
	listed, err := appProvider.ReadForBuyer(ctx, user)
	if err != nil || len(listed) != 2 {
		t.Fatalf("standalone mandates unavailable under app role: %+v %v", listed, err)
	}
	other, err := appProvider.ReadForBuyer(ctx, "00000000-0000-7000-8000-000000000099")
	if err != nil || len(other) != 0 {
		t.Fatalf("another buyer saw mandates: %+v %v", other, err)
	}
	remote.replay = &first
	input.SupplierOrganizationID = "00000000-0000-7000-8000-000000000010"
	if _, err := p.CreateAuthorizationSession(ctx, input); err == nil {
		t.Fatal("provider reference replay reassigned a mandate to another supplier")
	}
	var savedOrg string
	if err := pool.QueryRow(ctx, `SELECT supplier_organization_id::text FROM app.payment_mandates WHERE id=$1::uuid`, first.ID).Scan(&savedOrg); err != nil || savedOrg != org {
		t.Fatalf("replay changed original supplier: %s %v", savedOrg, err)
	}
}
