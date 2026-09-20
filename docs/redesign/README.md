# Kredit network redesign

Status: implementation blueprint. This document proposes the replacement product; it does not describe shipped functionality. No production reset, schema change or deployment is authorized by this document itself.

## Product decision

Kredit helps businesses manage credit they receive and credit they extend. Manufacturers are the initial distribution channel. Distributors and retailers remain independent businesses that can join several supplier networks and invite their own customers. Consumers have personal purchase accounts, not artificial business profiles.

Begin with seller-funded trade credit: each seller owns its credit decisions and bears its contractual receivable risk. External financing, guarantees and Kredit-funded lending are separate future products, requiring explicit decisions about responsibility and operating requirements. Direct debit is collection authorization, not a repayment guarantee.

Success means a manufacturer completes repeat trade cycles, a distributor independently activates selling, and its retailers complete real transactions. Invitations, imported records and dormant accounts are not active network growth.

## Evidence and scope

The current implementation contains separate organization memberships, buyer businesses owned by users, seller and buyer navigation, and an existing separate consumer purchase lifecycle. The redesign must address those assumptions across all domains. See [the exhaustive page and module disposition inventory](feature-map.md).

The previous isolation work was verified locally. Production deployment and provider behavior must not be inferred from those results. The assertion that production holds only demo data must be verified before choosing a destructive cutover. Default to a reversible, additive transition until then.

## Identity and authority

- User: authentication identity. A signed-in person is not automatically a business owner.
- Person: natural-person profile and evidence, kept separate from business records.
- Business: canonical trading entity, with legal form, trading names and verification evidence. Informal and sole-trader businesses must not be incorrectly described as incorporated legal persons.
- Membership: current authority to act for a business, with role, status, scope and approval ceiling. Removal terminates future authority, not historical attribution.
- Branch: operating subdivision with scoped staff access; it does not silently become a separate debtor.
- Capability: buying, business selling or consumer selling, each with eligibility and restrictions. Evidence reuse is conditional on purpose, freshness and authority.
- Trading relationship: directed supplier business to customer business. A pair can have reciprocal relationships, each with independent terms and exposure. Self-trade is rejected.
- Invitation: an unaccepted proposal linked to the inviting business's private partner record. It is not membership, verified ownership, accepted credit or a debit mandate.

Do not create a second canonical business for every invitation. Claiming an existing business requires appropriate authority verification. Do not merge by name, telephone or registration text alone. Conflicting claims enter restricted review; imports must not reveal whether arbitrary contacts already have accounts.

The business switcher must make the active business visible before every material action. Server requests verify membership, capability and transaction relationship independently of the UI. A personal purchase never inherits a business administrator's access merely because its purchaser is an employee.

## Financial model

Keep separate concepts for credit policy, policy versions, accepted terms, reservation, obligation, schedule, payment, allocation, settlement and reversal. A mutable relationship row must not be the only record of an accepted limit or debt.

Available credit is calculated from the effective approved limit less active reservations and relevant outstanding exposure. Store performance projections only with authoritative derivation and reconciliation. Define expiry, cancellation and conversion of reservations explicitly. Concurrent approvals must lock or otherwise serialize the same credit capacity.

Conversion of a reservation into an obligation releases the converted reservation in the same transaction that creates exposure. It must never count both amounts or temporarily free capacity. Partial deliveries convert only the agreed portion. Reducing a limit below existing exposure blocks new usage without rewriting existing debt.

Verified principal repayments release only eligible credit capacity, according to the accepted release rule. Fees do not reduce principal. Reversals reinstate the correct exposure; subsequent over-limit status blocks new usage and opens an exception. Provider success, settlement and seller bank receipt remain distinct facts.

Receivables are not cash. Downstream customer receipts do not automatically pay upstream supplier obligations. No cross-business netting or transferable credit limits. Ledger accounts require explicitly documented ownership and accounting perspective before any new postings are introduced.

## Connected lifecycles

Avoid a single status for an entire sale. Present a readable summary while retaining independent state machines:

| Record | Core progression | Exceptions |
|---|---|---|
| Relationship | Proposed, invited, accepted, active | Suspended, closed; historical obligations survive |
| Credit proposal | Draft, submitted, approved, accepted | Rejected, expired, superseded |
| Order | Draft, accepted, partly fulfilled, fulfilled | Cancelled remainder; reservation expiry |
| Shipment | Prepared, dispatched, delivered | Failed delivery, shortage, damage, return |
| Invoice/obligation | Issued under agreed trigger, partly paid, paid | Disputed, adjusted, reversed through evidence |
| Payment attempt | Prepared, submitted, pending, confirmed | Failed, unknown, reversed |
| Settlement | Expected, pending, reconciled | Shortfall, delayed, returned |
| Dispute | Open, evidence requested, reviewed, resolved | Escalated; narrowly scoped collection hold |

