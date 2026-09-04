# ADR 0004: Mono Sweep behind the collection boundary

Status: accepted for sandbox implementation; production certification pending
Date: 2 September 2026

## Decision

Keep obligations, acceptance, collection eligibility/policy, retries, ledger and
history in Kredit. Implement Mono HTTP and event vocabulary in
`internal/providers/mono`, adapting the existing generic mandate/collection
interfaces. Accepting a supplier agreement is independent of authorizing a bank
mandate. Require authoritative provider activation before release/collection.

Commit a reservation and attempt before debit submission. PostgreSQL serializes
obligation balance and shared mandate capacity. Unknown outcomes retain their
reservations and reconcile by saved request reference. Financial idempotency is
per attempt; delivery/event IDs deduplicate receipt processing. Individual Sweep
subevents cannot independently post an aggregate payment.

## Rationale

Providers and notification deliveries may retry or reorder messages; a bank
operation cannot participate in the database transaction. A durable intent and
controlled reconciliation avoid resubmission on uncertain outcomes and recover
when a payment commits before the attempt snapshot. The supplier-funded credit
agreement and financial history remain usable with a future provider adapter.

## Consequences

Ambiguous provider/customer operations may require operator reconciliation.
Manual payment confirmation conflicting with an unresolved debit must wait.
STANDARD staging is unavailable with Mono Sweep and is rejected; GRACE and
IMMEDIATE use accepted timing and authorized provider capabilities. Raw customer
BVNs and bank inventories are not persisted. Internal events support future
history analytics without introducing a public credit score.

All new enablement flags default off. Production Mono use is rejected until
real sandbox acceptance proves the entire required scenario. Local tests are
not a substitute for hosted authorization or provider contract certification.
