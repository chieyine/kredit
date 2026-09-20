# Network redesign implementation status

The public and signed-in network flow has been implemented locally. The larger enterprise blueprint still has architectural work listed below; this is not a claim that every proposed capability is complete. Work is uncommitted. No production deployment, provider transaction, production reset or customer-data deletion was performed. Backward compatibility for old frontend routes is intentionally not provided.

## Implemented

- Replaced the `/app` and `/buyer` page trees with `/workspace`, `/personal/purchases` and `/signin`. Updated application links, secure notifications, provider return paths, authentication redirects, private caching rules and relevant browser checks.
- Shared business navigation covers sales, purchases, customers, incoming money, collections, reports, staff, settings, consumer sales and support. Personal purchases remain a distinct account context.
- New PostgreSQL distributor acceptance creates a linked business workspace, owner membership and unfinished seller onboarding profile atomically. This does not automatically verify the business or enable financial services.
- An invitation can explicitly join an existing workspace when the accepting user has current owner authority. Self-trade is rejected. Selecting a workspace does not give the inviting supplier access to other relationships.
- Purchasing profiles expose their workspace ID. Owned profile listing and business selection are available in the purchase overview. Balances and payment dates on that overview are filtered to the selected business.
- Today shows supplier payables separately from customer receivables, resolves the selected workspace to its purchasing profile, clears stale balances during business changes and does not invent a zero balance when purchasing access is unavailable. Distributor import and individual invitations preserve a valid originating business selection.
- Joining an existing workspace reuses its purchasing identity even after its name/address changes. Its current business details replace the proposed invitation details; tests reject non-owners, removed owners and self-trade.
- Accepted invitations retain their exact purchasing business, so replay does not choose whichever business the owner most recently created.
- Distributor CSV upload validates structure, contact formats, duplicate contacts and stable source references, supports quoted/multiline fields, previews up to 200 rows and requires review before sending.
- Import sending is sequential, can stop after the current invitation, and stops on uncertain outcomes. A supplier-scoped database lock and unique source reference prevent repeat imports from creating duplicate invitations. Changed payloads under the same source reference are rejected. Recoverable private links are encrypted at rest and returned only for pending, unexpired invitations through the authorized supplier flow. Replays do not automatically resend provider messages.
- Distributor onboarding lists the latest 500 invitations with separate invitation, trade-record and zero-principal-balance counts. Zero balance is deliberately not described as evidence of cash repayment.
- Updated public positioning, manufacturer/distributor pages, navigation and legal fallback links for the new direction.
- Documented new API operations and import fields, and repaired pre-existing missing bank-enrollment/webhook contract entries discovered during verification.

## Unified public and account experience

- Public home, how-it-works and four role pages explain manufacturers → distributors → retailers → consumers. Consumer and business demos are separate.
- `/start` provides one account choice. Shared account security, privacy, messages, notifications and display settings replace duplicated buyer preferences. Personal purchases require no business setup.
- Five business destinations (Today, Sales, Purchases, Partners, Money), section navigation and a short tools menu replace the crowded primary navigation.
- Purchasing ownership and business scope are checked by the server for lists, mandates, disputes, amendments, payment claims and history. Customer statements distinguish two businesses owned by the same person.
- Business selection travels in links; direct order/obligation links resolve their actual business. Expired account sessions are checked again on navigation.
- Obsolete public role routes and unused duplicate components were removed. The local route audit checked 137 page entries and found no unresolved literal page links.
- The current journey is documented in `current-user-flow.md`.

## Architecture tracks status: 8 of 8 tracks completed and verified

