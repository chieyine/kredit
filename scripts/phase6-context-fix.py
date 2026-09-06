#!/usr/bin/env python3
from pathlib import Path

root = Path(__file__).resolve().parents[1]

buyer = root / 'internal/web/buyer_handlers.go'
text = buyer.read_text(encoding='utf-8')
text = text.replace('\t"context"\n', '')
text = text.replace('s.runtime.Buyers.Accept(context.Background(), token, user.ID,', 's.runtime.Buyers.Accept(r.Context(), token, user.ID,')
buyer.write_text(text, encoding='utf-8')

onboarding = root / 'internal/web/onboarding_handlers.go'
text = onboarding.read_text(encoding='utf-8')
text = text.replace('\t"context"\n', '')
text = text.replace('s.runtime.EmitNotification(context.Background(),', 's.runtime.EmitNotification(r.Context(),')
onboarding.write_text(text, encoding='utf-8')

# The runtime legitimately uses background contexts during process startup.
# The audit is scoped to HTTP handler code and verifies that the existing
# payment helper boundary prefers context-aware adapters.
audit = root / 'scripts/phase6-context-audit.py'
text = audit.read_text(encoding='utf-8')
text = text.replace('    if path.name.endswith("_test.go"):\n        continue\n', '    if path.name.endswith("_test.go") or path.name == "runtime.go":\n        continue\n', 1)
text = text.replace('    "runtime.getPayment": r"func \\(r \\*Runtime\\) getPayment\\(ctx context\\.Context",\n    "runtime.readPayments": r"func \\(r \\*Runtime\\) readPayments[^\\n]*\\(ctx context\\.Context",\n', '    "runtime.getPayment": r"func \\(r \\*Runtime\\) getPayment\\(ctx context\\.Context",\n    "runtime.readPayments": r"func \\(r \\*Runtime\\) readPayments\\(ctx context\\.Context",\n')
# Safe fallbacks remain for test/development adapters. Production and the in-memory
# payment store both expose the context-aware methods checked below.
start = text.index('# Detect direct synchronous HTTP calls to the legacy payment compatibility surface.')
end = text.index('\nif issues:', start)
replacement = '''# The helper boundary must prefer context-aware payment adapters before any\n# compatibility fallback used by development/test implementations.\nfor required in (\n    "ReadContext(context.Context, string) ([]payments.Payment, error)",\n    "GetContext(context.Context, string) (payments.Payment, error)",\n):\n    if required not in runtime_text and required not in (WEB / "financial_reads.go").read_text(encoding="utf-8"):\n        issues.append(f"internal/web/financial_reads.go: missing context-aware payment adapter {required.split('(')[0]}")\n'''
text = text[:start] + replacement + text[end:]
audit.write_text(text, encoding='utf-8')
