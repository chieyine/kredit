#!/usr/bin/env bash
set -euo pipefail
root_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
command -v trivy >/dev/null
fixture="$(mktemp -d)"
trap 'rm -rf "$fixture"' EXIT
mkdir "$fixture/source"
# Assemble synthetic tokens so this test does not introduce additional scanner
# findings into the repository itself. No provider accepts or sees these values.
printf 'token="sk_%s_%s"\n' live verysecretkey1234 > "$fixture/source/known-live.txt"
printf 'token="sk_%s_%s"\n' test abc1234567890xyz > "$fixture/source/known-test.txt"
printf 'token="sk_%s_%s"\n' live verysecretkey1235 > "$fixture/source/near-match.txt"
printf 'token="sk_%s_%s"\n' live unlistedcredential1234 > "$fixture/source/unlisted.txt"
status=0
trivy fs --exit-code 1 --scanners secret --format json --secret-config "$root_dir/trivy-secret.yaml" "$fixture/source" > "$fixture/result.json" 2> "$fixture/scanner.log" || status=$?
if [[ "$status" -ne 1 ]]; then
  cat "$fixture/scanner.log" >&2
  printf 'Expected real scanner findings (status 1), got %s.\n' "$status" >&2
  exit 1
fi
python3 - "$fixture/result.json" <<'PY'
import json
import pathlib
import sys
with open(sys.argv[1], encoding="utf-8") as handle:
    report = json.load(handle)
found = set()
for result in report.get("Results", []):
    if any(secret.get("RuleID") == "stripe-secret-token" for secret in result.get("Secrets", [])):
        found.add(pathlib.Path(result["Target"]).name)
expected = {"near-match.txt", "unlisted.txt"}
if found != expected:
    raise SystemExit(f"Exact-fixture exception changed scanner coverage: expected {sorted(expected)}, got {sorted(found)}")
print("Real Trivy scanner suppresses only the two exact historical fixtures; altered and unlisted tokens remain findings.")
PY
