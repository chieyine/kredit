#!/usr/bin/env bash
set -euo pipefail

if [[ -f .env ]]; then
	set -a
	source .env
	set +a
fi

if [[ "${ALLOW_DB_RESET:-false}" != "true" ]]; then
  printf '%s\n' 'Refusing reset. Set ALLOW_DB_RESET=true for an explicit local reset.' >&2
  exit 1
fi

: "${DATABASE_URL:=postgres://kredit:kredit@localhost:5432/kredit?sslmode=disable}"
psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -c 'DROP SCHEMA public CASCADE; CREATE SCHEMA public;'
go run ./cmd/migrate
