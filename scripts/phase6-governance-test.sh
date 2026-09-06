#!/usr/bin/env bash
set -euo pipefail

fail() {
  printf 'phase6 governance check failed: %s\n' "$1" >&2
  exit 1
}

# CI must install a real, pinned OpenAPI linter and require it at runtime.
grep -q '@redocly/cli@2\.51\.2' .github/workflows/ci.yml || fail 'Redocly CLI is not pinned in CI'
grep -q 'OPENAPI_LINT_STRICT: "1"' .github/workflows/ci.yml || fail 'strict OpenAPI lint is not enabled in CI'
grep -q "A real OpenAPI linter is required in CI" scripts/api-lint.sh || fail 'OpenAPI lint does not fail closed'

# Security tooling must not silently float to a future Trivy release.
grep -q 'version: v0\.74\.0' .github/workflows/ci.yml || fail 'Trivy is not pinned'

# SvelteKit must own script CSP so it can attach framework hashes/nonces. The
# application hook must not reintroduce a weaker script directive. Match actual
# directive/header syntax rather than comments that may discuss unsafe-inline.
grep -q "mode: 'auto'" web/svelte.config.js || fail 'SvelteKit CSP mode is not auto'
grep -q "'script-src': \['self'\]" web/svelte.config.js || fail 'script CSP is not restricted to self plus framework nonces/hashes'
if grep -Eiq "(['\"]script-src['\"]\s*:|script-src\s+)[^;\]}]*unsafe-inline" web/svelte.config.js web/src/hooks.server.ts; then
  fail 'script-src unsafe-inline was reintroduced'
fi
# Browser/proxy cancellation must reach the upstream Go request as well as the
# fixed timeout budget, and the synchronous request-context audit must remain in CI.
grep -q 'AbortSignal.any(\[event.request.signal, timeout\])' web/src/hooks.server.ts || fail 'proxy does not propagate request cancellation'
grep -q 'python3 scripts/phase6-context-audit.py' scripts/ci.sh || fail 'request-context audit is not enforced in CI'

# Analytics may use pseudonymous subject hashes, but obvious customer/bank/provider
# identifiers must remain rejected from metadata and HTTP persistence must expose a
# request-aware path.
grep -q 'forbiddenAnalyticsMetadata' internal/reports/store.go || fail 'analytics metadata deny-list is missing'
for field in '"phone"' '"email"' '"bvn"' '"nin"' '"bank_account"' '"provider_token"'; do
  grep -q "$field" internal/reports/store.go || fail "analytics sensitive-field guard missing: $field"
done
grep -q 'func (s \*Store) TrackContext(ctx context.Context' internal/reports/store.go || fail 'request-aware analytics persistence is missing'

# Upload completion must remain distinct from malware/scanner promotion and bounded
# by private-storage/object validation rather than treating upload success as clean.
grep -q 'StatePendingScan' internal/documents/store.go || fail 'document pending-scan state is missing'
grep -q 'CompleteScan' internal/documents/store.go || fail 'scanner promotion boundary is missing'
grep -q 'allowedType' internal/documents/store.go || fail 'document content-type validation is missing'

# Sensitive onboarding changes must retain recent MFA rather than relying only on a
# long-lived authenticated session.
grep -q 'requireFreshMFA' internal/web/onboarding_handlers.go || fail 'recent MFA enforcement is missing from onboarding changes'

# Sitemap timestamps are evidence, not decoration. Fixed pages have no fabricated
# lastmod and topic dates must derive from actual published article metadata.
if grep -q '<lastmod>2026-08-31</lastmod>' web/src/routes/sitemap.xml/+server.ts; then
  fail 'sitemap still contains a fabricated fixed lastmod'
fi
grep -q 'topicLastmod' web/src/routes/sitemap.xml/+server.ts || fail 'topic lastmod is not metadata-derived'

# Status documentation must track the actual migration frontier and distinguish
# engineering completion from external launch approval.
test -f db/migrations/086_phase5_financial_metrics.sql || fail 'expected migration 086 is missing'
grep -q 'Last updated: 6 September 2026' IMPLEMENTATION_STATUS.md || fail 'implementation status date is stale'
grep -q 'migrations run through \*\*086\*\*' IMPLEMENTATION_STATUS.md || fail 'implementation status migration frontier is stale'
grep -q 'Actual Mono sandbox certification remains open in issue #5' IMPLEMENTATION_STATUS.md || fail 'external provider gate is not explicit'

# Temporary self-modifying patch harnesses must not survive the phase.
if find scripts .github/workflows -maxdepth 1 -type f \( -name 'phase6-*-fix.py' -o -name 'phase6-*-fix.yml' -o -name 'phase6-status-sync.py' -o -name 'phase6-status-sync.yml' \) | grep -q .; then
  fail 'temporary Phase 6 patch harness remains in the repository'
fi

printf '%s\n' 'phase6_governance=passed'
