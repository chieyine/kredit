#!/usr/bin/env bash
set -euo pipefail
root_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
command -v trivy >/dev/null
fixture="$(mktemp -d)"
trap 'rm -rf "$fixture"' EXIT
mkdir "$fixture/source"
# Neutral filenames are deliberate: Trivy's built-in allow rules skip paths
# containing '-test'. Prove detection before exercising our own exceptions.
# Assemble inert credentials to avoid introducing new repository findings.
printf ' secretVal := "sk_%s_%s"\n' live verysecretkey1234 > "$fixture/source/archived-a.txt"
printf ' "secret": "sk_%s_%s",\n' test abc1234567890xyz > "$fixture/source/archived-b.txt"
printf ' token="sk_%s_%s"\n' live verysecretkey1235 > "$fixture/source/near-match.txt"
printf ' token="sk_%s_%s"\n' live verysecretkey1234a > "$fixture/source/extended.txt"
printf ' token="sk_%s_%s"\n' live unlistedcredential1234 > "$fixture/source/unlisted.txt"
printf '{}\n' > "$fixture/baseline.yaml"
for mode in baseline configured; do
  config="$fixture/baseline.yaml"
  [[ "$mode" == baseline ]] || config="$root_dir/trivy-secret.yaml"
  status=0
  trivy fs --exit-code 1 --scanners secret --format json --secret-config "$config" "$fixture/source" > "$fixture/$mode.json" 2> "$fixture/$mode.log" || status=$?
  if [[ "$status" -ne 1 ]]; then
    cat "$fixture/$mode.log" >&2
    printf 'Expected real scanner findings for %s (status 1), got %s.\n' "$mode" "$status" >&2
    exit 1
  fi
done
python3 - "$fixture/baseline.json" "$fixture/configured.json" <<'PY'
import json
import pathlib
import sys

def detected(path):
    with open(path, encoding="utf-8") as handle:
        report = json.load(handle)
    found = set()
    for result in report.get("Results", []):
        if any(secret.get("RuleID") == "stripe-secret-token" for secret in result.get("Secrets", [])):
            found.add(pathlib.Path(result["Target"]).name)
    return found

negative = {"near-match.txt", "extended.txt", "unlisted.txt"}
baseline_expected = negative | {"archived-a.txt", "archived-b.txt"}
for path, expected in ((sys.argv[1], baseline_expected), (sys.argv[2], negative)):
    actual = detected(path)
    if actual != expected:
        raise SystemExit(f"Scanner coverage mismatch: expected {sorted(expected)}, got {sorted(actual)}")
print("Real Trivy baseline detects all five fixtures; only the two exact archived values are excluded by the repository configuration.")
PY
