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
format_output="$(gofmt -l .)"
if [[ -n "$format_output" ]]; then
  printf '%s\n' "$format_output"
  printf '%s\n' 'Go files require formatting.' >&2
  exit 1
fi
go test ./...
go test -race ./internal/collections ./internal/credit ./internal/payments ./internal/reports
go vet ./...
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
