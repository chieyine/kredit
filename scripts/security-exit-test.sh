#!/usr/bin/env bash
# Exercise the release script's exit contract without network calls or scanners.
set -euo pipefail
root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
scratch="$(mktemp -d)"
trap 'rm -rf "$scratch"' EXIT
mkdir -p "$scratch/bin" "$scratch/work" "$scratch/cache"
for tool in govulncheck gosec staticcheck osv-scanner; do
  printf '#!/usr/bin/env bash\nexit 0\n' > "$scratch/bin/$tool"
done
cat > "$scratch/bin/go" <<'MOCK'
#!/usr/bin/env bash
if [[ "${1:-}" == env ]]; then printf '%s\n' "$MOCK_GOPATH"; fi
exit 0
MOCK
cat > "$scratch/bin/trivy" <<'MOCK'
#!/usr/bin/env bash
# Model Trivy's documented zero-by-default finding status.
if [[ "$MOCK_FINDING" == 1 && " $* " == *' --exit-code 1 '* ]]; then
  printf '%s\n' 'Synthetic detected finding.'
  exit 1
fi
exit 0
MOCK
chmod +x "$scratch/bin/"*
for finding in 1 0; do
  status=0
  (
    cd "$scratch/work"
    PATH="$scratch/bin:$PATH" MOCK_GOPATH="$scratch/go" MOCK_FINDING="$finding" \
      GOCACHE="$scratch/cache" XDG_CACHE_HOME="$scratch/cache" SECURITY_STRICT=1 \
      bash "$root/scripts/security.sh"
  ) > "$scratch/result.log" 2>&1 || status=$?
  if [[ "$status" != "$finding" ]]; then
    cat "$scratch/result.log" >&2
    printf 'Security exit contract: finding=%s status=%s\n' "$finding" "$status" >&2
    exit 1
  fi
done
printf '%s\n' 'Security exit contract passed with synthetic scanners (not a real security scan).'
