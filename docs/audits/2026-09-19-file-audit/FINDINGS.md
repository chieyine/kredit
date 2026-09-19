# Kredit source audit — 22 findings

Audit opened: 2026-09-19. Status: **Project-file review complete; findings remain open.** Five P1 findings and seventeen P2 findings. Static evidence only; no runtime reproduction.

Method: source reading and cross-file tracing only. No agents, tests, builds, linters, deployments or application/database execution. No application code changed. These findings are reasoned from source; none is represented as a runtime reproduction. Existing audit reports are not validation evidence.

## F001 — P1 — Identical journal retries fail because posting order is random

Files: [internal/ledger/postgres.go:143–148,201](</Users/macbookpro/Documents/Kredit.com/internal/ledger/postgres.go:143>); [internal/ledger/store.go:177–184](</Users/macbookpro/Documents/Kredit.com/internal/ledger/store.go:177>); [db/migrations/004_milestone3_credit_ledger.sql:16](</Users/macbookpro/Documents/Kredit.com/db/migrations/004_milestone3_credit_ledger.sql:16>).

`postTx` loads an existing transaction after a key conflict and calls `sameTransactionIntent`. The latter compares posting slices position by position. Database posting IDs default to random UUIDs, and the loader orders by those IDs, whereas the requested transaction uses debit/credit construction order. For example, a two-posting debit/credit journal can reload as credit/debit. Retrying the identical journal then returns “idempotency key was reused for a different ledger transaction.” This is a durable journal retry defect; outer domain guards may prevent some paths from reaching it, but the journal itself does not meet its promised replay contract.

Recommendation: persist an explicit posting ordinal or compare canonical posting multisets, preserving duplicate postings rather than collapsing them by account. Keep outbox serialization deterministic as well.

## F002 — P1 — Storage failures can produce a false successful sign-out

Files: [internal/web/auth_handlers.go:100–112](</Users/macbookpro/Documents/Kredit.com/internal/web/auth_handlers.go:100>); [internal/auth/postgres.go](</Users/macbookpro/Documents/Kredit.com/internal/auth/postgres.go>) (`SessionFromToken`, `RevokeSession`); [web/src/lib/api/client.ts:70–78](</Users/macbookpro/Documents/Kredit.com/web/src/lib/api/client.ts:70>).

After a valid session is read, a database failure during revocation returns HTTP 401. The handler returns before clearing cookies. The browser accepts every 401 as a successful sign-out, clears local storage and navigates to the signed-out page. Session lookup itself also maps database errors to invalid-session errors. Consequently, during a transient database failure, an unrevoked session and its HttpOnly cookie can survive a UI-confirmed logout. When storage recovers, the browser can again authenticate with that cookie, which is especially harmful on a shared device.

Recommendation: distinguish invalid/revoked credentials from infrastructure errors; return 503 for an unconfirmed revocation, and do not tell the browser logout succeeded. Define a safe cookie-clearing policy independently of durable revocation and report the resulting assurance accurately.

## F003 — P2 — Backup retention deletes unrelated old files

File: [cmd/backup-r2/main.go:200–206](</Users/macbookpro/Documents/Kredit.com/cmd/backup-r2/main.go:200>).

The retention loop removes every old non-directory entry in `--local-dir`. It never checks that the filename belongs to this command or is a backup/checksum. Pointing the configurable output at a shared backup directory removes unrelated archives or operational files older than the cutoff. Negative retention days are also accepted, placing the cutoff in the future and making newly written local backups eligible for deletion.

Recommendation: validate retention as positive, prune only explicitly recognized Kredit backup pairs, and surface removal errors. Prefer a private command-owned directory and collision-resistant filenames.

## F004 — P2 — Placeholder R2 configuration reports successful offsite backup

File: [cmd/backup-r2/main.go:126–129,200–210](</Users/macbookpro/Documents/Kredit.com/cmd/backup-r2/main.go:126>).

A placeholder account endpoint or access key skips the entire upload branch, then prunes old local files and exits successfully. This bypasses the later failure check that correctly rejects other offsite replication failures. A scheduled run with template configuration can therefore appear healthy indefinitely while no remote backup exists.

Recommendation: reject placeholder configuration before dumping, or require an explicit local-only mode that cannot be mistaken for a completed offsite backup. Do not prune under an unconfirmed replication result.

## F005 — P2 — Interrupted bank authorization sends customers to a missing support page

