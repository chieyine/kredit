#!/usr/bin/env bash
set -euo pipefail

root_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$root_dir"
source "$root_dir/scripts/load-env.sh"
load_env_defaults "$root_dir/.env"

database_admin_url="${DATABASE_DIRECT_URL:-${DATABASE_URL:-}}"
: "${database_admin_url:?DATABASE_DIRECT_URL or DATABASE_URL is required}"
role_admin_url="${DATABASE_ROLE_ADMIN_URL:-$database_admin_url}"

app_password="${KREDIT_APP_DB_PASSWORD:-kredit-app-development-only}"
worker_password="${KREDIT_WORKER_DB_PASSWORD:-kredit-worker-development-only}"
backup_password="${KREDIT_BACKUP_DB_PASSWORD:-kredit-backup-development-only}"

psql "$role_admin_url" -X -v ON_ERROR_STOP=1 -f infra/postgres/roles.sql
psql "$role_admin_url" -X -v ON_ERROR_STOP=1 \
  -v app_login_password="$app_password" \
  -v worker_login_password="$worker_password" \
  -v backup_login_password="$backup_password" \
  -v demote_owner=true \
  -f infra/postgres/development-logins.sql

printf '%s\n' 'Development database roles configured.'
