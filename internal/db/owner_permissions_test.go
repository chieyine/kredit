package db

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestOwnerProtectionsSurviveRoleProvisioning(t *testing.T) {
	if os.Getenv("KREDIT_INTEGRATION") != "1" || os.Getenv("DATABASE_URL") == "" {
		t.Skip("integration database required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	pool, err := Open(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	for _, role := range []string{"kredit_app", "kredit_worker"} {
		for _, target := range []struct{ table, privilege string }{
			{"app.platform_owner_guard", "SELECT"}, {"app.platform_owner_guard", "UPDATE"}, {"app.platform_owner_guard", "INSERT"},
			{"app.platform_settings_history", "UPDATE"}, {"app.platform_settings_history", "DELETE"}, {"app.platform_settings_history", "TRUNCATE"},
			{"app.platform_role_assignments", "TRUNCATE"}, {"app.users", "TRUNCATE"},
		} {
			var allowed bool
			if err := pool.Raw().QueryRow(ctx, `SELECT has_table_privilege($1,$2,$3)`, role, target.table, target.privilege).Scan(&allowed); err != nil {
				t.Fatal(err)
			}
			if allowed {
				t.Errorf("%s must not have %s on %s", role, target.privilege, target.table)
			}
		}
	}
}
