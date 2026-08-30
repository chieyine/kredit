#!/usr/bin/env bash
set -euo pipefail

before="$(mktemp -d)"
trap 'rm -rf "$before"' EXIT
cp -R db/generated/. "$before/"
bash scripts/sqlc-generate.sh
if ! diff -ru "$before" db/generated >/dev/null; then
  printf '%s\n' 'db/generated is stale; run scripts/sqlc-generate.sh and commit the result.' >&2
  diff -ru "$before" db/generated || true
  exit 1
fi
printf '%s\n' 'sqlc generated output is up to date.'
