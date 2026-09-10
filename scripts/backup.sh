#!/usr/bin/env bash
set -euo pipefail
umask 077

backup_database_url="${BACKUP_DATABASE_URL:-${DATABASE_DIRECT_URL:-${DATABASE_URL:-}}}"
: "${backup_database_url:?BACKUP_DATABASE_URL, DATABASE_DIRECT_URL, or DATABASE_URL is required}"
backup_dir="${BACKUP_DIR:-$PWD/.tmp/backups}"
mkdir -p "$backup_dir"
timestamp="$(date -u +%Y%m%dT%H%M%SZ)"
# BSD and GNU mktemp both randomize a template ending in XXXXXX.
# Keep the archive extension inside a private, uniquely allocated directory.
archive_dir="$(mktemp -d "$backup_dir/kredit-$timestamp-XXXXXX")"
output="$archive_dir/backup.dump"
# Retain object ACLs, especially revoked PUBLIC access on SECURITY DEFINER
# functions. Restoring an ACL-stripped archive can silently restore defaults.
# The destination must have the same named roles provisioned before restore.
# Runtime RLS-scoped credentials cannot produce a complete backup.
pg_dump --format=custom --no-owner --file="$output" "$backup_database_url"
chmod 600 "$output"
if command -v sha256sum >/dev/null 2>&1; then
  sha256sum "$output" > "$output.sha256"
elif command -v shasum >/dev/null 2>&1; then
  shasum -a 256 "$output" > "$output.sha256"
else
  printf '%s\n' 'sha256sum or shasum is required to write backup integrity evidence.' >&2
  exit 1
fi
chmod 600 "$output.sha256"
printf 'backup=%s\n' "$output"
printf 'checksum=%s\n' "$output.sha256"
