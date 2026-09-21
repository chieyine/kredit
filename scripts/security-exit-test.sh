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
# Keep the release scope and fail-on-findings flags explicit.
[[ " $* " == *' --exit-code 1 '* ]] || exit 2
[[ " $* " == *' --scanners vuln,secret,misconfig '* ]] || exit 2
[[ " $* " == *' --skip-dirs .tmp '* ]] || exit 2
if [[ "$MOCK_FINDING" == 1 ]]; then
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
# A future tracked file must not be hidden by the build-cache exclusion.
mkdir -p "$scratch/project/scripts" "$scratch/project/.tmp"
cp "$root/scripts/trivy-source-scan.sh" "$scratch/project/scripts/"
git -C "$scratch/project" init --quiet
printf 'synthetic\n' > "$scratch/project/.tmp/tracked.txt"
git -C "$scratch/project" add --force .tmp/tracked.txt
if PATH="$scratch/bin:$PATH" MOCK_FINDING=0 bash "$scratch/project/scripts/trivy-source-scan.sh" > "$scratch/scope.log" 2>&1; then
  printf 'Tracked cache content was silently excluded.\n' >&2
  exit 1
fi
grep -q 'Refusing cache exclusion: tracked files exist under .tmp' "$scratch/scope.log"
printf '%s\n' 'Security exit and tracked-cache contracts passed with synthetic scanners (not a real security scan).'
