#!/usr/bin/env bash
set -euo pipefail

root_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$root_dir"
source "$root_dir/scripts/load-env.sh"
load_env_defaults "$root_dir/.env"

if [[ "${ALLOW_DB_RESET:-false}" != "true" ]]; then
  printf '%s\n' 'Refusing reset. Set ALLOW_DB_RESET=true for an explicit local reset.' >&2
  exit 1
fi

case "${APP_ENV:-}" in
  development|test) ;;
  *) printf '%s\n' 'Reset requires APP_ENV=development or APP_ENV=test.' >&2; exit 1 ;;
esac

database_admin_url="${DATABASE_DIRECT_URL:-${DATABASE_URL:-}}"
: "${database_admin_url:?DATABASE_DIRECT_URL or DATABASE_URL is required for reset}"
psql "$database_admin_url" -v ON_ERROR_STOP=1 -c 'BEGIN; DROP SCHEMA IF EXISTS app CASCADE; DROP SCHEMA IF EXISTS ledger CASCADE; DROP SCHEMA IF EXISTS jobs CASCADE; DROP SCHEMA public CASCADE; CREATE SCHEMA public; COMMIT;'
DATABASE_DIRECT_URL="$database_admin_url" go run ./cmd/migrate

psql "$database_admin_url" -v ON_ERROR_STOP=1 -f infra/postgres/roles.sql
