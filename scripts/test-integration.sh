#!/usr/bin/env bash
set -euo pipefail

export GOCACHE="${GOCACHE:-$PWD/.tmp/go-cache}"
mkdir -p "$GOCACHE"

if [[ -z "${DATABASE_URL:-}" ]]; then
	printf '%s\n' 'DATABASE_URL is required for integration tests.' >&2
	exit 1
fi

# Preserve the application connection before replacing DATABASE_URL with the
# fixture owner. Check every required connection before seed or test writes.
if [[ -z "${APP_DATABASE_URL:-}" && -n "${DATABASE_DIRECT_URL:-}" && "$DATABASE_URL" != "$DATABASE_DIRECT_URL" ]]; then
	export APP_DATABASE_URL="$DATABASE_URL"
fi
: "${APP_DATABASE_URL:?APP_DATABASE_URL must point to the restricted application login for the isolated test database}"
: "${RIVER_DATABASE_URL:?RIVER_DATABASE_URL must point to the restricted worker login for the isolated test database}"

# Integration setup and cleanup deliberately use the migration/test owner.
# Individual authorization tests enter the restricted runtime roles themselves.
export DATABASE_URL="${TEST_DATABASE_URL:-${DATABASE_DIRECT_URL:-$DATABASE_URL}}"

# Several cross-domain assertions intentionally use the deterministic
# acceptance fixtures. Seeding here also verifies that a fresh migrated
# database can be prepared twice without duplicate records.
go run ./cmd/seed
go run ./cmd/seed

# Database packages share the same acceptance fixtures. Serial package
# execution prevents one package's cleanup from racing another package while
# still exercising every PostgreSQL adapter, not a hand-maintained subset.
# Include integration-tagged tests inside internal packages as well as the
# cross-domain tests. Omitting the tag here silently drops auth/audit regressions.
KREDIT_INTEGRATION=1 go test -p 1 -tags=integration ./...
