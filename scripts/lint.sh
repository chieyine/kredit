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

if command -v golangci-lint >/dev/null 2>&1; then
  golangci-lint run
else
  printf '%s\n' 'golangci-lint is not installed; gofmt and go vet completed.'
fi
