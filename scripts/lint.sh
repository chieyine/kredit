#!/usr/bin/env bash
set -euo pipefail

export GOCACHE="${GOCACHE:-$PWD/.tmp/go-cache}"
mkdir -p "$GOCACHE"

format_output="$(gofmt -l .)"
if [[ -n "$format_output" ]]; then
  printf '%s\n' "$format_output"
  printf '%s\n' 'Go files require formatting.' >&2
  exit 1
fi

go vet ./...

# .golangci.yml enables errcheck, staticcheck, unused and ineffassign. Skipping
# them silently means the repository is configured for checks that never run, so
# CI and release builds fail closed when the linter is missing.
if command -v golangci-lint >/dev/null 2>&1; then
  golangci-lint run
  exit 0
fi

if [[ "${CI:-0}" == "true" || "${CI:-0}" == "1" || "${SECURITY_STRICT:-0}" == "1" ]]; then
  printf '%s\n' 'golangci-lint is required for CI and release builds; install it from https://golangci-lint.run.' >&2
  exit 1
fi

printf '%s\n' 'golangci-lint is not installed; gofmt and go vet completed. Install it before opening a pull request.'
