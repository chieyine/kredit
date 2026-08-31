package main

import (
	"fmt"
	"os"

	"kredit/internal/config"
	"kredit/internal/readiness"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "production configuration rejected: %v\n", err)
		os.Exit(1)
	}
	report := readiness.Evaluate(cfg)
	if !report.Ready {
		fmt.Fprintf(os.Stderr, "production readiness rejected; missing gates: %v\n", report.Missing)
		os.Exit(1)
	}
	fmt.Printf("production configuration accepted for %s (%s); %d readiness gates passed\n", cfg.Environment, cfg.Version, len(report.Gates))
}
