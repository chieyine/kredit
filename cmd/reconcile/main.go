package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"kredit/internal/db"
	"kredit/internal/ledger"
)

func main() {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		fail(fmt.Errorf("DATABASE_URL is required for reconciliation"))
	}
	database, err := db.Open(context.Background(), databaseURL)
	if err != nil {
		fail(err)
	}
	defer database.Close()
	if err := database.CheckSchema(context.Background()); err != nil {
		fail(err)
	}
	if err := database.CheckPersistenceContract(context.Background()); err != nil {
		fail(err)
	}
	report, err := ledger.Reconcile(context.Background(), database.Raw())
	if err != nil {
		fail(err)
	}
	encoded, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		fail(err)
	}
	fmt.Println(string(encoded))
	if len(report.UnbalancedIDs) > 0 || report.DebitKobo != report.CreditKobo {
		os.Exit(2)
	}
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
