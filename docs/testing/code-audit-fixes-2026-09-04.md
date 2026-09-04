# Kredit code-audit fixes — 4 September 2026

## Outcome

All 30 actionable findings from the [3 September 2026 file-by-file audit](./code-audit-2026-09-03.md) are resolved in the current working tree: 8 P1 findings and 22 P2 findings. The API and worker compile, the complete Go suite passes, all database-tagged integration tests pass against PostgreSQL 18, the frontend type check and production build pass, and every enabled browser test has passed after the final corrections.

This work was completed without subagents. The original audit and its 664-file register remain unchanged as the before-state evidence. This document records the implemented state and verification evidence.

## Resolution register

| ID | Resolution | Primary verification |
|---|---|---|
| A01 | Report payment reads now return and propagate errors instead of failing compilation or presenting an empty successful report. The payments screen also suppresses success/empty content after a failed read. | Complete Go suite; report error regression; browser failed-read regression |
| A02 | Drawdown cancellation now requires the authorized trade-line ID and actor inside the mutation. Cross-line and blank-actor attempts fail. | Trade-line regression tests; complete integration suite |
| A03 | Organization authorization requires an active membership. Suspended and invited memberships cannot pass the access boundary. | Memory and PostgreSQL organization tests under runtime access rules |
| A04 | Neither memory nor PostgreSQL role changes can demote the owner. | Owner-demotion unit and PostgreSQL integration regressions |
| A05 | Schedule creation preserves the exact accepted first collection instant and derives later installments from it. A narrowly scoped migration repairs matching open legacy schedules. | Schedule and runtime regressions; fresh migration run |
| A06 | Presigned uploads require create-only `If-None-Match: *`, the declared content type, and server-side encryption; the required signed headers are returned to the client. | S3 signing unit test |
| A07 | PostgreSQL buyer invitations persist the token hash and can be created, previewed, and accepted under the restricted runtime role. | Buyer PostgreSQL integration regression |
| A08 | Obligation balance changes now synchronize drawdown exposure and reusable trade-line capacity. Memory payments and adjustments use the same invariant. | Payment, reversal, adjustment, trade-line, and database-trigger regressions |
| A09 | Limit reductions persist the approved and available limits together while allowing existing exposure above a newly reduced limit. | PostgreSQL reduce-and-reload regression |
| A10 | Dispute and operational principal adjustments reduce collectible schedule items atomically and roll back together on failure. | Memory atomicity tests and database dispute/operations tests |
| A11 | The worker runs durable deemed-acceptance processing at startup and on its maintenance interval. Eligibility is reloaded from PostgreSQL and requires persisted delivery evidence. | Worker tests, credit tests, migration 070 guard, complete integration suite |
| A12 | Mono cancellation affects only trade lines linked to the cancelled internal mandate and propagates mutation errors. | Handler/runtime tests and complete Go suite |
| A13 | Finance MFA updates preserve existing owner MFA evidence; owner evidence is cleared only when owner MFA becomes false. | Onboarding regression tests |
| A14 | Unrelated onboarding edits preserve the original terms and privacy acceptors. Actor fields change only on an explicit consent mutation. | Onboarding database tests in the complete integration suite |
| A15 | Approved recovery requests deliver a private continuation through the verified channel before state is committed. The URL keeps the raw token in its fragment, and restricted runtime lookup now establishes the user's RLS context. | Delivery-failure atomicity test, recovery PostgreSQL test, restricted-role auth test |
| A16 | Recovery completion atomically revokes active MFA methods, sessions, and recovery codes, allowing controlled factor replacement. | Auth and user-control unit/integration tests |
| A17 | An unverified TOTP enrollment can be replaced; verified enrollment remains protected. | Unit and PostgreSQL integration regressions |
| A18 | Notification settings submit only accepted write fields plus `expected_version`. | Product-flow browser test |
| A19 | Exact preview routes are exempt from the command idempotency requirement while mutations remain protected. | Middleware regression tests and operations browser flows |
| A20 | Draft date-time values use timezone-safe conversion and retain the same instant when edited in another timezone. | New York timezone browser regression |
| A21 | Definitively rejected invitation and credit requests receive a new idempotency key; uncertain server failures retain the original key. | Browser retry-key regression |
| A22 | Document JSON endpoints accept the envelope needed for a full 2 MiB decoded file while enforcing the decoded-file limit. | Full-size document-body HTTP regression |
| A23 | Due, overdue, and ageing totals allocate only the current outstanding balance across open installments and reject inconsistent inputs. | Report calculation and PostgreSQL tests |
| A24 | Accepting another supplier invitation reuses the buyer's existing person, business, representative, and matching verification records. | Buyer unit and PostgreSQL integration regressions |
| A25 | Audit listing safely scans events whose request ID is null. | PostgreSQL audit regression |
| A26 | Audit event JSON uses the snake-case field names consumed by the activity UI. | Serialization regression |
| A27 | Startup now requires schema version 72 and checks the tables, columns, and functions needed by runtime authentication and financial integrity. | Fresh PostgreSQL 18 migration from empty database; persistence-contract tests |
| A28 | Buyers can report a receipt problem with a reason; this opens the issue path without activating the obligation. | Product-flow browser regression |
| A29 | Seller-specific reminder consent is checked when a reminder is queued and again immediately before scheduled delivery. Withdrawal suppresses queued reminders. | Notification unit and PostgreSQL integration regressions |
| A30 | The invoice action reads the signed-download response and navigates to its document URL, with visible failure handling. | Browser download regression |

