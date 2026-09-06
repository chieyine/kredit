#!/usr/bin/env python3
"""Audit synchronous HTTP paths for accidental background-context fallbacks.

This is intentionally conservative: it does not forbid background contexts in workers,
commands, tests or process-startup infrastructure. It protects HTTP handlers and the
context-aware financial adapters used directly by them.
"""
from __future__ import annotations

from pathlib import Path
import re
import sys

ROOT = Path(__file__).resolve().parents[1]
WEB = ROOT / "internal" / "web"

issues: list[str] = []

# HTTP handlers must never manufacture a background context; browser/request
# cancellation must be allowed to flow from r.Context(). runtime.go is excluded
# because its Background contexts are process-startup/runtime-construction work.
for path in sorted(WEB.glob("*.go")):
    if path.name.endswith("_test.go") or path.name == "runtime.go":
        continue
    text = path.read_text(encoding="utf-8")
    if "context.Background()" in text or "context.TODO()" in text:
        issues.append(f"{path.relative_to(ROOT)}: HTTP code manufactures a background context")

# Protect the context-aware payment boundary. Compatibility fallbacks are kept for
# development/test adapters, but the helper must prefer the request-aware methods.
financial_text = (WEB / "financial_reads.go").read_text(encoding="utf-8")
required_context_helpers = {
    "getPayment": r"func \(r \*Runtime\) getPayment\(ctx context\.Context",
    "readPayments": r"func \(r \*Runtime\) readPayments\(ctx context\.Context",
}
for label, pattern in required_context_helpers.items():
    if not re.search(pattern, financial_text):
        issues.append(f"internal/web/financial_reads.go: missing context-aware {label} helper")

for required in (
    "GetContext(context.Context, string) (payments.Payment, error)",
    "ReadContext(context.Context, string) ([]payments.Payment, error)",
):
    if required not in financial_text:
        issues.append(f"internal/web/financial_reads.go: missing request-aware payment adapter {required.split('(')[0]}")

# Analytics persistence reached from HTTP must use the request-aware API.
for path in sorted(WEB.glob("*.go")):
    if path.name.endswith("_test.go"):
        continue
    body = path.read_text(encoding="utf-8")
    if "Reports.Track(" in body:
        issues.append(f"{path.relative_to(ROOT)}: legacy analytics Track bypasses request context")

if issues:
    print("Phase 6 request-context audit failed:", file=sys.stderr)
    for issue in issues:
        print(f"- {issue}", file=sys.stderr)
    sys.exit(1)

print("Phase 6 request-context audit passed.")
