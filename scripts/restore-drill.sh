#!/usr/bin/env bash
set -euo pipefail
umask 077

: "${RESTORE_DATABASE_URL:?RESTORE_DATABASE_URL is required and must target an isolated restore database}"
backup_file="${1:?usage: scripts/restore-drill.sh /absolute/path/to/backup.dump}"
if [[ "$backup_file" != /* ]]; then printf 'backup path must be absolute\n' >&2; exit 1; fi
if [[ ! -f "$backup_file" ]]; then printf 'backup file not found\n' >&2; exit 1; fi
if [[ -n "${DATABASE_URL:-}" && "$RESTORE_DATABASE_URL" == "$DATABASE_URL" ]]; then printf 'restore target must differ from DATABASE_URL\n' >&2; exit 1; fi
if [[ ! -f "$backup_file.sha256" ]]; then printf 'backup checksum file is required\n' >&2; exit 1; fi
drill_root="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
python3 "$drill_root/scripts/verify_backup.py" "$backup_file"
: "${RESTORE_EXPECTED_FINGERPRINT:?source fingerprint from a quiesced database is required}"
[[ -f "$RESTORE_EXPECTED_FINGERPRINT" ]] || { printf 'source fingerprint not found\n' >&2; exit 1; }
# Refuse populated targets, including aliases of the source. No --clean, DROP,
# trigger disabling or RLS disabling on the target is used by this drill.
target_objects="$(psql "$RESTORE_DATABASE_URL" -XAt -v ON_ERROR_STOP=1 -c "SELECT count(*) FROM pg_catalog.pg_class c JOIN pg_catalog.pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname NOT IN ('pg_catalog','information_schema') AND n.nspname !~ '^pg_' AND c.relkind IN ('r','p','v','m','f');")"
if [[ "$target_objects" != "0" ]]; then printf 'restore target must be an empty database\n' >&2; exit 1; fi
printf 'restoring into the explicitly configured empty target...\n'
# Preserve archived grants/revocations. Named roles must exist in the target
# cluster. Legacy ACL-stripped archives fail the source security comparison.
pg_restore --exit-on-error --no-owner --dbname="$RESTORE_DATABASE_URL" "$backup_file"
psql "${RESTORE_ROLE_ADMIN_URL:-$RESTORE_DATABASE_URL}" -X -v ON_ERROR_STOP=1 -f "$drill_root/infra/postgres/roles.sql"
runtime_grants="$(psql "$RESTORE_DATABASE_URL" -XAt -v ON_ERROR_STOP=1 -c "SELECT has_schema_privilege('kredit_app','app','USAGE') AND has_table_privilege('kredit_app','app.idempotency_records','SELECT,INSERT,UPDATE') AND NOT has_table_privilege('kredit_app','app.idempotency_records','DELETE') AND has_function_privilege('kredit_app','app.delete_expired_idempotency_record(text,text)','EXECUTE') AND has_schema_privilege('kredit_worker','jobs','USAGE');")"
[[ "$runtime_grants" == "t" ]] || { printf 'restored runtime grants are incomplete\n' >&2; exit 1; }
actual_dir="$(mktemp -d)"
trap 'rm -rf "$actual_dir"' EXIT
DATABASE_URL="$RESTORE_DATABASE_URL" python3 "$drill_root/scripts/recovery_fingerprint.py" capture "$actual_dir/restored.json" --quiesced
python3 "$drill_root/scripts/recovery_fingerprint.py" compare "$RESTORE_EXPECTED_FINGERPRINT" "$actual_dir/restored.json"
printf 'runtime_grants=passed\n'
printf 'restore_drill=passed (isolated logical recovery; not production/PITR certification)\n'
