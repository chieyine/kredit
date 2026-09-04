# Kredit code audit — 3 September 2026

**Result: the current working tree is not ready to run or release reliably.** This review identifies **30 actionable defects: 8 high-priority (P1) and 22 normal-priority (P2)**. The API/worker compilation failure is reproduced. Several independent authorization, financial-state, persistence and interface defects remain behind that build failure.

This was an audit only, performed without subagents. No application implementation, configuration, migration or existing test file was changed. Existing uncommitted work was reviewed as it stood. This report and its file register are new audit artifacts; temporary probes and build caches are under ignored output directories.

## Scope and how to read this report

The accompanying [file-by-file register](</Users/macbookpro/Documents/Kredit.com/docs/testing/code-audit-2026-09-03-files.csv>) accounts for **664 repository-owned files**. It contains each file’s path, line count, review method, validation status, associated findings and SHA-256. The original application/support inventory contained 663 files; the final register additionally includes `Claude outputs/ci.yml`, an archived workflow draft that is not the active GitHub workflow. Dependencies, Git internals, caches and generated build output are excluded.

Handwritten backend behavior, frontend logic/components/routes, persistence boundaries, permissions, financial flows, providers, notifications, deployment scripts and supporting contracts were reviewed through source reads and cross-file traces. Generated source was checked through the compiler/type/contract tools rather than manually inspecting every generated line. SQL received schema-wide searches and targeted query/constraint/policy review; it was not executed. Test files received package execution where possible and focused coverage inspection, not an assertion-by-assertion proof. Documentation and editorial content received structural/reference checks, not legal or financial fact checking. The register preserves these distinctions instead of marking every file “passed.”

A blank finding cell means no separate actionable finding was recorded for that file under the stated method. It does **not** certify that the file or every execution path is correct. In particular, the build and database constraints below prevent a claim that every app component works end to end.

P1 means a build blocker, significant authorization/data-integrity risk or core financial/onboarding failure that should be addressed before relying on the app. P2 means a concrete functional, reporting, persistence or operational defect. Verification descriptions distinguish executed probes from source/schema conclusions. Findings behind A01 describe the affected path once the compilation blocker is corrected; they do not claim the broken server was exercised.

## Validation actually performed

| Check | Result | Practical limit |
|---|---|---|
| `go test ./...` | Failed: API, worker and internal/web do not compile; 46 other package results were `ok` | PostgreSQL-gated tests were skipped; cached results are shown where Go reused them |
| SvelteKit synchronization and Svelte check | Passed, 0 errors and 0 warnings | Does not validate dynamically typed API payloads or browser interactions |
| Frontend production build | Passed using the configured Vercel adapter | No deployed frontend/backend integration or container build |
| Go formatting | 248 Go files checked; none required formatting | Style only |
| Shell syntax | 31 shell scripts passed | No claim that every script was executed |
| Repository integrity/brand/reference checker | Passed for 664 files | Structural checks, not a behavior audit |
| API product-contract synchronization | Passed: 116 explicit frontend calls, 191 backend routes, 190 API operations | Method/path matching does not prove response fields, headers or state transitions |
| Frontend API coverage checker | Passed for 188 routes; 16 intentionally have no separate screen | Coverage accounting, not interaction testing |
| Content audit | Passed: 100 articles, 203,810 blog words | Completeness/structure; factual claims not verified |
| Focused local behavior probes | Reproduced cancellation, owner demotion, schedule-date error, MFA-readiness regression, reporting overstatement, preferences rejection, preview-header requirement and draft timezone drift | Store/serialization/predicate probes; not live HTTP, PostgreSQL or provider tests |
| Original-file hash comparison | All 663 initial files unchanged | New audit artifacts are outside that original baseline |

**Not executed:** PostgreSQL 18 migration/integration scenarios, complete API/browser end-to-end flows, live Mono/identity/payment/notification/storage calls, backup restoration, load testing, full race/fuzz campaigns, dependency-vulnerability scan completion, container deployment and production certification. The local PostgreSQL installations were older than the required version, and the Docker daemon was unavailable. No database was reset or modified to work around that limitation. The pnpm check wrapper attempted a dependency reinstall and could not complete in this environment; the available local Svelte/Vite tools were then run successfully.

The original repository checker fingerprint was `9d6de06341af952f8ba998d1557a46fe8222c5fcaf248ccaf17442a9ac8acd60`. Per-file hashes in the register are the authoritative snapshot identifiers for this report.

## Findings index

