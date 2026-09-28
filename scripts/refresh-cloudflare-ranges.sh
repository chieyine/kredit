#!/usr/bin/env bash
set -euo pipefail

# Generate a validated candidate by default. --apply activates it in the
# standalone Compose deployment, whose Caddy admin API is disabled.
root_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
apply=false
if [[ "${1:-}" == --apply ]]; then apply=true; shift; fi
caddyfile="${1:-$root_dir/infra/environments/Caddyfile.prod}"
[[ $# -le 1 && -f "$caddyfile" ]] || { printf 'Usage: %s [--apply] [Caddyfile]\n' "$0" >&2; exit 1; }
caddyfile="$(cd "$(dirname "$caddyfile")" && pwd)/$(basename "$caddyfile")"
deployment_dir="$root_dir/infra/environments"
compose=(docker compose --env-file "$deployment_dir/.env.production" -f "$deployment_dir/docker-compose.prod.yml")
# Upgraded installations pin their images in the generated release override.
if [[ -f "$deployment_dir/docker-compose.release.yml" ]]; then
  compose+=(-f "$deployment_dir/docker-compose.release.yml")
fi
compose+=(--profile standalone)
if $apply && [[ "$caddyfile" != "$root_dir/infra/environments/Caddyfile.prod" ]]; then
  printf 'Activation supports only the production Compose Caddyfile.\n' >&2; exit 1
fi

# Serialize the full read/validate/activate sequence.
lock_dir="$caddyfile.refresh-lock"
mkdir "$lock_dir" || { printf 'Another refresh is active.\n' >&2; exit 1; }
tmp_dir="$(mktemp -d)" || { rmdir "$lock_dir"; exit 1; }
trap 'rm -rf "$tmp_dir"; rmdir "$lock_dir"' EXIT
curl --fail --silent --show-error --max-time 30 https://www.cloudflare.com/ips-v4 > "$tmp_dir/v4"
curl --fail --silent --show-error --max-time 30 https://www.cloudflare.com/ips-v6 > "$tmp_dir/v6"
python3 - "$caddyfile" "$tmp_dir" <<'PY'
import ipaddress
import pathlib
import sys

target, work = pathlib.Path(sys.argv[1]), pathlib.Path(sys.argv[2])
ranges = []
for family in (4, 6):
    lines = (work / f'v{family}').read_text().splitlines()
    if not lines:
        raise SystemExit(f'No IPv{family} ranges returned')
    for line in lines:
        network = ipaddress.ip_network(line.strip(), strict=True)
        if network.version != family or network.prefixlen == 0:
            raise SystemExit(f'Unexpected IPv{family} range')
        ranges.append(str(network))
if len(ranges) < 15 or len(set(ranges)) != len(ranges):
    raise SystemExit('Incomplete or duplicate proxy ranges')
lines = target.read_text().splitlines(keepends=True)
starts = [i for i, line in enumerate(lines) if '# BEGIN CLOUDFLARE RANGES' in line]
ends = [i for i, line in enumerate(lines) if '# END CLOUDFLARE RANGES' in line]
if len(starts) != 1 or len(ends) != 1 or starts[0] >= ends[0]:
    raise SystemExit('Expected one ordered pair of proxy-range markers')
candidate = lines[:starts[0]+1] + ['        trusted_proxies static ' + ' '.join(ranges) + '\n'] + lines[ends[0]:]
(work / 'candidate').write_text(''.join(candidate))
PY

# Validate with the deployed image and certificate mounts before replacement.
"${compose[@]}" run --rm --no-deps -v "$tmp_dir/candidate:/etc/caddy/Caddyfile:ro" caddy \
  caddy validate --config /etc/caddy/Caddyfile --adapter caddyfile
if ! $apply; then
  cp "$tmp_dir/candidate" "$caddyfile.candidate"
  printf 'Validated candidate: %s.candidate\nRun with --apply to back up and activate it.\n' "$caddyfile"
  exit 0
fi

# Preserve the bind-mounted inode and retain the previous configuration.
backup="$caddyfile.backup.$(date -u +%Y%m%dT%H%M%SZ)"
cp -p "$caddyfile" "$backup"
if ! cat "$tmp_dir/candidate" > "$caddyfile" || ! "${compose[@]}" restart caddy; then
  cat "$backup" > "$caddyfile"
  "${compose[@]}" restart caddy || true
  printf 'Activation failed; restored %s. Inspect the proxy before retrying.\n' "$backup" >&2
  exit 1
fi
printf 'Activated validated proxy ranges. Previous configuration: %s\n' "$backup"
