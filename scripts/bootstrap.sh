#!/usr/bin/env bash
set -euo pipefail

root_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$root_dir"

if [[ ! -f .env ]]; then
  cp .env.example .env
  printf 'Created .env from .env.example; review development values before use.\n'
fi

source "$root_dir/scripts/load-env.sh"
load_env_defaults "$root_dir/.env"

printf '%s\n' 'Checking local toolchain...'
bash scripts/check-tools.sh

printf '%s\n' 'Starting local infrastructure...'
docker compose up -d postgres minio mailpit provider-simulator

printf '%s\n' 'Preparing Go dependencies and schema...'
go mod download
pnpm install --frozen-lockfile
go run ./cmd/migrate
bash scripts/configure-development-database.sh
go run ./cmd/seed

printf '%s\n' 'Running available generators...'
bash scripts/generate.sh

printf '%s\n' 'Bootstrap complete.'
