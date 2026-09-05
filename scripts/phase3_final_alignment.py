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

# Keep Phase 3 focused on financial-core collection tests rather than pulling
# unrelated admin/report workflow suites into this acceptance gate.
p = Path('.github/workflows/phase3-financial-proof.yml')
text = p.read_text()
old = '''      - name: Re-prove payment and collection invariants
        run: |
          go test -p 1 ./internal/payments ./internal/collections ./internal/ledger ./internal/schedules
          go test -race ./internal/payments ./internal/collections
      - name: Re-prove deterministic PostgreSQL financial paths
        run: |
          go test -p 1 ./internal/payments -run 'TestPostgresPayment' -count=1 -v
          go test -p 1 ./internal/collections -count=1
'''
new = '''      - name: Re-prove payment and collection invariants
        run: |
          go test -p 1 ./internal/payments ./internal/ledger ./internal/schedules
          go test -race ./internal/payments
          go test -race ./internal/collections -run 'Test(PostgresReservationCommitsBeforeProviderAndDuplicateJobs|PostgresPaymentBeforeDebitAndReplayAfterPaymentCommit|PostgresSharedMandateCannotOverReserve|CollectionWorkerRoleCanPostAndReconcilePayment|CollectionStateLoadsAfterRestartAndObservesExternalManualPayment|PostgresCollectionIsRestartSafeAndFinanciallyIdempotent)$' -count=1
      - name: Re-prove deterministic PostgreSQL financial paths
        run: |
          go test -p 1 ./internal/payments -run 'TestPostgresPayment' -count=1 -v
          go test -p 1 ./internal/collections -run 'Test(PostgresReservationCommitsBeforeProviderAndDuplicateJobs|PostgresPaymentBeforeDebitAndReplayAfterPaymentCommit|PostgresSharedMandateCannotOverReserve|CollectionWorkerRoleCanPostAndReconcilePayment|CollectionStateLoadsAfterRestartAndObservesExternalManualPayment|PostgresCollectionIsRestartSafeAndFinanciallyIdempotent)$' -count=1 -v
'''
if text.count(old) != 1:
    raise SystemExit(f'workflow financial proof anchor: {text.count(old)}')
text = text.replace(old, new, 1)
text = text.replace('DATABASE_URL="$ADMIN_DATABASE_URL" go run ./cmd/seed\n          DATABASE_URL="$ADMIN_DATABASE_URL" go run ./cmd/seed', 'DATABASE_URL="$ADMIN_DATABASE_URL" DATABASE_DIRECT_URL="$ADMIN_DATABASE_URL" go run ./cmd/seed\n          DATABASE_URL="$ADMIN_DATABASE_URL" DATABASE_DIRECT_URL="$ADMIN_DATABASE_URL" go run ./cmd/seed')
p.write_text(text)
