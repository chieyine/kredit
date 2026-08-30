#!/usr/bin/env bash
set -euo pipefail

export GOCACHE="${GOCACHE:-$PWD/.tmp/go-cache}"
mkdir -p "$GOCACHE"

go test ./...
if [[ -n "${DATABASE_URL:-}" ]]; then
	KREDIT_INTEGRATION=1 go test ./internal/schedules ./internal/documents ./internal/support ./internal/paymentclaims ./internal/payments ./internal/tradelines ./internal/credit
	KREDIT_INTEGRATION=1 go test -tags=integration ./tests/integration
else
	printf '%s\n' 'DATABASE_URL is required for integration tests.' >&2
	exit 1
fi