| ID | Priority | Finding |
|---|---|---|
| A01 | P1 | API and worker do not compile |
| A02 | P1 | Cancellation authorizes a URL trade line but mutates an unrelated drawdown |
| A03 | P1 | Suspended members retain organization access |
| A04 | P1 | An administrator can demote the business owner |
| A05 | P1 | Generated collection dates can precede the accepted debit date |
| A06 | P1 | A reusable upload URL can replace a document after it passes scanning |
| A07 | P1 | PostgreSQL buyer invitations omit a required token hash |
| A08 | P1 | Repayments do not restore trade-line capacity |
| A09 | P2 | Lowering a trade-line limit violates the persisted balance constraint |
| A10 | P2 | Dispute adjustments leave the repayment schedule overstated |
| A11 | P2 | Deemed acceptance is implemented but never invoked by the running app |
| A12 | P2 | One revoked Mono mandate suspends unrelated trade lines for the same buyer |
| A13 | P2 | Changing finance MFA readiness clears valid owner MFA readiness |
| A14 | P2 | Unrelated onboarding edits overwrite who accepted terms and privacy |
| A15 | P2 | Production recovery never delivers its required continuation credentials |
| A16 | P2 | Completing recovery leaves the lost authenticator as the required factor |
| A17 | P2 | An abandoned TOTP enrollment cannot be restarted |
| A18 | P2 | Notification preferences cannot be saved from the settings page |
| A19 | P2 | Protected operations previews omit a required idempotency key |
| A20 | P2 | Editing a draft silently shifts its collection time |
| A21 | P2 | Correcting a rejected form reuses a key tied to the old payload |
| A22 | P2 | The invoice form allows files too large for its JSON endpoint |
| A23 | P2 | Overdue and near-term totals include future installment principal |
| A24 | P2 | An existing buyer cannot accept another supplier invitation |
| A25 | P2 | One audit event without a request ID hides the whole organization audit list |
| A26 | P2 | Activity screen reads field names the audit endpoint does not emit |
| A27 | P2 | Database readiness accepts a schema older than authentication requires |
| A28 | P2 | Buyers cannot report receipt problems before confirming a normal sale |
| A29 | P2 | Seller-specific reminder withdrawal is not enforced during delivery |
| A30 | P2 | The invoice “Open” link opens JSON instead of the document |

## Detailed findings

### A01 · P1 · API and worker do not compile

**Locations:** [internal/web/runtime.go:371](</Users/macbookpro/Documents/Kredit.com/internal/web/runtime.go:371>); [internal/reports/store.go:34](</Users/macbookpro/Documents/Kredit.com/internal/reports/store.go:34>)

The runtime assigns paymentStore.List, which returns ([]payments.Payment, error), to reports.Source.Payments, which accepts only a slice-returning function. Both report-store constructions fail compilation. This prevents building or starting the API and worker and prevents all internal/web tests from running. Carry the error-aware contract through the report boundary; do not discard payment-read errors to silence the compiler.

**Verification:** Reproduced by go test ./... at runtime.go:371 and :373.

**Regression check after correction:** Build both binaries and exercise report generation with a payment-store read failure.

### A02 · P1 · Cancellation authorizes a URL trade line but mutates an unrelated drawdown

**Locations:** [internal/web/credit_handlers.go:978](</Users/macbookpro/Documents/Kredit.com/internal/web/credit_handlers.go:978>); [internal/web/credit_handlers.go:1004](</Users/macbookpro/Documents/Kredit.com/internal/web/credit_handlers.go:1004>); [internal/tradelines/store.go:563](</Users/macbookpro/Documents/Kredit.com/internal/tradelines/store.go:563>); [internal/tradelines/postgres.go:231](</Users/macbookpro/Documents/Kredit.com/internal/tradelines/postgres.go:231>); [db/migrations/034_trade_line_runtime_policy.sql:4](</Users/macbookpro/Documents/Kredit.com/db/migrations/034_trade_line_runtime_policy.sql:4>)

Both cancellation handlers validate the lineID supplied in the URL, then mutate drawdownID without proving that it belongs to that line. CancelDrawdown only requires a nonempty actor; the PostgreSQL adapter loads the actual line from the drawdown. A user with their own authorized line and a different pending/confirmed drawdown ID can therefore cancel that other sale, release its reservation, and receive its returned data. The runtime-wide table policy does not supply the missing ownership check. Enforce the authorized line and actor within the mutation transaction.

**Verification:** Store probe cancelled a victim drawdown using an unrelated actor. HTTP and PostgreSQL execution were not possible; the handler-to-store path was traced in source.

**Regression check after correction:** Cross-organization and cross-buyer cancellation must fail without changing either line, reservation, or notification state.

### A03 · P1 · Suspended members retain organization access

**Locations:** [internal/organizations/postgres.go:167](</Users/macbookpro/Documents/Kredit.com/internal/organizations/postgres.go:167>); [internal/web/organization_handlers.go:210](</Users/macbookpro/Documents/Kredit.com/internal/web/organization_handlers.go:210>)

PostgreSQL Membership returns every status except removed, including suspended and invited. requireOrganizationAccess checks only the returned role. Suspending a member therefore leaves their existing role permissions usable through direct API requests; any separate MFA requirement still applies. The memory adapter requires active membership, so its tests do not represent this production behavior. Require active status at both the repository and authorization boundary.

**Verification:** Static repository/query/authorization trace; database execution unavailable.

**Regression check after correction:** After suspension, an existing session must be denied reads and writes for that organization; invited users must not gain active permissions.

### A04 · P1 · An administrator can demote the business owner

**Locations:** [internal/organizations/postgres.go:317](</Users/macbookpro/Documents/Kredit.com/internal/organizations/postgres.go:317>); [internal/organizations/store.go:250](</Users/macbookpro/Documents/Kredit.com/internal/organizations/store.go:250>)

