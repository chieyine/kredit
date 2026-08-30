package jobs

import "testing"

func TestMaintenanceArgsKind(t *testing.T) {
	if got := (MaintenanceArgs{}).Kind(); got != KindMaintenance {
		t.Fatalf("kind = %q, want %q", got, KindMaintenance)
	}
}

func TestMaintenanceOperationsIncludeScheduleEvaluation(t *testing.T) {
	if OpEvaluateSchedules == "" {
		t.Fatal("schedule evaluation operation must be configured")
	}
}

func TestJobClassesUseDedicatedQueuesAndRetryBudgets(t *testing.T) {
	if got := (FinancialArgs{}).InsertOpts().Queue; got != QueueFinancial {
		t.Fatalf("financial queue = %q", got)
	}
	if got := (ProviderWebhookArgs{}).InsertOpts().MaxAttempts; got != 10 {
		t.Fatalf("provider max attempts = %d", got)
	}
	if got := (DocumentArgs{}).InsertOpts().Queue; got != QueueDocuments {
		t.Fatalf("document queue = %q", got)
	}
}
