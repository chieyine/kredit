# Kredit platform code audit — 27 September 2026

**Implementation status:** recommendations are now implemented under the one-person-per-business rule. See [IMPLEMENTATION.md](IMPLEMENTATION.md) for the final changes, migration 213 and limits. F02’s original root-cause claim is corrected below; F01’s staff restoration was superseded by the user’s product decision.

The pre-implementation production-code audit reached complete cumulative coverage: **870 source/configuration paths had review evidence for their contents at audit completion**. Of these, **197 received full current-file inspection**; **673 retain the earlier complete-file review with an exact current-content hash match**. This is not a claim that all 870 files were freshly reread in this pass.

Kredit has a sensible core architecture, but the latest attempt to simplify its screens disconnected essential workflows and introduced misleading financial summaries. Ten finding groups were fixed in the initial pass. The completion pass found **six additional issues, F11–F16**, covering local setup, release checks, proxy maintenance and two usability problems. All six were subsequently implemented; their original rationale is retained below. The recommendations below distinguish those defects from optional structural cleanup.

The right direction is **fewer decisions, accurate balances, visible next steps and reusable workflows**. A smaller menu alone is insufficient if a person cannot finish setup, review a sale, resolve a payment or find an earlier purchase.

## Scope and evidence

Baseline: `7fef1d0b780c73b79daa3cee0a2a2a1e950b6939`. The working tree was clean when this review began. One reviewer; no agents; no tests written or executed.

[FILES.csv](FILES.csv) records every baseline tracked file plus the new payment-summary component: **1,465 rows**. It includes the audit-stage initial and final content hashes, review depth, notes, finding references, completion basis and whether the previous review still matches the file. This audit's own report/register are output artifacts and are excluded from their own inventory. This is a preserved pre-implementation snapshot: its hashes and 870-path count are not a claim about every later implementation file.

- **869 existing source/configuration files**, approximately **145,656 lines**, were inventoried and pattern-screened. The count includes both paths to the same symlinked API contract; it is not 145,656 unique logical source lines.
- **846 files** matched the exact reviewed content hashes in the 26 September audit at the start. That is prior review provenance, not evidence of a fresh manual reading or of correctness.
- The completed register has **870 source/configuration paths**, including the new payment-summary component. **197 received a complete current-source read** across the initial and completion passes: configuration, CI, infrastructure, scripts, command entry points, migrations 001–047, every changed/new production file since the previous complete review, and the additional dependencies traced for the findings.
- **28 current source files differ from or are absent from the previous complete-review register. All 28 were read in full.** There is no changed/new production file relying only on a stale earlier review.
- **673 source/configuration files retain an exact match to the earlier complete-file review**: 40 also received focused fresh inspection, and 633 received current static screening. Their earlier review is linked explicitly in the register. They were **not freshly read in full** in this pass.
- **341 test/support files** and **254 documentation, evidence, media, placeholder or dependency files** are recorded separately. Test behavior, dependency implementations and historical audit artifacts were not re-audited. Relevant product/architecture and build guidance was read as context.

Completion means the cumulative production-source coverage has no missing or stale file review, not that every repository file received a new manual reading. Earlier review evidence is the checked-in [26 September register](../2026-09-26-source-audit/coverage.json), specifically its `reviewed_sha256` and `status: reviewed` fields. Reusing that evidence does not independently prove its original review quality. Tests/support, historical documentation, media and dependency implementations remain outside the production-code behavior review. This audit does not establish production readiness or prove every financial/concurrency path correct.

## What the platform is

Kredit records trade credit financed by the seller's own goods. A business proposes a sale, its customer reviews and accepts the agreement, separate bank permission is established where required, and release, receipt, repayment and disputes remain attached to the transaction. Kredit is not supplying loan capital in this model.

The implementation also supports retailer-to-person purchases. Those have separate agreements, payment events, delivery, cancellation and refund rules. They must appear in the simple workspace without being mistaken for the business-credit ledger.

The main layers are:

| Layer | Responsibility | Assessment |
|---|---|---|
| SvelteKit web | Public pages, account/workspace/admin views, same-origin API proxy | Keep the framework and shared visual system. Strengthen response contracts and recovery paths. |
| Go API | Authentication, authorization, orchestration and domain commands | Keep the modular monolith. Avoid adding services merely to reduce file size. |
| Domain packages | Credit, purchases, ledger, payments, collections, identity and operations | Preserve explicit business rules and audit history. Cache/persistence variants add maintenance cost. |
| PostgreSQL | Durable state, money records, tenant restrictions and transaction guards | Preserve migration history and database protections; hiding a page cannot replace them. |
| Worker/providers | Queues, notices, collection/reconciliation and external services | Unknown outcomes require reconciliation, not another blind submission. |