ChangeRole rejects assigning the owner role and changing the actor’s own role, but does not protect a target who already is the owner. An administrator with team-management permission can change the owner to viewer. The UI hides the owner controls and ChangeStatus protects owners, but neither protects this API mutation. Check the target’s current role atomically before changing it.

**Verification:** Memory-store probe changed the owner to viewer successfully. The PostgreSQL UPDATE has the same missing guard.

**Regression check after correction:** An administrator attempting to change the owner’s role must receive a denial and leave ownership intact.

### A05 · P1 · Generated collection dates can precede the accepted debit date

**Locations:** [internal/credit/postgres.go:392](</Users/macbookpro/Documents/Kredit.com/internal/credit/postgres.go:392>); [internal/schedules/store.go:199](</Users/macbookpro/Documents/Kredit.com/internal/schedules/store.go:199>); [internal/web/runtime.go:224](</Users/macbookpro/Documents/Kredit.com/internal/web/runtime.go:224>)

Activation takes the calendar date from DueDate and only the hour/minute from the accepted CollectionAt, then schedule creation adds GraceHours again. It discards the accepted collection calendar date. For a 10 September due date, accepted collection on 11 September at 05:00 Lagos and six grace hours, the generated collection time is 10 September at 11:00: eighteen hours early. Collection eligibility uses the generated schedule; other notice/provider gates may still delay an actual debit. Preserve the accepted timestamp and derive installment dates from an explicit, consistent rule.

**Verification:** Real schedule-store probe reproduced the eighteen-hour discrepancy. No live debit was attempted.

**Regression check after correction:** Compare accepted terms with persisted and worker-visible collection timestamps for one-time, equal, custom, timezone and grace-period cases.

### A06 · P1 · A reusable upload URL can replace a document after it passes scanning

**Locations:** [internal/documents/s3.go:67](</Users/macbookpro/Documents/Kredit.com/internal/documents/s3.go:67>); [internal/documents/scanner.go:77](</Users/macbookpro/Documents/Kredit.com/internal/documents/scanner.go:77>); [internal/documents/store.go:225](</Users/macbookpro/Documents/Kredit.com/internal/documents/store.go:225>); [internal/web/document_upload_slot_handlers.go:68](</Users/macbookpro/Documents/Kredit.com/internal/web/document_upload_slot_handlers.go:68>)

The direct-upload URL authorizes PUT to one mutable object key for ten minutes. Neither the signature nor scan metadata binds the accepted bytes to an immutable object version or required checksum. A client can upload clean bytes, allow scanning to mark the document CLEAN, then overwrite the same key before the PUT URL expires. SignedDownload checks the old CLEAN state but serves the new object. Finalize uploads into an immutable/version-bound object and bind the scan and download to those exact bytes.

**Verification:** Static S3 signing, scanning and download trace. No live object-storage exploit was run.

**Regression check after correction:** Reusing a completed upload URL must fail or must never change the bytes served by a CLEAN document.

### A07 · P1 · PostgreSQL buyer invitations omit a required token hash

**Locations:** [internal/buyers/postgres.go:102](</Users/macbookpro/Documents/Kredit.com/internal/buyers/postgres.go:102>); [db/migrations/003_milestone2_buyers_identity.sql:94](</Users/macbookpro/Documents/Kredit.com/db/migrations/003_milestone2_buyers_identity.sql:94>)

CreateInvitation generates a raw token but omits token_hash from its INSERT and parameters. The schema requires a non-null unique token_hash and supplies no default; no subsequent migration removes this requirement. Every creation through this durable adapter fails before returning a usable invitation. Persist the hash of the generated token in the same transaction. The similarly named generated SQL query includes the hash, but it is not the implementation used here.

**Verification:** Static SQL/schema contradiction; PostgreSQL was unavailable. Existing buyer PostgreSQL tests cover nil-pool behavior and encryption rather than a successful invitation INSERT.

**Regression check after correction:** Create and redeem a real invitation against a fully migrated database; assert no raw token is stored.

### A08 · P1 · Repayments do not restore trade-line capacity

**Locations:** [internal/tradelines/store.go:593](</Users/macbookpro/Documents/Kredit.com/internal/tradelines/store.go:593>); [internal/tradelines/postgres.go:253](</Users/macbookpro/Documents/Kredit.com/internal/tradelines/postgres.go:253>); [internal/payments/postgres.go:36](</Users/macbookpro/Documents/Kredit.com/internal/payments/postgres.go:36>)

Drawdown activation increases current exposure, but no production caller invokes UpdateOutstanding when payments, reversals, write-offs or dispute adjustments change the obligation balance. Cross-repository searches found no repayment-driven SQL/trigger updating trade-line exposure either. A fully repaid drawdown therefore continues consuming capacity and inflates utilization until some separate correction is made. Update the linked line’s exposure atomically with balance changes, accounting for the previous outstanding value on repeated partial payments.

**Verification:** Static end-to-end call-site and SQL write-path analysis; no live repayment run.

**Regression check after correction:** Activate a drawdown, partially repay, repay fully and reverse a payment; verify the line’s exposure and available capacity after every step.

