#!/usr/bin/env bash
set -euo pipefail

# Evidence review does not authorize deployment. The protected human approval
# remains separate. Empty references or a green unit suite are not sign-off.
bash scripts/phase5-evidence-gate.sh

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
require_command node
require_command pnpm
require_command docker
require_command psql
require_command rg
require_command python3

if command -v tofu >/dev/null 2>&1; then
	run_gate 'OpenTofu formatting' tofu fmt -check -recursive infra/environments
elif command -v terraform >/dev/null 2>&1; then
	run_gate 'Terraform formatting' terraform fmt -check -recursive infra/environments
else
	printf 'MISSING TOOL: tofu or terraform\n' >&2
	failures=$((failures + 1))
fi

run_gate 'Repository and product contract audit' pnpm run audit
run_gate 'Implementation evidence contract' bash scripts/implementation-plan-conformance-test.sh
run_gate 'Environment precedence contract' bash scripts/load-env-test.sh
run_gate 'Go unit tests' go test ./...
run_gate 'Go vet' go vet ./...
run_gate 'Go production builds' go build ./cmd/...
run_gate 'API contract lint' bash scripts/api-lint.sh
run_gate 'Compose configuration' docker compose config --quiet

if [[ "${APP_ENV:-}" != "production" ]]; then
	printf 'MISSING PRODUCTION MODE: APP_ENV=production is required for release certification.\n' >&2
	failures=$((failures + 1))
else
	run_gate 'Production runtime configuration' go run ./cmd/configcheck --stored
fi

run_gate 'Published legal metadata' node scripts/legal-publication-check.mjs
for launch_flag in FEATURE_APPROVED_RETENTION_POLICY FEATURE_PRODUCTION_PILOT FEATURE_REAL_IDENTITY FEATURE_REAL_COLLECTIONS; do
	if [[ "${!launch_flag:-}" != "true" ]]; then
		printf 'MISSING LAUNCH CAPABILITY: %s=true is required.\n' "$launch_flag" >&2
		failures=$((failures + 1))
	fi
done
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
	run_gate 'Database field inventory' bash scripts/data-inventory-check.sh
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
	printf '\nRelease checks FAILED with %d unmet gate(s).\n' "$failures" >&2
	exit 1
fi
printf '\nLocal release and evidence checks passed. Protected human deployment approval is still required.\n'
