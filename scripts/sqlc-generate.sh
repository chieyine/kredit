#!/usr/bin/env bash
set -euo pipefail

sqlc_version="${SQLC_VERSION:-v1.31.0}"
if command -v sqlc >/dev/null 2>&1; then
  sqlc generate
  exit 0
fi

# Go-installed developer tools are not always on PATH in desktop/CI shells.
# Prefer the pinned binary when it is already present before attempting a
# network-backed module execution.
if command -v go >/dev/null 2>&1; then
  gobin="$(go env GOPATH 2>/dev/null)/bin/sqlc"
  if [[ -x "$gobin" ]]; then
    "$gobin" generate
    exit 0
  fi
fi

# Keep generation reproducible on clean machines without requiring a global
# install. Go verifies the module checksum and caches the tool for subsequent
# runs. This path needs network access only on the first invocation.
if command -v go >/dev/null 2>&1; then
  if go run "github.com/sqlc-dev/sqlc/cmd/sqlc@${sqlc_version}" generate; then
    exit 0
  fi
fi

if [[ "${SQLC_ALLOW_SKIP:-0}" == "1" ]]; then
  printf '%s\n' 'sqlc is unavailable; generation was explicitly skipped with SQLC_ALLOW_SKIP=1.'
  exit 0
fi
printf '%s\n' "sqlc ${sqlc_version} is required; install it or rerun with network access." >&2
exit 1