### A09 · P2 · Lowering a trade-line limit violates the persisted balance constraint

**Locations:** [internal/tradelines/postgres.go:445](</Users/macbookpro/Documents/Kredit.com/internal/tradelines/postgres.go:445>); [internal/tradelines/store.go:657](</Users/macbookpro/Documents/Kredit.com/internal/tradelines/store.go:657>); [db/migrations/007_milestone6_trade_lines.sql:24](</Users/macbookpro/Documents/Kredit.com/db/migrations/007_milestone6_trade_lines.sql:24>)

ReduceLimit changes ApprovedLimitKobo and AvailableLimitKobo in memory. The persistence UPSERT updates available_limit_kobo but omits approved_limit_kobo from its conflict-update clause. On an existing line, a genuine reduction leaves the old approved limit in PostgreSQL and violates the available = approved − exposure − reserved constraint, rolling back the operation. Persist both changed values together.

**Verification:** Static mutation/UPSERT/check-constraint trace.

**Regression check after correction:** Reduce an existing PostgreSQL line’s unused capacity and reload it; assert both approved and available limits changed consistently.

### A10 · P2 · Dispute adjustments leave the repayment schedule overstated

**Locations:** [internal/disputes/postgres.go:147](</Users/macbookpro/Documents/Kredit.com/internal/disputes/postgres.go:147>); [internal/operations/postgres.go:287](</Users/macbookpro/Documents/Kredit.com/internal/operations/postgres.go:287>)

The durable dispute adjustment posts a ledger entry and reduces the obligation/snapshot balance without reducing the unpaid schedule items. For a 100-unit obligation adjusted down by 20, the schedule still asks for 100; paying the remaining 80 leaves a residual schedule amount despite a paid obligation. The collection engine’s outstanding cap limits collection, but does not repair the contradictory schedule or reconciliation result. Apply the same transactionally consistent schedule reduction required for other principal adjustments.

**Verification:** Static transaction trace; compared with the schedule handling in operation adjustments.

**Regression check after correction:** Resolve a dispute with an adjustment, pay the remainder, and assert zero obligation balance, zero unpaid scheduled amount and no reconciliation discrepancy.

### A11 · P2 · Deemed acceptance is implemented but never invoked by the running app

**Locations:** [internal/credit/postgres.go:900](</Users/macbookpro/Documents/Kredit.com/internal/credit/postgres.go:900>); [internal/credit/store.go:743](</Users/macbookpro/Documents/Kredit.com/internal/credit/store.go:743>)

AutoActivateMatured has no production call site in the API, worker or jobs; only tests invoke it. Consequently a released sale awaiting the configured deemed-acceptance timeout remains pending without a buyer action. The PostgreSQL method additionally searches only the process-local request map, so simply adding a timer would miss unloaded requests after a restart. Schedule durable processing that queries eligible rows and performs the normal guarded activation.

**Verification:** Repository-wide call-site search and worker/runtime review.

**Regression check after correction:** Release a sale, restart the worker, advance past its acceptance deadline and verify exactly one durable activation; reported receipt issues must remain blocked.

### A12 · P2 · One revoked Mono mandate suspends unrelated trade lines for the same buyer

**Locations:** [internal/web/mono_handlers.go:93](</Users/macbookpro/Documents/Kredit.com/internal/web/mono_handlers.go:93>)

The revoked/blocked mandate path iterates every buyer line and suspends it without checking which mandate the line uses. A buyer with two supplier-specific mandates loses access to both lines when only one mandate is cancelled. It also attempts to set each line’s mandate state using the provider reference, while ignoring errors. Restrict the mutation to lines bound to the affected internal mandate and handle persistence failures.

**Verification:** Static event-to-line mutation trace; no provider event was sent.

**Regression check after correction:** With two independent mandates and lines, revoke one and assert that only its bound line changes.

### A13 · P2 · Changing finance MFA readiness clears valid owner MFA readiness

**Locations:** [internal/onboarding/store.go:291](</Users/macbookpro/Documents/Kredit.com/internal/onboarding/store.go:291>)

When financeMFA changes while ownerMFA remains true and already has a timestamp, SyncSecurity enters the mutation, skips the “ownerMFA and timestamp is zero” branch, and clears the timestamp in else. This incorrectly marks an already protected owner as incomplete and can temporarily block onboarding readiness. Preserve an existing timestamp whenever ownerMFA is true; clear it only when ownerMFA is false.

**Verification:** Store probe: owner MFA was true before the finance-state change and false afterwards.

**Regression check after correction:** Toggle only finance readiness in both directions and assert that valid owner MFA evidence is preserved.

### A14 · P2 · Unrelated onboarding edits overwrite who accepted terms and privacy

**Locations:** [internal/onboarding/postgres.go:140](</Users/macbookpro/Documents/Kredit.com/internal/onboarding/postgres.go:140>)

The shared profile UPDATE sets terms_accepted_by and privacy_accepted_by to the current mutation actor whenever the corresponding existing acceptance timestamp is non-null. Editing settlement or security after another person accepted the documents therefore attributes the old acceptance to the new actor, or can clear it for a non-UUID system actor. Preserve the original acceptor unless recording a new explicit consent event.