Accepted terms define the debt and due-date trigger. Customer silence is not receipt acceptance. Failure to confirm enters an evidence-based escalation route. A shortage report is a claim; it does not automatically create a financial credit note. Approved adjustments are versioned, attributable and reconcile to both parties' views.

Cancellation affects the permitted unfulfilled portion. Returns and refunds do not erase delivery or payment history. Changes to payment dates require preserved acceptance and policy controls. Multiple invoices can receive allocations from one confirmed payment without double counting; unmatched funds remain visible for reconciliation.

## Workspace and screens

Business navigation: Today, Sales, Purchases, Partners, Money, Reports, Team and Settings. Personal purchases and platform operations remain separate contexts. Do not display all enterprise features to every retailer.

| Screen | Primary action | Required detail |
|---|---|---|
| Today | Complete the next authorized task | Due date, business, counterparty, amount, reason, last update; no invented balances |
| Sales | Create or review a sale | Business/consumer separation, approvals, fulfillment, aging |
| Purchases | Review terms or confirm a delivery | Supplier balances, available limits, upcoming dues and disputes |
| Partner directory | Invite or open a relationship | Suppliers/customers, local customer reference, branch and account manager |
| Relationship | Review statement or manage terms | Accepted versions, exposure, trades, communications and authority |
| Trade detail | Act on the current milestone | Items, shipment evidence, agreement, invoice, payment and dispute timeline |
| Money | Reconcile a payment or inspect a collection | Reported versus verified versus settled amounts; incoming/outgoing separation |
| Reports | Investigate an exception or export | Defined measures, reporting period, freshness and access scope |
| Team | Invite staff or change authority | Roles, branch restrictions, ceilings, approval separation and revocation |
| Settings | Finish a capability requirement | Bank accounts, verification, billing, integrations and notification preferences |

Every screen needs loading, empty, restricted, stale, failed and retry states. Action confirmations identify the business, counterparty and exact consequence. Money uses integer kobo in storage and clear currency display. Mobile primary tasks must not depend on wide tables, hover or color alone. Keyboard access, focus, readable contrast and clear error recovery are release requirements.

## Complete journeys

1. Manufacturer: create/claim business, prove authority, configure receiving account and trade policy, invite staff, import distributors, review proposals, send approved invitations, complete initial trade and reconcile repayment.
2. Distributor: open invitation, authenticate, establish business authority, review supplier-specific terms, complete relevant bank authorization, accept trade, inspect delivery, repay and obtain a reconciled statement.
3. Distributor becomes seller: activate selling within the same business; reuse applicable evidence; complete missing settlement, authority and terms requirements; invite a retailer and create its first sale.
4. Retailer: manage supplier purchases, then optionally activate business or consumer selling. Buying remains usable while selling setup is incomplete unless an independently justified restriction applies.
5. Consumer: open personal offer, understand full price and dates, accept, report payment, receive confirmation, track delivery, request return/refund or escalate a complaint. Existing consumer financial separation remains intact.
6. Staff: accept membership, enter the correct business/branch, perform only allowed actions, request approval above their ceiling; losing membership immediately blocks future business actions.
7. Operations: triage a case, inspect minimum necessary evidence, perform an authorized evidenced correction, notify affected parties through approved workflows and close with an audit trail.

## Manufacturer import and enterprise controls

Upload, validate, preview, approve, invite, monitor. Capture source file hash, row identity, batch ID and ERP reference. Explicitly parse naira versus kobo, dates and country formats. Show exposure totals and row-level errors. Retry a row without duplicating relationships or notifications. Cancelling a batch stops unsent invitations without erasing accepted relationships.

Opening balances are a distinct accounting import with provenance, approvals and counterparty reconciliation. Importing a roster creates neither debt nor bank authorization. Invitations say terms are proposed and subject to acceptance; they must not imply guaranteed financing.

Support account-manager assignment, territory and branch views, configurable approval ceilings and separation of proposal from approval. A relationship suspension blocks new trading as specified but does not hide outstanding obligations. Bulk actions show affected counts and amounts and preserve per-record outcomes.

## Expansion and commercial design

Offer selling activation when it is relevant, not only after an arbitrary number of purchases. Explain the concrete benefit: manage customers, terms, statements and collections. Do not promise automatic eligibility or a fabricated credit score.

Track invited, accepted, ready, first completed trade, repeat trade and activated-as-seller separately. Define attribution without giving an inviter account control or private downstream data. Organic invitations and paid referral rewards are separate programmes.

Pricing proposals should identify who pays, for what event, and how reversals affect fees. Preserve accepted fees and billing evidence. Validate price willingness and cost to serve in the pilot; do not label transaction revenue as profit or acquisition cost as zero.

## Integrations, operations and public experience

Publish versioned contracts only after domain ownership is settled. Map external business/customer/order identifiers per integration. Authenticate and scope inbound events, protect against replay, retry safely and quarantine conflicting or out-of-order events. An ERP must not silently overwrite an accepted agreement. Provide delivery logs and redacted diagnostics, not secrets.

