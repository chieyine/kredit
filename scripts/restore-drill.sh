#!/usr/bin/env bash
set -euo pipefail
umask 077

: "${RESTORE_DATABASE_URL:?RESTORE_DATABASE_URL is required and must target an isolated restore database}"
backup_file="${1:?usage: scripts/restore-drill.sh /absolute/path/to/backup.dump}"
if [[ "$backup_file" != /* ]]; then printf 'backup path must be absolute\n' >&2; exit 1; fi
if [[ ! -f "$backup_file" ]]; then printf 'backup file not found: %s\n' "$backup_file" >&2; exit 1; fi
if [[ -n "${DATABASE_URL:-}" && "$RESTORE_DATABASE_URL" == "$DATABASE_URL" ]]; then printf 'restore target must differ from DATABASE_URL\n' >&2; exit 1; fi
if [[ ! -f "$backup_file.sha256" ]]; then printf 'backup checksum file is required\n' >&2; exit 1; fi
if [[ -f "$backup_file.sha256" ]]; then
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum --check "$backup_file.sha256"
  elif command -v shasum >/dev/null 2>&1; then
    expected="$(awk '{print $1}' "$backup_file.sha256")"
    actual="$(shasum -a 256 "$backup_file" | awk '{print $1}')"
    [[ "$expected" == "$actual" ]] || { printf 'backup checksum mismatch\n' >&2; exit 1; }
  else
    printf '%s\n' 'sha256sum or shasum is required to verify backup integrity.' >&2
    exit 1
  fi
fi
# Refuse any populated target, including an alias of the source database.
# Restore drills use fresh databases; they must never erase an existing ledger.
target_objects="$(psql "$RESTORE_DATABASE_URL" -XAt -v ON_ERROR_STOP=1 -c "SELECT count(*) FROM pg_catalog.pg_class c JOIN pg_catalog.pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname NOT IN ('pg_catalog','information_schema') AND n.nspname !~ '^pg_' AND c.relkind IN ('r','p','v','m','f');")"
if [[ "$target_objects" != "0" ]]; then printf 'restore target must be an empty database\n' >&2; exit 1; fi
printf 'restoring into the explicitly configured empty target...\n'
pg_restore --exit-on-error --no-owner --dbname="$RESTORE_DATABASE_URL" "$backup_file"
restore_role_admin_url="${RESTORE_ROLE_ADMIN_URL:-$RESTORE_DATABASE_URL}"
psql "$restore_role_admin_url" -X -v ON_ERROR_STOP=1 -f infra/postgres/roles.sql
psql "$RESTORE_DATABASE_URL" -v ON_ERROR_STOP=1 -c 'SELECT current_database(), NOW();'
psql "$RESTORE_DATABASE_URL" -v ON_ERROR_STOP=1 -c 'SELECT version_id FROM goose_db_version ORDER BY id DESC LIMIT 1;'
runtime_grants="$(psql "$RESTORE_DATABASE_URL" -XAt -v ON_ERROR_STOP=1 -c "SELECT has_schema_privilege('kredit_app','app','USAGE') AND has_table_privilege('kredit_app','app.idempotency_records','SELECT,INSERT,UPDATE') AND NOT has_table_privilege('kredit_app','app.idempotency_records','DELETE') AND has_function_privilege('kredit_app','app.delete_expired_idempotency_record(text,text)','EXECUTE') AND has_schema_privilege('kredit_worker','jobs','USAGE');")"
[[ "$runtime_grants" == "t" ]] || { printf 'restored runtime grants are incomplete\n' >&2; exit 1; }
printf 'runtime_grants=passed\n'
printf 'restore_drill=passed\n'
