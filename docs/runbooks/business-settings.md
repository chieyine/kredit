# Business settings

The admin configuration centre is available at `/admin/settings`. It requires an active platform administrator, policy manager or approver and MFA; changes require fresh MFA, CSRF protection, and an idempotency key.

## Available controls

| Area | Settings | Effect |
| --- | --- | --- |
| Collections | New collection switch, automatic collection, automatic retry, maximum attempts | The worker checks active settings for new work; collection transactions also enforce the new-collection switch and attempt limit. Reconciliation remains available while paused. |
| Payment claims | Enable new off-platform payment claims | Both HTTP and database writes enforce this. Existing claims can still be reviewed. |
| Notices | Delivered notice minimum, upcoming-payment reminder lead time, mandate-expiry reminder lead time | Changes affect new debit reservations and future notification discovery. Existing queued/sent reminders are not recalled. The deployment's delivered-notice floor still applies. |
| Limits | Supplier count, buyer count, new principal, buyer exposure, drawdowns per line per Lagos day, allowed industries | Database write guards enforce business counts, principal, aggregate exposure and daily reservations. Exposure is checked when goods are released or a drawdown is reserved, and includes released ordinary sales awaiting receipt. Tightening limits does not suppress recognition of an existing goods commitment. Existing balances and accepted amounts are preserved. Cancelled drawdowns still count toward the daily cap. |
| Review flag | Enhanced-review threshold | Flags new credit requests for enhanced review. This is a review flag, not proof of approval or a newly implemented individual-credit approval workflow. |
| Corrections | Cumulative correction threshold | Write-offs and fee waivers reaching the threshold require a recorded proposal and independent approval through `/admin/approvals`. It can be lowered, but cannot exceed the existing ₦10,000 ceiling. |
| Fees | Supplier base and collection rates, entered as percentages | New credit drafts and new drawdown offers record both rates and their policy revision. Rates are included in agreement disclosures. Activation journals and collected-payment fees use those recorded terms. |

There are 18 configurable fields. Monetary policy values are whole kobo; 100 kobo is ₦1. Rates use basis points; 100 basis points is 1%. The page accepts naira and percentages with up to two decimal places and converts them exactly into the stored units. Zero means no additional admin cap only where the field explicitly permits it. Hard validation bounds remain in code.

## Changing a policy

1. Refresh the settings and review the current revision.
2. Edit the values, review the displayed changes, give a business reason, and choose a future effective date in Lagos time, within one year.
3. Submit the proposal. No active setting changes at this point.
4. A different active approver or platform administrator reviews the exact immutable proposal and records an approval or rejection before its effective date. A revoked or suspended proposer cannot have an old proposal approved.
5. An approved proposal takes effect according to database time. API/worker policy reads do not require a restart. A transaction already in progress uses the policy it read; bank submissions already made cannot be recalled by the switch.

Only one pending or future scheduled change is permitted at a time. A stale revision, conflicting proposal identifier, incomplete JSON snapshot, or invalid combination is rejected. A scheduled change can be cancelled before it becomes effective. To revert an active policy, submit a new proposal using the desired previous values; history is never edited. Settings display the last 100 changes. `/admin/history` searches all retained changes with pagination, original values, named actors, decision reasons and a CSV export of each page.

For an urgent customer-specific stop, the existing risk-hold controls remain available. The business settings workflow requires independent approval even when pausing all new collections.

## Deployment boundary

Database-backed runtimes seed initial policy values from deployment configuration exactly once. Subsequent environment changes do not silently replace saved settings. The startup check validates both active and scheduled policies against deployment ceilings. Lower admin limits before tightening deployment ceilings; otherwise the new deployment will fail readiness rather than run with conflicting approvals.

Secrets, provider endpoints, provider capability/real-money enablement, Mono certification, partial Sweep capability, identity/WhatsApp integrations, live supplier billing, legal/retention/launch approvals, currency and ledger invariants remain deployment controls. An admin switch cannot manufacture provider approval or implement an unavailable provider capability. Approved deployment count/exposure/principal/attempt ceilings, industry restrictions, enhanced-review threshold and notice floor continue to constrain policy changes.

## Existing agreements and published prices

Fee policy changes affect new offers only. Even an existing draft keeps its recorded rates, so refreshing or sending it cannot silently reprice it. Older records without explicit fee terms retain the historical 50-basis-point base and collection rates. The existing legacy drawdown wording/hash compatibility remains intact.

Public pricing reads `/api/v1/pricing` without caching. Buyer and supplier offer screens display the offer's recorded rates. Fee rows record the actual basis-point rate as well as the amount. Receipts, activation journals, obligations and schedules commit together on the ordinary credit activation path; a rejected limit cannot leave a separate activation journal behind.

## Deployment and rollback

Apply migrations 061 through 066 with the migration owner, then apply `infra/postgres/roles.sql` according to the deployment procedure. API and worker startup require schema version 66, the policy functions/tables and the fee-term columns. The role template covers policy authorization and sequence permissions.

The demo seed command is development-only and initializes deployment defaults before synthetic fixtures. Use a disposable database for the integration suite; it intentionally creates financial records and exercises immutable history.

Migration rollback is supported before feature use. Migration 063 refuses to restore the previous commitment rules while policy history or principal/exposure caps are in use. Migration 062 refuses to remove non-default offer fee terms; migration 061 refuses to discard any policy decision history. Once those records exist, repair with a forward migration. Do not drop columns or disable history guards to force a downgrade.

Policy history contains administrator identifiers and reasons. It follows financial/audit retention holds pending the approved retention register; do not place credentials or customer bank details in free-text reasons.
