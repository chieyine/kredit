# Admin workflows

## Where to work

- **Approval inbox** (`/admin/inbox`): open policies, financial proposals, disputes, financial reviews, support, recovery and privacy cases. The inbox shows only categories permitted by the signed-in operator's active roles. Claim or release ownership and set a review deadline with a recorded reason. Business decision deadlines and proposal expiries are enforced separately.
- **Financial changes** (`/admin/approvals`): load an obligation or credit-request reference, review current balances, and propose a principal write-off, supplier fee waiver or changes to unpaid repayment dates.
- **Business settings** (`/admin/settings`): enter money in naira and fees as percentages. Preview counts and descriptions before proposing or approving. Counts describe current records; execution always rechecks eligibility and limits.
- **Change history** (`/admin/history`): search all retained proposals and assignments by person, reason, reference and category. Inspect previous/proposed values and decision events. Export the selected page (100 records) as CSV; paginate to older records. CSV neutralizes spreadsheet formulas and preserves numeric values.
- **Work needing attention** (`/admin/attention`): role-scoped overdue review counts, unresolved settlements, uncertain debits, failed notices, and upcoming mandate expiries. Provider operators can review up to 100 detail records in each operational category.
- **Buyer date changes** (`/buyer/amendments`): the authenticated buyer reviews independently approved dates and explicitly accepts or rejects them. Accessible from buyer navigation and durable amendment notifications.

## Roles

| Role | Powers |
| --- | --- |
| Financial operator | Read platform money and financial proposals; propose corrections and repayment dates; cancel their open proposals. |
| Policy manager | Read policies and propose settings; cancel pending/scheduled policies. Cannot approve policies. |
| Approver | Read policies, money and financial proposals; independently approve/reject changes. Cannot propose financial or policy changes. |
| Access administrator | Manage the admin team and find users. Cannot grant/revoke platform-administrator or access-administrator roles; these need a platform administrator. Cannot change their own roles. |
| Support agent | Existing support powers and support inbox. |
| Compliance reviewer | Existing compliance, provider, privacy and recovery powers and related review inbox. |
| Dispute reviewer | Existing dispute powers and dispute inbox. |
| Platform administrator | All administrator functions, subject to independent approval, buyer-consent and financial invariants. |

Multiple roles combine their permitted functions. In delegated-team mode, independent approval requires a different person. Solo-owner mode permits the current platform owner to approve their own policy or financial proposal; buyer consent and financial invariants still apply. Financial approvers cannot be the buyer or an active member of the supplier business. Permissions and active account/role status are checked again when applying a stored change. Team role changes require recent MFA, CSRF, an idempotency key and an audit reason. The interface includes identity confirmation for expired verification sessions.

## Financial corrections

1. A financial operator or platform administrator loads current obligation details and submits an immutable proposal with an expiry within 30 days.
2. The authorized approver reviews the exact amount and reason under the configured governance mode. The transaction locks the obligation, schedule and proposal and rechecks balances, allocations, accrued fees and pending debits.
3. An approved write-off posts one balanced journal, reduces the outstanding balance and latest unpaid schedule amounts, updates the credit projection, records decision evidence, and queues existing financial notices in one transaction. Paid allocation references remain available for reversals. Disputed instalments and pending buyer payment claims must be resolved before principal write-off.
4. A fee waiver cannot exceed accrued, unwaived fees. Caller-supplied approver IDs never authorize a correction. Direct supplier corrections remain limited by the cumulative policy threshold; larger amounts use this workflow.

Concurrent duplicate approval attempts cannot create duplicate journals. A changed snapshot, expired proposal, revoked proposer, conflicting financial reservation or missing authority rejects application. Cancel and submit a fresh proposal against current details. Applied records cannot be edited or deleted.

## Repayment dates

Amendments change dates for all existing unpaid instalments in their existing order. They preserve principal, paid allocations, recorded fees, grace periods, original agreement records and item references. New dates must allow the current delivered-notice period and be within five years. They do not restructure instalment amounts or write a new agreement over the accepted one.

Independent admin approval moves the proposal to `awaiting_buyer` and queues a durable notification. The original schedule continues while consent is pending. The buyer reviews previous and proposed dates, confirms their identity if necessary, and explicitly accepts or rejects. Acceptance rechecks the exact snapshot, proposer and approver authority, expiry, current notice policy, disputes and unresolved collection reservations. Changed balances require a new proposal. The update, buyer identity, decision reason and notification commit together.

Changed dates produce new reminder identities and invalidate prior pre-debit notice matching. Provider mandate scope and actual notice delivery remain independently enforced at collection time. An amendment does not renew a mandate or create provider authorization.

## Deployment and limits

Apply all migrations through 148 using the migration owner. API and worker readiness require schema 148 and the new tables, views and functions. The migrations grant runtime access when the application roles exist. Standard roles.sql grants cover installations where roles are created later.

Migration 064 deliberately refuses rollback because approval/consent history must survive. Use a forward correction. No real provider calls are needed to deploy these code paths, but live Mono operation still requires separately available credentials, provider access and buyer-hosted authorization. No legal or provider certification is implied by an administrator decision.

