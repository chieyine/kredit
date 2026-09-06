package observability

import (
	"context"
	"testing"
)

func TestPhase5FinancialMetricsRejectMissingDatabase(t *testing.T) {
	if value, err := DurableFinancialMetrics(context.Background(), nil); err == nil || value != "" {
		t.Fatalf("missing database emitted healthy metrics: %q, %v", value, err)
	}
}
