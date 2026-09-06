#!/usr/bin/env bash
set -euo pipefail

if command -v redocly >/dev/null 2>&1; then
  REDOCLY_TELEMETRY=off REDOCLY_SUPPRESS_UPDATE_NOTICE=true redocly lint api/openapi.yaml
  exit 0
fi

if command -v spectral >/dev/null 2>&1; then
  spectral lint api/openapi.yaml
  exit 0
fi

if [[ "${OPENAPI_LINT_STRICT:-0}" == "1" || "${CI:-false}" == "true" ]]; then
  printf '%s\n' 'A real OpenAPI linter is required in CI (Redocly or Spectral).' >&2
  exit 1
fi

printf '%s\n' 'No OpenAPI linter installed; running a local structural smoke check only.'
grep -q '^openapi: 3\.1\.0$' api/openapi.yaml
grep -q '^paths:' api/openapi.yaml
