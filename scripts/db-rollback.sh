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

: "${DATABASE_URL:=postgres://kredit:kredit@localhost:5432/kredit?sslmode=disable}"
psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -c 'DROP TABLE IF EXISTS app_meta; DROP TABLE IF EXISTS schema_migrations;'