File: [web/src/routes/buyer/bank-authorization/[reference]/+page.svelte:31](</Users/macbookpro/Documents/Kredit.com/web/src/routes/buyer/bank-authorization/[reference]/+page.svelte:31>).

The STARTED state tells the customer not to retry because their request may have reached the bank, then offers `/buyer/support`. There is no corresponding route or matching redirect in the project route inventory. The recovery link therefore leads to a missing page precisely when the customer needs assistance.

Recommendation: link to an existing support/contact flow and include the saved authorization reference safely.

## F006 — P2 — Authorized operators receive an incomplete bank-recovery queue

Files: [internal/providers/bankdebit/recovery.go](</Users/macbookpro/Documents/Kredit.com/internal/providers/bankdebit/recovery.go>) (`Pending`); [internal/web/mandate_recovery_handlers.go](</Users/macbookpro/Documents/Kredit.com/internal/web/mandate_recovery_handlers.go>) (`listMandateAuthorizations`); [internal/access/roles.go](</Users/macbookpro/Documents/Kredit.com/internal/access/roles.go>); [db/migrations/154_bank_debit_enrollment.sql:16–17](</Users/macbookpro/Documents/Kredit.com/db/migrations/154_bank_debit_enrollment.sql:16>).

Both HTTP and repository checks authorize `PermissionProviderOperations`, which includes platform administrators and compliance reviewers. `Pending` sets the operator identity and selects STARTED enrollments. The forced RLS policy grants cross-user SELECT only to platform_owner, however. A non-owner administrator/reviewer sees only their own enrollments, usually none, despite being authorized to operate this recovery surface. The queue returns success rather than reporting missing access, leaving interrupted customer authorizations invisible.

Recommendation: align the RLS read policy with the explicitly intended recovery roles, or restrict the endpoint to owners and make that limit visible. Preserve owner scoping for customer reads.

## F007 — P2 — Consumer sale creators cannot reopen the sales they create

Files: [internal/consumer/store.go](</Users/macbookpro/Documents/Kredit.com/internal/consumer/store.go>) (`Create`, `Get`, `List`); [internal/access/roles.go](</Users/macbookpro/Documents/Kredit.com/internal/access/roles.go>) (`Can`); [internal/web/consumer_handlers.go](</Users/macbookpro/Documents/Kredit.com/internal/web/consumer_handlers.go>).

Create permits administrator/sales memberships through `PermissionCreateCredit`, and release permits those same roles. Get/List require `PermissionReadFinancial`, which excludes administrator and sales. Such users can create an offer, but reopening its detail returns 404 and loading the list returns 503. The database read policy explicitly includes both roles, so the application permission check creates this inconsistency. Sales staff cannot reliably carry out the release workflow through the UI.

Recommendation: define consumer-sale operational read access for the roles allowed to create/release offers, with any necessary field-level financial restrictions; do not broaden unrelated financial permissions accidentally.

## F008 — P2 — Consumer retry helper discards the key for an in-progress write

Files: [web/src/lib/consumer.ts](</Users/macbookpro/Documents/Kredit.com/web/src/lib/consumer.ts>) (`Mutation.send`); [web/src/lib/components/ConsumerSales.svelte](</Users/macbookpro/Documents/Kredit.com/web/src/lib/components/ConsumerSales.svelte>) (`create`, `retry`); [internal/web/server.go](</Users/macbookpro/Documents/Kredit.com/internal/web/server.go>) (`withIdempotency`); [internal/consumer/store.go](</Users/macbookpro/Documents/Kredit.com/internal/consumer/store.go>) (`Create`).

The browser clears its pending identity on every response below 500, including HTTP 409 `idempotency_in_progress`. That response means the original mutation can still commit. After a timeout and an early retry, the next submit obtains a new key. Consumer offer creation has no durable original-intent key of its own, so both requests can produce separate offers. The helper also keeps the identity only in memory, losing it on refresh, and clears it before the caller validates the success payload.

Recommendation: retain the exact key/body through in-progress, unknown and invalid-success outcomes; persist the minimal intent across refreshes and clear it only for a confirmed result or a definitive rejection before a write.

## F009 — P1 — Every pilot scorecard fails on an unimplemented metric

Files: [internal/reports/analytics.go:111](</Users/macbookpro/Documents/Kredit.com/internal/reports/analytics.go:111>), [internal/reports/analytics.go:148](</Users/macbookpro/Documents/Kredit.com/internal/reports/analytics.go:148>); [db/migrations/122_scorecard_projections.sql:60](</Users/macbookpro/Documents/Kredit.com/db/migrations/122_scorecard_projections.sql:60>); [internal/reports/payment_metrics.go:5](</Users/macbookpro/Documents/Kredit.com/internal/reports/payment_metrics.go:5>).

