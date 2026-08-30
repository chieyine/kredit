# ADR 0002: PostgreSQL and the ledger are authoritative

- Status: accepted
- Date: 2026-08-16

## Decision

PostgreSQL is authoritative for domain state and the balanced ledger is
authoritative for money-derived balances. Provider systems are authoritative
only for their own mandate, debit, and settlement states. Cached projections,
browser state, and channel messages must be reproducible or non-authoritative.

## Consequences

- All financially material writes are transactional and idempotent.
- Reversals are new events; accepted obligations are never deleted.
- Reconciliation and ledger-rebuild tooling are required before collections.

