#!/usr/bin/env bash
set -euo pipefail

# Go server code and Go types are hand-written; see
# docs/adr/0005-hand-written-http-and-sql.md. The OpenAPI document stays
# canonical and is enforced against the implemented routes by
# scripts/product-contract-sync.mjs.
#
# TypeScript client types are no longer generated from it. The one page that
# used them typed its response as `any` — the operation it called is declared
# with additionalProperties — so an 18,000-line file was being rebuilt before
# every dev server, build, check and test run to provide no type safety at all.
# Pages now declare the shape they read.

if command -v go >/dev/null 2>&1; then
  go generate ./...
fi
