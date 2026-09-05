from pathlib import Path

# Payment proof: persisted state constants are lowercase.
p = Path('internal/payments/postgres_test.go')
text = p.read_text().replace("state='RECOGNIZED'", "state='recognized'")
p.write_text(text)

# Standalone collection restart test must use the same tenant-aware payment
# wrapper as the shared collection fixture.
p = Path('internal/collections/postgres_test.go')
text = p.read_text()
old = 'paymentStore := payments.NewPostgresStore(pool, outbox.NewStore(pool), nil)'
new = 'paymentStore := &tenantPaymentStore{PostgresStore: payments.NewPostgresStore(pool, outbox.NewStore(pool), nil), ctx: ctx}'
if text.count(old) != 1:
    raise SystemExit(f'postgres collection payment store anchor: {text.count(old)}')
text = text.replace(old, new, 1)
p.write_text(text)
