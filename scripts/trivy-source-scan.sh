#!/usr/bin/env bash
set -euo pipefail
root_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$root_dir"
# .tmp is the existing ignored local build/module cache, not application source.
# Go toolchain fixtures downloaded there must not become application findings.
# Fail rather than silently hide any tracked source in that excluded directory.
tracked_cache="$(git ls-files -- .tmp)"
if [[ -n "$tracked_cache" ]]; then
  printf 'Refusing cache exclusion: tracked files exist under .tmp.\n' >&2
  exit 1
fi
# Do not exclude owned code, infrastructure, archived patches or dependency
# manifests. Findings still fail the gate at every severity. The manifests are
# also checked independently by govulncheck, OSV and the package audit.
exec trivy fs --exit-code 1 --scanners vuln,secret,misconfig --skip-dirs .tmp .
