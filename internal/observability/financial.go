package observability

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

var financialMetricNames = []string{
	"ledger_discrepancies", "balance_discrepancies", "schedule_discrepancies",
	"collection_payment_discrepancies", "settlement_discrepancies",
	"collection_unknown_states", "notification_dead_letters", "outbox_delivery_failures",
	"provider_webhook_oldest_unprocessed_seconds", "collection_oldest_unresolved_seconds",
	"river_pending_jobs", "river_discarded_jobs", "active_obligations", "negative_outstanding_balances",
}

// DurableFinancialMetrics returns platform-wide database aggregates, not process
// counters or tenant-filtered guesses. Missing/invalid instrumentation is an
// error, never a fabricated zero. The SQL function exposes no customer records.
func DurableFinancialMetrics(ctx context.Context, pool *pgxpool.Pool) (string, error) {
	if pool == nil {
		return "", errors.New("financial monitoring database is unavailable")
	}
	rows, err := pool.Query(ctx, `SELECT metric,value FROM app.phase5_financial_metrics()`)
	if err != nil {
		return "", err
	}
	defer rows.Close()
	allowed := make(map[string]bool, len(financialMetricNames))
	for _, name := range financialMetricNames {
		allowed[name] = true
	}
	values := make(map[string]float64, len(allowed))
	for rows.Next() {
		var name string
		var value float64
		if err := rows.Scan(&name, &value); err != nil {
			return "", err
		}
		if _, duplicate := values[name]; !allowed[name] || duplicate || value < 0 || math.IsNaN(value) || math.IsInf(value, 0) {
			return "", errors.New("invalid financial monitoring result")
		}
		values[name] = value
	}
	if err := rows.Err(); err != nil {
		return "", err
	}
	if len(values) != len(allowed) {
		return "", errors.New("financial monitoring result is incomplete")
	}
	names := append([]string(nil), financialMetricNames...)
	sort.Strings(names)
	var out strings.Builder
	for _, name := range names {
		fmt.Fprintf(&out, "# TYPE kredit_%s gauge\nkredit_%s %g\n", name, name, values[name])
	}
	return out.String(), nil
}