The baseline has 132 Svelte page routes, 312 registered backend routes identified by the route checker, and 212 migrations. These explain why simplifying navigation did not substantially simplify the implementation.

## Findings and initial changes

These sections retain the audit-stage sequence. The implementation record gives the final behavior.

### F01 — P1: required setup and approval workflows were redirected away

`internal/onboarding/store.go` requires saved default credit terms and finance-staff MFA. Its readiness response links to `/workspace/settings/credit-policy` and `/workspace/team`. `web/src/lib/features.ts` redirected both back to Settings, which did not offer those controls. A new business could be told to complete a step it could never open.

The same redirect table hid purchasing identity/bank setup, staff purchasing permissions, schedule amendments, history and credit approvals while backend requirements and existing records remained active.

**Changed:** restored these necessary routes; added contextual Settings links; restored the draft-sale approval link, buyer verification link and obligation amendment link. Main navigation stays simple. The inaccurate claim that only an owner can sign in was removed. The full-workspace flag is now described accurately as bypassing redirects, not changing permissions or restoring another navigation menu.

Files: `web/src/lib/features.ts`, workspace Settings, supplier sale detail, buyer order detail and obligation detail.

### F02 — P2: unverified customer amounts silently defaulted to zero (corrected)

The page substituted zero for an absent `outstanding_kobo`. The original audit incorrectly inferred that the HTTP customer list omitted balances from the narrower `buyers.Customer` directory type. In fact, `listOrganizationCustomers` already augmented that directory with authorized obligation balances. The claim that every real debtor necessarily appeared as owing zero is withdrawn.

**Changed:** require a validated balance instead of substituting zero. The final implementation uses one authorized database projection behind the existing endpoint and removes the initial extra full-credit-list request. Buyer business identity remains separate from its owner's other businesses.

Files: Customers, `records.ts`, `internal/reports/customers.go` and the customer HTTP handler.

### F03 — P1: combined balances could silently omit personal purchases

The dashboard treated failed or unfinished personal-sale reads as an empty list while showing a combined total. Both it and Customers ignored `next_cursor`; the Go endpoint limits a page to 100 records. Older debts could disappear from totals. The dashboard's overdue amount included business credit only, even while the main balance combined both domains. Its future-payment count also included unaccepted/cancelled personal offers and used a label inconsistent with the business endpoint's overdue-inclusive horizon.

**Changed:** added one shared, bounded, cancellable pagination reader. It validates page structure and rejects repeated records/cursors instead of returning a partial total. Purchase amounts and schedules are validated as nonnegative safe integers; the new summaries use exact integer arithmetic. Combined totals are unavailable until both sources finish successfully. Personal overdue amounts are included and capped at the current outstanding balance. Due counts exclude inactive personal offers and are labelled “Due now or within 7 days.”

Files: `web/src/lib/consumer.ts`, Today and Customers.

**Follow-up implemented:** workspace personal totals now use a dedicated repeatable-read database summary with a 15-second query deadline. Detailed history still uses bounded pagination. Business and personal domains remain separate requests; a combined screen is not one atomic cross-domain snapshot.

### F04 — P1: a failed send could still display “Sent”

The simple form created a draft, attempted to send it, swallowed any send error, and navigated with `given=1`. Sale detail turned that query parameter into a message claiming the customer had been sent the sale. The form also skipped review of the saved fee terms before sending. If creation succeeded but navigation failed, the submit path could create another draft.

**Changed:** the form performs one creation and opens the saved draft for review. The existing record shows exact terms/fees and owns the Send action. It says “Sale saved,” not “Sent.” The form retains a confirmed saved-record link and prevents another creation if navigation fails. Existing mutation identity handling remains in use.

Files: Give goods on credit and supplier sale detail.

### F05 — P1: personal offers invented stock and handover evidence

The simple form unconditionally sent `stock_reserved: true`, a stock reference claiming goods had been handed over, and terms saying handover had already occurred. The seller had not confirmed those facts, and the consumer model treats immediate fulfillment as handover after acceptance.