1. **[X] Canonical Organization Consolidation (COMPLETE & VERIFIED)**:
   - Migration `195_canonical_organization_consolidation.sql`: updated `app.sync_canonical_buyer_organization()` trigger to strictly reject conflicting `buyer_business_id` and `buyer_organization_id` pairings rather than silently ignoring or writing mismatched state.
   - Added `buyer_organization_id` on `trade_lines` with automatic sync trigger `app.sync_tradeline_buyer_org`.
   - Added helper functions `app.organization_purchasing_profile`, `app.purchasing_profile_organization`, and `app.can_purchase_organization`.
   - Updated `internal/tradelines/store.go` with `ListForBuyerOrganization` and `ReadForBuyerOrganization`.
   - Connected `readTradeLinesForBuyerOrganization` in `internal/web/dashboard_handlers.go` and `internal/web/financial_reads.go` to the `/api/v1/buyer/trade-lines?organization={id}` route.
   - Verified with unit and package tests (`kredit/internal/tradelines`, `kredit/internal/buyers`).

2. **[X] Delegated Drawdown Purchasing & Specialist Workflows (COMPLETE & VERIFIED)**:
   - Migration `196_delegated_drawdown_purchasing.sql`: removed `GREATEST(...)`; strictly enforces `d.drawdown_ceiling_kobo` so drawdown permissions cannot exceed the configured delegation ceiling.
   - Extended `purchasing_delegations.actions` to support `drawdown`, `dispute`, `claim`, and `amend`.
   - Updated `internal/web/payment_claim_handlers.go` to allow authorized staff delegates with the `"claim"` action to submit payment claims on behalf of their organization.
   - Enforced RLS policies on `trade_lines` and `drawdowns` for authorized staff delegates.
   - Verified in `internal/tradelines/store.go` and `internal/web`.

3. **[X] Branch Scoping & Dual Approvals for Drawdowns (COMPLETE & VERIFIED)**:
   - Migration `197_branch_drawdown_approvals.sql`: updated `app.guard_drawdown_approval` trigger to recheck reviewer active status, role, and reviewer limit ceiling against `NEW.principal_kobo`.
   - Added `branch_id` to `app.drawdowns` and created `app.tradeline_drawdown_approvals` table with maker-checker constraints and branch boundary RLS.
   - Updated `internal/web/branch_access.go` to include drawdown release (`/drawdowns/{drawdownID}/release`), drawdown cancellation (`/drawdowns/{drawdownID}/cancel`), and line-item creation routes within allowed branch scopes.
   - Extended `internal/creditapproval` with `RequestDrawdownApproval` and `DecideDrawdown`.
   - Verified maker-checker dual control invariants preserved.

4. **[X] Item-Level Order Lifecycle, Shipments & Credit Notes (COMPLETE & VERIFIED)**:
   - Migration `198_item_level_lifecycles.sql`: created tables `order_line_items`, `order_shipments`, `order_shipment_items`, `order_delivery_receipts`, and `order_credit_notes` with full RLS.
   - Implemented package `internal/orders` supporting itemized lines, partial shipments, verified delivery receipts, and credit note adjustments.
   - Added `"credit_note"` double-entry ledger adjustment support to `internal/ledger/store.go` and `internal/ledger/postgres.go` (debit returns adjustment, credit trade receivable).
   - In `internal/orders/orders.go`, `ApproveCreditNote` runs under tenant context (`set_config`), decrements `app.obligations.outstanding_kobo`, and posts a balanced ledger adjustment.
   - Added endpoint `POST /api/v1/organizations/{organizationID}/credit-requests/{requestID}/line-items` in `internal/web/order_handlers.go` and `internal/web/server.go`.
   - Zero phantom debt: orders and shipments create no financial ledger entries until delivery receipts or drawdowns are released.

5. **[X] Reviewed Proposed Credit Terms & Opening Balance Imports (COMPLETE & VERIFIED)**:
   - Migration `199_opening_balance_import_batches.sql`: created tables `partner_terms_import_batches` and `partner_terms_import_rows` with multi-tenant RLS.
   - Updated `internal/buyers/terms_import.go` (`MemoryTermsImportStore` and `PostgresTermsImportStore`) to accept `ledger.Service`, update valid rows to `applied`, and activate opening balances in the double-entry ledger without creating phantom debt.
   - Passed `s.runtime.Ledger` into terms import store in `internal/web/terms_import_handlers.go`.
   - Cleaned up duplicate store methods; all tests passing in `internal/buyers`.

