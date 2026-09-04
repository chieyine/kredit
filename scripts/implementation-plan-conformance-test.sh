#!/usr/bin/env bash
set -euo pipefail

root_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$root_dir"

temp_dir="$(mktemp -d "${TMPDIR:-/tmp}/kredit-plan-conformance.XXXXXX")"
trap 'rm -rf "$temp_dir"' EXIT

bash scripts/implementation-plan-conformance.sh

invalid_manifest="$temp_dir/workstream-evidence.tsv"
awk 'BEGIN { FS=OFS="\t" } $1 == "TL-DRAWDOWN" { $3="complete"; $8="-"; $9="-"; $10="2026-08-29" } { print }' docs/product/workstream-evidence.tsv > "$invalid_manifest"

if bash scripts/implementation-plan-conformance.sh "$invalid_manifest" > /dev/null 2>&1; then
  printf '%s\n' 'expected complete workstream without evidence to fail' >&2
  exit 1
fi

printf '%s\n' 'Implementation plan conformance self-test passed.'