**Changed:** display the delivery/return terms before submission, require the seller to confirm stock reservation and delivery capability, and describe reservation after acceptance without claiming completed handover. Share copy describes an offer. The original consumer state machine still records actual release and receipt separately.

File: `web/src/routes/workspace/give/+page.svelte`.

### F06 — P2: a grouped customer's balance opened only one purchase

Today and Customers combined multiple personal purchases into one customer balance but linked to whichever individual sale was first in the fetched records. The detail page could therefore show a smaller amount and hide the other contributing purchases.

**Changed:** grouped rows open the existing consumer-sales list filtered to the exact customer contact and current business. That view reads all pages, so its displayed records explain the grouped balance. Unrelated offer-creation controls are omitted in this filtered view.

Files: Today, Customers and `ConsumerSales.svelte`.

### F07 — P2: Money in omitted personal receipts and reported payments

The primary Money in screen queried only business-credit payments and claims while describing its totals as all customer payments. A seller using the new “A person” flow could receive money or a claim with nothing appearing there.

**Changed:** label the existing business-payment total accurately and add a separate personal-purchase summary using the same pagination/decoding helper. It includes receipts after reversals and refunds and links reported payments to their existing authorized review screen. Shared claim matching is reused by the purchase detail and the summary. Consumer money display uses the existing exact kobo formatter.

Files: `ConsumerPayments.svelte`, `ConsumerPurchase.svelte`, `consumer.ts`, Money in.

### F08 — P2: unrelated read failures hid actionable records

Today hid its entire attention section unless seven resources loaded, including payment history that was not otherwise needed on that screen. One failed payment-history read could hide a successfully loaded dispute. Personal payment claims, refunds, returns and release work were absent from the attention list.

**Changed:** show actions from each successfully loaded resource with separate failure notices for unavailable sources. Include personal-sale exceptions and handover work. Remove the unused payment-history request from both client loading and server prefetch. Bank-permission follow-up remains visible.

Files: Today and its server loader.

### F09 — P2: creation and invitation recovery had dead ends

The new form silently substituted default credit terms after a preference read failed, left a previously selected business buyer active while adding another, lacked a refresh action after a customer joined, and relied on navigating to the same URL to reset a completed personal-sale form. Fresh MFA was required by the consumer endpoint but the form offered no inline route to complete it.

**Changed:** business drafts require a confirmed settings read; choosing a new customer clears the old selection; joining instructions include an explicit customer refresh; “Give another credit” resets the completed form; the existing identity-confirmation component is available on the page. The submitted consumer amount respects the existing backend limit. Copy describes link creation/sharing without asserting message delivery.

File: Give goods on credit.

### F10 — P3: current documentation described a different product surface

The current flow described five main destinations and migration 194, while the code used the three-destination interface and a startup contract through migration 212. The active page list omitted newer routes and did not distinguish retained pages from redirects.

**Changed:** updated the current user flow, regenerated the page map with default redirect behavior, labelled older implementation notes as historical, and pointed the repository's current-audit link here.

## Additional findings from the completion pass

These were open at audit completion and are now implemented. The original findings and recommendations below explain the changes; see the implementation record for final status and verification boundaries.

### F11 — P2: documented development startup skips the upload bucket

`scripts/bootstrap.sh` and `scripts/dev.sh` start `postgres minio mailpit provider-simulator`, omitting `minio-init`. That omitted service is the only local setup step that creates `kredit-local`. The scripts run API and worker processes on the host, so the Compose dependencies attached to the containerized API/worker do not repair this omission. `internal/documents/s3.go` uses the configured bucket without creating it. A clean installation can therefore start the application with no upload bucket. Bootstrap also does not explicitly wait for PostgreSQL readiness before migration; dependency downloads merely make that race less likely.

**Recommendation:** give both scripts one shared local-infrastructure preparation step that waits for PostgreSQL and successfully completes `minio-init` before continuing. Keep bucket provisioning outside application requests. Validate the scripts statically here; actual clean-volume startup remains an unexecuted operational check.

### F12 — P2: the saved-configuration checker validates before loading saved configuration

`cmd/configcheck/main.go` always calls `config.Load()` before checking `--stored`. In production, that performs full validation of the environment-only configuration, including object-storage/provider settings. The API and worker instead call `config.LoadBootstrap()`, read the encrypted settings and then call `config.ApplyStoredConnections()`, which performs complete validation of the merged configuration. A deployment using saved connections can therefore run successfully while the release checker rejects its environment defaults before it reads those same connections.

