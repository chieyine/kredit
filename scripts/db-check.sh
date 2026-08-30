#!/usr/bin/env bash
set -euo pipefail

if [[ -f .env ]]; then
	set -a
	source .env
	set +a
fi

: "${DATABASE_URL:=postgres://kredit:kredit@localhost:5432/kredit?sslmode=disable}"
psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -c 'SELECT current_database(), NOW();'
