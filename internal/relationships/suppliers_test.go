package relationships

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestBuyerSupplierDirectoryIncludesUnactivatedLimitsAndHistoricalConsent(t *testing.T) {
	if os.Getenv("KREDIT_INTEGRATION") != "1" || os.Getenv("DATABASE_URL") == "" {
		t.Skip("isolated integration database required")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	buyer, other, supplier, line := uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString()
	for _, id := range []string{buyer, other} {
		if _, err = pool.Exec(ctx, `INSERT INTO app.users(id,normalized_email) VALUES($1::uuid,$2)`, id, id+"@directory.test"); err != nil {
			t.Fatal(err)
		}
	}
	if _, err = pool.Exec(ctx, `INSERT INTO app.organizations(id,legal_name,business_type,business_address,industry) VALUES($1::uuid,'Directory supplier','limited_company','Test address','retail')`, supplier); err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, `INSERT INTO app.trade_lines(id,supplier_organization_id,buyer_user_id,buyer_business_id,approved_limit_kobo,available_limit_kobo,cadence,default_grace_hours,start_at,end_at,state,terms_version) VALUES($1::uuid,$2::uuid,$3::uuid,$4::uuid,1000,1000,'friday',24,now(),now()+interval '30 days','PENDING_MANDATE','fixture-v1')`, line, supplier, buyer, uuid.NewString()); err != nil {
		t.Fatal(err)
	}
	cfg, err := pgxpool.ParseConfig(os.Getenv("DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	cfg.MaxConns = 1
	cfg.AfterConnect = func(ctx context.Context, c *pgx.Conn) error { _, err := c.Exec(ctx, `SET ROLE kredit_app`); return err }
	runtime, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer runtime.Close()
	store := NewPostgresStore(runtime)
	items, err := store.Suppliers(ctx, buyer)
	if err != nil || len(items) != 1 || items[0].ID != supplier || items[0].LegalName != "Directory supplier" {
		t.Fatalf("buyer directory: %+v %v", items, err)
	}
	items, err = store.Suppliers(ctx, other)
	if err != nil || len(items) != 0 {
		t.Fatalf("foreign supplier leaked: %+v %v", items, err)
	}
	if _, err = store.Record(ctx, buyer, supplier, "payment_reminders", "v1", strings.Repeat("a", 64), true); err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, `DELETE FROM app.trade_lines WHERE id=$1::uuid`, line); err != nil {
		t.Fatal(err)
	}
	if _, err = store.Record(ctx, buyer, supplier, "payment_reminders", "v1", strings.Repeat("b", 64), false); err != nil {
		t.Fatalf("former relationship could not withdraw consent: %v", err)
	}
	items, err = store.Suppliers(ctx, buyer)
	if err != nil || len(items) != 1 {
		t.Fatalf("historical consent seller disappeared: %+v %v", items, err)
	}
	if allowed, err := store.AllowsReminders(ctx, buyer, supplier); err != nil || allowed {
		t.Fatalf("withdrawal ineffective: %v %v", allowed, err)
	}
	// Immutable consent evidence deliberately remains in the isolated audit DB.
}
