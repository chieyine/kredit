#!/usr/bin/env bash
set -euo pipefail

# Release certification is intentionally fail-closed.  A green unit suite is
# not a production-v1 certificate when domain persistence, browser checks, or
# external approvals are missing.

export GOCACHE="${GOCACHE:-$PWD/.tmp/go-cache}"
export CI="${CI:-true}"
mkdir -p "$GOCACHE"

failures=0
require_command() {
	local command_name="$1"
	if ! command -v "$command_name" >/dev/null 2>&1; then
		printf 'MISSING TOOL: %s\n' "$command_name" >&2
		failures=$((failures + 1))
	fi
}

run_gate() {
	local label="$1"
	shift
	printf 'GATE: %s\n' "$label"
	if ! "$@"; then
		printf 'FAILED: %s\n' "$label" >&2
		failures=$((failures + 1))
	fi
}

require_command go
require_command pnpm
require_command docker

run_gate 'Go unit tests' go test ./...
run_gate 'Go vet' go vet ./...
run_gate 'API contract lint' bash scripts/api-lint.sh
run_gate 'SQLC drift check' bash scripts/sqlc-check.sh
run_gate 'Compose configuration' docker compose config --quiet

if [[ -d web/node_modules ]]; then
	run_gate 'Svelte checks' pnpm --dir web check
	run_gate 'Production web build' pnpm --dir web build
	if pnpm --dir web exec playwright --version >/dev/null 2>&1; then
		run_gate 'Browser end-to-end tests' pnpm --dir web exec playwright test
	else
		printf 'MISSING BROWSER: Playwright is not installed.\n' >&2
		failures=$((failures + 1))
	fi
else
	printf 'MISSING FRONTEND DEPS: web/node_modules is absent.\n' >&2
	failures=$((failures + 1))
fi

if [[ -z "${DATABASE_URL:-}" ]]; then
	printf 'MISSING DATABASE: DATABASE_URL is required for release certification.\n' >&2
	failures=$((failures + 1))
else
	run_gate 'Database persistence contract' bash scripts/db-check.sh
fi

if rg -n 'DomainAggregatesDurable:[[:space:]]*false' internal/web/runtime.go >/dev/null 2>&1; then
	printf 'MISSING DURABILITY: runtime still advertises process-local domain aggregates.\n' >&2
	failures=$((failures + 1))
fi

for evidence in SECURITY_REVIEW_REFERENCE DPIA_REFERENCE LEGAL_APPROVAL_REFERENCE PEN_TEST_REFERENCE BACKUP_RESTORE_REFERENCE PROVIDER_CERTIFICATION_REFERENCE SUPPORT_TRAINING_REFERENCE LAUNCH_APPROVAL_REFERENCE; do
	if [[ -z "${!evidence:-}" ]]; then
		printf 'MISSING APPROVAL: %s\n' "$evidence" >&2
		failures=$((failures + 1))
	fi
done

if (( failures > 0 )); then
	printf '\nRelease certification FAILED with %d unmet gate(s).\n' "$failures" >&2
	exit 1
fi

printf '\nRelease certification PASSED: all local and external gates supplied.\n'
