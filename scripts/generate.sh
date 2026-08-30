#!/usr/bin/env bash
set -euo pipefail

bash scripts/openapi-generate.sh
bash scripts/sqlc-generate.sh

if command -v go >/dev/null 2>&1; then
  go generate ./...
fi

