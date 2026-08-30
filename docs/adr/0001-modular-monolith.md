# ADR 0001: Modular monolith for production v1

- Status: accepted
- Date: 2026-08-16

## Decision

Kredit v1 uses one Go codebase, one PostgreSQL source of truth, and three
deployable processes: web, API, and worker. Domain modules remain explicit in
code while release and transaction boundaries stay coordinated.

## Rationale

The product has agreement, mandate, ledger, collection, dispute, and audit
operations that need strong transactional consistency. A modular monolith keeps
local development and observability tractable while preserving seams for a
future split if scale, compliance isolation, or team ownership requires it.

## Consequences

- Domain modules must not reach into each other's persistence details.
- Financial commands use database transactions and explicit locks.
- External providers are adapters, not domain dependencies.
- Splitting a module later requires an ADR and a consistency plan.

