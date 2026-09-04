#!/usr/bin/env bash
set -euo pipefail

root_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$root_dir"
source "$root_dir/scripts/load-env.sh"
load_env_defaults "$root_dir/.env"

: "${DATABASE_URL:?DATABASE_URL is required; set it explicitly or configure .env}"
psql "$DATABASE_URL" -X -v ON_ERROR_STOP=1 -c "SELECT current_database(), current_user, session_user, current_setting('server_version') AS server_version, inet_server_port() AS server_port, NOW();"
