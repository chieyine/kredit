#!/usr/bin/env bash
set -euo pipefail

root_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$root_dir"

# Both host-run entry points need the same database and upload prerequisites.
docker compose up -d --wait --wait-timeout 120 postgres minio mailpit provider-simulator
docker compose run --rm --no-deps minio-init
