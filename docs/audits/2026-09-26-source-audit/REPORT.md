# Source audit — 26 September 2026

**Status: complete file-by-file source review, with fixes applied in this working tree. This is not a production certification or a guarantee of future correctness.**

Baseline: `62fe2ebe92de5f985b05d7b7ff710e891b6fb356`. The working tree was clean when review began. Review and fixes were performed by one reviewer without agents or tests, as requested. Builds, type checking, linting and syntax parsing are the only automated checks used. No migration, deployment, database write or provider operation was executed.

The final inventory contains **866 reviewed source/configuration files**: 859 files present at the baseline and 7 newly authored files. There are no partial or pending files in the defined source scope. The inventory was expanded during the final cross-check to include extensionless configuration, committed environment examples, the web manifest and the secondary API contract. [coverage.json](coverage.json) records each path, snapshot hash and final reviewed hash. `reviewed` means the whole file was inspected; exact shared API objects were compared with the already-read canonical contract and every differing field was inspected. It does not certify all behavior or dependencies.

The review covered Go application and worker code, Svelte/TypeScript frontend code, SQL migrations and seed/configuration, API definitions, build/deployment scripts, infrastructure and workflow configuration. Tests and test-support files, third-party implementations, lockfile contents, generated output, actual local secrets, historical audit/evidence snapshots and most prose/media are excluded. Test runner and workflow configuration was read without invoking tests. Relevant deployment guidance was also reviewed and updated. Historical audit documents are not treated as proof that this revision is correct. Source-pattern matches in `screening-candidates.json` are investigation aids, not validated findings.

The fixes below are grouped as A01–A33. All are local source changes. No commit, deployment, database migration, provider request or production operation was performed.

## Fixes in this working tree

### A02 / A03 — P1: deploy the actual release and require a readable backup

The old cutover copied stale-capable host binaries that the supplied containers did not use, could skip the worker and converted restart failure into success. Its backup caller also expected an unrelated gzip filename and could hide a dump failure. The replacement deploys one clean committed source archive, verifies the transferred checksum, builds API/worker/migrator together under unique release image tags, stops writers, and uses the shared custom-archive backup interface before any migration. Backup completion requires successful `pg_dump`, full archive decoding with `pg_restore` and a checksum sidecar. The dedicated backup login explicitly assumes its non-inherited backup role. Both running images and health states are verified.

Prior source, image IDs/tags and the backup manifest are retained. Failure after stopping writers leaves them stopped; no automatic schema downgrade or restoration occurs. A generated Compose override and systemd drop-in select the verified release on reboot. Staging cannot replace image tags selected by the running release. Administrative commands after cutover must include that override; both deployment guides now say so. The cutover runbook now matches the script and describes its actual checks. No remote command, container build, backup or deployment was executed during this audit. Docker/Compose behavior and restoration remain unexecuted verification limits.

### A01 — P1: keep command identity stable across session changes

Request deduplication previously included the session ID and mutable access state in its identity. A committed request whose response was lost could execute again after MFA or reauthentication. Memory and PostgreSQL stores now reserve by method, resource, actor and key; authorization metadata stays attached to the original record for response disclosure. Migration `211_stable_idempotency_identity.sql` indexes that identity, serializes reservations (including older writers) and retains legacy duplicates for explicit recovery rather than choosing an outcome or repeating a write. Expiration keeps A04's protection for uncertain outcomes.

Replays under changed access return a specific conflict and instruct the user to read the current resource. The response scope now also accounts for MFA freshness, membership lifecycle versions, delegation expiry, branch scopes/reassignments and platform-role assignment identity. Unchanged-access replays retain the original response behavior. These checks are database snapshots, not atomic locks over the entire HTTP response. Apply migration 211 with writers stopped before starting updated processes. Existing expired completed 200–499 outcomes retain the documented 24-hour reuse contract; records already deleted cannot be reconstructed. No migrations have been applied.

### A04 — P1: preserve uncertain idempotency outcomes

Migration 200 allowed expiration to delete reservations that never recorded a response. The HTTP reservation, domain commit and response recording are separate operations; a crash between the latter two leaves no response even though the business operation succeeded. Age alone cannot justify executing it again.