**Recommendation:** select `LoadBootstrap()` when `--stored` is supplied, retain `Load()` for environment-only checking, and retain the final validation inside `ApplyStoredConnections()`. Keep the read-only database lookup and deadline. Do not bypass final readiness checks.

### F13 — P2: release tooling requires legal settings the application no longer uses

`web/src/lib/server/legal-config.ts` returns the checked-in `legalPublication`; `.env.example` already identifies the old legal environment variables as legacy. However, `scripts/release-certify.sh` still requires those variables and `LEGAL_DOCUMENTS_ACTIVE=true`. Terraform also requires public legal metadata inputs, passes them to the web container and defaults its document versions to v1 while the published/onboarding versions are v2. These inputs cannot change the displayed publication, and a valid deployment can be blocked by their absence.

**Recommendation:** remove the obsolete runtime inputs and validate the actual published metadata against the backend's current document versions. Retain the separate legal-approval evidence gate; publication metadata and professional approval are different checks. This is a configuration-consistency finding, not an assessment of the legal text.

### F14 — P2: proxy-range refresh can replace a working configuration before validation

`scripts/refresh-cloudflare-ranges.sh` moves the generated candidate over the target before calling `caddy validate`. Validation is skipped entirely when a host Caddy binary is unavailable; a reload failure prints a warning but exits successfully. The documented production deployment runs Caddy in Compose with `admin off`, so the script's host `caddy reload` path does not match that deployment's activation mechanism. Its address filter checks character shape and count, rather than validating IPv4/IPv6 CIDRs.

**Recommendation:** validate every returned range and the complete candidate before replacing the current file. Separate candidate generation from an explicit supported activation command, or support the documented Compose deployment directly. Preserve the previous file and report activation failures with a nonzero exit. No running proxy was contacted or changed during this audit.

### F15 — P2: low-data mode gives mobile navigation a dark background with dark text

`web/src/lib/product-tools.ts` applies the persisted `low-data` class to the document. `web/src/app.css` then sets `.mobile-nav` to `#0b0d12`. `PortalNav.svelte` still resolves ordinary navigation text from the light product theme's `--color-foreground` (`#14161b`). The override makes inactive destinations almost indistinguishable from their background when low-data mode is enabled. Disabling motion and blur does not require changing that background.

**Recommendation:** keep the mobile bar's existing light background in low-data mode; retain the opaque dark override only for the dark site header. This conclusion follows from the CSS cascade, not a browser/device accessibility run.

### F16 — P2: existing customer credit-limit records still redirect away

`features.ts` parks the entire `/workspace/sales/limits` prefix. The backend still sends seller links to `/workspace/sales/limits/{id}` when a drawdown is confirmed, ready for release or changed. Those links reach the dashboard instead of the existing agreement and its release/cancellation controls. Hiding new credit-limit entry points does not remove the need to finish or inspect an existing line. This affects businesses with existing lines or an enabled drawdown capability; it is not a claim that such records exist in the current deployment.

**Recommendation:** keep existing line detail routes reachable. If creation should stay hidden in the simple interface, distinguish the collection/creation entry point from record detail routes. Continue enforcing feature and authority checks on the server, and provide a contextual path to existing records without adding another primary navigation tab.

## Maintainability recommendations — implemented within the stated boundaries

These are the original code-structure recommendations. Each has an implementation and an explicit boundary in [IMPLEMENTATION.md](IMPLEMENTATION.md); they did not justify rewriting stable financial code wholesale.

