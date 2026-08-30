#!/usr/bin/env bash
set -euo pipefail

root_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$root_dir"

if [[ -f .env ]]; then
	set -a
	source .env
	set +a
fi

docker compose up -d postgres minio mailpit provider-simulator

tmp_dir="${TMPDIR:-/tmp}/kredit-dev"
mkdir -p "$tmp_dir"

cleanup() {
  jobs -pr | xargs -r kill 2>/dev/null || true
}
trap cleanup EXIT INT TERM

go run ./cmd/api >"$tmp_dir/api.log" 2>&1 &
go run ./cmd/worker >"$tmp_dir/worker.log" 2>&1 &
# Bind the dev server to loopback unless explicitly exposed; the Vite proxy
# reaches the API on this machine either way.
pnpm --dir web dev --host "${DEV_WEB_HOST:-127.0.0.1}"
