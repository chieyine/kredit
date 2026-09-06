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
# application hook must not reintroduce a weaker script-src unsafe-inline policy.
grep -q "mode: 'auto'" web/svelte.config.js || fail 'SvelteKit CSP mode is not auto'
grep -q "'script-src': \['self'\]" web/svelte.config.js || fail 'script CSP is not restricted to self plus framework nonces/hashes'
if grep -q "script-src[^\n]*unsafe-inline" web/svelte.config.js web/src/hooks.server.ts; then
  fail 'script-src unsafe-inline was reintroduced'
fi
# Browser/proxy cancellation must reach the upstream Go request as well as the
# fixed timeout budget.
grep -q 'AbortSignal.any(\[event.request.signal, timeout\])' web/src/hooks.server.ts || fail 'proxy does not propagate request cancellation'

# Sitemap timestamps are evidence, not decoration. Fixed pages have no fabricated
# lastmod and topic dates must derive from actual published article metadata.
if grep -q '<lastmod>2026-08-31</lastmod>' web/src/routes/sitemap.xml/+server.ts; then
  fail 'sitemap still contains a fabricated fixed lastmod'
fi
grep -q 'topicLastmod' web/src/routes/sitemap.xml/+server.ts || fail 'topic lastmod is not metadata-derived'

printf '%s\n' 'phase6_governance=passed'
