#!/usr/bin/env bash
set -euo pipefail
root_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
fixture="$(mktemp -d "${TMPDIR:-/tmp}/kredit-db-guard.XXXXXX")"
trap 'rm -rf "$fixture"' EXIT

for action in reset rollback; do
  if env APP_ENV=production ALLOW_DB_RESET=true ALLOW_DB_ROLLBACK=true bash "$root_dir/scripts/db-$action.sh" > "$fixture/$action.log" 2>&1; then
    printf 'production %s was allowed\n' "$action" >&2; exit 1
  fi
  grep -q 'requires APP_ENV=development or APP_ENV=test' "$fixture/$action.log"
done

printf 'synthetic backup\n' > "$fixture/backup.dump"
shasum -a 256 "$fixture/backup.dump" > "$fixture/backup.dump.sha256"
mkdir "$fixture/bin"
cat > "$fixture/bin/psql" <<'SHIM'
#!/usr/bin/env bash
printf '1\n'
SHIM
cat > "$fixture/bin/pg_restore" <<'SHIM'
#!/usr/bin/env bash
printf 'pg_restore must not be invoked for a populated target\n' >&2
exit 99
SHIM
chmod +x "$fixture/bin/psql" "$fixture/bin/pg_restore"
if env PATH="$fixture/bin:$PATH" DATABASE_URL='' RESTORE_DATABASE_URL='postgres://synthetic@localhost/isolated' bash "$root_dir/scripts/restore-drill.sh" "$fixture/backup.dump" > "$fixture/restore.log" 2>&1; then
  printf 'populated restore target was accepted\n' >&2; exit 1
fi
grep -q 'restore target must be an empty database' "$fixture/restore.log"
if grep -q 'pg_restore must not' "$fixture/restore.log"; then exit 1; fi
rm "$fixture/backup.dump.sha256"
if env RESTORE_DATABASE_URL='postgres://synthetic@localhost/isolated' bash "$root_dir/scripts/restore-drill.sh" "$fixture/backup.dump" > "$fixture/checksum.log" 2>&1; then exit 1; fi
grep -q 'backup checksum file is required' "$fixture/checksum.log"
printf 'Database destructive-operation guards passed without connecting to a database.\n'
