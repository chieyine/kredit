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

Multiple roles combine their permitted functions. Independent approval always requires a different person. Financial approvers cannot be the buyer or an active member of the supplier business. Permissions and active account/role status are checked again when applying a stored change. Team role changes require recent MFA, CSRF, an idempotency key and an audit reason. The interface includes identity confirmation for expired verification sessions.

## Financial corrections

1. A financial operator or platform administrator loads current obligation details and submits an immutable proposal with an expiry within 30 days.
2. Another approver reviews the exact amount and reason. The transaction locks the obligation, schedule and proposal and rechecks balances, allocations, accrued fees and pending debits.
3. An approved write-off posts one balanced journal, reduces the outstanding balance and latest unpaid schedule amounts, updates the credit projection, records decision evidence, and queues existing financial notices in one transaction. Paid allocation references remain available for reversals. Disputed instalments and pending buyer payment claims must be resolved before principal write-off.
4. A fee waiver cannot exceed accrued, unwaived fees. Caller-supplied approver IDs never authorize a correction. Direct supplier corrections remain limited by the cumulative policy threshold; larger amounts use this workflow.

Concurrent duplicate approval attempts cannot create duplicate journals. A changed snapshot, expired proposal, revoked proposer, conflicting financial reservation or missing authority rejects application. Cancel and submit a fresh proposal against current details. Applied records cannot be edited or deleted.

## Repayment dates

Amendments change dates for all existing unpaid instalments in their existing order. They preserve principal, paid allocations, recorded fees, grace periods, original agreement records and item references. New dates must allow the current delivered-notice period and be within five years. They do not restructure instalment amounts or write a new agreement over the accepted one.

Independent admin approval moves the proposal to `awaiting_buyer` and queues a durable notification. The original schedule continues while consent is pending. The buyer reviews previous and proposed dates, confirms their identity if necessary, and explicitly accepts or rejects. Acceptance rechecks the exact snapshot, proposer and approver authority, expiry, current notice policy, disputes and unresolved collection reservations. Changed balances require a new proposal. The update, buyer identity, decision reason and notification commit together.

Changed dates produce new reminder identities and invalidate prior pre-debit notice matching. Provider mandate scope and actual notice delivery remain independently enforced at collection time. An amendment does not renew a mandate or create provider authorization.

## Deployment and limits

Apply migrations 064–066 after the existing 061–063 policies/commitment migrations, using the migration owner. API and worker readiness require schema 66 and the new tables, views and functions. The migrations grant runtime access when the application roles exist. Standard roles.sql grants cover installations where roles are created later.

Migration 064 deliberately refuses rollback because approval/consent history must survive. Use a forward correction. No real provider calls are needed to deploy these code paths, but live Mono operation still requires separately available credentials, provider access and buyer-hosted authorization. No legal or provider certification is implied by an administrator decision.

Review deadlines are operational targets. They do not redefine contractual or statutory deadlines. Free-text reasons and retained before/after snapshots are audit data; use the approved retention and access policies. No secrets belong in proposal notes.
