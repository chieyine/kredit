package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"kredit/internal/db"
	"kredit/internal/ledger"
)

func main() {
	databaseURL, err := databaseURLFromEnv(os.Getenv)
	if err != nil {
		fail(err)
	}
	database, err := db.OpenAsRole(context.Background(), databaseURL, "kredit_worker")
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

func databaseURLFromEnv(getenv func(string) string) (string, error) {
	databaseURL := strings.TrimSpace(getenv("RIVER_DATABASE_URL"))
	if databaseURL == "" {
		return "", fmt.Errorf("RIVER_DATABASE_URL is required for reconciliation")
	}
	return databaseURL, nil
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
