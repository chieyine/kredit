# Audit remediation — 20 September 2026

Base: `1bfc68ab338f630e2e023ce079e9227f2285152d`. Review: draft PR #16.
This is targeted remediation of a partial audit, not a whole-repository production certification.

## Behaviour changes

- Create restricted, non-login runtime role identities before migrations reference them. Object grants remain after schema migrations. Existing runtime roles are not elevated.
- Import staging/read/review uses authenticated, transaction-local tenant identity. Review permission is selected from the actual route; independent authorised reviewers are required. Source identity is the SHA-256 of typed canonical rows, not the idempotency header plus row count.
- **Import approval only reviews proposed terms/opening balances. It does not apply them to live obligations or the ledger.** Responses explicitly state `financial_application: not_applied`. A separate authorised application workflow must resolve a genuine buyer obligation, evidence and accounting event, then commit the entire financial aggregate atomically. It is not implemented here. Never treat import row IDs as obligation IDs.
- Shipments, line items, receipts and credit notes are constrained to their parent order and supplier. Shipment quantity changes are transactional and serialized; receipts have exact replay protection and a narrow database transition, not general buyer update rights.
- New child-record policies preserve parent and branch restrictions. Reconciliation workers can read selected-tenant credit notes but cannot create or alter evidence.
- Credit-note approval uses one transaction for the journal, schedule and outstanding balance. A regression injects failure after the journal write to verify rollback.
- Failed history reads remain visible errors. Development memory stores persist per server. Shipment list queries include their items without per-shipment database queries.
- Spreadsheet presentation CSV protects formula-leading untrusted text; `ToMachineCSV` explicitly preserves machine data. Recipients should still import text columns as text.
- Startup requires migration 201 and the relevant new tables/functions.

## Required maintenance cutover

Migration 201 and the compatible API/worker must be released together. Older binaries do not populate the new shipment-item order identity and previously updated fulfilled quantities themselves; mixing old writers with the new database trigger is unsupported.

1. Review CI and close unresolved gates before approval. Back up the database and rehearse restore in a separate environment.
2. Pause new writes and stop/drain old API and worker processes. Reconcile unresolved provider operations; do not abandon external debits.
3. On a representative restored staging database, check for mismatched order/supplier/obligation references, duplicate shipment receipts and invalid import evidence. The migration deliberately fails rather than deleting or guessing how to repair ambiguous records.
4. Apply schema migrations and runtime grants using the authorised migration/admin connection; start only the compatible application and worker.
5. Verify restricted-role startup, financial reconciliation, receipt/dispatch replay and branch access before resuming traffic.
6. Use a reviewed forward fix if rollback is needed; do not drop evidence guards or disable RLS.

No production data migration, provider operation, merge, or deployment was performed by this remediation.

## Verification limits

The dedicated workflow exercises fresh migration and replay, domain and handler unit tests, restricted-role PostgreSQL integration tests and audit race tests. Existing wider CI remains required. An early run caught a PL/pgSQL variable collision with CURRENT_ROLE; the variable is now actor_role. Existing schedule fixtures were corrected from unsupported CURRENT to UNPAID. The load-test job now supplies an allowed origin matching its isolated API; request-origin enforcement was not weakened.

The main CI log also reported a tenant-matrix coverage failure referring to a test file not retrievable at the inspected commit. That diagnostic is not waived. Verify its source and the seven new tables' access inventory before treating repository-wide CI as passed. Legal-basis and retention approvals are not fabricated or implied by this code change.
