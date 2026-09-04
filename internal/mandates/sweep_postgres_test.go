package mandates

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type reusableRemote struct {
	*MockProvider
	creates   int
	beforeGet func()
}

func (p *reusableRemote) Name() string { return "sandbox-mandate" }
func (p *reusableRemote) CreateAuthorizationSession(ctx context.Context, in AuthorizationInput) (Mandate, error) {
	p.creates++
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
}