New migration `207_preserve_uncertain_idempotency.sql` allows expiration-based deletion only for completed HTTP 200–499 responses. Incomplete reservations and server-error outcomes remain available for reconciliation, including a panic after a business commit. The memory store uses the same rule. The final startup check requires migration 212, covering this and the subsequent fixes.

**Rollout:** apply migrations through 212 with writers stopped before starting the updated API/worker. It has been reviewed as source only and has not been applied. Its down migration intentionally does not restore unsafe deletion. Existing grants are preserved by `CREATE OR REPLACE`; public execution remains revoked. The fix cannot recover evidence already deleted. Uncertain records need controlled reconciliation and retention management; they are intentionally not made retryable by a timer. Completed 200–499 records still have the existing expiry contract. Cross-session deduplication is addressed separately in A01.

### A05 — P2: preserve frontend client attribution through ingress

`Caddyfile.prod` stripped the three signed client-IP envelope headers produced by the frontend proxy. With the documented HTTPS API route through Caddy, the API could not recover the visitor IP and rate limits could group unrelated visitors under frontend egress addresses.

The configuration now preserves the signed envelope. The API still verifies its HMAC, timestamp, method and request URI. Unsigned forwarding claims remain stripped or replaced. The production deployment guide now describes this boundary. Caddy has not been reloaded or executed locally.

### A06 — P2: keep fetch cancellation active during body reads

`web/src/lib/api/reliable.ts` cleared its timer and removed the caller's cancellation listener when `fetch` returned headers. Callers could then hang consuming a delayed response body, beyond the intended timeout, or continue after cancellation.

`boundedFetch` now uses a deadline signal combined with the caller's signal. Both remain attached while the response is consumed. This uses `AbortSignal.timeout` and `AbortSignal.any`, consistent with the project's modern browser/runtime APIs.

### A07 — P2: suppress database values in error diagnostics

PostgreSQL driver errors can contain submitted values without the words `postgres` or `pgx`. `SafeError` previously recognized SQLSTATE only inside a separate keyword match, permitting those messages to retain user data.

`internal/platform/logging/logging.go` now recognizes SQLSTATE before keyword filtering and retains only SQLSTATE/constraint identifiers. `internal/web/http_helpers.go` also treats SQLSTATE as an internal-detail marker. This is a focused fix, not a proof that every logging or stored-error path is free of personal data.

### A08 — P2: propagate session metadata and refresh failures

`SessionFromToken` silently ignored failure to open the metadata transaction, read MFA metadata, refresh the idle timestamp or commit. It could report a refreshed session even when persistence failed.

`internal/auth/postgres.go` now returns the existing session-unavailable error category for those failures, uses bounded transaction cleanup independent of request cancellation, and reports the new timestamp only after commit. No session schema or authentication policy was changed.

### A09 — P2: honor the selected migration directory

`scripts/run-migrations.sh` validated and announced its directory argument but did not pass it to `cmd/migrate`; the command always selected `db/migrations` under its working directory.

The command now accepts `--migrations-dir`, validates options and the directory before opening a database connection, and retains the existing `down` restriction. The wrapper passes its selected directory to either the compiled command or `go run`. Default invocations and `migrate down` remain supported. Existing compiled migrator binaries must be rebuilt when updating the wrapper; the old executable does not understand the new flag and will fail rather than silently use another directory.

### A10 — P2: bound credit cache lifetime and release failed listings

`internal/credit/postgres.go` removed evicted requests but retained their mandate entries. Eviction now removes unreferenced mandate aliases while preserving mandates used by other cached requests. A failed listing after pinning some rows also leaked those pins; failed hydration now releases the partially loaded list.

### A11 — P2: snapshot shared values before releasing their locks

Credit persistence now uses the copied request in its snapshot for authority and persistence decisions. Previously it read a mutable cache pointer after releasing the lock. The development trade-line store now copies its newly created result before releasing the lock for the same reason. These changes do not establish that all aggregate cache races have been resolved.

### A12 — P2: compare billing receipt replays at stored precision