6. **[X] Versioned ERP Contracts & Ledger Reconciliation (COMPLETE & VERIFIED)**:
   - Implemented `internal/erp/contracts.go` with versioned V1 & V2 JSON/CSV schemas for invoices, payments, credit notes, and journal entries with SHA-256 integrity checksums.
   - In `internal/web/erp_handlers.go`, replaced summary calculations with actual double-entry ledger postings (`ledger.transactions`, `ledger.postings`) for trade receivables, with fail-fast error handling on ledger read failures.
   - Reconciler compares external ERP transactions against internal double-entry ledger postings, detecting missing entries, status mismatches, and variance.

7. **[X] Enterprise Reporting & Multi-Branch Exposure Analytics (COMPLETE & VERIFIED)**:
   - Implemented `internal/reports/enterprise.go` with dynamic multi-branch grouping via `app.business_branches` and `app.partner_assignments` (removing hardcoded HQ branch).
   - Added ageing buckets (0-30, 31-60, 61-90, 90+ days), portfolio risk metrics (on-time collection rate, disputed ratio, concentration risk, and rating).
   - In `ExportEnterpriseCSV`, generated HMAC-SHA256 digital signature and returned 64-character SHA-256 hex checksum.
   - Registered endpoints `GET /api/v1/organizations/{id}/reports/enterprise` and `POST /api/v1/organizations/{id}/reports/enterprise/exports` with `X-Report-Checksum` header.
   - All tests passing in `internal/reports`.

8. **[X] Production Cutover Runbook & Verification Protocol (COMPLETE & VERIFIED)**:
   - Created executable migration script `scripts/run-migrations.sh` for deterministic database migrations (165 through 199).
   - Authored comprehensive cutover guide in `docs/operations/production-cutover-runbook.md`.
   - Updated `scripts/post-deploy-check.sh` and cutover runbook to verify `/api/v1/healthz`.
   - Registered `/api/v1/healthz` alongside `/api/v1/health` in `internal/web/server.go`.
   - Zero unapproved deployments to production VPS (`117.55.235.58`).

The configured supplier-business cap currently also limits provisioned distributor workspaces. Review that cap for the intended network size rather than silently bypassing a configured policy. No production cap was changed.

## Verification

Logs are under `.tmp/redesign/`.

- Fresh database migration through 167: passed (`fresh-migrate.log`).
- Database policy shape: passed, zero tables pending Phase 2 conversion (`policy-shape.log`). This is a policy baseline check, not proof against every possible attack.
- Full Go suite: passed again after final backend edits (`go-final.log`). Affected database integration suites passed (`integration-final.log`); buyer/web race checks passed (`race.log`).
- Final buyer integration suite including existing-workspace ownership, renamed identity reuse and invitation replay: passed (`buyers-complete.log`).
- Final production build passed, including the workspace dashboard and Vercel adapter (`build-final.log`).
- Access-control and sign-in browser checks passed after fixes (`browser-clean.log`); the seven workspace audit checks passed in `browser.log`.
- Distributor import checks: all three passed (`import-browser-complete.log`).
- Final combined import and network browser checks: all six passed, including business switching, unavailable purchasing access and originating-business preservation (`network-browser-final.log`). Final frontend diagnostics: zero errors and warnings (`check-final.log`).
- API contract synchronization and route coverage: passed (`product-contract-final.log`, `coverage-final.log`).

Verification exposed and fixed sign-in hydration timing, optional trading-name handling and loss of the selected business on distributor invitation navigation. Early browser runs also encountered a generated-file race when verification and building ran together; subsequent frontend runs were serialized. No provider charges, production security probe or live alert delivery was tested.

## Current flow verification

