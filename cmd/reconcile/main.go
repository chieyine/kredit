package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"kredit/internal/db"
	"kredit/internal/ledger"
)

// Exit codes: 0 balanced, 1 could not run, 2 the ledger does not balance.
func main() {
	balanced, err := run()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if !balanced {
		os.Exit(2)
	}
}

// A scheduled run must end even when the database stops answering, so the
// whole check is bounded; reconciliation of a healthy ledger takes seconds.
const reconcileTimeout = 10 * time.Minute

func run() (bool, error) {
	databaseURL, err := databaseURLFromEnv(os.Getenv)
	if err != nil {
		return false, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), reconcileTimeout)
	defer cancel()
	database, err := db.OpenAsRole(ctx, databaseURL, "kredit_worker")
	if err != nil {
		return false, err
	}
	defer database.Close()
	if err := database.CheckSchema(ctx); err != nil {
		return false, err
	}
	if err := database.CheckPersistenceContract(ctx); err != nil {
		return false, err
	}
	report, err := ledger.Reconcile(ctx, database.Raw())
	if err != nil {
		return false, err
	}
	encoded, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return false, err
	}
	fmt.Println(string(encoded))
	return len(report.UnbalancedIDs) == 0 && report.DebitKobo == report.CreditKobo, nil
}

func databaseURLFromEnv(getenv func(string) string) (string, error) {
	databaseURL := strings.TrimSpace(getenv("RIVER_DATABASE_URL"))
	if databaseURL == "" {
		return "", fmt.Errorf("RIVER_DATABASE_URL is required for reconciliation")
	}
	return databaseURL, nil
}
