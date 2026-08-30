#!/usr/bin/env bash
set -euo pipefail
umask 077

: "${DATABASE_URL:?DATABASE_URL is required}"
backup_dir="${BACKUP_DIR:-$PWD/.tmp/backups}"
mkdir -p "$backup_dir"
timestamp="$(date -u +%Y%m%dT%H%M%SZ)"
output="$backup_dir/kredit-$timestamp.dump"
pg_dump --format=custom --no-owner --no-privileges --file="$output" "$DATABASE_URL"
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
