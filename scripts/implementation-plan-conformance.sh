#!/usr/bin/env bash
set -euo pipefail

root_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$root_dir"

manifest="${1:-docs/product/workstream-evidence.tsv}"
plan="docs/product/readme-completion-plan.md"
gap_audit="docs/operations/readme-gap-audit.md"

expected_ids=(
  FOUNDATION-CONTRACT-LOCK
  TL-DRAWDOWN
  SUPPLIER-ONBOARDING
  NOTIFICATION-PREFERENCES
  ACCOUNT-RECOVERY
  PRIVACY-RIGHTS
  DATA-INVENTORY
  OPS-CONTROLS
  INTERACTIVE-FLOWS
  WCAG-AA
  PRODUCT-ANALYTICS
)

failures=0
fail() {
  printf 'implementation plan conformance: %s\n' "$1" >&2
  failures=$((failures + 1))
}

if [[ ! -f "$manifest" ]]; then
  fail "missing evidence manifest $manifest"
  exit 1
fi

header=$'workstream_id\twave\tstatus\tproduct_owner\tengineering_owner\tcompliance_owner\toperations_owner\tcode_evidence\ttest_evidence\treviewed_at'
first_line="$(sed -n '1p' "$manifest")"
if [[ "$first_line" != "$header" ]]; then
  fail "evidence manifest header does not match the locked schema"
fi

seen_ids=""
while IFS=$'\t' read -r id wave status product_owner engineering_owner compliance_owner operations_owner code_evidence test_evidence reviewed_at extra; do
  [[ "$id" == "workstream_id" || -z "$id" ]] && continue
  if [[ -n "${extra:-}" ]]; then
    fail "$id has more than ten columns"
  fi
  if grep -Fqx -- "$id" <<< "$seen_ids"; then
    fail "$id appears more than once"
  fi
  seen_ids="${seen_ids}${id}"$'\n'

  case "$status" in
    not_started|in_progress|complete) ;;
    *) fail "$id has invalid status $status" ;;
  esac
  [[ "$wave" =~ ^[0-6]$ ]] || fail "$id has invalid wave $wave"

  for owner in "$product_owner" "$engineering_owner" "$compliance_owner" "$operations_owner"; do
    if [[ -z "$owner" || "$owner" == "-" || "$owner" == "TBD" ]]; then
      fail "$id is missing an accountable owner"
    fi
  done

  if [[ "$status" == "complete" ]]; then
    if [[ -z "$code_evidence" || "$code_evidence" == "-" ]]; then
      fail "$id is complete without code/document evidence"
    fi
    if [[ -z "$test_evidence" || "$test_evidence" == "-" ]]; then
      fail "$id is complete without test evidence"
    fi
    [[ "$reviewed_at" =~ ^[0-9]{4}-[0-9]{2}-[0-9]{2}$ ]] || fail "$id is complete without a review date"

    for evidence_group in "$code_evidence" "$test_evidence"; do
      IFS=';' read -ra evidence_paths <<< "$evidence_group"
      for evidence_path in "${evidence_paths[@]}"; do
        if [[ ! -e "$evidence_path" ]]; then
          fail "$id references missing evidence $evidence_path"
        fi
      done
    done
  fi

  if ! grep -Fq -- "$id" "$plan"; then
    fail "$id is missing from the implementation plan"
  fi
  if [[ "$id" != "FOUNDATION-CONTRACT-LOCK" ]] && ! grep -Fq -- "$id" "$gap_audit"; then
    fail "$id is missing from the gap audit"
  fi
done < "$manifest"

for id in "${expected_ids[@]}"; do
  if ! grep -Fqx -- "$id" <<< "$seen_ids"; then
    fail "$id is missing from the evidence manifest"
  fi
done

if [[ "$failures" -ne 0 ]]; then
  printf 'Implementation plan conformance failed with %d issue(s).\n' "$failures" >&2
  exit 1
fi

printf 'Implementation plan conformance passed.\n'
