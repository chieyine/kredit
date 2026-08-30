#!/usr/bin/env bash
set -euo pipefail

if command -v redocly >/dev/null 2>&1; then
  redocly lint api/openapi.yaml
  exit 0
fi

if command -v spectral >/dev/null 2>&1; then
  spectral lint api/openapi.yaml
  exit 0
fi

printf '%s\n' 'No OpenAPI linter installed; performed structural file check.'
grep -q '^openapi: 3\.1\.0$' api/openapi.yaml
grep -q '^paths:' api/openapi.yaml

