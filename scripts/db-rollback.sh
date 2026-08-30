#!/usr/bin/env bash
set -euo pipefail

if [[ -f .env ]]; then
	set -a
	source .env
	set +a
fi

if [[ "${ALLOW_DB_ROLLBACK:-false}" != "true" ]]; then
  printf '%s\n' 'Refusing rollback. Set ALLOW_DB_ROLLBACK=true for an explicit local rollback.' >&2
  exit 1
fi

: "${DATABASE_URL:=postgres://kredit:kredit@localhost:5432/kredit?sslmode=disable}"
psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -c 'DROP TABLE IF EXISTS app_meta; DROP TABLE IF EXISTS schema_migrations;'
