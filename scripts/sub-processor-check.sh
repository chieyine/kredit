#!/usr/bin/env bash
set -euo pipefail

# Inventory production-source HTTPS references. This is a conservative static
# check, not a proof of every dynamically configured outbound destination.
# Negative authorization tests must not become registered production processors.
root_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$root_dir"
register="docs/compliance/sub-processors.md"
[[ -s "$register" ]] || { printf 'sub-processor register is missing: %s\n' "$register" >&2; exit 1; }
[[ -d internal && -d cmd ]] || { printf 'production source directories are missing\n' >&2; exit 1; }

# Reserved/test domains and owned services are not third-party processors.
# The exact documentation hosts below appear in explanatory source comments;
# api.paystack.co and the other actual API hosts are NOT exempt.
ignore='(^|\.)(example|test|invalid|localhost)$|(^|\.)example\.(com|net|org)$|(^|\.)kredit\.(ng|test)$|^(localhost|opentelemetry\.io|docs\.mono\.co|developer\.flutterwave\.com|paystack\.com)$'
raw="$(mktemp)"
trap 'rm -f "$raw"' EXIT
status=0
grep -rhoE 'https://[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}' --include='*.go' --exclude='*_test.go' internal cmd > "$raw" || status=$?
# grep status 1 means no matches; every actual read/tool error fails closed.
if [[ "$status" -gt 1 ]]; then
  printf 'Unable to inspect production HTTPS references.\n' >&2
  exit "$status"
fi
hosts=()
while IFS= read -r host; do
  [[ -n "$host" ]] && hosts+=("$host")
done < <(sed 's|https://||' "$raw" | tr '[:upper:]' '[:lower:]' | sort -u | grep -Ev "$ignore" || true)
missing=0
for host in "${hosts[@]}"; do
  if ! grep -Fq -- "\`$host\`" "$register"; then
    printf 'undeclared outbound host: %s\n' "$host" >&2
    printf '  record its processor, data sent and transfer evidence in %s\n' "$register" >&2
    missing=$((missing + 1))
  fi
done
if [[ "$missing" -ne 0 ]]; then
  printf 'Sub-processor check failed with %d undeclared host(s).\n' "$missing" >&2
  exit 1
fi
printf 'Sub-processor register covers production Go HTTPS references (%d checked; test files excluded).\n' "${#hosts[@]}"
