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
require_command psql
require_command rg

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
run_gate 'SQLC drift check' bash scripts/sqlc-check.sh
run_gate 'Compose configuration' docker compose config --quiet

if [[ "${APP_ENV:-}" != "production" ]]; then
	printf 'MISSING PRODUCTION MODE: APP_ENV=production is required for release certification.\n' >&2
	failures=$((failures + 1))
else
	run_gate 'Production runtime configuration' go run ./cmd/configcheck
fi

for legal_value in LEGAL_ENTITY_NAME LEGAL_SERVICE_ADDRESS LEGAL_CONTACT_EMAIL PRIVACY_CONTACT_EMAIL LEGAL_EFFECTIVE_DATE TERMS_VERSION PRIVACY_VERSION; do
	if [[ -z "${!legal_value:-}" ]]; then
		printf 'MISSING PUBLIC LEGAL VALUE: %s\n' "$legal_value" >&2
		failures=$((failures + 1))
	fi
done
if [[ "${LEGAL_DOCUMENTS_ACTIVE:-}" != "true" ]]; then
	printf 'MISSING LEGAL ACTIVATION: LEGAL_DOCUMENTS_ACTIVE=true is required.\n' >&2
	failures=$((failures + 1))
fi
for launch_flag in FEATURE_APPROVED_RETENTION_POLICY FEATURE_PRODUCTION_PILOT FEATURE_REAL_IDENTITY FEATURE_REAL_COLLECTIONS; do
	if [[ "${!launch_flag:-}" != "true" ]]; then
		printf 'MISSING LAUNCH CAPABILITY: %s=true is required.\n' "$launch_flag" >&2
		failures=$((failures + 1))
	fi
done
if [[ -n "${LEGAL_EFFECTIVE_DATE:-}" && ! "${LEGAL_EFFECTIVE_DATE}" =~ ^[0-9]{4}-[0-9]{2}-[0-9]{2}$ ]]; then
	printf 'INVALID LEGAL DATE: LEGAL_EFFECTIVE_DATE must use YYYY-MM-DD.\n' >&2
	failures=$((failures + 1))
fi
if [[ "${TERMS_VERSION:-}" != "supplier-terms-v1" || "${PRIVACY_VERSION:-}" != "privacy-v1" ]]; then
	printf 'LEGAL VERSION MISMATCH: document versions must match supplier onboarding.\n' >&2
	failures=$((failures + 1))
fi
if [[ -n "${LEGAL_CONTACT_EMAIL:-}" && ! "${LEGAL_CONTACT_EMAIL}" =~ ^[^[:space:]@]+@[^[:space:]@]+\.[^[:space:]@]+$ ]]; then
	printf 'INVALID LEGAL EMAIL: LEGAL_CONTACT_EMAIL is not a valid email address.\n' >&2
	failures=$((failures + 1))
fi
if [[ -n "${PRIVACY_CONTACT_EMAIL:-}" && ! "${PRIVACY_CONTACT_EMAIL}" =~ ^[^[:space:]@]+@[^[:space:]@]+\.[^[:space:]@]+$ ]]; then
	printf 'INVALID PRIVACY EMAIL: PRIVACY_CONTACT_EMAIL is not a valid email address.\n' >&2
	failures=$((failures + 1))
fi

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
	printf '\nRelease certification FAILED with %d unmet gate(s).\n' "$failures" >&2
	exit 1
fi

printf '\nRelease certification PASSED: all local and external gates supplied.\n'
