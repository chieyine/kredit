from pathlib import Path

p = Path('internal/collections/sweep_postgres_test.go')
text = p.read_text()
replacements = {
    'return s.PostgresStore.RecordContext(s.ctx, input)': 'return s.RecordContext(s.ctx, input)',
    'return s.PostgresStore.ReverseContext(s.ctx, paymentID, actor, reason)': 'return s.ReverseContext(s.ctx, paymentID, actor, reason)',
    'return s.PostgresStore.ReadContext(s.ctx, obligationID)': 'return s.ReadContext(s.ctx, obligationID)',
    'return e.PostgresEngine.GetAttemptContext(e.ctx, attemptID)': 'return e.GetAttemptContext(e.ctx, attemptID)',
}
for old, new in replacements.items():
    if text.count(old) != 1:
        raise SystemExit(f'unexpected count for {old!r}: {text.count(old)}')
    text = text.replace(old, new)
p.write_text(text)

p = Path('internal/web/financial_reads.go')
text = p.read_text()
obsolete = '''func (r *Runtime) readCollectionsAttempts(id string) ([]collections.Attempt, error) {
\treturn r.readCollectionsAttemptsContext(context.Background(), id)
}
'''
if text.count(obsolete) != 1:
    raise SystemExit(f'obsolete helper count: {text.count(obsolete)}')
text = text.replace(obsolete, '')
p.write_text(text)
