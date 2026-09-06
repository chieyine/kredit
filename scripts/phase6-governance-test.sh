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

# Sitemap timestamps are evidence, not decoration. Fixed pages have no fabricated
# lastmod and topic dates must derive from actual published article metadata.
if grep -q '<lastmod>2026-08-31</lastmod>' web/src/routes/sitemap.xml/+server.ts; then
  fail 'sitemap still contains a fabricated fixed lastmod'
fi
grep -q 'topicLastmod' web/src/routes/sitemap.xml/+server.ts || fail 'topic lastmod is not metadata-derived'

printf '%s\n' 'phase6_governance=passed'
