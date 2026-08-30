#!/usr/bin/env bash
set -euo pipefail
umask 077

: "${RESTORE_DATABASE_URL:?RESTORE_DATABASE_URL is required and must target an isolated restore database}"
backup_file="${1:?usage: scripts/restore-drill.sh /absolute/path/to/backup.dump}"
if [[ "$backup_file" != /* ]]; then printf 'backup path must be absolute\n' >&2; exit 1; fi
if [[ ! -f "$backup_file" ]]; then printf 'backup file not found: %s\n' "$backup_file" >&2; exit 1; fi
if [[ -n "${DATABASE_URL:-}" && "$RESTORE_DATABASE_URL" == "$DATABASE_URL" ]]; then printf 'restore target must differ from DATABASE_URL\n' >&2; exit 1; fi
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
printf 'restoring into the explicitly configured isolated target...\n'
pg_restore --clean --if-exists --exit-on-error --no-owner --dbname="$RESTORE_DATABASE_URL" "$backup_file"
psql "$RESTORE_DATABASE_URL" -v ON_ERROR_STOP=1 -c 'SELECT current_database(), NOW();'
psql "$RESTORE_DATABASE_URL" -v ON_ERROR_STOP=1 -c 'SELECT version_id FROM goose_db_version ORDER BY id DESC LIMIT 1;'
printf 'restore_drill=passed\n'
