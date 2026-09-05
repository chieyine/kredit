#!/usr/bin/env bash
set -euo pipefail

export GOCACHE="${GOCACHE:-$PWD/.tmp/go-cache}"
mkdir -p "$GOCACHE"
export XDG_CACHE_HOME="${XDG_CACHE_HOME:-$PWD/.tmp/cache}"
mkdir -p "$XDG_CACHE_HOME"

SECURITY_STRICT="${SECURITY_STRICT:-0}"

# Go-installed scanners commonly live in GOPATH/bin, which is not always on
# PATH in desktop shells or minimal CI runners.
if command -v go >/dev/null 2>&1; then
  gobin="$(go env GOPATH 2>/dev/null)/bin"
  if [[ -d "$gobin" ]]; then
    PATH="$gobin:$PATH"
    export PATH
  fi
fi

run_scanner() {
	local name="$1"
	shift
	if command -v "$name" >/dev/null 2>&1; then
		output_file="$(mktemp)"
		if "$@" >"$output_file" 2>&1; then
			cat "$output_file"
			rm -f "$output_file"
			return
		fi
		# A scanner compiled with an older Go patch release cannot analyze a
		# newer module. Treat that as an unavailable tool in best-effort local
		# checks, while the strict release gate still fails closed.
		if [[ "$SECURITY_STRICT" != "1" ]] && grep -q -E 'Staticcheck was built with|module requires at least go' "$output_file"; then
			printf '%s\n' "$name is incompatible with the active Go toolchain; scan skipped (set SECURITY_STRICT=1 to fail closed)."
			rm -f "$output_file"
			return
		fi
		cat "$output_file" >&2
		rm -f "$output_file"
		return 1
	fi
	if [[ "$SECURITY_STRICT" == "1" ]]; then
		printf '%s\n' "$name is required when SECURITY_STRICT=1." >&2
		exit 1
	fi
	printf '%s\n' "$name is not installed; scan skipped (set SECURITY_STRICT=1 to fail closed)."
}

go vet ./...

run_scanner govulncheck govulncheck ./...

# Generated sqlc files are historical generated artifacts and are not an
# appropriate source for credential-pattern findings. The remaining exclusions
# are narrow, reviewed scanner exceptions rather than global rule disables:
# - auth/store.go G115 serializes the low byte of a monotonically shifted TOTP
#   counter; G505 is HMAC-SHA1 required for broad RFC 6238 authenticator
#   compatibility, not password hashing or a signature primitive.
# - auth_handlers.go G124 flags the intentionally browser-readable CSRF cookie
#   and cannot infer that Secure=true outside development. The session cookie
#   remains HttpOnly and both cookies are SameSite=Lax.
# - the G705 paths write attachment/plain-text responses with explicit safe
#   Content-Types; none of those bytes are rendered as HTML by Kredit.
GOSEC_PATH_EXCLUSIONS='internal/auth/store.go:G115,G505;internal/web/auth_handlers.go:G124;internal/web/user_control_handlers.go:G705;internal/web/reports_handlers.go:G705;internal/web/credit_handlers.go:G705'
run_scanner gosec gosec -exclude-generated --exclude-rules="$GOSEC_PATH_EXCLUSIONS" ./...

run_scanner staticcheck staticcheck ./...
run_scanner osv-scanner osv-scanner scan source -r .
run_scanner trivy trivy fs --scanners vuln,secret,misconfig .

if command -v rg >/dev/null 2>&1; then
	if rg -n --hidden --glob '!.tmp/**' --glob '!node_modules/**' --glob '!.git/**' --glob '!README.md' --glob '!IMPLEMENTATION_PLAN.md' --glob '!*.lock' '(BEGIN (RSA|OPENSSH) PRIVATE KEY|AKIA[0-9A-Z]{16}|password\s*=\s*"[^"$]+")' .; then
		printf '%s\n' 'Potential secret material found.' >&2
		exit 1
	fi
else
	if grep -rnE --exclude-dir={.tmp,node_modules,.git} --exclude={README.md,IMPLEMENTATION_PLAN.md,'*.lock'} '(BEGIN (RSA|OPENSSH) PRIVATE KEY|AKIA[0-9A-Z]{16}|password[[:space:]]*=[[:space:]]*"[^"$]+")' .; then
		printf '%s\n' 'Potential secret material found.' >&2
		exit 1
	fi
fi
