#!/usr/bin/env bash
set -euo pipefail

root_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$root_dir"
export GOCACHE="${GOCACHE:-$PWD/.tmp/go-cache}"
mkdir -p "$GOCACHE"

# Every stored seed corpus runs as an ordinary test, so regressions found by a
# previous fuzzing session stay enforced on every build.
go test ./...

# Randomised fuzzing is bounded so the gate stays usable in CI. Go accepts one
# -fuzz target per invocation, so each target is run on its own.
duration="${FUZZTIME:-20s}"
targets="$(grep -rn --include='*_test.go' '^func Fuzz' . \
  | sed -E 's#^\./##; s#/[^/]*_test\.go:[0-9]+:func (Fuzz[A-Za-z0-9_]*).*#\t\1#' \
  | sort -u)"

if [[ -z "$targets" ]]; then
  printf '%s\n' 'no fuzz targets found' >&2
  exit 1
fi

count=0
while IFS=$'\t' read -r package target; do
  [[ -z "$target" ]] && continue
  printf 'fuzzing %s in ./%s for %s\n' "$target" "$package" "$duration"
  go test "./${package}" -run "^${target}\$" -fuzz "^${target}\$" -fuzztime "$duration"
  count=$((count + 1))
done <<< "$targets"

printf 'fuzzed %d target(s) for %s each.\n' "$count" "$duration"
