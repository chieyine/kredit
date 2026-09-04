#!/usr/bin/env bash
set -euo pipefail

export GOCACHE="${GOCACHE:-$PWD/.tmp/go-cache}"
export CI="${CI:-true}"
mkdir -p "$GOCACHE"

bash scripts/api-lint.sh
bash scripts/readme-conformance.sh
bash scripts/implementation-plan-conformance-test.sh
bash scripts/load-env-test.sh
pnpm run audit
go test ./...
go test -race ./internal/collections ./internal/credit ./internal/payments ./internal/reports
# lint.sh runs gofmt, go vet and golangci-lint, and fails closed when the linter
# is missing. Calling it here is what makes .golangci.yml enforceable.
bash scripts/lint.sh
bash scripts/security.sh

if [[ ! -d web/node_modules ]]; then
	printf '%s\n' 'Frontend dependencies are not installed.' >&2
	exit 1
fi
pnpm --dir web check
pnpm --dir web build
pnpm --dir web test

if [[ -n "${DATABASE_URL:-}" ]]; then
	bash scripts/test-integration.sh
	bash scripts/data-inventory-check.sh
fi
