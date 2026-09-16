#!/usr/bin/env bash
set -euo pipefail

# PostgreSQL combines permissive policies with OR. A tenant-scoped policy sitting
# beside a permissive blanket role check is therefore not an isolation boundary:
# `tenant_predicate OR current_user IN ('kredit_app','kredit_worker')` is always
# true for the roles the application actually connects as.
#
# Migration 081 handled this correctly for the core money tables by DROPPING the
# blanket policies before adding tenant ones. This gate exists so the remaining
# tables stay visible, and so no new table silently repeats the pattern. It reads
# the live catalogue rather than the migration text, because what matters is the
# shape of the policies that actually exist.
#
# Tables still carrying the pattern are listed in the baseline below. The gate
# fails when a table appears that is not in the baseline, and also when a table
# in the baseline has been fixed but not removed from it, so the list can only
# shrink.

root_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$root_dir"
: "${DATABASE_URL:?DATABASE_URL is required}"

baseline="docs/compliance/rls-permissive-baseline.txt"
[[ -s "$baseline" ]] || { printf 'RLS baseline is missing: %s\n' "$baseline" >&2; exit 1; }

actual="$(mktemp)"
expected="$(mktemp)"
trap 'rm -f "$actual" "$expected"' EXIT

psql "$DATABASE_URL" -X -v ON_ERROR_STOP=1 -A -t -c "
SELECT schemaname||'.'||tablename
FROM pg_policies
WHERE schemaname IN ('app','ledger')
GROUP BY schemaname, tablename
HAVING count(*) FILTER (
         WHERE permissive = 'PERMISSIVE'
           AND (qual ILIKE '%current_organization_id%' OR qual ILIKE '%current_user_id%')
       ) > 0
   AND count(*) FILTER (
         WHERE permissive = 'PERMISSIVE'
           AND qual ILIKE '%current_user%kredit_app%'
           AND qual NOT ILIKE '%current_organization_id%'
           AND qual NOT ILIKE '%current_user_id%'
       ) > 0
ORDER BY 1" > "$actual"

grep -vE '^\s*(#|$)' "$baseline" | sort > "$expected"
sort -o "$actual" "$actual"

if ! diff -u "$expected" "$actual" > /dev/null; then
  printf 'Row-level security policy shape changed.\n\n' >&2
  printf 'Lines starting with + are tables that newly OR a blanket role check over a\n' >&2
  printf 'tenant policy. Lines starting with - are tables that have been fixed and must\n' >&2
  printf 'be removed from %s.\n\n' "$baseline" >&2
  diff -u "$expected" "$actual" >&2 || true
  exit 1
fi

count="$(grep -c . "$actual" || true)"
printf 'RLS policy shape matches the recorded baseline (%s table(s) still pending Phase 2 conversion).\n' "$count"
