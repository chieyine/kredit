#!/usr/bin/env bash
set -euo pipefail

# Every external host Kredit's own code sends data to must be recorded in the
# sub-processor register. The field-level data inventory proves what is stored;
# it cannot see a new outbound transfer, because calling a third party adds no
# database column. That is exactly how an undeclared model provider reached
# production, so this gate closes the gap from the other side.

root_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$root_dir"

register="docs/compliance/sub-processors.md"
[[ -s "$register" ]] || { printf 'sub-processor register is missing: %s\n' "$register" >&2; exit 1; }

# Test and documentation hosts are not transfers. Reserved TLDs (RFC 2606 and
# RFC 6761) plus Kredit's own domains are the only exemptions, so a real
# provider can never be excused by naming.
ignore='(^|\.)(example|test|invalid|localhost)$|(^|\.)example\.(com|net|org)$|(^|\.)kredit\.(ng|test)$|^(localhost|opentelemetry\.io|docs\.mono\.co|developer\.flutterwave\.com)$'

hosts=()
while IFS= read -r line; do
  [[ -n "$line" ]] && hosts+=("$line")
done < <(
  grep -rhoE 'https://[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}' --include='*.go' internal cmd \
    | sed 's|https://||' \
    | sort -u \
    | grep -Ev "$ignore" || true
)

missing=0
for host in "${hosts[@]}"; do
  [[ -z "$host" ]] && continue
  if ! grep -Fq -- "\`$host\`" "$register"; then
    printf 'undeclared outbound host: %s\n' "$host" >&2
    printf '  add it to %s with its processor, the data sent and the transfer basis\n' "$register" >&2
    missing=$((missing + 1))
  fi
done

if [[ "$missing" -ne 0 ]]; then
  printf 'Sub-processor check failed with %d undeclared host(s).\n' "$missing" >&2
  exit 1
fi

printf 'Sub-processor register covers every outbound host in Go source (%d checked).\n' "${#hosts[@]}"
