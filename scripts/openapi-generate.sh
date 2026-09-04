#!/usr/bin/env bash
set -euo pipefail

# Go server code and Go types are hand-written; see
# docs/adr/0005-hand-written-http-and-sql.md. The OpenAPI document stays
# canonical and is enforced against the implemented routes by
# scripts/product-contract-sync.mjs. Only the TypeScript client types are
# generated from it.
mkdir -p web/src/lib/api/generated

openapi_typescript="$(command -v openapi-typescript || true)"
if [[ -x ./node_modules/.bin/openapi-typescript ]]; then
  openapi_typescript=./node_modules/.bin/openapi-typescript
fi
if [[ -n "$openapi_typescript" ]]; then
  "$openapi_typescript" api/openapi.yaml -o web/src/lib/api/generated/schema.d.ts
else
  printf '%s\n' 'openapi-typescript is not installed; TypeScript generation skipped.'
fi