`internal/billing/invoices.go` now normalizes receipt dates to PostgreSQL microseconds before validation, storage and replay comparison. An identical request carrying nanoseconds could previously conflict with its saved result. The billing bank-receipt flow already used this normalization.

### A13 — P2: scope development purchasing profile lookup

`internal/buyers/workspaces.go` now requires the requested profile's owner to match the caller in its memory implementation. Previously a workspace ID alone selected another user's business. The PostgreSQL implementation already checks purchasing authority.

The audit event persistence error path now uses `logging.SafeError` as part of A07, preventing SQLSTATE diagnostics from bypassing that sanitization.

### A14 — P2: fail explicitly when the approval inbox cannot be read

`internal/creditapproval/store.go` overwrote query and row-decoding failures while collecting drawdown approvals, and omitted the cursor error check. It now returns those errors and closes the cursor before reading reviewer limits. Decisions also reject pending approvals whose drawdown is no longer awaiting confirmation or goods release; exact completed decision replays retain their prior behavior.

### A15 — P2: preserve actor context in branch-restricted financial reads

Supplier payment-claim listings discarded the authenticated user when setting the organization. They now preserve that actor, so branch policies evaluate the actual caller. Financial-change decisions now establish transaction-local tenant identity before their first proposal lookup; previously proposals in organizations with branch restrictions could be hidden even from an authorized reviewer or buyer. Approval-list reads now also preserve the signed-in actor. The delegated purchasing permission check for payment claims now establishes that actor in SQL and distinguishes infrastructure failures from denials.

### A16 — P2: supply the current date when interpreting messages

The WhatsApp AI prompt asked the provider to calculate relative due dates without supplying today’s date. Both text and voice requests now include the current Nigerian calendar date from the parser clock. Structured results still require the existing user review and validation; model output is not proof of financial intent.

### A17 — P2: reconcile submitted collections before unrelated maintenance

`internal/web/collection_jobs.go` now queues provider reconciliation first. A referral, reminder or policy-maintenance failure previously returned before enqueueing checks of bank requests already in flight.

### A18 — P1: authorize the audience of business notifications

Organization broadcasts previously selected members by role alone, without branch restrictions; queued delivery did not recheck membership after revocation. Broadcasts now retain an explicit organization-audience flag and require current, active, company-wide financial access before queueing and again before delivery. The check evaluates the recipient directly rather than inheriting a worker's branch bypass. Buyer notices retain their separate recipient context.

Migration `208_notification_organization_audience.sql` backfills known broadcast templates and preserves this boundary on rollback. Apply it before starting updated processes; it has not been applied here. Already delivered information cannot be recalled. A membership change racing after the final check can still occur; this change does not claim atomicity across a database authorization read and an external messaging provider.

### A19 — P1: recheck role-management authority under the lifecycle lock

Role revocation took an assignment row lock before the global authority lock used by the database trigger. It also relied on the earlier HTTP authorization check. Revocation now acquires the shared lifecycle lock and validates current authority first, preventing stale authority after concurrent revocation and inconsistent lock ordering. Grants and revocations of administrator/access-administrator roles now recheck their stronger permission inside the transaction; granting one's own role is also rejected at that boundary.

### A20 — P1: keep drawdown approval valid through its intended lifecycle

The drawdown approval fingerprint included reservation, obligation and goods-release fields that change after review. This invalidated an otherwise approved drawdown during release/activation. Migration `209_drawdown_approval_intent.sql` excludes those lifecycle fields and recalculates existing fingerprints from their stored proposals, preserving the originally reviewed terms. It serializes threshold/ceiling changes with the release check, locks current reviewer authority, checks ceilings at decision time and makes the captured proposal immutable. Persisting an already released sale does not require its historical reviewer to retain access forever. Existing mismatched proposals remain mismatched.

### A21 — P2: honor delegated payment-claim authority in the database

The API exposed delegated `claim` permission but the database locking capability required the original buyer user. Migration `210_delegated_payment_claim_lock.sql` accepts current delegated claim authority, locks it against revocation and retains the branch/current-business restrictions. It gives no permission to update debt values; the claim still records the actual submitting user. API and worker startup now require migration 212. Migrations 207–212 have not been applied. Quiesce old writers before applying and starting updated binaries, including for notification audience backfilling and drawdown intent conversion.

