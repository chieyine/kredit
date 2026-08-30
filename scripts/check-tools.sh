#!/usr/bin/env bash
set -euo pipefail

missing=0
for tool in go node pnpm docker psql task; do
  if ! command -v "$tool" >/dev/null 2>&1; then
    printf 'missing required tool: %s\n' "$tool" >&2
    missing=1
  fi
done

if [[ "$missing" -ne 0 ]]; then
  exit 1
fi

go version
node --version
pnpm --version
