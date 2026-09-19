package db

import (
	"context"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"os"
	"testing"
)

func TestScopedDatabaseClearsPooledIdentityAndReleasesTransactions(t *testing.T) {
	if os.Getenv("KREDIT_INTEGRATION") != "1" {
		t.Skip("isolated integration database required")
	}
	cfg, err := pgxpool.ParseConfig(os.Getenv("DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	cfg.MaxConns = 1
	pool, err := pgxpool.NewWithConfig(t.Context(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	store := &ScopedDatabase{Pool: pool}
	ctx := WithTenantContext(t.Context(), "11111111-1111-4111-8111-111111111111", "22222222-2222-4222-8222-222222222222")
	sql := `SELECT COALESCE(current_setting('app.current_user_id',true),''),COALESCE(current_setting('app.current_organization_id',true),'')`
	check := func(row pgx.Row, wantUser, wantOrg string) {
		t.Helper()
		var u, o string
		if err := row.Scan(&u, &o); err != nil || u != wantUser || o != wantOrg {
			t.Fatalf("identity=(%s,%s): %v", u, o, err)
		}
	}
	identity, _ := TenantFromContext(ctx)
	check(store.QueryRow(ctx, sql), identity.UserID, identity.OrganizationID)
	check(store.QueryRow(t.Context(), sql), "", "")
	rows, err := store.Query(ctx, sql)
	if err != nil {
		t.Fatal(err)
	}
	if !rows.Next() {
		t.Fatal("scoped query returned no identity")
	}
	var user, org string
	if err = rows.Scan(&user, &org); err != nil || user != identity.UserID || org != identity.OrganizationID {
		t.Fatalf("query identity: %s %s %v", user, org, err)
	}
	if rows.Next() {
		t.Fatal("extra result")
	}
	if err = rows.Err(); err != nil {
		t.Fatal(err)
	}
	check(store.QueryRow(t.Context(), sql), "", "")
	if _, err = store.Exec(ctx, `SELECT 1`); err != nil {
		t.Fatal(err)
	}
	check(store.QueryRow(t.Context(), sql), "", "")
	tx, err := store.BeginTx(ctx, pgx.TxOptions{AccessMode: pgx.ReadOnly})
	if err != nil {
		t.Fatal(err)
	}
	check(tx.QueryRow(ctx, sql), identity.UserID, identity.OrganizationID)
	if err = tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	check(store.QueryRow(t.Context(), sql), "", "")
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	if err = store.QueryRow(cancelled, sql).Scan(&user, &org); err == nil {
		t.Fatal("cancelled query succeeded")
	}
	check(store.QueryRow(t.Context(), sql), "", "")
	if pool.Stat().AcquiredConns() != 0 {
		t.Fatal("transaction leaked a pooled connection")
	}
}