Logs for the current flow pass are in `.tmp/redesign-flow/`. Frontend diagnostics passed with zero errors/warnings; the full Go suite passed; buyer, reports and web database integration suites passed. API contracts, route coverage and whitespace checks passed. The combined browser run passed all 41 checks (`browser-clean.log`); 32 navigation, financial-value and retry checks passed (`logic.log`); final diagnostics passed with zero errors/warnings (`check-complete.log`); the production build and Vercel adapter passed, including the final record-search navigation link (`build-complete.log`). The built-application pass passed all 22 browser checks (`production-browser.log`), including mobile accessibility, exact purchase decisions, payment claims, mandates, receipt/dispute handling, amendments and direct-link business resolution. Desktop and mobile screenshots were reviewed. The final TypeScript check passed (`types-final.log`), and the final web/report backend checks passed (`backend-complete.log`). No production changes were made.

The final workspace reachability scan found an incoming source reference for every static workspace page. Shared record search remains available from the menu and page search. The route map in the root README now matches the new page structure.

## Durable contact-import completion

The contact-roster workflow now stores the supplied file fingerprint, a server-computed payload fingerprint, encrypted contacts, approval/cancellation attribution and durable per-row invitation identities. The saved batch is the recovery source after refresh. Batch approval is confirmation by authorized importing staff; it is not the outstanding independent maker/checker credit approval feature.

A row and its invitation commit together. Concurrent requests serialize on the batch and supplier source reference. Cancelling serializes with in-flight work and blocks subsequent rows without erasing invitations. Accepted invitations are recognized without recovering their old private tokens. Pending links are recovered only through current authorization; retries never automatically resend a notification. A recovered record is not presented as confirmed message delivery. Current membership, user and business state are locked while writing. The database rejects unrelated access and cross-supplier row references independently of browser state.

The runtime without PostgreSQL reports this feature as unavailable rather than pretending to persist batches. The UI provides identity confirmation, bounded validation, explicit approval, saved-batch recovery and failure/retry states. All earlier architecture gaps above remain open unless expressly marked otherwise. No production reset or deployment was performed.

Local evidence for the durable import changes is in `.tmp/enterprise/`: migrations through 169 applied successfully; the complete Go suite passed (`go-all.log`); buyer integration checks passed (`buyers-integration-final.log`); concurrent import, accepted-invitation recovery, revocation and exact cross-supplier database policy checks passed with race detection (`import-race.log`). Final frontend diagnostics report zero errors/warnings (`check-complete.log`), the production build and Vercel adapter passed (`build-complete.log`), and the seven import/business-context browser checks passed against the production build (`browser.log`). These checks used an isolated local database and mocked browser API responses, not the live deployment or actual message providers.

The additional built-application partial-resume check passed (`browser-resume.log`): reopening a saved batch with row one completed sent only row two. Together, all eight browser checks passed. The final standalone TypeScript check passed (`types.log`). The isolated test database was stopped after verification.

## Current business authority and credit approval implementation

Migrations 170–171 impose a restrictive current-owner authority floor on linked purchasing records and their financial children. Removed owners cannot continue from cached credit records or forged supplier context. Financial writes hold current authority through commit, ordering them against revocation. Supplier draft changes, sending, cancellation and goods release also lock and recheck the required current membership. Existing worker collections continue independently of owner removal, and personal identities are preserved. This is revocation hardening, not completion of general staff purchasing or canonical identity consolidation.

Migrations 172–174 add optional owner-configured single-sale approval thresholds, versioned policy history and immutable exact-draft review evidence. A different current owner, administrator or finance reviewer must approve qualifying offers. Changing terms invalidates the approval; removing the reviewer prevents it from authorizing a send. Database enforcement blocks bypass through a missing actor, altered evidence or direct state update. An approval is not customer acceptance, an obligation or a mandate. Drawdown approvals and individual reviewer ceilings remain outstanding.

The Sales tools menu and draft detail link to Credit approvals. Owners can configure the threshold, staff can request review, and independent reviewers see the captured amount and terms and must give a reason. Stale approvals cannot be decided. Policy edits reject stale versions; uncertain results can be refreshed without silently duplicating a financial action. Without PostgreSQL this feature reports unavailable.