The scorecard unconditionally requests `days_to_payment` through `app.pilot_metric`. The SQL CASE has no branch for this key and raises `unknown scorecard metric`. The application returns immediately on that error, discarding the whole report even with an empty database. The separately defined `daysToPaymentSQL` constant is not called by this loop. No later migration replaces the function.

Recommendation: implement the metric in the authoritative SQL function and keep the application's requested metric keys aligned with that function. Source-confirmed; no tests or database execution performed.

## F010 — P2 — Buyer-silence guardrail excludes current automatic activations

Files: [db/migrations/122_scorecard_projections.sql:58](</Users/macbookpro/Documents/Kredit.com/db/migrations/122_scorecard_projections.sql:58>); [internal/reports/analytics.go:138](</Users/macbookpro/Documents/Kredit.com/internal/reports/analytics.go:138>); [db/migrations/112_system_acceptance_evidence.sql](</Users/macbookpro/Documents/Kredit.com/db/migrations/112_system_acceptance_evidence.sql>); [internal/credit/postgres.go:484](</Users/macbookpro/Documents/Kredit.com/internal/credit/postgres.go:484>).

The metric labelled “Activations from buyer silence” counts only receipt confirmations tagged `deemed_acceptance_auto_activated`. The current automatic-recognition path records separate `app.system_acceptances` evidence; the receipt trigger explicitly rejects that old tag. Consequently, once F009 is repaired, the guardrail still reports zero for current automatic activations (or reflects only legacy records), hiding precisely the behavior it claims to monitor.

Recommendation: derive the numerator from current system-acceptance evidence joined to activated obligations, define the matching denominator/window, and include legacy records only through an explicit compatible mapping.

## F011 — P2 — Optional report-view events always fail with nil metadata

Files: [internal/reports/store.go](</Users/macbookpro/Documents/Kredit.com/internal/reports/store.go>) (`TrackContext`); [internal/web/reports_handlers.go](</Users/macbookpro/Documents/Kredit.com/internal/web/reports_handlers.go>) (`trackOptionalActivity` and report handlers); [db/migrations/050_product_analytics.sql:17](</Users/macbookpro/Documents/Kredit.com/db/migrations/050_product_analytics.sql:17>).

The report-view handlers pass nil metadata. `TrackContext` marshals that nil map as JSON `null` and inserts it into `analytics_events.metadata`, whose constraint requires a JSON object. PostgreSQL rejects the insert. The caller discards the error, so allowed report/history views silently disappear from usage analytics. The memory adapter accepts the same input, concealing the production-only behavior.

Recommendation: normalize absent metadata to an empty object before serialization and expose aggregate failure telemetry without failing the user's report request.

## F012 — P1 — Terraform production web deployment omits a required signing secret

Files: [infra/environments/main.tf](</Users/macbookpro/Documents/Kredit.com/infra/environments/main.tf>) (web container environment); [web/src/lib/server/legal-config.ts:11](</Users/macbookpro/Documents/Kredit.com/web/src/lib/server/legal-config.ts:11>); [web/src/hooks.server.ts:6](</Users/macbookpro/Documents/Kredit.com/web/src/hooks.server.ts:6>).

The API and worker receive the runtime secret, but the web container receives only explicitly listed environment values. That list sets APP_ENV=production and never supplies FRONTEND_PROXY_SIGNING_KEY. The server hook calls assertLaunchWebConfig at module load, which throws when this key is absent. Therefore a production deployment created from this Terraform module cannot serve the app successfully or pass its homepage readiness probe. The Dockerfile does not supply the key either.

Recommendation: inject the matching signing key into the web container using a narrowly scoped secret-key reference, and include this requirement in the Terraform deployment contract. This finding concerns the checked-in Kubernetes module; it does not assert that the separate Vercel/VPS installation has the same omission.

## F013 — P2 — Whitespace bypasses seller feedback membership checks

Files: [internal/web/feedback_handlers.go:20–28](</Users/macbookpro/Documents/Kredit.com/internal/web/feedback_handlers.go:20>); [internal/feedback/store.go:53–95](</Users/macbookpro/Documents/Kredit.com/internal/feedback/store.go:53>); [db/migrations/050_product_analytics.sql:30–63](</Users/macbookpro/Documents/Kredit.com/db/migrations/050_product_analytics.sql:30>); [internal/reports/analytics.go:93](</Users/macbookpro/Documents/Kredit.com/internal/reports/analytics.go:93>).

