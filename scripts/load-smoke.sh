#!/usr/bin/env bash
set -euo pipefail

: "${BASE_URL:=http://localhost:8080}"
if ! command -v k6 >/dev/null 2>&1; then printf '%s\n' 'k6 is not installed; load smoke skipped.' >&2; exit 2; fi
k6 run --env BASE_URL="$BASE_URL" tests/load/smoke.js