Review deadlines are operational targets. They do not redefine contractual or statutory deadlines. Free-text reasons and retained before/after snapshots are audit data; use the approved retention and access policies. No secrets belong in proposal notes.


## Super-admin launch setup

Open `/admin/setup` as the platform owner. Confirm your identity with an authenticator when prompted. The page lists provider connections, private storage, document scanning, legal content and launch evidence, with API and worker configuration status. “Recorded” or “applied” is not a successful provider transaction or an approval from the provider.

Runtime settings are encrypted. With `ADMIN_CONFIG_AUTO_APPLY=true` and the production restart supervisor, the API and worker gracefully restart to apply changed runtime versions. Messaging settings apply to new events directly; existing events retain their original encrypted connection. A brief service interruption is possible while processes restart.

- `/admin/message-submissions`: review uncertain native message sends. Record the original accepted message reference or close without resending. Neither action proves delivery.
- `/admin/settlement-review`: review the registered account holder against the verified business. Approve only the unchanged provider-backed destination. For a held registration, permit a retry only after the original provider confirms no sub-account was created.
- `/admin/provider-work`: review pending provider operations and follow their recovery links.

An initial deployment still needs the database, encryption roots, server/domain setup and first owner bootstrap. Provider account activation, sender/template approval, domain records and legal decisions take place with the relevant provider or reviewer. Enter the resulting supported settings in admin; never substitute invented references.

Current source work still lacks complete native Mono identity and payment-level settlement/billing flows. Do not treat these admin pages as confirmation that the whole platform is production ready. See `docs/platform-improvements/IMPLEMENTATION.md` for the current completion record. No implementation-stage tests or migrations have run.


## Platform fee bills

Open **Fee billing** in super admin. A seller's invoice preference stays pending until the owner approves that exact arrangement and supplies verified Kredit receiving-bank instructions. Weekly periods close Monday at midnight in Lagos; monthly periods close on the first day. The worker bills eligible unbilled fees from closed periods, including missed periods, with a seven-day payment window. No fees are duplicated when the worker restarts.

Use **Record a received payment** only after checking an actual credit in Kredit's bank statement. Enter the bank transaction reference, naira amount, Lagos timestamp and evidence reference. A seller's transfer screenshot alone is insufficient. A repeated bank reference cannot create another receipt. Payments are for platform fees and leave buyer debt unchanged.

Waivers and recognised collection reversals appear as credits. **Record a completed refund** records a refund already completed through the bank, up to the bill's credit balance. It does not send money. Receipts and refunds are immutable evidence; uncertain outcomes should be refreshed before retrying. Incorrect bank evidence must be escalated for controlled correction, never overwritten.

The seller can read and print the latest 100 bills in **Settings → Kredit fees**. Split-settlement and authorised-debit preferences are still pending their dedicated implementation and cannot be activated through invoice approval.

## Previous collection accounts

In **Platform settings → Saved collection accounts**, save a generic account before switching the active provider. Its saved copy must exactly match the active connection. Once switched, existing collection attempts continue to use their original account; new attempts use the active provider. Use a distinct name for each provider account. Blank credential fields preserve the saved values only at the same address. An address change requires new credentials. Keep accounts while any reconciliation or return can still arrive. Mono continues to use its dedicated configuration; native account rotation remains unfinished.

## Fee bank permissions and settlement

Use **Fee billing** to review every saved fee setup, including interrupted registrations that were not yet selected as a billing preference. The seller starts setup in **Settings → Kredit fees**, completes the hosted bank permission, selects its saved setup and saves the preference. Approval verifies the original provider's customer, mandate, lifetime ceiling, reference, dates and readiness. Enter verified receiving-bank instructions in the review evidence; those instructions support transfer payment if a bill cannot be debited.

A pending debit is checked using its original reference. Do not pay or retry it while its outcome is unknown. Failed or below-minimum bills remain payable by bank transfer. A provider dispute/reversal signal blocks conflicting receipts and subsequent automatic fee debits until reviewed. **Review provider evidence** requires both recorded bank evidence and a matching provider lookup. **Record a completed bank reversal** reopens the fee bill without changing buyer debt.

The fee bank movement panel distinguishes provider-held fees from confirmed cash. Record each completed bank receipt or return once using its bank reference. A receipt here does not charge the seller again. Seller proceeds use **Seller settlements**, which records the original destination, customer payment, actual payout and any return separately.

A rejected identity check can be reopened after an evidence-backed appeal in **Verification recovery**. The old decision is preserved; reopening never verifies the person or business. Expired checks require fresh evidence.

Before changing financial providers, save the old named account under **Launch setup → Saved collection accounts**. Fee permissions and unresolved collections remain bound to it. New credentials for another account should use a distinct account name. Retain old identity connections similarly until their work is reconciled.


Mesaj activation also requires a valid HTTPS certificate on its API host. The main website's certificate alone does not establish that the API certificate is valid. Kredit does not bypass TLS validation; configure an approved alternative messaging connector if the account's API remains unavailable.

Split-settlement arrangements also issue a monthly bill for any fees left unpaid. Previously deducted amounts are shown separately, including any later reversal. The receiving-bank instructions supplied when approving the arrangement are used for these bills.