The HTTP handler checks business membership only when the raw `area` equals `seller`. The store subsequently trims this value and accepts ` seller ` as seller feedback. An authenticated caller with valid CSRF and idempotency headers can therefore submit `{"area":" seller ","organization_id":"<another business UUID>","screen":"overview","answer":"no"}` without membership in that business. The store records the supplied organization hash through `app.record_product_event`, which does not validate organization membership. Monthly deduplication limits one response per caller/business/month but does not restore authorization. This permits cross-business feedback attribution and misleading organization-scoped feedback totals (once F009 is repaired); it does not grant access to financial records.

Recommendation: normalize and validate the input before authorization, and enforce seller membership against the normalized business identifier at the storage boundary as well.

## F014 — P2 — Supplier sale listings expose buyer bank authorization links

Files: [internal/credit/reads.go:10–12,38–62](</Users/macbookpro/Documents/Kredit.com/internal/credit/reads.go:10>); [internal/web/financial_reads.go:15–20](</Users/macbookpro/Documents/Kredit.com/internal/web/financial_reads.go:15>); [internal/web/credit_handlers.go:183–197](</Users/macbookpro/Documents/Kredit.com/internal/web/credit_handlers.go:183>); [internal/credit/store.go:991–993,1031–1033](</Users/macbookpro/Documents/Kredit.com/internal/credit/store.go:991>).

The durable supplier listing returns the full stored View, including `mandate.authorization_url`, without the redaction applied by both development supplier-list and single-sale readers. The HTTP listing selects this durable reader and returns its output directly to organization readers. A buyer who initiates authorization before acceptance can therefore have their hosted bank authorization URL disclosed to the supplier. Mono and Paystack adapters populate this field with hosted URLs. This is a disclosure of a buyer-only authorization link; provider-side ability to complete authorization with the URL alone has not been established.

Recommendation: apply the same supplier-safe projection consistently to every supplier read, removing authorization URLs before serializing responses.

## F015 — P2 — Restored credit mandates cannot be accepted by their provider reference

Files: [internal/credit/postgres.go:885–888](</Users/macbookpro/Documents/Kredit.com/internal/credit/postgres.go:885>); [internal/credit/store.go:749–750,796–798](</Users/macbookpro/Documents/Kredit.com/internal/credit/store.go:749>); [internal/web/credit_handlers.go:494–510](</Users/macbookpro/Documents/Kredit.com/internal/web/credit_handlers.go:494>); [web/src/routes/buyer/credit-requests/[requestID]/+page.svelte:68](</Users/macbookpro/Documents/Kredit.com/web/src/routes/buyer/credit-requests/[requestID]/+page.svelte:68>).

Mandate creation indexes each mandate by both its internal ID and provider ID. Restoring a saved credit View only installs the internal-ID entry. The buyer page submits the provider ID when accepting a sale with an existing mandate, and Accept looks up that exact key. After a restart, or when the acceptance reaches another API replica, this lookup fails with a mandate ownership/ceiling error even though the saved mandate belongs to the buyer. The handler's preceding GetForBuyer restores the View but does not restore the missing provider alias. This affects the API-supported authorization-before-acceptance flow. The current page normally offers authorization after acceptance, so this is not a claim that every ordinary UI acceptance is blocked. Acceptance with an empty mandate reference is a separate supported path.

Recommendation: resolve the submitted provider reference against the authoritative mandate bound to this request, or restore and consistently refresh both aliases. Avoid dependence on which API replica created the mandate.

## F016 — P2 — A provider lookup failure prevents recording local cancellation

Files: [internal/mandates/provider.go:268–273,387–406](</Users/macbookpro/Documents/Kredit.com/internal/mandates/provider.go:268>); [internal/web/mandate_handlers.go:32–43](</Users/macbookpro/Documents/Kredit.com/internal/web/mandate_handlers.go:32>); [internal/web/runtime.go:862–876](</Users/macbookpro/Documents/Kredit.com/internal/web/runtime.go:862>).

CancelMandate calls GetMandate before BlockMandate. GetMandate performs a remote lookup for a nonterminal permission and returns immediately when that lookup fails. Consequently, an authenticated buyer's cancellation can fail before the local PAUSED state and cancellation_requested marker are saved. The endpoint reports failure, but the local collection permission remains active. Collection also checks the provider, so this does not imply a debit during a total provider outage; after lookup recovers, collection can proceed because the cancellation request was never recorded.

