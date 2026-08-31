#!/usr/bin/env bash

# Load a dotenv file as defaults without replacing values already supplied by
# the caller. This keeps CI, release, and one-off operator configuration above
# local developer configuration. The parser deliberately supports the simple
# KEY=value contract used by this repository and never evaluates file content.
load_env_defaults() {
	local env_file="${1:-.env}"
	local line key value

	[[ -f "$env_file" ]] || return 0

	while IFS= read -r line || [[ -n "$line" ]]; do
		line="${line%$'\r'}"
		[[ "$line" =~ ^[[:space:]]*$ || "$line" =~ ^[[:space:]]*# ]] && continue
		line="${line#export }"
		[[ "$line" == *=* ]] || continue

		key="${line%%=*}"
		value="${line#*=}"
		key="${key#"${key%%[![:space:]]*}"}"
		key="${key%"${key##*[![:space:]]}"}"
		[[ "$key" =~ ^[A-Za-z_][A-Za-z0-9_]*$ ]] || {
			printf 'Invalid environment key in %s: %s\n' "$env_file" "$key" >&2
			return 1
		}

		# An explicitly supplied value, including an empty one, always wins.
		declare -p "$key" >/dev/null 2>&1 && continue
		if [[ ${#value} -ge 2 && "$value" == \"*\" && "$value" == *\" ]]; then
			value="${value:1:${#value}-2}"
		elif [[ ${#value} -ge 2 && "$value" == \'*\' && "$value" == *\' ]]; then
			value="${value:1:${#value}-2}"
		fi
		printf -v "$key" '%s' "$value"
		export "$key"
	done < "$env_file"
}
