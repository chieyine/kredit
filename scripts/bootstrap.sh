#!/usr/bin/env bash
set -euo pipefail

root_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$root_dir"

if [[ ! -f .env ]]; then
  cp .env.example .env
  printf 'Created .env from .env.example; review development values before use.\n'
fi

set -a
source .env
set +a

printf '%s\n' 'Checking local toolchain...'
bash scripts/check-tools.sh

printf '%s\n' 'Starting local infrastructure...'
docker compose up -d postgres minio mailpit provider-simulator

printf '%s\n' 'Preparing Go dependencies and schema...'
go mod download
pnpm install --frozen-lockfile
go run ./cmd/migrate
go run ./cmd/seed

printf '%s\n' 'Running available generators...'
bash scripts/generate.sh

printf '%s\n' 'Bootstrap complete.'
