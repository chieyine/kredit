#!/usr/bin/env bash
set -euo pipefail

root_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$root_dir"
: "${DATABASE_URL:?DATABASE_URL is required}"

inventory="docs/compliance/data-inventory.tsv"
[[ -s "$inventory" ]] || { printf 'data inventory is missing\n' >&2; exit 1; }
expected="$(mktemp)"
actual="$(mktemp)"
trap 'rm -f "$expected" "$actual"' EXIT
psql "$DATABASE_URL" -A -F $'\t' -t -c "SELECT c.table_schema,c.table_name,c.column_name FROM information_schema.columns c JOIN information_schema.tables t USING(table_schema,table_name) WHERE t.table_type='BASE TABLE' AND c.table_schema IN ('app','ledger','river','jobs') ORDER BY 1,2,c.ordinal_position" > "$expected"
tail -n +2 "$inventory" | cut -f1-3 > "$actual"
if ! diff -u "$expected" "$actual"; then
  printf 'data inventory does not cover the current production schema; regenerate and review it\n' >&2
  exit 1
fi
awk -F '\t' 'NR>1 { if (NF != 16 || $4=="" || $7=="" || $8=="" || $12=="" || $13=="" || $16=="") { print "incomplete inventory row " NR > "/dev/stderr"; bad=1 } } END { exit bad }' "$inventory"
printf 'Field-level data inventory covers every production column.\n'