**Verification:** Static shared-mutation SQL analysis.

**Regression check after correction:** Have one actor accept, another edit an unrelated field, then verify original consent actor, version and timestamp remain unchanged.

### A15 · P2 · Production recovery never delivers its required continuation credentials

**Locations:** [internal/web/user_control_handlers.go:86](</Users/macbookpro/Documents/Kredit.com/internal/web/user_control_handlers.go:86>); [internal/web/user_control_handlers.go:212](</Users/macbookpro/Documents/Kredit.com/internal/web/user_control_handlers.go:212>); [internal/notifications/store.go:503](</Users/macbookpro/Documents/Kredit.com/internal/notifications/store.go:503>); [web/src/routes/recover/+page.svelte:7](</Users/macbookpro/Documents/Kredit.com/web/src/routes/recover/+page.svelte:7>)

The request ID is returned only in development. On approval, the raw completion token is likewise returned only in development, then disappears after its hash is stored. The notification helper sends neither a continuation URL nor the token, and the recovery templates do not even interpolate the reference. The public screen needs the request ID to submit evidence and the completion token to finish, so the standard production flow cannot progress. Deliver short-lived, recipient-bound recovery continuations through the verified channel.

**Verification:** Static HTTP response, token lifecycle, notification template and screen trace.

**Regression check after correction:** Run the complete non-development recovery flow using delivered messages without developer fields or database/operator intervention.

### A16 · P2 · Completing recovery leaves the lost authenticator as the required factor

**Locations:** [internal/usercontrol/store.go:349](</Users/macbookpro/Documents/Kredit.com/internal/usercontrol/store.go:349>); [internal/web/user_control_handlers.go:173](</Users/macbookpro/Documents/Kredit.com/internal/web/user_control_handlers.go:173>); [internal/auth/postgres.go:387](</Users/macbookpro/Documents/Kredit.com/internal/auth/postgres.go:387>)

Completion changes the recovery request state and revokes backup codes; the handler then revokes sessions. Neither step retires the inaccessible MFA method nor grants a controlled replacement enrollment. A user recovering from a lost authenticator still needs that same authenticator for protected actions, and the active-method uniqueness rule prevents starting another enrollment. Tie successful recovery to a tightly scoped factor-replacement flow and make its state changes recoverable on partial failure.

**Verification:** Static recovery/authentication lifecycle review.

**Regression check after correction:** Recover an account after losing TOTP, enroll and verify a replacement under restricted authority, then confirm the old factor and sessions no longer work.

### A17 · P2 · An abandoned TOTP enrollment cannot be restarted

**Locations:** [internal/auth/postgres.go:407](</Users/macbookpro/Documents/Kredit.com/internal/auth/postgres.go:407>); [db/migrations/002_milestone1_auth_org.sql:73](</Users/macbookpro/Documents/Kredit.com/db/migrations/002_milestone1_auth_org.sql:73>)

BeginTOTPEnrollment always inserts an unrevoked method. The unique index covers unverified methods too. Starting enrollment, losing the displayed secret and retrying therefore hits a uniqueness violation, which is misleadingly returned as “user not found.” Provide a safe resume or replacement path for unverified enrollment without weakening protection for already verified factors.

**Verification:** Static INSERT/index/UI lifecycle trace.

**Regression check after correction:** Start enrollment, reload before verification, restart and successfully verify; a verified factor must still require the proper replacement authorization.

### A18 · P2 · Notification preferences cannot be saved from the settings page

**Locations:** [web/src/routes/app/settings/notifications/+page.svelte:7](</Users/macbookpro/Documents/Kredit.com/web/src/routes/app/settings/notifications/+page.svelte:7>); [internal/web/user_control_handlers.go:26](</Users/macbookpro/Documents/Kredit.com/internal/web/user_control_handlers.go:26>); [internal/web/http_helpers.go:18](</Users/macbookpro/Documents/Kredit.com/internal/web/http_helpers.go:18>)

The page spreads the entire GET preferences object into its PUT body, including opted_out and version, while preferenceUpdate accepts neither field. The strict decoder rejects the request before saving. Construct the write payload from the accepted fields, using expected_version for concurrency.

**Verification:** An exact request-shape/strict-decoder probe returned json: unknown field "opted_out".

**Regression check after correction:** Load preferences from the real API, change one setting and save the returned object through the screen successfully.

### A19 · P2 · Protected operations previews omit a required idempotency key

**Locations:** [web/src/routes/admin/controls/+page.svelte:17](</Users/macbookpro/Documents/Kredit.com/web/src/routes/admin/controls/+page.svelte:17>); [web/src/routes/admin/jobs/+page.svelte:8](</Users/macbookpro/Documents/Kredit.com/web/src/routes/admin/jobs/+page.svelte:8>); [web/src/routes/admin/provider-events/+page.svelte:6](</Users/macbookpro/Documents/Kredit.com/web/src/routes/admin/provider-events/+page.svelte:6>); [internal/web/server.go:220](</Users/macbookpro/Documents/Kredit.com/internal/web/server.go:220>)

