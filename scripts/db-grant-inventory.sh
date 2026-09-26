#!/usr/bin/env bash
# Regenerates docs/compliance/runtime-grant-inventory.txt from a migrated
# database that has had infra/postgres/roles.sql applied.
set -euo pipefail
root_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"; cd "$root_dir"
: "${DATABASE_URL:?DATABASE_URL is required}"
out="docs/compliance/runtime-grant-inventory.txt"
temporary="$(mktemp "${out}.XXXXXX")"
trap 'rm -f "$temporary"' EXIT
{ sed -n '1,/^# Regenerate/p' "$out"
  psql "$DATABASE_URL" -X -v ON_ERROR_STOP=1 -A -t -F' ' -c "
    SELECT grantee, table_schema||'.'||table_name,
           string_agg(DISTINCT lower(privilege_type), ',' ORDER BY lower(privilege_type))
    FROM information_schema.role_table_grants
    WHERE grantee IN ('kredit_app','kredit_worker') AND table_schema IN ('app','ledger','jobs')
    GROUP BY grantee, table_schema, table_name ORDER BY 1,2" | sort
} > "$temporary"
mv "$temporary" "$out"
printf 'Wrote %s (%s grants).\n' "$out" "$(grep -cvE '^\s*(#|$)' "$out")"
