#!/usr/bin/env bash
set -euo pipefail

root_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$root_dir"
: "${DATABASE_URL:?DATABASE_URL is required}"

# Validate explicit records before querying the schema. A schema field is not
# automatically inventoried just because the database happens to contain it.
python3 scripts/data-inventory-test.py
expected="$(mktemp)"
trap 'rm -f "$expected"' EXIT
psql "$DATABASE_URL" -X -v ON_ERROR_STOP=1 -A -F $'\t' -t -c "SELECT c.table_schema,c.table_name,c.column_name FROM information_schema.columns c JOIN information_schema.tables t USING(table_schema,table_name) WHERE t.table_type='BASE TABLE' AND c.table_schema IN ('app','ledger','river','jobs') ORDER BY 1,2,c.ordinal_position" > "$expected"
python3 scripts/data-inventory.py --check-schema "$expected"