Recommendation: validate ownership and load local mandate facts without remote I/O, durably block new collections and save cancellation intent, then attempt and reconcile remote cancellation.

## F017 — P1 — Default monthly instalment terms can become impossible to activate after release

Files: [web/src/routes/app/credit/new/+page.svelte:17,103,149](</Users/macbookpro/Documents/Kredit.com/web/src/routes/app/credit/new/+page.svelte:17>); [internal/credit/store.go:1131–1144](</Users/macbookpro/Documents/Kredit.com/internal/credit/store.go:1131>); [internal/credit/postgres.go:502–527](</Users/macbookpro/Documents/Kredit.com/internal/credit/postgres.go:502>); [internal/schedules/store.go:211–222,589–594](</Users/macbookpro/Documents/Kredit.com/internal/schedules/store.go:211>); [internal/web/credit_terms_handlers.go:25–50](</Users/macbookpro/Documents/Kredit.com/internal/web/credit_terms_handlers.go:25>).

The advanced sale form defaults monthly instalments to last_day, while its label describes a fallback for months where the selected day does not exist. The schedule generator instead moves every due date, including the first, to month end. The accepted first collection instant remains based on the original selected payment day. For example, a first payment on September 15 with 24 extra hours produces a September 16 collection instant, but activation generates a September 30 first due date and rejects collection before that day. Credit creation validates count/cadence/policy but does not generate the schedule; terms preview validates only the first collection instant. The sale can therefore be sent, accepted and released before receipt confirmation fails and the activation transaction rolls back. Sent terms are immutable, so an ordinary draft edit cannot recover that sale.

Recommendation: resolve the intended month-end semantics consistently in UI and domain logic, generate and validate the entire exact schedule before sending the agreement, and bind those validated dates to acceptance.

## F018 — P2 — Credit-line expiry blocks receipt of goods already dispatched

Files: [internal/tradelines/store.go:521–524,768–780](</Users/macbookpro/Documents/Kredit.com/internal/tradelines/store.go:521>); [internal/tradelines/postgres.go:222](</Users/macbookpro/Documents/Kredit.com/internal/tradelines/postgres.go:222>); [internal/web/credit_handlers.go:1083–1086](</Users/macbookpro/Documents/Kredit.com/internal/web/credit_handlers.go:1083>).

A buyer confirming receipt without a problem must pass the same current-line eligibility check as a new drawdown. For a purchase accepted and dispatched before the line end time but delivered after it, that check returns “trade line has expired.” The purchase remains GOODS_RELEASED with its capacity reserved and no obligation activated. EndAt has no amendment operation, and Resume also rejects an expired line, so normal application actions cannot finish this already-dispatched sale. Cancellation and reservation expiry cover only pre-release purchases.

Recommendation: distinguish permission to initiate new exposure from completing an already-agreed and dispatched purchase. Preserve receipt evidence and provide an audited completion path after line expiry without granting new borrowing capacity.

## F019 — P2 — Reporting a drawdown delivery problem has no resolution path

Files: [internal/tradelines/store.go:515–519,540–548,610–619,781–805](</Users/macbookpro/Documents/Kredit.com/internal/tradelines/store.go:515>); [internal/tradelines/postgres.go:445](</Users/macbookpro/Documents/Kredit.com/internal/tradelines/postgres.go:445>); [db/migrations/046_trade_line_drawdown_lifecycle.sql:86–97](</Users/macbookpro/Documents/Kredit.com/db/migrations/046_trade_line_drawdown_lifecycle.sql:86>); [internal/jobs/client.go:306–314](</Users/macbookpro/Documents/Kredit.com/internal/jobs/client.go:306>).

Reporting a delivery issue moves the drawdown to RECEIPT_ISSUE_REPORTED and inserts an OPEN drawdown_receipt_disputes row. Subsequent receipt calls accept only an identical issue replay; a corrected no_issue receipt is rejected. Cancellation allows only pending/confirmed purchases, and expiry excludes released reservations. The repository has no command or worker to resolve/cancel this separate dispute record or leave the issue state; the ordinary obligation-dispute service cannot help because no obligation was created. Even when the seller remedies the problem or accepts returned goods, the purchase and reserved credit capacity remain stuck.

Recommendation: add an authorized, audited resolution flow that either records the buyer’s confirmed corrected delivery and activates once, or closes the returned/cancelled purchase and releases its reservation atomically. Preserve the original issue history.

