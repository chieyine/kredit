#!/usr/bin/env bash
set -euo pipefail

if [[ -d web/node_modules ]]; then
  pnpm --dir web test
else
	printf '%s\n' 'Frontend dependencies are not installed.' >&2
	exit 1
fi
