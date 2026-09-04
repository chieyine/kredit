package observability

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

// DurableFinancialMetrics exposes current database facts as gauges. These are
// not process counters: resolving a case or replaying a job can decrease them.
func DurableFinancialMetrics(ctx context.Context, pool *pgxpool.Pool) (string, error) {
	var ledgerDrift, balanceDrift, scheduleDrift, collectionDrift, settlementDrift, unknown, deadLetters, failedOutbox int64
	var webhookAge float64
	err := pool.QueryRow(ctx, `SELECT
 (SELECT count(*) FROM app.financial_discrepancies WHERE kind='ledger'),
 (SELECT count(*) FROM app.financial_discrepancies WHERE kind='balance'),
 (SELECT count(*) FROM app.financial_discrepancies WHERE kind='schedule'),
 (SELECT count(*) FROM app.financial_discrepancies WHERE kind='collection_payment'),
 (SELECT count(*) FROM app.financial_discrepancies WHERE kind IN ('settlement','settlement_missing','provider_reversal','settlement_without_payment')),
 (SELECT count(*) FROM app.collection_attempts WHERE state='UNKNOWN'),
 (SELECT count(*) FROM app.notifications WHERE state='failed' AND delivery_attempts>=8),
 (SELECT count(*) FROM app.outbox_events WHERE state='failed' AND attempts>=8),
 COALESCE((SELECT GREATEST(0,extract(epoch FROM now()-min(received_at))) FROM app.provider_webhook_inbox WHERE state IN ('received','processing','failed')),0)`).Scan(&ledgerDrift, &balanceDrift, &scheduleDrift, &collectionDrift, &settlementDrift, &unknown, &deadLetters, &failedOutbox, &webhookAge)
	if err != nil {
		return "", err
	}
	var out strings.Builder
	for _, v := range []struct {
		name  string
		value int64
	}{{"ledger_discrepancies", ledgerDrift}, {"balance_discrepancies", balanceDrift}, {"schedule_discrepancies", scheduleDrift}, {"collection_payment_discrepancies", collectionDrift}, {"settlement_discrepancies", settlementDrift}, {"collection_unknown_states", unknown}, {"notification_dead_letters", deadLetters}, {"outbox_delivery_failures", failedOutbox}} {
		fmt.Fprintf(&out, "# TYPE kredit_%s gauge\nkredit_%s %d\n", v.name, v.name, v.value)
	}
	fmt.Fprintf(&out, "# TYPE kredit_provider_webhook_oldest_unprocessed_seconds gauge\nkredit_provider_webhook_oldest_unprocessed_seconds %g\n", webhookAge)
	return out.String(), nil
}
