#!/usr/bin/env bash
set -euo pipefail

# Regenerates the trusted_proxies list in infra/environments/Caddyfile.prod
# from Cloudflare's published ranges, then reloads Caddy.
#
# Why this exists: the list was hardcoded. With trusted_proxies_strict, a range
# Cloudflare adds later stops being trusted, {client_ip} silently becomes
# Cloudflare's address, and every user behind it shares one rate-limit key.
# The symptom is users locked out of sign-in with nothing in the logs.
#
# Run weekly from cron. Exits non-zero without touching the file if either
# fetch fails, so a network blip cannot empty the trust list.

caddyfile="${1:-infra/environments/Caddyfile.prod}"
[[ -f "$caddyfile" ]] || { printf 'Caddyfile not found: %s\n' "$caddyfile" >&2; exit 1; }

v4="$(curl --fail --silent --show-error --max-time 30 https://www.cloudflare.com/ips-v4)"
v6="$(curl --fail --silent --show-error --max-time 30 https://www.cloudflare.com/ips-v6)"

ranges="$(printf '%s\n%s\n' "$v4" "$v6" | grep -E '^[0-9a-fA-F:.]+/[0-9]+$' | tr '\n' ' ' | sed 's/ $//')"
# A short list means the fetch returned something unexpected. Refuse rather
# than narrowing the trust list to whatever parsed.
count="$(printf '%s' "$ranges" | wc -w)"
[[ "$count" -ge 15 ]] || { printf 'Refusing: only %s ranges parsed.\n' "$count" >&2; exit 1; }

tmp="$(mktemp)"; trap 'rm -f "$tmp"' EXIT
awk -v repl="        trusted_proxies static $ranges" '
  /# BEGIN CLOUDFLARE RANGES/ { print; print repl; skip = 1; next }
  /# END CLOUDFLARE RANGES/   { skip = 0 }
  !skip                        { print }
' "$caddyfile" > "$tmp"

grep -q 'trusted_proxies static' "$tmp" || { printf 'Refusing: generated file has no trusted_proxies line.\n' >&2; exit 1; }
mv "$tmp" "$caddyfile"; trap - EXIT
printf 'Updated %s with %s Cloudflare ranges.\n' "$caddyfile" "$count"

if command -v caddy >/dev/null 2>&1; then
  caddy validate --config "$caddyfile" --adapter caddyfile
  caddy reload --config "$caddyfile" --adapter caddyfile 2>/dev/null \
    || printf 'Validated, but reload failed (not running here?).\n' >&2
fi
