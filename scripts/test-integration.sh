#!/usr/bin/env bash
set -euo pipefail

export GOCACHE="${GOCACHE:-$PWD/.tmp/go-cache}"
mkdir -p "$GOCACHE"

if [[ -z "${DATABASE_URL:-}" ]]; then
	printf '%s\n' 'DATABASE_URL is required for integration tests.' >&2
	exit 1
fi

# Several cross-domain assertions intentionally use the deterministic
# acceptance fixtures. Seeding here also verifies that a fresh migrated
# database can be prepared twice without duplicate records.
go run ./cmd/seed
go run ./cmd/seed

# Database packages share the same acceptance fixtures. Serial package
# execution prevents one package's cleanup from racing another package while
# still exercising every PostgreSQL adapter, not a hand-maintained subset.
KREDIT_INTEGRATION=1 go test -p 1 ./...
KREDIT_INTEGRATION=1 go test -p 1 -tags=integration ./tests/integration