Provider mandates remain tied to their actual provider, account, consent, amount limits and lifetime. An unknown submission keeps its original reference and is reconciled before another attempt. No assumed mandate portability or indiscriminate provider failover.

Operations console groups work into onboarding/identity, trade disputes, financial reconciliation, provider recovery, privacy/access and system health. Maintain restricted access, reasons, approval where required, MFA and immutable evidence. Support impersonation is not a default feature.

Public website explains the manufacturer entry point and separate value for distributors and retailers, with truthful capability claims and role-specific calls to action. Update demo, pricing, help, notification copy and onboarding together. Do not present future functionality as live. Preserve consumer terms, complaints and privacy paths.

## Architecture and source ownership

Retain the modular Go application, PostgreSQL, financial ledger, audit, idempotency, outbox and worker boundaries. Redesign identity and authorization assumptions throughout them rather than exempting retained code from review.

- `internal/organizations`, `buyers`, `relationships`, `access`: canonical business, memberships, partner claims and directed relationships.
- `credit`, `tradelines`, `businesspolicy`, `schedules`: policy versions, capacity, accepted obligations and timing.
- `payments`, `collections`, `settlement`, `ledger`, `corrections`: financial authority, allocations, reconciliation and reversals.
- `consumer`: separate personal contract, delivery and refund lifecycle.
- `onboarding`, `identity`, `mandates`: reusable evidence with capability-specific authority and original-provider provenance.
- `web`, `web/src/routes`, shared client contracts: unified business context, screens and server-side authorization.
- `platformops`, `support`, `usercontrol`, `reports`, `notifications`, `whatsapp`: operational recovery, privacy, scoped projections and communications.

Do not simply rename organization IDs to business IDs. Inventory every foreign key, policy, function, worker claim, notification link, export and public token. Decide contractual debtor/creditor identity separately from acting-user identity and tenant context.

## Implementation sequence and acceptance gates

1. Contract and identity foundation: schema design, authority matrix, immutable transaction identities and import mappings. Gate: one business buys and sells; staff revocation and unrelated-network access are enforced in API and database.
2. End-to-end trade slice: manufacturer to distributor through accepted terms, capacity reservation, delivery, debt, payment and reconciliation. Gate: concurrent reservations, partial receipt, unknown debit and reversal behave correctly.
3. Unified workspace: business switcher, Today, purchases/sales and relationship statements using actual scoped data. Gate: no duplicate onboarding, no mixed-business cached views and complete mobile action paths.
4. Manufacturer operations: imports, staff approvals, branches, statements and batch management. Gate: retrying an import or invitation creates no duplicate liability or message.
5. Downstream expansion: selling activation and distributor-to-retailer trade. Gate: retailer completes a real cycle without the manufacturer obtaining downstream access.
6. Consumer and operational completion: incorporate retained personal journeys, returns, recovery, privacy, billing, reports, support and public pages. Gate: every feature in the disposition map has an implemented destination or an explicit retirement decision.
7. Integration and launch: ERP contracts, monitoring, backup/restore, staged production rollout and connected-provider verification. Gate: balances reconcile, critical alerts reach the configured receiver, and authorized rollback procedures are rehearsed.

No calendar estimate is asserted before dependency and schema mapping. Each stage must have a working vertical slice, not just new tables and empty pages.

## Cutover and verification

Take a verified backup and inventory deployed versions, credentials by location, provider references and actual stored data. If demo-only status is confirmed and a reset is explicitly approved, a clean baseline is an option. Otherwise use additive migrations, explicit mapping and reconciled transition. Existing signed agreements, external references and ledger facts are never silently rewritten.

Record pre/post counts and financial totals per business and relationship. Compatibility links may redirect only after authorization and context are resolved. Financial rollback uses forward correction where actions cannot safely be undone; deploying an older binary against an incompatible schema is not a rollback plan.

Verification covers maker/checker bypass, revoked staff, guessed IDs, reciprocal relationships, multiple businesses per person, duplicate/replayed callbacks, concurrent reservations, partial deliveries, disputed amounts, payment reversal, provider outage, notification retry, privacy exports and consumer refunds. Existing checks remain a baseline, not proof of the new model.

Pilot with one manufacturer and a small authorized distributor cohort. Define on-time repayment denominator, activation duration, reconciliation effort, exceptions per trade, support cost and buyer-to-seller conversion. Expand only when real completed cycles support it.

## Original first engineering deliverable

Create the foreign-key and authorization dependency map and a concrete schema/API change proposal for canonical businesses and memberships. This must precede deleting `/buyer`, merging identity tables or resetting a database. The blueprint and page disposition map are complete planning artifacts; implementation gates above remain pending.

## Implementation record

See [implementation status](implementation-status.md) for the code delivered and the remaining architecture work. The user has waived backward compatibility; old `/app` and `/buyer` page routes have been removed rather than redirected. API naming has not been globally renamed. No live database reset is implied or performed.