The controls, jobs and provider-event screens POST to /ops/commands/preview without Idempotency-Key. The middleware matches /ops/commands and requires the header; its special preview exemption covers only business-policy previews. The request is rejected before the preview handler, so preview-dependent actions cannot proceed. Make the preview contract consistent between client and middleware.

**Verification:** Exact middleware predicate probe returned true for the preview route; request headers inspected in all three screens.

**Regression check after correction:** Open each affected screen and complete preview → authorized apply with the actual middleware enabled.

### A20 · P2 · Editing a draft silently shifts its collection time

**Locations:** [web/src/routes/app/credit/[id]/+page.svelte:26](</Users/macbookpro/Documents/Kredit.com/web/src/routes/app/credit/[id]/+page.svelte:26>); [web/src/routes/app/credit/[id]/+page.svelte:36](</Users/macbookpro/Documents/Kredit.com/web/src/routes/app/credit/[id]/+page.svelte:36>)

The draft editor fills datetime-local using a UTC ISO string with its timezone stripped. Saving then interprets that value as browser-local time. In Lagos, an unchanged 09:00 collection becomes 08:00 local after one save; repeating the edit repeats the drift. Format the form value in the intended local timezone and serialize that same timezone consistently.

**Verification:** Node probe with TZ=Africa/Lagos reproduced a −60 minute change for an unchanged draft value.

**Regression check after correction:** Round-trip an unchanged draft in Lagos and at least one other timezone and assert identical stored instants.

### A21 · P2 · Correcting a rejected form reuses a key tied to the old payload

**Locations:** [web/src/routes/app/credit/new/+page.svelte:28](</Users/macbookpro/Documents/Kredit.com/web/src/routes/app/credit/new/+page.svelte:28>); [web/src/routes/buyer-invitations/[token]/+page.svelte:50](</Users/macbookpro/Documents/Kredit.com/web/src/routes/buyer-invitations/[token]/+page.svelte:50>); [internal/web/server.go:199](</Users/macbookpro/Documents/Kredit.com/internal/web/server.go:199>)

The sale and buyer-invitation forms keep their idempotency keys after a definite server rejection. The server stores completed error responses and binds the key to the body. Correcting a rejected sale or replacing a wrong invitation OTP sends a different body with the old key and receives a conflict, trapping the user until a reset/reload. Preserve the key for an uncertain identical retry, but create a new logical request when a definitively rejected payload is corrected.

**Verification:** Static frontend key lifecycle and middleware/hash-conflict trace.

**Regression check after correction:** Submit a validly shaped but rejected request, correct it without reloading and succeed; a network retry of the unchanged request must remain duplicate-safe.

### A22 · P2 · The invoice form allows files too large for its JSON endpoint

**Locations:** [web/src/routes/app/credit/new/+page.svelte:26](</Users/macbookpro/Documents/Kredit.com/web/src/routes/app/credit/new/+page.svelte:26>); [internal/web/http_helpers.go:16](</Users/macbookpro/Documents/Kredit.com/internal/web/http_helpers.go:16>)

The form accepts invoices up to 2 MiB and base64-encodes them into JSON. The shared decoder caps the entire JSON request at 1 MiB, so even an approximately 768 KiB file plus metadata exceeds the limit. Users can select files explicitly advertised as supported and still receive a server rejection. Use the existing direct-upload flow or align the UI limit with encoded body size and metadata overhead.

**Verification:** Static client/decoder size calculation; 4/3 base64 expansion makes the mismatch deterministic.

**Regression check after correction:** Exercise invoice uploads just below and above the effective limit and at the advertised maximum.

### A23 · P2 · Overdue and near-term totals include future installment principal

**Locations:** [internal/reports/store.go:516](</Users/macbookpro/Documents/Kredit.com/internal/reports/store.go:516>); [internal/reports/store.go:638](</Users/macbookpro/Documents/Kredit.com/internal/reports/store.go:638>); [internal/reports/store.go:648](</Users/macbookpro/Documents/Kredit.com/internal/reports/store.go:648>)

Reporting selects the earliest unpaid installment to classify the obligation, then adds the entire outstanding obligation balance to overdue, due-today or due-this-week totals. If only one of two equal installments is overdue, both are reported overdue. Ageing similarly assigns the whole balance to the earliest installment bucket. Calculate these amounts from remaining installment balances in the relevant date window.

**Verification:** Report-store probe: 5,000 kobo overdue and 5,000 kobo future was reported as 10,000 kobo overdue.

**Regression check after correction:** Mix paid, overdue, due-today and future installments and compare every summary/ageing total with the unpaid schedule amounts.

### A24 · P2 · An existing buyer cannot accept another supplier invitation

**Locations:** [internal/buyers/postgres.go:206](</Users/macbookpro/Documents/Kredit.com/internal/buyers/postgres.go:206>); [db/migrations/003_milestone2_buyers_identity.sql:4](</Users/macbookpro/Documents/Kredit.com/db/migrations/003_milestone2_buyers_identity.sql:4>)

