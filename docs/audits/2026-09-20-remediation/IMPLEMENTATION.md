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

On candidate `9a24c1d42af0c2e224073091e41e62017a3ff3d3`, the tenant-isolation, financial-proof, provider-verification, production-assurance, request-context and audit-remediation workflows passed. Main CI and the product audit were not yet cleared at that checkpoint. These isolated tests are not provider certification, legal approval, live-system assurance or a complete file-by-file review.


## Browser and contract follow-through

The next changes quote malformed OpenAPI descriptions, preserve text visibility during reveal animations, repair dark-surface text contrast and small touch targets, and remove a conflicting public page title. Browser tests use a loopback-only read-only public API fixture only when neither a real base URL nor an internal API is supplied. It supplies initial-publication/pricing reads, rejects writes and credentials, and fails all unknown endpoints. Explicit real-stack tests are never replaced by this fixture. Missing, malformed and unavailable legal publications still return errors and noindex; no legal content or approval was manufactured.

Old route assertions and evidence pointers now identify the actual workspace/account pages. Shared discovery classification keeps private workspaces and directly shared deck/referral links out of indexing. The content and contract checks still fail on unknown public routes or unrecorded backend endpoints. Source hygiene repairs only newline bytes, not earlier audit claims or review dates.

### Functionality still not shipped as a browser workflow

The API registry now explicitly identifies API-only operations rather than claiming that an unrelated screen implements them: item-line creation, item-level buyer receipt confirmation, independent credit-note approval, independent drawdown review requests, proposed-terms batch staging/review, and ERP comparison. The deliveries screen does implement reads, shipment dispatch and credit-note drafting; it does not implement all those other actions. These API-only capabilities remain product completion work, not a passed end-to-end user journey.

ERP comparison still needs a separate accounting and tenant-context review before operational reliance. Opening balances must remain `not_applied` until a verified evidence-linked financial-application workflow exists. Complete JSON response schemas, broad access matrices, real browser-to-database flows, all wider CI gates, and deployment/restore rehearsal remain required for release clearance.
