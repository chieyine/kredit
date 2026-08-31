#!/usr/bin/env bash
set -euo pipefail

: "${BASE_URL:?Set BASE_URL to the production HTTPS origin.}"
: "${LEGAL_ENTITY_NAME:?Set LEGAL_ENTITY_NAME to the approved public operator name.}"

if [[ ! "$BASE_URL" =~ ^https:// ]] || [[ "$BASE_URL" =~ (localhost|127\.0\.0\.1|\[::1\]) ]]; then
	printf 'BASE_URL must be a non-local HTTPS origin.\n' >&2
	exit 1
fi

check_dir="$(mktemp -d)"
trap 'rm -rf "$check_dir"' EXIT

fetch() {
	local path="$1"
	local name="$2"
	curl --fail --silent --show-error --location --max-time 20 \
		--dump-header "$check_dir/$name.headers" \
		--output "$check_dir/$name.body" \
		"${BASE_URL%/}$path"
}

fetch / home
fetch /legal/privacy privacy
fetch /legal/terms terms
fetch /robots.txt robots
fetch /sitemap.xml sitemap

for header in 'strict-transport-security:' 'x-content-type-options: nosniff' 'x-frame-options: DENY'; do
	if ! grep -qiF "$header" "$check_dir/home.headers"; then
		printf 'Missing production response header: %s\n' "$header" >&2
		exit 1
	fi
done

for document in privacy terms; do
	if grep -qiF 'Complete pre-launch draft' "$check_dir/$document.body"; then
		printf '%s still shows pre-launch legal wording.\n' "$document" >&2
		exit 1
	fi
	if ! grep -qF "$LEGAL_ENTITY_NAME" "$check_dir/$document.body"; then
		printf '%s does not show the approved legal entity.\n' "$document" >&2
		exit 1
	fi
done

grep -qF 'Sitemap:' "$check_dir/robots.body"
grep -qF '<urlset' "$check_dir/sitemap.body"

printf 'Post-deployment check passed for %s.\n' "$BASE_URL"
