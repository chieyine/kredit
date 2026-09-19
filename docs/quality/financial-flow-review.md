# Financial flow review and evidence map

Prepared 2026-09-19. This is a source trace, not runtime verification. No automated tests, application exercises or provider calls were run. The rows name existing enforcement points and the evidence still needed before describing the flow as proven.

| Flow | Source enforcement inspected | Required future evidence |
| --- | --- | --- |
| Payment recognition and partial payment | `internal/payments/postgres.go`, `RecordTx`: authorized tenant context, obligation/request row locks, idempotency comparison before and after locking, outstanding-amount ceiling, schedule allocations, balance update and journal posting in the caller transaction. | Lost-response replay, simultaneous partial payments, commit failure and restart must preserve one payment, exact allocations and matching outstanding totals. |
| Journal retry | `internal/ledger/store.go`, `sameTransactionIntent`: exact posting multiset, header/reference equality and posting-count checks. | Reordered persisted postings replay without duplicating the journal; changed amounts/accounts and altered duplicate counts are rejected. |
| Buyer transfer report | `internal/web/payment_claim_handlers.go`, `decidePaymentClaim`: supplier permission and business scope, confirmation through `PaymentClaims.Confirm`, separate rejection path. `internal/paymentclaims/store.go` builds a `payment-claim:<id>` recognition identity. | A report alone does not reduce debt. Confirmation/replay creates one recognized payment; rejection and expired holds have the intended collection effects. |
| Payment reversal | `internal/payments/postgres.go`, `ReverseContext`: guarded reversal flow with an appended reversal identity and prior allocation records. | Replay produces one reversal, preserves the original event and restores only the applicable balance, allocation and fee effects. |
| Unknown bank collection | `internal/collections/postgres.go`, `Reconcile`: provider reconciliation through the existing transactional engine path. Monitoring counts unknown attempts and oldest unresolved age. | Provider timeout followed by success, duplicate/out-of-order webhook, partial result and crash/restart must not issue another debit while the first outcome is unresolved. |
| Mandate cancellation | `internal/mandates/provider.go`, `CancelMandate`: durable local block precedes remote cancellation. | Provider unavailability cannot reopen eligibility; already-submitted debits remain separately tracked; reconciliation does not undo cancellation intent. |
| Delivery and activation | `internal/tradelines/store.go`, `RecordDrawdownReceipt` and `CancelDrawdown`; `internal/tradelines/postgres.go`, `persistAggregateTx`: explicit confirmation/return resolution and dispute/audit persistence. | Concurrent confirmation/cancellation, expired lines and transaction failure preserve one terminal outcome, original evidence, correct reserved capacity and at most one obligation. |
| Seller settlement | `internal/settlement/receipts.go`, `Record`: bank-reference evidence under platform-owner authority. The interface now retains mutation identity and validates the matching receipt in the returned payout. | A timeout/retry records one bank receipt; a different amount/direction under an existing reference is rejected; recording settlement does not reduce buyer debt again. |
| Financial-review handover | `internal/platformops/financial_review.go`: serialized refresh/decision, locked current authority, assigned-reviewer release, platform-owner takeover, immutable events; resolution checks current discrepancies. | Simultaneous handover, role revocation and resolution cannot lose ownership history or hide a live difference. |

## What the source trace does not establish

Session/MFA changes alter HTTP idempotency scope. Domain deduplication must be demonstrated separately for each command; keeping a browser key does not prove every backend effect is exactly once. A success HTTP status does not establish provider settlement. An aggregate ledger total does not establish per-transaction balance or agreement with payment/schedule projections.

Provider behavior, database isolation under real concurrency, deployment readiness, actual backup restoration, notification delivery and real-user comprehension remain unverified. Existing test files are not evidence that those scenarios were executed against this patch. Keep the original audit's `REVIEW-LIMITS.md` with this record, including known fixture gaps.

## Release evidence record

For each later authorized exercise record: immutable commit/images; schema version; environment and named operator; controlled starting records; exact scenario; resulting domain/journal/provider references; observed result; unresolved differences; reviewer and date. Do not copy raw credentials, bank authorization URLs or personal evidence into public reports. Leave unexecuted rows explicitly pending.
