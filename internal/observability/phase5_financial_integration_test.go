//go:build integration

package observability

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	"kredit/internal/db"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPhase5GlobalMetricsPreserveTenantIsolation(t *testing.T) {
	if os.Getenv("KREDIT_PHASE5_INTEGRATION") != "1" {
		t.Skip("run the permanent Phase 5 isolated-database gate")
	}
	ctx := context.Background()
	adminURL, appURL := os.Getenv("DATABASE_URL"), os.Getenv("PHASE5_APP_DATABASE_URL")
	if adminURL == "" || appURL == "" {
		t.Fatal("Phase 5 requires separate fixture-admin and restricted app URLs")
	}
	admin, err := pgxpool.New(ctx, adminURL)
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close()
	app, err := db.OpenAsRole(ctx, appURL, "kredit_app")
	if err != nil {
		t.Fatal(err)
	}
	defer app.Close()
	var expected, visible int64
	if err := admin.QueryRow(ctx, `SELECT count(*) FROM app.obligations WHERE lifecycle_status='ACTIVE' AND outstanding_kobo>0`).Scan(&expected); err != nil || expected == 0 {
		t.Fatalf("nonempty seeded financial portfolio is required: %d %v", expected, err)
	}
	if err := app.Raw().QueryRow(ctx, `SELECT count(*) FROM app.obligations`).Scan(&visible); err != nil || visible != 0 {
		t.Fatalf("unscoped app can see financial rows: %d %v", visible, err)
	}
	metrics, err := DurableFinancialMetrics(ctx, app.Raw())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(metrics, fmt.Sprintf("kredit_active_obligations %d\n", expected)) {
		t.Fatalf("global monitoring silently lost tenant rows: expected=%d", expected)
	}
	for _, name := range financialMetricNames {
		if strings.Count(metrics, "# TYPE kredit_"+name+" gauge\n") != 1 {
			t.Fatalf("missing/duplicate gauge %s", name)
		}
	}
	var publicExecute bool
	if err := admin.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(COALESCE(p.proacl,acldefault('f',p.proowner))) a WHERE p.oid='app.phase5_financial_metrics()'::regprocedure AND a.grantee=0 AND a.privilege_type='EXECUTE')`).Scan(&publicExecute); err != nil || publicExecute {
		t.Fatalf("global metrics function must not be executable by PUBLIC: %v %v", publicExecute, err)
	}
}
