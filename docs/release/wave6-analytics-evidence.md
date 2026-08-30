# Wave 6 product analytics evidence

Evidence date: 29 August 2026

## Implemented control

- Migration 050 upgrades the first-party event envelope with schema version,
  deterministic deduplication, organisation hashing, source and server record
  time, bounded metadata, forbidden sensitive fields and query indexes.
- Migration 051 completes the locked Wave 0 vocabulary, instruments the
  authoritative payment-mandate store, collection submission, repeat sales and
  supplier retention, and adds loss/support-rate/accessibility guardrails.
- Transactional triggers cover supplier/customer onboarding, credit,
  acceptance, mandates, release/receipt, obligations, payments and claims,
  schedule delinquency, collections/recovery, trade lines/drawdowns, disputes
  and support intervention. Exact historical facts are backfilled idempotently.
- The live pilot scorecard calculates three primary KPIs, eight drivers and eight
  guardrails from authoritative records, with definitions, sources, date and
  supplier filters, freshness and zero-tolerance reconciliation.
- The scorecard endpoint is compliance-role restricted, AAL2 protected and
  audited; the admin interface exposes reconciliation status without exposing
  raw identifiers or event payloads.

## Verification

- clean database: migrations 001–051 and deterministic seed apply;
- privacy: raw subject IDs are hashed; sensitive metadata and invalid names are
  rejected;
- replay: two calls with the same product-event key persist exactly one event;
- reconciliation: seeded authoritative source counts match product events at
  tolerance zero;
- endpoint: AAL1 and unassigned access fail, AAL2 compliance access succeeds;
- UI/type checks: the dedicated Playwright scorecard journey passes and Svelte
  diagnostics report zero errors and zero warnings.

The KPI-design workflow influenced the outcome by limiting the primary set to
three decision-grade KPIs, separating drivers from safety guardrails, spelling
out source/definition ownership, and marking targets as baseline-dependent
instead of inventing unsupported goals.

## External gate

Data Protection Lead approval of analytics lawful basis and retention, pilot
owner approval of targets, and target-environment dashboard observation remain
release gates. They do not leave repository-owned instrumentation or
reconciliation work open.
