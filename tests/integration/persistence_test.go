//go:build integration

package integration

import (
	"context"
	"os"
	"testing"

	"kredit/internal/db"
	"kredit/internal/ledger"
)

func TestMigratedDatabasePersistenceContractAndLedger(t *testing.T) {
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		t.Fatal("DATABASE_URL is required for integration tests")
	}
	database, err := db.Open(context.Background(), url)
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	if err := database.CheckSchema(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := database.CheckPersistenceContract(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, err := ledger.Reconcile(context.Background(), database.Raw()); err != nil {
		t.Fatal(err)
	}
}