### A22 — P1: isolate credit changes until their commit succeeds

Credit draft/create/send/review/accept/mandate/release operations mutated the shared cache before committing. Pins did not serialize commands, and readers could refresh or expose that uncommitted graph. These operations now use a private aggregate copied from a fresh authorized database read, retain the database version check, and install the result only after persistence. Portal reads use their own committed copy; supplier lists read PostgreSQL directly. An unchanged review or existing mandate no longer attempts an incorrect version decrement on persistence. Older lifecycle revisions cannot replace a newer cached revision. The shared cache remains a routing hint for legacy method signatures, so eviction between a prior read and a command may still require refreshing; it cannot substitute for the fresh command read.

### A23 — P2: load public payment-link balances after restart

The signed public payment-link endpoint depended on a prior cache read in the same process. A restart or different replica returned a missing-sale error, and a retained cache could show an old balance. Migration `212_public_payment_intent.sql` exposes only the page's existing display fields through an exact-reference database projection. The HTTP handler still verifies the short-lived signature first. The current balance comes from normalized obligations and private aggregate details are not returned. API/worker startup requires migration 212; it has not been applied.

### A24 — P2: use the frontend's local health endpoint in Kubernetes

The web deployment's probes now use `/healthz`, matching the container's existing liveness contract, rather than rendering the homepage and its dependencies.

### A25 — P2: bind generated review claims to exact file contents

`build-review-register.py` classified new or changed files as fully read based on their directory. It now carries a review depth forward only when explicit prior evidence contains the same path and SHA-256; changed or new files are unreviewed. The data-inventory generator now writes a marked proposal instead of overwriting the explicit base inventory and duplicating fields maintained in its additions manifest. The new notification audience column is explicitly inventoried with review-pending defaults. Neither generator was run.

### A26 — P2: fail grant-inventory generation on database errors

`db-grant-inventory.sh` placed its query group on the left side of `&&`, suppressing Bash's exit-on-error behavior, and could print success after a failed query. It now writes through a unique temporary file, propagates the query failure and replaces the inventory only on success. It was reviewed and syntax-checked only; no database was queried.

### A27 — P2: keep development services local and initialization bounded

The local Compose API and web ports listened on every host interface while using development authentication. Both now bind loopback, matching the database, object-storage and simulator ports. Object-storage initialization stops after 60 failed attempts. The local web health check uses its dedicated endpoint. The isolated recovery workflow also preserves its database query timeout options when invoking the containerized client. No containers or workflows were run.

### A28 — P2: decode referral confirmations as referral responses

DSA enrollment, updates and payout actions, plus business referral confirmation, used the default purchase decoder. Successful server writes were therefore shown as unconfirmed and their request identities retained. Each mutation caller now supplies an explicit decoder; referral actions validate their saved confirmation or prepared payout reference, while consumer actions validate purchases. Shared consumer reads now use the existing safe problem-response handler. Fee-authorization reads and actions discard responses after a business change and clear identity/form details when the scope changes. Deck navigation preserves the router's browser-history state. Sign-out cleanup also covers older underscore/hyphen Kredit storage keys, including referral and feedback state.

### A29 — P2: keep customer, business and page identity accurate

Payment lists and search results used the sale ID as their row identity, even when several payments belonged to that sale. They now require and key on `payment_id`. Customer credit-limit selection now identifies both the owner and the customer business, uses the selection event's current value and clears the prior mandate when the customer changes.

Changing the business in admin billing clears the prior receipt, review and pending action state. Invitation token changes clear the selected purchasing workspace and expired-link state. Leaving partner import stops the remaining batch from submitting further invitations after the current request. Purchasing consent links open the exact versions recorded by the form. Team-read failures no longer claim that the business has no staff.

The dashboard and referral page tolerate unavailable browser storage; failure to remove a local referral hint no longer turns a confirmed server result into an error. Four demo labels now use the foreground color on their light panels rather than the primary button text color. These UI changes were type-checked and linted, without browser execution.

### A30 — P2: reject incomplete enterprise report results

