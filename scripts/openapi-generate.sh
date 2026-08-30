#!/usr/bin/env bash
set -euo pipefail

oapi_codegen="$(command -v oapi-codegen || true)"
if [[ -z "$oapi_codegen" && -x "./.tmp/bin/oapi-codegen" ]]; then
  oapi_codegen="./.tmp/bin/oapi-codegen"
fi
if [[ -z "$oapi_codegen" ]]; then
  printf '%s\n' 'oapi-codegen is not installed; OpenAPI generation is configured but skipped.'
  exit 0
fi

mkdir -p api/generated web/src/lib/api/generated
# The HTTP implementation uses the standard library ServeMux. Generate the
# contract types here; handlers remain explicit and are checked against the
# same OpenAPI document instead of generating an incompatible Chi server.
generated_file="$(mktemp)"
trap 'rm -f "$generated_file"' EXIT
"$oapi_codegen" -generate types -package generated api/openapi.yaml > "$generated_file"
sed '/^WARNING: You are using an OpenAPI 3.1.x specification/d' "$generated_file" > api/generated/types.gen.go

openapi_typescript="$(command -v openapi-typescript || true)"
if [[ -x ./node_modules/.bin/openapi-typescript ]]; then
  openapi_typescript=./node_modules/.bin/openapi-typescript
fi
if [[ -n "$openapi_typescript" ]]; then
  "$openapi_typescript" api/openapi.yaml -o web/src/lib/api/generated/schema.d.ts
else
  printf '%s\n' 'openapi-typescript is not installed; TypeScript generation skipped.'
fi
