package platformsettings

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Explicitly opt in: these fixtures belong to a dedicated disposable database.
func TestOwnerLifecycleDatabaseInvariant(t *testing.T) {
	if os.Getenv("KREDIT_OWNER_GUARD_TEST") != "1" {
		t.Skip("dedicated owner lifecycle database required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	var name string
	var port int
	var owners int
	if err = pool.QueryRow(ctx, `SELECT current_database(),inet_server_port(),(SELECT count(*) FROM app.platform_role_assignments WHERE role='platform_owner')`).Scan(&name, &port, &owners); err != nil {
		t.Fatal(err)
	}
	if name != "kredit_owner_lifecycle_test" || port != 55432 || owners != 0 {
		t.Fatal("refusing fixtures outside empty disposable owner-test database")
	}
	first, second := uuid.NewString(), uuid.NewString()
	for _, id := range []string{first, second} {
		if _, err = pool.Exec(ctx, `INSERT INTO app.users(id,normalized_email,status) VALUES($1::uuid,$2,'active')`, id, id+"@example.test"); err != nil {
			t.Fatal(err)
		}
	}
	grant := func(id string) {
		t.Helper()
		if _, err := pool.Exec(ctx, `INSERT INTO app.platform_role_assignments(user_id,role,granted_by,reason) VALUES($1::uuid,'platform_owner',$1::uuid,'synthetic lifecycle regression')`, id); err != nil {
			t.Fatal(err)
		}
	}
	grant(first)
	for _, tc := range []struct{ name, sql string }{
		{"suspension", `UPDATE app.users SET status='suspended' WHERE id=$1::uuid`},
		{"closure", `UPDATE app.users SET status='closed' WHERE id=$1::uuid`},
		{"lockout", `UPDATE app.users SET status='locked' WHERE id=$1::uuid`},
		{"immediate expiry", `UPDATE app.platform_role_assignments SET expires_at=now()-interval '1 minute' WHERE user_id=$1::uuid AND role='platform_owner'`},
		{"future expiry", `UPDATE app.platform_role_assignments SET expires_at=now()+interval '1 day' WHERE user_id=$1::uuid AND role='platform_owner'`},
		{"revocation", `UPDATE app.platform_role_assignments SET revoked_at=now() WHERE user_id=$1::uuid AND role='platform_owner'`},
		{"demotion", `UPDATE app.platform_role_assignments SET role='support_agent' WHERE user_id=$1::uuid AND role='platform_owner'`},
		{"deletion", `DELETE FROM app.platform_role_assignments WHERE user_id=$1::uuid AND role='platform_owner'`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tx, err := pool.Begin(ctx)
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = tx.Rollback(ctx) }()
			if _, err = tx.Exec(ctx, tc.sql, first); err == nil {
				t.Fatal("last effective owner could be removed")
			}
		})
	}
	grant(second)
	t.Run("concurrent removals", func(t *testing.T) {
		tx1, err := pool.Begin(ctx)
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = tx1.Rollback(ctx) }()
		if _, err = tx1.Exec(ctx, `UPDATE app.platform_role_assignments SET revoked_at=now() WHERE user_id=$1::uuid AND role='platform_owner'`, first); err != nil {
			t.Fatal(err)
		}
		tx2, err := pool.Begin(ctx)
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = tx2.Rollback(ctx) }()
		result := make(chan error, 1)
		go func() {
			_, err := tx2.Exec(ctx, `UPDATE app.platform_role_assignments SET revoked_at=now() WHERE user_id=$1::uuid AND role='platform_owner'`, second)
			result <- err
		}()
		select {
		case err := <-result:
			if err == nil {
				t.Fatal("competing transaction removed the other owner before the first committed")
			}
			t.Fatalf("unexpected early error: %v", err)
		case <-time.After(500 * time.Millisecond):
		}
		if err = tx1.Commit(ctx); err != nil {
			t.Fatal(err)
		}
		if err = <-result; err == nil {
			t.Fatal("competing removal succeeded after the first commit")
		}
		if err = pool.QueryRow(ctx, `SELECT count(*) FROM app.platform_role_assignments r JOIN app.users u ON u.id=r.user_id WHERE role='platform_owner' AND revoked_at IS NULL AND expires_at IS NULL AND status='active'`).Scan(&owners); err != nil || owners != 1 {
			t.Fatalf("owner invariant: %d, %v", owners, err)
		}
	})
}
