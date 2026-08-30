package ledger

import (
	"context"
	"testing"
)

func TestReconcileRequiresDatabase(t *testing.T) {
	if _, err := Reconcile(context.Background(), nil); err == nil {
		t.Fatal("expected reconciliation to require a database")
	}
}