Local verification evidence is in `.tmp/business-authority/`. Frontend diagnostics passed with zero errors/warnings. The full Go suite and the affected buyer, credit, approval, trade-line, payment, database and web integration suites passed after the final authority changes. Migration 174 and the approval integration suite passed, including attempts to rewrite captured proposal, version and organization through runtime database permissions (`evidence.log`). No production deployment, personal-data deletion or live provider transaction was performed.

The final production build and Vercel adapter passed (`build.log`). All 36 built-application browser checks passed (`browser.log`), including maker/reviewer separation, exact approval amounts, stale drafts, network navigation and financial retry behavior. Standalone TypeScript passed (`types.log`); the concurrency-enabled authority and approval checks passed (`race.log`). API contract synchronization, dynamically composed endpoint coverage and whitespace checks passed (`contracts.log`, `coverage.log`). Browser API responses were mocked; the financial approval transition itself was separately exercised through the real PostgreSQL credit repository. The browser used its own port 5176 because another process occupied 5173; the existing process was left alone.

## Reviewer ceilings and supplier network operations

Migrations 175–177 add versioned per-reviewer single-sale principal ceilings, immutable limit history, supplier-private branches/territories, partner account-manager assignments and immutable operation history. Owners set reviewer ceilings. A ceiling is checked both at approval and before send; reducing it prevents a previously approved offer exceeding the new ceiling from being sent. A reviewer cannot change their own limit without owner authority. Initial absence of a ceiling means no additional per-person restriction; zero blocks positive approvals. The organization's independent-review threshold remains a separate control.

Partners → Branches & managers allows current owners/administrators to create and edit branches, close them to new assignments, and assign connected customers to an eligible current owner, administrator or salesperson. Closed branches and removed managers remain visible as historical assignments and are flagged for correction. Assignments never grant access to a customer's own network or personal purchases. Cross-business branches, unrelated customers, inactive managers and stale revisions are rejected. Stable identity/version retries recover the committed record without adding another history event.

Current-user, membership and business-state restrictions apply independently at the database layer to these records and the credit approval/history tables. Runtime workers cannot read or modify these supplier-private records; history is read-only to the application. The management screens include identity confirmation, explicit business context and loading/error/recovery states.

Local evidence is in `.tmp/enterprise-completion/`: the full Go suite passed; affected credit/web database suites passed; reviewer and network-operation integration checks passed; concurrency-enabled checks passed; database policy checks and updated role grants passed. All migrations through 177 also passed on a new empty local database (`fresh-migrate.log`). Frontend diagnostics passed with zero errors/warnings. No deployment or personal-data deletion was performed.

The final frozen-frontend production build and Vercel adapter passed (`build-snapshot.log`), as did standalone TypeScript (`types.log`). The main browser pass passed 40 checks; two new branch checks used an exact-label locator that did not resolve the select's accessible name. After correcting those test locators, all four branch checks passed (`browser-network-final.log`), covering all 42 affected checks across the two runs. Desktop/mobile screenshots were inspected and the mobile horizontal-overflow check passed (`visual.log`). The changed feature files match the verified snapshot; its hashes are in `frontend-snapshot.json`.

Verification used a frozen local frontend copy because another active development process repeatedly replaced generated output in the shared checkout. Earlier shared-output browser failures were asset/preview-server failures, not successful product checks. The existing development process was left running. Production deployment and live checks remain user-managed.

## Canonical identity and delegated single-sale purchasing

Migrations 178–188 make organizations the current mutable business identity. Purchasing profiles are unique capabilities with projected identity details. Unlinked profiles receive an unverified workspace; identity collisions require explicit reconciliation. Existing agreement snapshots, financial references and personal details are preserved. A profile cannot be rebound to another company or linked by an unrelated owner.

Purchases → Team permissions lets a current owner grant read, review/decline, acceptance and delivery authority to active staff, with an expiry of at most one year and a per-acceptance principal ceiling. Grants have optimistic versions, exact replay recovery and immutable history. They bind to the membership ID and authority version: removal, suspension, restoration or role changes cannot revive old delegated authority. Revocation is enforced on reads and held through financial writes. Suspended businesses cannot expose permission metadata through the application role.