Accepting an invitation always inserts a new person for the authenticated user. persons.user_id is unique, so a buyer already onboarded through another invitation fails the second acceptance rather than acquiring a relationship with another supplier. Reuse the existing verified person and intentionally select/reuse the appropriate business, preserving tenant relationship boundaries.

**Verification:** Static acceptance INSERT and uniqueness-constraint trace; database execution unavailable.

**Regression check after correction:** Onboard the same user with two suppliers and verify both relationships, one stable person identity and no cross-supplier data leakage.

### A25 · P2 · One audit event without a request ID hides the whole organization audit list

**Locations:** [internal/audit/store.go:150](</Users/macbookpro/Documents/Kredit.com/internal/audit/store.go:150>); [internal/audit/store.go:141](</Users/macbookpro/Documents/Kredit.com/internal/audit/store.go:141>); [internal/web/credit_handlers.go:512](</Users/macbookpro/Documents/Kredit.com/internal/web/credit_handlers.go:512>)

Append stores an empty RequestID as SQL NULL, and normal credit audit events leave it empty. ListForOrganization selects nullable request_id without COALESCE and scans it into a Go string. The scan fails and the method returns nil for the entire result, so legitimate audit history appears empty. Handle nullable columns explicitly and propagate read failures instead of presenting them as no records.

**Verification:** Static nullability/write/read trace; database execution unavailable.

**Regression check after correction:** List a mixture of events with and without request IDs and verify every event is returned; a true query error must be distinguishable from an empty history.

### A26 · P2 · Activity screen reads field names the audit endpoint does not emit

**Locations:** [internal/audit/store.go:16](</Users/macbookpro/Documents/Kredit.com/internal/audit/store.go:16>); [internal/web/organization_handlers.go:201](</Users/macbookpro/Documents/Kredit.com/internal/web/organization_handlers.go:201>); [web/src/routes/app/activity/+page.svelte:15](</Users/macbookpro/Documents/Kredit.com/web/src/routes/app/activity/+page.svelte:15>)

Audit.Event has no JSON tags, so the endpoint emits Action and At. The screen reads item.action and item.created_at. Even when the list is nonempty, the action label is missing and its timestamp becomes Invalid Date. Define a stable response DTO or JSON tags and consume the same contract in the page.

**Verification:** Static Go JSON serialization contract and Svelte field access comparison; independent of the database NULL issue.

**Regression check after correction:** Render an actual audit response on the activity screen and assert readable action text and a valid event time.

### A27 · P2 · Database readiness accepts a schema older than authentication requires

**Locations:** [internal/db/persistence_contract.go:188](</Users/macbookpro/Documents/Kredit.com/internal/db/persistence_contract.go:188>); [db/migrations/068_session_idle_and_mfa_throttle.sql:1](</Users/macbookpro/Documents/Kredit.com/db/migrations/068_session_idle_and_mfa_throttle.sql:1>); [db/migrations/069_shared_authentication_rate_limits.sql:1](</Users/macbookpro/Documents/Kredit.com/db/migrations/069_shared_authentication_rate_limits.sql:1>)

The minimum migration remains 66, although the code uses session/MFA additions from 68 and shared authentication-rate-limit capabilities from 69. The required columns/functions list does not cover these additions, so a version-66 database can pass this persistence gate and then fail authentication operations. Require the actual schema floor and explicitly check essential authentication capabilities.

**Verification:** Static readiness contract versus migration and authentication code comparison.

**Regression check after correction:** Startup against version 66 and 68 must fail clearly; a fresh schema through 69 must satisfy all required capabilities.

### A28 · P2 · Buyers cannot report receipt problems before confirming a normal sale

**Locations:** [web/src/routes/buyer/credit-requests/[requestID]/+page.svelte:58](</Users/macbookpro/Documents/Kredit.com/web/src/routes/buyer/credit-requests/[requestID]/+page.svelte:58>); [internal/web/credit_handlers.go:482](</Users/macbookpro/Documents/Kredit.com/internal/web/credit_handlers.go:482>)

At RECEIPT_CONFIRMATION_PENDING, the page offers only “Yes, I got the goods.” Its problem-report form is rendered only after an obligation exists, which requires receipt confirmation. The backend supports a receipt-issue state, but this screen never submits it. A buyer with missing/damaged goods must either stop or falsely confirm receipt to reach the later dispute form. Expose the receipt-issue action and reason before activation, as the trade-line receipt screen already does.

**Verification:** Static screen-state/backend transition comparison.

**Regression check after correction:** For an accepted, released normal sale with no obligation, report a receipt issue and verify collection remains blocked without confirming receipt.

### A29 · P2 · Seller-specific reminder withdrawal is not enforced during delivery

**Locations:** [web/src/routes/buyer/permissions/+page.svelte:7](</Users/macbookpro/Documents/Kredit.com/web/src/routes/buyer/permissions/+page.svelte:7>); [internal/relationships/store.go:33](</Users/macbookpro/Documents/Kredit.com/internal/relationships/store.go:33>); [internal/notifications/store.go:316](</Users/macbookpro/Documents/Kredit.com/internal/notifications/store.go:316>); [internal/web/outbox_notifications.go:84](</Users/macbookpro/Documents/Kredit.com/internal/web/outbox_notifications.go:84>)

