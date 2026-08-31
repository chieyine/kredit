#!/usr/bin/env bash
set -euo pipefail

root_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
source "$root_dir/scripts/load-env.sh"

fixture="$(mktemp "${TMPDIR:-/tmp}/kredit-env.XXXXXX")"
trap 'rm -f "$fixture"' EXIT

printf '%s\n' \
	'KREDIT_EXISTING=file-value' \
	'KREDIT_DEFAULT=default-value' \
	'KREDIT_QUOTED="value with spaces"' \
	'KREDIT_EQUALS=left=right' > "$fixture"

export KREDIT_EXISTING='caller-value'
load_env_defaults "$fixture"

[[ "$KREDIT_EXISTING" == 'caller-value' ]]
[[ "$KREDIT_DEFAULT" == 'default-value' ]]
[[ "$KREDIT_QUOTED" == 'value with spaces' ]]
[[ "$KREDIT_EQUALS" == 'left=right' ]]

printf '%s\n' 'Environment default loading passed.'
