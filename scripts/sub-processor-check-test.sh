#!/usr/bin/env bash
set -euo pipefail
root_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
fixture="$(mktemp -d)"
trap 'rm -rf "$fixture"' EXIT
mkdir -p "$fixture/scripts" "$fixture/internal/provider" "$fixture/cmd" "$fixture/docs/compliance"
cp "$root_dir/scripts/sub-processor-check.sh" "$fixture/scripts/"
printf '`api.paystack.co`\n' > "$fixture/docs/compliance/sub-processors.md"
cat > "$fixture/internal/provider/client.go" <<'GO'
package provider
// Documentation is not an outbound transfer: https://paystack.com/docs/.
const endpoint = "https://API.PAYSTACK.CO"
GO
cat > "$fixture/internal/provider/client_test.go" <<'GO'
package provider
const rejected = "https://unregistered-negative-test.com"
GO
bash "$fixture/scripts/sub-processor-check.sh" > "$fixture/result.log"
# A new production host must still fail, even when tests are excluded.
printf '\nconst unexpected = "https://unregistered-production-provider.com"\n' >> "$fixture/internal/provider/client.go"
if bash "$fixture/scripts/sub-processor-check.sh" > "$fixture/result.log" 2>&1; then
  printf 'Undeclared production host was accepted.\n' >&2
  exit 1
fi
grep -q 'undeclared outbound host: unregistered-production-provider.com' "$fixture/result.log"
rm "$fixture/docs/compliance/sub-processors.md"
if bash "$fixture/scripts/sub-processor-check.sh" > "$fixture/result.log" 2>&1; then
  printf 'Missing processor register was accepted.\n' >&2
  exit 1
fi
grep -q 'sub-processor register is missing' "$fixture/result.log"
printf 'Sub-processor gate rejects missing registers and new production hosts without treating negative tests as providers.\n'