## Additional defects found while fixing and validating

The repair run exposed four further defects, all corrected:

- Migration 047 used a UUID function through the wrong schema on PostgreSQL 18; it now uses `pg_catalog.gen_random_uuid()`.
- Migration 067 contained procedural SQL without the migration tool's statement boundary markers; a clean database can now apply it.
- The new financial-integrity migrations were renumbered to 071 and 072 to preserve the existing migration 070 deemed-acceptance evidence.
- The public demo could lose an immediate click before client hydration. Its controls remain disabled until their actions are attached.

## Final verification

| Check | Result |
|---|---|
| Fresh PostgreSQL 18 schema | Migrations 001 through 072 applied successfully |
| `go test ./...` | Passed for every package |
| Database-tagged `go test -p=1 -tags=integration ./...` | Passed for every package, including restricted runtime-role regressions |
| Svelte check | 0 errors and 0 warnings; the final changed demo component was rechecked after its hydration fix |
| Frontend production build | Passed with the configured Vercel adapter after the final changes |
| Focused audit browser tests | 3 passed: timezone preservation, signed invoice navigation, and idempotency retry behavior |
| Complete browser suite | 77 passed and one production-configuration test skipped in the full run; its three discovered failures were corrected and then passed together, covering the payment error state, complete demo journey, and mobile customer navigation |
| Patch hygiene | Go formatting passed; the independent review pass found and fixed the recovery RLS lookup and early demo interaction race |

External provider calls were verified at the repository's contract/adaptor boundary. No live Mono, object-storage, email, SMS, or payment-provider transaction was sent.

## 4 September K01–K22 follow-up audit

The later [4 September file-by-file audit](./code-audit-2026-09-04.md)
identified 22 additional findings. This register supersedes any broad
"everything is resolved" interpretation of the earlier A01–A30 result.

