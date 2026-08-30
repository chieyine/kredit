#!/usr/bin/env bash
set -euo pipefail

if [[ -d web/node_modules ]] && pnpm --dir web exec playwright --version >/dev/null 2>&1; then
  pnpm --dir web exec playwright test
else
	printf '%s\n' 'Playwright dependencies are not installed.' >&2
	exit 1
fi