1. **M01 — Move summary computation into authorized read projections.** Today loads several resources and full consumer event histories; Customers loads the credit list for balances. `consumer.Store.List` loads event history separately for each sale. A small summary query can return counts, balances and actionable references without shipping every event to the browser. Preserve scope and failure semantics when doing this.
2. **M02 — Retire duplicate sale entry implementations deliberately.** `workspace/give`, `sales/new` and `sales/quick` remain separate implementations even though the latter two redirect by default. Their draft recovery, preferences and validation have already drifted. Keep one owner of each rule before removing a route that remains available under the full-workspace flag. Redirecting the advanced form also removes its invoice/installment inputs from the default journey; that capability decision remains explicit.
3. **M03 — Split the largest files by responsibility.** At baseline, `internal/web/credit_handlers.go` has 1,837 lines, `admin/platform-settings/+page.svelte` 1,495, credit store/persistence files over 1,300 each, and sale detail over 1,000. Separate request decoding, lifecycle commands and read projections; separate page sections with distinct state. Avoid moving code solely to satisfy a line limit.
4. **M04 — Converge frontend data access.** `api/client.ts`, `api/reliable.ts`, `admin-client.ts` and `consumer.ts` have overlapping error, fetch and decoding responsibilities. Continue moving money reads through validated responses and commands through the established mutation-intent contract. The generic `WorkspacePage` deliberately only validates objects; per-page field contracts remain a maintenance responsibility.
5. **M05 — Keep navigation and product capability separate.** Seventeen default redirect rules remain. They do not remove the underlying API or data. Team, branch, approval and provider controls must remain enforced server-side. Do not treat a hidden menu item as a disabled capability. The user subsequently explicitly retired staff capability; that decision is implemented in authorization and migration 213, not only navigation. F16 is the remaining concrete record-access defect found in this pass.
6. **M06 — Reduce documentation sprawl without rewriting history.** Keep the current flow, route map and operations guide short and authoritative. Retain old audits as historical evidence; do not copy their completion claims into a new audit without content hashes and stated review depth. This review maintains one report, one CSV and one checks record rather than another source-code dump.
7. **M07 — Make infrastructure dependency selection reproducible.** `.gitignore` excludes `.terraform.lock.hcl`, while the Terraform provider version uses a range. Retain the provider lockfile generated for the actual supported platforms. Container/action pins remain explicitly baselined debt; update their reviewed baseline deliberately rather than inventing digests or changing providers during a source audit.
8. **M08 — Keep static contract checks honest and focused.** The route checker covers direct frontend calls, not every mutation wrapper or runtime response. The frontend coverage script's path/name matching is not proof of method-level or end-to-end coverage. Prefer one shared route/operation inventory, document each check's limits, and avoid maintaining overlapping scripts whose green outputs mean different things. The same ownership principle applies to the global stylesheet's broad `!important` overrides: move components to existing tokens gradually instead of adding another overriding stylesheet.

## Implementation order and retained boundaries

1. Repair F11–F16 as small, independent changes. Preserve existing release evidence, authorization and financial checks.
2. Address M01/M04 together: one authorized read model and shared validated client contract for balances and next actions.
3. Consolidate entry forms under M02 without silently losing invoices, installment schedules or existing draft recovery.
4. Apply M03/M05–M08 incrementally where ownership becomes clearer. A shorter file or menu is not by itself a reason to rewrite stable financial code.

The completion trace found no missing drawdown balance in the credit-list path: `ActivateTradeLineDrawdown` creates a credit request and obligation, and the production report reads those obligations under a repeatable-read transaction. Original migrations with broad policies were treated as migration history, not reported as current vulnerabilities without considering subsequent replacements. No new service split, financial-rule relaxation or migration-history rewrite is recommended.

The modular monolith, integer money, explicit transitions, durable audit evidence, scoped access, idempotent commands and shared design system are worth keeping. The objective is a small, coherent daily workflow over those controls.

## Pre-launch simplification

The user subsequently confirmed that the platform has not launched and only a testing account exists. Compatibility redirects, staff-history APIs, staff/delegation/branch/approval modules, obsolete draft recovery and older runtime-format fallbacks are now removed. Migration 213 drops the unused staff schema instead of archiving it. The original findings above remain audit history; [IMPLEMENTATION.md](IMPLEMENTATION.md) is the current disposition.

## Verification and limits

Final static/build results are recorded in [CHECKS.md](CHECKS.md). No unit, integration, end-to-end, browser or provider tests were run. No database, migration, deployment, provider-service change or production operation was performed. Application dependencies are unchanged; the infrastructure provider lockfile was added and Terraform configuration validation completed without a backend.

The audit-stage changes were frontend behavior and documentation. The subsequent implementation also changes read projections, authorization, owner credit-note confirmation, operational scripts and schema requirements; see the separate implementation record. Compiler/lint/build success cannot establish actual browser usability, provider behavior or transactional correctness. The local Node and Go versions differ from the declared deployment toolchain; exact-version compatibility remains unverified.

The audit baseline required 212 migrations; the implementation now requires 213 in `internal/db/persistence_contract.go`. This review does not establish which migrations are applied in any environment. Previous claims about unapplied migrations are historical, not a fresh database observation.