| ID | Disposition | Implemented result |
|---|---|---|
| K01 | Fixed | Supplier payment APIs reject collection-origin payments; payment and database boundaries require a matching active collection attempt and worker provenance. |
| K02 | Fixed | Supplier routes cannot adjudicate disputes; independent platform dispute review remains available. |
| K03 | Fixed | Dispute effect is server-owned and only one active aggregate dispute may block an obligation; migration 073 fails early with an actionable preflight if legacy duplicates require manual consolidation. |
| K04 | Fixed | Jobs, collections, account suspension and risk holds use separate command permissions resolved before preview or execution. |
| K05 | Fixed for the current case model | Support agents are limited to cases assigned through the existing case owner; global search and directories require an administrator/access administrator. |
| K06 | **Open architectural blocker** | Document metadata now has tenant/worker RLS, but older permissive API-role policies still OR around tenant policies on several financial domains. A repository-wide context/function migration and PostgreSQL 18 role-matrix proof are still required. |
| K07 | Fixed | Upload completion, expiry, leases, retry quarantine and quotas are durable; quota count and insert are atomic under a tenant advisory lock (and one mutex in memory). |
| K08 | Fixed | Read-only organization members cannot open disputes or add evidence. |
| K09 | Fixed | Sensitive organization actions require AAL2 verified within 15 minutes. |
| K10 | Fixed | MFA step-up atomically rotates the bearer session and CSRF state and revokes the old session. |
| K11 | Fixed | Accepted TOTP counters are persisted and replayed/older counters are rejected under lock. |
| K12 | Fixed for active-key separation | Session hashing, OTP authentication and field encryption consume separate configured keys; remaining capability domains use labelled derived keys. Operational dual-read rotation remains a deployment procedure, not a completed live-key rotation. |
| K13 | Fixed | Public callbacks are signals only. They cannot supply a missing provider collection ID or post money; server-to-server reconciliation validates the returned collection identity. |
| K14 | Fixed | Buyer acknowledgements are bound to a specific delivered/read notification and immutable connector receipt. Worker access to acknowledgement evidence is SELECT-only. Deemed acceptance remains disabled. |
| K15 | Fixed | Mandate lookup and restore preserve supplier scope, and migration 073 removes the old supplier-unscoped database function. |
| K16 | Fixed | The supplier can poll KYB status; provider ID and subject ID must match, and the result is applied with provider-reference/version compare-and-swap semantics. |
| K17 | Fixed | Worker scheduling was removed and the database rejects silence-based activation. Product documentation now states the same pilot rule. |
| K18 | Fixed | Approved legal pages become indexable under the same activation state used by sitemap/robots generation. |
| K19 | **Partially fixed** | Request contexts now reach payment operations, financial reads, document storage and KYB/provider calls. Many legacy repository interfaces still create background contexts and require a deliberate code-wide interface migration. |
| K20 | Fixed | CI scanner versions are pinned. |
| K21 | Fixed | Release/status documentation no longer freezes obsolete migration/test counts or contradicts the deemed-acceptance policy. |
| K22 | Fixed | Legacy CSV line endings and monitoring whitespace were normalized; repository integrity and `git diff --check` pass. |

### Independent post-patch review

A fresh read-only reviewer initially found callback-ID poisoning, unbound notice
acknowledgements, missing worker RLS, an old mandate-function overload, mandate
restore scope loss, stale KYB result application, duplicate-dispute migration
risk and a concurrent upload-quota race. Each was corrected. The follow-up
review verified those corrections and repeated the focused collection,
document, mandate, identity, onboarding, web and dispute tests. It independently
confirmed that K06 remains open repository-wide.

### Current verification evidence

- Focused security and financial Go packages pass.
- `go vet ./...` passes.
- Svelte check passes with 0 errors and 0 warnings; the production build passes.
- Repository integrity, product/API contract sync, frontend route coverage,
  content audit and `git diff --check` pass.
- The full Go run passes all code assertions except three listener-based tests
  blocked by this managed sandbox's prohibition on local test listeners.
- A fresh PostgreSQL 14 compatibility run applies migrations 001–055 and then
  correctly stops at migration 056 because PostgreSQL 14 lacks the required
  `security_invoker` view option. Migrations 056–077 and the final runtime-role
  matrix must be executed on PostgreSQL 18 in CI or the target environment.

The repository must not be described as fully remediated or production-ready
while K06 and the remaining K19 interface migration are open, or until
production data passes migration 073's duplicate-active-dispute preflight.
