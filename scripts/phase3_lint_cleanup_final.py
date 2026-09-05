from pathlib import Path
p = Path('internal/collections/sweep_postgres_test.go')
text = p.read_text()
replacements = {
    'return s.PostgresStore.GetContext(s.ctx, paymentID)': 'return s.GetContext(s.ctx, paymentID)',
    'return s.PostgresStore.RebuildContext(s.ctx, obligationID)': 'return s.RebuildContext(s.ctx, obligationID)',
}
for old, new in replacements.items():
    if text.count(old) != 1:
        raise SystemExit(f'unexpected count for {old!r}: {text.count(old)}')
    text = text.replace(old, new)
p.write_text(text)