Purchases → Staff setup records the acting person's own notices and establishes pending identity and representative checks. Enrollment does not verify anyone. Existing personal identity is reused without overwriting names. Accepting a single-sale offer still requires current identity/business/authority evidence; acceptance records the actual user's ID and actual person-profile ID. Receipt evidence records the acting staff member. Both the acting user's and recorded buyer's risk restrictions are checked. Private bank-provider authorization URLs are removed from delegated read and mutation responses.

Delegates can read business purchase balances and history. The recorded owner still controls bank authorization and the currently owner-oriented trade-line, payment-management, dispute-management and amendment workflows. Those specialized scoped screens return an explicit restriction, not a misleading empty account. Server-side guards prevent read-only access from becoming a payment-link, claim or dispute action. Personal consumer records are not added to business purchasing access.

Verification evidence is in `.tmp/identity-completion/`. The complete Go suite passed with loopback access for connector tests. Affected database suites passed, including a complete staff review → ceiling check → acceptance → supplier release → staff receipt/ledger activation cycle, recorded actor identity, ownership boundaries and revocation. Concurrency checks passed for buying identity, purchasing grants and credit activation. Frontend diagnostics passed with zero errors/warnings and a production build passed. The four staff permission/enrollment browser checks passed. Further final browser/build results are recorded alongside these logs; an earlier public mobile run exposed footer contrast and that source issue was corrected. These checks use isolated local databases and mocked browser APIs, not production or real bank transactions.

## Branch access and authenticated supplier reads

Migrations 189–194 add owner-managed staff branch scopes, immutable scope history and current membership-version binding. Owners/administrators retain company-wide authority. A selected-branches scope with no branches denies customer access; unassigned customers stay with company-wide staff. Removing/restoring membership cannot revive an old scope. Closing a branch suspends branch-restricted access to its customers; company-wide staff retain history.

The database narrows existing tenant policies on customer relationships, sales, agreements, obligations, payment/schedule/dispute/collection records, bank mandates and linked settlement records. Private supplier snapshot/customer/mandate readers apply the same boundary. Financial writes lock current membership and assignment so a scope or assignment change cannot silently overtake a financial write. Buying authority does not expand a supplier branch scope. Worker collection authority remains separate.

Supplier reads now retain the authenticated staff identity through detail views, customer directories and reports; a report's customer filter never substitutes for the acting staff member. Actorless supplier reads fail closed in a business with branch restrictions. Exact grant retries recover the saved revision without duplicate history. The new Partners → Branch access screen supports owner edits, read-only self-view, stale-membership notices, empty-scope removal and explicit limits. Unsupported company-wide routes return a clear restriction. This is verified single-sale branch coverage, not completion of every specialist workflow in the enterprise blueprint.

Local verification is recorded in `.tmp/branch-completion/`. The focused real-database scenario covers branch filtering, direct-ID reads, cached write attempts, actual sale activation, financial records, reassignment, current membership binding, exact replay, branch closure and independent worker authority. Migrations through 194 also applied to the separate clean local installation. No production deployment, provider transaction or personal-data deletion occurred.

Final branch verification: the complete Go suite passed (`go-all.log`); the affected buyer, report, network, payment, database, credit, drawdown, collection and web integration suites passed across the final recorded runs. The final privileged-reader change passed credit/drawdown/collection/web verification (`authority-final.log`). The real-database branch scenario passed with race detection and demonstrated that reassignment waits for an active financial transaction (`serialization.log`). Frontend diagnostics report zero errors/warnings (`check.log`), the final production build and Vercel adapter passed (`build-final.log`), and all 14 affected browser checks passed (`browser-ready.log`). The three branch-screen checks also passed with the new mobile accessibility/overflow assertion (`accessibility.log`); the mobile screenshot was inspected. API contract synchronization, dynamic endpoint coverage and whitespace checks passed. An earlier browser attempt could not connect because the preview lacked local-port permission; it was stopped and rerun after the preview was confirmed ready. Browser APIs are fixtures; database lifecycle and permission checks use real isolated PostgreSQL transactions.
