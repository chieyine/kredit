#!/usr/bin/env bash
set -euo pipefail

root_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$root_dir"

migrations_dir="${1:-$root_dir/db/migrations}"

if [[ ! -d "$migrations_dir" ]]; then
  printf 'Error: migrations directory not found: %s\n' "$migrations_dir" >&2
  exit 1
fi

if [[ -f "$root_dir/scripts/load-env.sh" && -f "$root_dir/.env" ]]; then
  source "$root_dir/scripts/load-env.sh"
  load_env_defaults "$root_dir/.env"
fi

if [[ -z "${DATABASE_DIRECT_URL:-}" && -z "${DATABASE_URL:-}" ]]; then
  printf 'Error: DATABASE_DIRECT_URL or DATABASE_URL must be set.\n' >&2
  exit 1
fi

printf 'Running database migrations from: %s\n' "$migrations_dir"

if [[ -x "$root_dir/bin/kredit-migrate" ]]; then
  "$root_dir/bin/kredit-migrate" --migrations-dir "$migrations_dir"
else
  go run "$root_dir/cmd/migrate" --migrations-dir "$migrations_dir"
fi

printf 'All database migrations applied successfully.\n'