The permissions screen persists a seller-specific payment_reminders consent and says that seller can no longer send optional reminders. Delivery checks only the user’s global preferences; neither the outbox translation nor notification service consults relationship consents. Optional upcoming-payment reminders can therefore continue despite the displayed withdrawal. Apply the latest supplier/buyer consent before scheduling or delivering optional reminders, while retaining the intended treatment of required notices.

**Verification:** Static consent-write, global-preference and routine-reminder delivery trace.

**Regression check after correction:** Withdraw reminders for supplier A while keeping supplier B enabled; upcoming reminders must be suppressed only for A, including queued deliveries.

### A30 · P2 · The invoice “Open” link opens JSON instead of the document

**Locations:** [web/src/routes/app/credit/[id]/+page.svelte:74](</Users/macbookpro/Documents/Kredit.com/web/src/routes/app/credit/[id]/+page.svelte:74>); [internal/web/document_handlers.go:81](</Users/macbookpro/Documents/Kredit.com/internal/web/document_handlers.go:81>)

The credit detail page navigates directly to the document download API in a new tab. That endpoint returns a JSON object containing the signed URL rather than redirecting or streaming the file. The user sees metadata/JSON instead of the invoice. Fetch the authorized URL and navigate to it, or provide a deliberately redirecting endpoint with the same access and scan checks.

**Verification:** Static anchor target and HTTP response comparison; applies when invoice_document_id is present.

**Regression check after correction:** Open an attached, CLEAN invoice from the sale page and verify it displays the actual PDF/image; pending or rejected documents must remain unavailable.

## File register coverage

| Category | Files | Method |
|---|---:|---|
| API contract | 2 | Endpoint/method and explicit client-call consistency checks; not every request/response shape proven |
| Archived draft | 1 | Read and compared with active workflow; not executed by GitHub from this location |
| Configuration and dependency metadata | 18 | Configuration/toolchain/build dependency inspection; no completed dependency vulnerability or deployment certification |
| Database fixtures | 1 | Fixture/source and test setup review; not loaded into a database |
| Database migrations | 69 | Schema-wide static searches and targeted constraint, function, policy and call-site review; not a migration execution |
| Database queries | 14 | Manual SQL query review against callers/schema; generated output compiled; no live SQL execution |
| Documentation and evidence | 63 | Integrity, links and supporting contract references; not substantive legal or historical evidence validation |
| Editorial content code | 1 | Content completeness/structure checks and content-loader source review; editorial claims not fact-checked |
| Frontend implementation | 162 | Manual handwritten logic/component/route review plus Svelte type checks, build and route/contract scans; no full browser verification |
| Generated code | 20 | Repository integrity and compiler/type/contract checks; generated bodies not manually reviewed line by line |
| Go implementation | 121 | Manual source and cross-file behavior review; Go formatting and package checks |
| Go tests | 110 | Package test execution where buildable; focused coverage inspection, not exhaustive manual assertion review |
| Infrastructure and orchestration | 18 | Static configuration review; infrastructure not provisioned or deployed |
| Repository scripts | 35 | Source review and syntax/structural checks; only specifically listed checks executed |
| Static assets | 10 | Inventory, build linkage and structural checks; no exhaustive visual/pixel inspection |
| Test assets | 19 | Repository structure/reference inspection; execution status listed separately |

## Evidence and remaining confidence limits

The local probes used actual exported store implementations. Because internal/web could not compile, the preview predicate and notification request type were extracted from the current source into an isolated probe; these two results validate the predicate and strict decoding, not a running HTTP response. The timezone probe evaluates the exact conversion pattern under Africa/Lagos.

```text
foreign-actor drawdown cancellation: state=CANCELLED error=<nil>
overdue reporting: actual due=5000 reported=10000 error=<nil>
owner demotion: role=viewer error=<nil>
owner MFA before=true after finance change=false
agreed collection=2026-09-11 05:00:00 +0100 WAT generated=2026-09-10 11:00:00 +0100 WAT delta=-18h0m0s
admin preview requires key: true
notification UI roundtrip: json: unknown field "opted_out"
{"original":"2026-09-11T09:00:00+01:00","field":"2026-09-11T08:00","saved":"2026-09-11T07:00:00.000Z","deltaMinutes":-60}
```

The most consequential testing gap is the difference between memory stores and durable adapters. The membership and owner checks, invitation inserts, schedule/accounting projection updates and nullable database reads need real PostgreSQL coverage. Some files named postgres_test.go test only construction or nil-pool behavior; their package passing does not prove the adapter works with PostgreSQL.

The active CI workflow also does not explicitly install golangci-lint even though scripts/lint.sh requires it in CI. The archived workflow draft does include that installation. Clean-runner tool availability was not verified here, so this is recorded as a deployment-check gap rather than counted as an independently reproduced CI failure.

Recommended correction order is A01, then authorization/isolation findings A02–A04, accepted-date and immutable-document integrity A05–A06, core onboarding/financial state A07–A12, and the remaining recovery, settings, reporting and interface defects. The regression checks above specify what must be demonstrated after corrections; no fixes have been made in this audit.
