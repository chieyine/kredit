#!/usr/bin/env bash
set -euo pipefail

# Two shapes of row-level security failure, both read from the live catalogue
# rather than from migration text, because what matters is the policy that
# actually exists.
#
# GATE 1 - a tenant policy OR'd away by a permissive blanket role check.
#   PostgreSQL combines permissive policies with OR, so
#   `tenant_predicate OR current_user IN ('kredit_app','kredit_worker')` is
#   always true for the roles the application connects as. Migration 081
#   handled this for the core money tables and 158-163 finished the rest.
#
# GATE 2 - a table that carries a tenant column, has RLS enabled, and has no
#   tenant policy at all. Gate 1 cannot see these: it only fires where a tenant
#   policy already exists to be OR'd away, so a table that never had one passes
#   silently. That blind spot covered app.support_cases, app.operation_actions
#   and app.provider_customer_bindings - all of which hold customer data and
#   relied entirely on the application remembering its WHERE clause.
#
# Both gates fail on anything not in their baseline, and also when a baseline
# entry has been fixed but not removed, so the lists can only shrink.

root_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$root_dir"
: "${DATABASE_URL:?DATABASE_URL is required}"

ored_baseline="docs/compliance/rls-permissive-baseline.txt"
missing_baseline="docs/compliance/rls-missing-tenant-policy-baseline.txt"
[[ -s "$ored_baseline" ]] || { printf 'RLS baseline is missing: %s\n' "$ored_baseline" >&2; exit 1; }
[[ -s "$missing_baseline" ]] || { printf 'RLS baseline is missing: %s\n' "$missing_baseline" >&2; exit 1; }

actual="$(mktemp)"; expected="$(mktemp)"
trap 'rm -f "$actual" "$expected"' EXIT

compare() {
  local label="$1" baseline="$2" explain_add="$3" explain_del="$4"
  { grep -vE '^\s*(#|$)' "$baseline" || true; } | sort > "$expected"
  sort -o "$actual" "$actual"
  if ! diff -u "$expected" "$actual" > /dev/null; then
    printf '%s\n\n' "$label" >&2
    printf '%s\n' "$explain_add" >&2
    printf '%s\n\n' "$explain_del" >&2
    diff -u "$expected" "$actual" >&2 || true
    exit 1
  fi
  printf '%s: matches baseline (%s entr(y|ies)).\n' "$label" "$(grep -c . "$actual" || true)"
}

# --- Gate 1: blanket role policy OR'd over a tenant policy -------------------
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
compare 'Gate 1 (tenant policy OR-ed away by a blanket role policy)' "$ored_baseline" \
  '+ lines are tables that newly OR a blanket role check over a tenant policy.' \
  "- lines are tables that have been fixed and must be removed from $ored_baseline."

# --- Gate 2: tenant column, RLS on, no tenant policy ------------------------
# A column named for an owner is the signal that rows belong to someone. If a
# table has one and no policy mentions the tenant settings, the database is
# enforcing nothing and a forgotten WHERE clause is a cross-tenant read.
psql "$DATABASE_URL" -X -v ON_ERROR_STOP=1 -A -t -c "
SELECT n.nspname||'.'||c.relname
FROM pg_class c
JOIN pg_namespace n ON n.oid = c.relnamespace AND n.nspname IN ('app','ledger')
WHERE c.relkind = 'r'
  AND c.relrowsecurity
  AND EXISTS (
        SELECT 1 FROM pg_attribute a
        WHERE a.attrelid = c.oid AND a.attnum > 0 AND NOT a.attisdropped
          AND a.attname IN ('organization_id','supplier_organization_id',
                            'buyer_user_id','buyer_business_id','user_id','agent_id')
      )
  AND NOT EXISTS (
        SELECT 1 FROM pg_policy p
        WHERE p.polrelid = c.oid
          AND p.polpermissive
          AND (pg_get_expr(p.polqual, p.polrelid) ILIKE '%current_organization_id%'
            OR pg_get_expr(p.polqual, p.polrelid) ILIKE '%current_user_id%')
      )
ORDER BY 1" > "$actual"
compare 'Gate 2 (tenant column with no tenant policy)' "$missing_baseline" \
  '+ lines are tables carrying a tenant column with no row-level tenant policy.' \
  "- lines are tables that now have one and must be removed from $missing_baseline."