The frontend supplied healthy ratings, zero figures and a “Verified” label when required enterprise-report fields were absent. It now validates the organization, timestamp, fingerprint format, rating, bounded ratios, counts and all ageing buckets before displaying the report. An unavailable risk report gets its own error message while the other successful reports remain usable. `NO_DATA` is displayed explicitly. Export downloads reject redirects and non-CSV responses. Fingerprint format validation is not a cryptographic authenticity check.

### A31 — P2: require verified delivery state and mutation confirmations

The deliveries page could display an empty delivery history after a failed read and allowed actions without a successfully loaded record. It now separates loading, failed and loaded state, validates integer quantities, and gates actions on the current verified read. Shipment and credit-note creation must return their expected shapes; approval must return the same note ID and approved state. An arbitrary JSON object no longer clears the mutation identity or confirms success.

### A32 — P2: preserve exact saved credit-policy amounts

The credit-policy form converted a decimal monetary string through JavaScript `Number` and substituted a hard-coded credit limit when the response was incomplete. It now keeps the money input as decimal text, uses the existing exact kobo conversion and rejects missing or invalid saved terms. Loading failures keep the form unavailable for saving.

### A33 — P2: align the API contract and remove its drifting duplicate

The canonical contract now includes all supported purchasing delegation actions and drawdown ceilings, the instalment-preview inputs, typed referral overview fields, and the existing reconciliation release/takeover actions and request-recovery search result. Feedback `yes`/`no` values are quoted for YAML parser compatibility. Fee-report and idempotency descriptions match the actual behavior.

`docs/api/openapi.yaml` was a second, outdated full definition even though its README called it a pointer. All legacy objects were compared with the canonical definition; supported operations described only in the older copy were reconciled against the implementation. It is now a relative symlink to `api/openapi.yaml`, retaining the existing documentation path without another editable contract. Consumers must preserve repository symlinks.

## Verification and limits

Final checks are recorded in [checks.json](checks.json):

- Go application build: passed for every command and its production dependencies.
- Backend static lint: passed, 0 issues, with test loading and unused analysis disabled. Unused analysis is excluded because declarations referenced only by excluded tests would otherwise be misreported. The initial final-pass attempt timed out after cache-write warnings; the completed run used a writable cache, two execution threads and a longer timeout.
- Frontend type/Svelte check: passed, 0 errors and 0 warnings, including the final edits.
- Frontend ESLint: passed for 259 source files, 0 errors and 0 warnings. The last two edited files were checked again after their final changes.
- Shell, Python, JSON and YAML syntax: passed for 75 files. Shell scripts were parsed, not executed; Python was parsed without imports or execution.
- Go formatting and patch whitespace: passed. Changed frontend source was formatted.
- OpenAPI validation: passed with 0 errors and **434 warnings**. These comprise 243 missing operation descriptions, 133 missing documented 4xx responses, 53 missing summaries, 2 unused schemas, 1 unspecified license, 1 path-shape warning for role routes with different HTTP methods, and 1 missing-success warning for the deliberately disabled schedule-replacement endpoint. These documentation gaps are retained as explicit maintenance debt; no license, response behavior or redundant prose was invented to silence the tool.

The local toolchain differs from the declared deployment versions: Go 1.27.1 versus 1.26.8, and Node 24.11.0 versus the required >=24.20.0 <25. The checks therefore do not establish compatibility with the exact deployment toolchain. The installed pnpm version is 11.17.0.

**Migrations 207–212 are source-reviewed but unapplied.** Quiesce old API/worker writers, retain a readable backup and apply all pending migrations with the matching release before starting updated processes. The new startup boundary requires migration 212. The migrations intentionally retain integrity protections on `down`; generic schema rollback is inappropriate. The source audit did not execute Docker, PostgreSQL migrations, Caddy, Terraform deployment, a browser, provider calls or recovery operations.

No tests were run or written, and no agents were used. Compiler/type/lint success and complete reading cannot prove concurrency safety, SQL execution against the deployed database, external-provider behavior, browser behavior or crash recovery. The code is materially improved and the defined file-by-file audit is complete; perfection and immunity to future changes cannot be established by this review.