## F020 — P2 — A declined or cancelled offer makes the buyer balance permanently unconfirmed

Files: [web/src/routes/buyer/+page.svelte:24,41,65–70](</Users/macbookpro/Documents/Kredit.com/web/src/routes/buyer/+page.svelte:24>); [web/src/lib/money.ts:20–24](</Users/macbookpro/Documents/Kredit.com/web/src/lib/money.ts:20>); [internal/credit/reads.go:38,60–63](</Users/macbookpro/Documents/Kredit.com/internal/credit/reads.go:38>); [internal/credit/store.go:649–665,689–705,1040–1054](</Users/macbookpro/Documents/Kredit.com/internal/credit/store.go:649>).

The buyer overview sums every returned credit request, using zero for only DRAFT, SENT and BUYER_REVIEWING when no obligation exists. A normally declined or cancelled offer has no obligation, remains in the complete buyer history, and therefore contributes null. sumKobo propagates that null to the whole total. One such historical offer causes “Not confirmed” and “Payment dates could not be confirmed” indefinitely, even when every active balance and payment date loaded successfully. Other legitimate pre-activation states have the same issue.

Recommendation: distinguish states that legitimately have no activated debt from an ACTIVE record missing required obligation data. Sum verified obligations and known zero-debt states, while retaining the failure state for incomplete active records.

## F021 — P2 — Manual payment records replace the actual payment date with a UI timestamp

Files: [web/src/routes/buyer/credit-requests/[requestID]/+page.svelte:83–86,144](</Users/macbookpro/Documents/Kredit.com/web/src/routes/buyer/credit-requests/[requestID]/+page.svelte:83>); [web/src/lib/api/mutation.ts:21,36,45](</Users/macbookpro/Documents/Kredit.com/web/src/lib/api/mutation.ts:21>); [internal/paymentclaims/store.go:117–125,217–218](</Users/macbookpro/Documents/Kredit.com/internal/paymentclaims/store.go:117>); [web/src/routes/app/payments/+page.svelte:83](</Users/macbookpro/Documents/Kredit.com/web/src/routes/app/payments/+page.svelte:83>); [web/src/routes/app/credit/[id]/+page.svelte:114–120,184](</Users/macbookpro/Documents/Kredit.com/web/src/routes/app/credit/[id]/+page.svelte:114>).

The buyer can report a transfer made earlier, but the form asks only for amount and reference. It submits the mutation object's creation time as paid_at. The claim service preserves that timestamp and copies it into the recognized payment when the seller confirms receipt; the seller screen labels it “Payment day.” A transfer made yesterday and reported today is therefore stored and presented with the wrong payment date, without either party being offered a correction during confirmation. The seller’s direct cash/transfer entry also omits a payment-date input and sends the same MutationIntent.createdAt value. Both entry paths therefore invent the payment date from form activity. Retry identity should stabilize the request, not invent evidence of when money moved.

Recommendation: collect and validate the actual transfer date/time, keep it stable across retries, and distinguish it from the report creation timestamp. Let the seller verify that date against the bank evidence before recognition.

## F022 — P2 — Saved normal payment terms are ignored by both sale creation forms

Files: [web/src/routes/app/settings/credit-policy/+page.svelte:8–12](</Users/macbookpro/Documents/Kredit.com/web/src/routes/app/settings/credit-policy/+page.svelte:8>); [web/src/routes/app/credit/quick/+page.svelte:59–63,77,84–102,126](</Users/macbookpro/Documents/Kredit.com/web/src/routes/app/credit/quick/+page.svelte:59>); [web/src/routes/app/credit/new/+page.svelte:16,46–47,60–79,103](</Users/macbookpro/Documents/Kredit.com/web/src/routes/app/credit/new/+page.svelte:16>); [internal/onboarding/store.go:337](</Users/macbookpro/Documents/Kredit.com/internal/onboarding/store.go:337>).

The settings page promises that saved payment days and extra hours will be filled into the next sale. Neither creation form loads the onboarding profile. The quick form always previews and submits 24 grace hours, with no control to change them, and leaves the payment date empty. The full form also starts at 24 hours and a blank date; its sample action hardcodes 30 days. For a seller who saved 72 extra hours, the normal quick-sale flow still creates terms permitting collection after only 24 hours. The checked agreement exposes those terms, but the configured defaults never influence them.

Recommendation: load the selected business’s saved terms before initializing either form, derive the initial payment day in Lagos time, and use the saved grace value consistently in preview, review and submission. Reinitialize defaults when switching business without overwriting deliberate edits.
