#!/usr/bin/env bash
set -euo pipefail

# infra/postgres/roles.sql says:
#
#   "New tables are deliberately NOT auto-granted to either runtime role: a
#    migration must make an explicit privilege decision before new data
#    becomes reachable."
#
# ALTER DEFAULT PRIVILEGES does deliver that - but only until roles.sql runs
# again, and every deploy path runs it *after* migrations (ci.yml,
# scripts/db-reset.sh, scripts/configure-development-database.sh). Its
# `GRANT SELECT, INSERT, UPDATE ON ALL TABLES` then sweeps up every table the
# migrations just created. Demonstrated on a live database: a table created
# after roles.sql has no grants, and has INSERT,SELECT,UPDATE the moment
# roles.sql is re-run.
#
# Rather than drop the blanket grant - which would need all 203 existing
# migrations to name their own grants - this gate makes the reach of the
# runtime roles a reviewed fact. A table becoming readable or writable by
# kredit_app or kredit_worker must appear in the inventory, which is a diff a
# human approves. That is the property the comment was claiming.

root_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$root_dir"
: "${DATABASE_URL:?DATABASE_URL is required}"

inventory="docs/compliance/runtime-grant-inventory.txt"
[[ -s "$inventory" ]] || { printf 'Inventory is missing: %s\n' "$inventory" >&2; exit 1; }

actual="$(mktemp)"; expected="$(mktemp)"
trap 'rm -f "$actual" "$expected"' EXIT

psql "$DATABASE_URL" -X -v ON_ERROR_STOP=1 -A -t -F' ' -c "
SELECT grantee, table_schema||'.'||table_name,
       string_agg(DISTINCT lower(privilege_type), ',' ORDER BY lower(privilege_type))
FROM information_schema.role_table_grants
WHERE grantee IN ('kredit_app','kredit_worker')
  AND table_schema IN ('app','ledger','jobs')
GROUP BY grantee, table_schema, table_name
ORDER BY 1,2" > "$actual"

{ grep -vE '^\s*(#|$)' "$inventory" || true; } | sort > "$expected"
sort -o "$actual" "$actual"

if ! diff -u "$expected" "$actual" > /dev/null; then
  printf 'Runtime role reach changed.\n\n' >&2
  printf '+ lines are tables the runtime roles can now touch, or widened privileges.\n' >&2
  printf '  Confirm the migration intended it, then record it in %s.\n' "$inventory" >&2
  printf -- '- lines are grants that disappeared; confirm that was intended.\n\n' >&2
  diff -u "$expected" "$actual" >&2 || true
  exit 1
fi
printf 'Runtime grants match the reviewed inventory (%s grant(s)).\n' "$(grep -c . "$actual" || true)"
