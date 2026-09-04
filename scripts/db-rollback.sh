#!/usr/bin/env bash
set -euo pipefail

root_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$root_dir"
source "$root_dir/scripts/load-env.sh"
load_env_defaults "$root_dir/.env"

if [[ "${ALLOW_DB_ROLLBACK:-false}" != "true" ]]; then
  printf '%s\n' 'Refusing rollback. Set ALLOW_DB_ROLLBACK=true for an explicit local rollback.' >&2
  exit 1
fi

case "${APP_ENV:-}" in
  development|test) ;;
  *) printf '%s\n' 'Rollback requires APP_ENV=development or APP_ENV=test.' >&2; exit 1 ;;
esac

database_admin_url="${DATABASE_DIRECT_URL:-${DATABASE_URL:-}}"
: "${database_admin_url:?DATABASE_DIRECT_URL or DATABASE_URL is required for rollback}"
DATABASE_DIRECT_URL="$database_admin_url" go run ./cmd/migrate down
