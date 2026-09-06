#!/usr/bin/env python3
"""Audit synchronous HTTP paths for accidental background-context fallbacks.

This is intentionally conservative: it does not forbid background contexts in workers,
commands, tests or asynchronous infrastructure. It protects the HTTP package and the
context-aware financial adapters used directly by HTTP handlers.
"""
from __future__ import annotations

from pathlib import Path
import re
import sys

ROOT = Path(__file__).resolve().parents[1]
WEB = ROOT / "internal" / "web"

issues: list[str] = []

# HTTP handlers must never manufacture a background context; browser/request
# cancellation must be allowed to flow from r.Context().
for path in sorted(WEB.glob("*.go")):
    if path.name.endswith("_test.go"):
        continue
    text = path.read_text(encoding="utf-8")
    if "context.Background()" in text or "context.TODO()" in text:
        issues.append(f"{path.relative_to(ROOT)}: HTTP code manufactures a background context")

# Protect the context-aware payment adapter that Phase 3 introduced. Compatibility
# methods may still exist for non-HTTP callers, but HTTP helpers must route through
# the context-aware surface.
web_text = "\n".join(
    p.read_text(encoding="utf-8") for p in sorted(WEB.glob("*.go")) if not p.name.endswith("_test.go")
)
required_context_helpers = {
    "runtime.getPayment": r"func \(r \*Runtime\) getPayment\(ctx context\.Context",
    "runtime.readPayments": r"func \(r \*Runtime\) readPayments[^\n]*\(ctx context\.Context",
}
runtime_text = (WEB / "runtime.go").read_text(encoding="utf-8")
for label, pattern in required_context_helpers.items():
    if not re.search(pattern, runtime_text):
        issues.append(f"internal/web/runtime.go: missing context-aware {label} helper")

# Detect direct synchronous HTTP calls to the legacy payment compatibility surface.
# The runtime helper is allowed to hold the type assertion/fallback boundary.
for legacy in ("Payments.Record(", "Payments.Reverse(", "Payments.Get(", "Payments.List(", "Payments.Rebuild("):
    for path in sorted(WEB.glob("*.go")):
        if path.name.endswith("_test.go") or path.name == "runtime.go":
            continue
        if legacy in path.read_text(encoding="utf-8"):
            issues.append(f"{path.relative_to(ROOT)}: direct legacy call {legacy[:-1]} bypasses request context")

if issues:
    print("Phase 6 request-context audit failed:", file=sys.stderr)
    for issue in issues:
        print(f"- {issue}", file=sys.stderr)
    sys.exit(1)

print("Phase 6 request-context audit passed.")
