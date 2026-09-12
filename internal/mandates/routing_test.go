package mandates

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type namedTestRemote struct {
	*reusableRemote
	name string
}

func (p *namedTestRemote) Name() string { return p.name }
func TestSavedMandateAccountRouting(t *testing.T) {
	dsn := os.Getenv("NATIVE_IDENTITY_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("requires disposable test database")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	var user, business, org string
	if err = pool.QueryRow(ctx, `INSERT INTO app.users(normalized_email) VALUES($1) RETURNING id::text`, fmt.Sprintf("routing-%d@example.test", time.Now().UnixNano())).Scan(&user); err != nil {
		t.Fatal(err)
	}
	if err = pool.QueryRow(ctx, `INSERT INTO app.businesses(owner_user_id,legal_name,business_type,business_address,industry) VALUES($1::uuid,'Route Buyer','limited_company','Test','Test') RETURNING id::text`, user).Scan(&business); err != nil {
		t.Fatal(err)
	}
	if err = pool.QueryRow(ctx, `INSERT INTO app.organizations(legal_name,business_type,business_address,industry) VALUES('Route Seller','limited_company','Test','Test') RETURNING id::text`).Scan(&org); err != nil {
		t.Fatal(err)
	}
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatal(err)
	}
	cfg.ConnConfig.RuntimeParams["role"] = "kredit_app"
	app, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer app.Close()
	oldRemote := &namedTestRemote{reusableRemote: &reusableRemote{MockProvider: NewMockProvider()}, name: "old-account"}
	activeRemote := &namedTestRemote{reusableRemote: &reusableRemote{MockProvider: NewMockProvider()}, name: "new-account"}
	old := NewPostgresProviderWithRemote(app, oldRemote)
	active := NewPostgresProviderWithRemote(app, activeRemote)
	first, err := old.CreateAuthorizationSession(ctx, AuthorizationInput{UserID: user, BusinessID: business, SupplierOrganizationID: org, AmountCeiling: 100000, RequiredUntil: time.Now().AddDate(0, 1, 0)})
	if err != nil {
		t.Fatal(err)
	}
	router, err := NewRouter(app, active, old)
	if err != nil {
		t.Fatal(err)
	}
	oldReads, newReads := 0, 0
	oldRemote.beforeGet = func() { oldReads++ }
	activeRemote.beforeGet = func() { newReads++ }
	result, err := router.GetMandate(ctx, first.ProviderID)
	if err != nil || result.Provider != old.Name() || oldReads != 1 || newReads != 0 {
		t.Fatal("saved mandate used active account", err)
	}
	if _, err = router.GetMandate(WithProvider(ctx, active.Name()), first.ProviderID); err == nil || newReads != 0 {
		t.Fatal("explicit wrong account contacted provider")
	}
	missing, _ := NewRouter(app, active)
	if _, err = missing.GetMandate(ctx, first.ProviderID); err == nil || newReads != 0 {
		t.Fatal("missing account fell back")
	}
}
