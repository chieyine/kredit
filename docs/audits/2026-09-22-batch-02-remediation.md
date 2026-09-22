# Batch 2 source remediation — 22 September 2026

Base: `35926ae0c93f5f5d9fac1a9c663f55cbd6ad29d1`, the unmerged Batch 1 repair branch.
Scope: K11, K12, K13, the additional K09 locations, and the D02-01 date contract.

This is a source-change record. It is not a statement that the entire repository, all locking paths, or the deployed system have been verified.

## K11 — Financial lock ordering

The reviewed existing-obligation paths now acquire the obligation before its credit request. `db.LockObligationRequest` uses separate statements, preserves tenant/RLS boundaries, and does not derive or elevate an actor from the target row. Payment recording and balance rebuilding use this helper. Reversal first reads the authorized payment to locate the aggregate, locks the obligation and request, and then rereads and locks the payment. It verifies that the payment still belongs to that same obligation before mutation.

`orders.lockSupplierOrder` retains the original active-user, membership, allowed-role, organization-status and selected-tenant checks. It first locks authority records against revocation, then any existing obligation, then the credit request. This applies to the common order boundary, including credit-note creation and approval; it avoids leaving the old request-first path in a sibling order writer.

An obligation can be created while an order transaction waits for its request lock. After obtaining that lock, the order boundary checks the canonical obligation again. A changed association causes `ErrConflict` before acquiring a late obligation lock. The owning transaction must roll back; the code does not automatically retry a financial mutation.

The reviewed claim-confirmation caller already takes its obligation lock before invoking `RecordTx`. The credit-note approval caller takes the common order lock before its note lock. The relevant migration 201 authority/credit-note trigger and foreign-key definitions were inspected. This correction is not a guarantee that every unreviewed credit, trade-line, collection, administrative or trigger path in the repository is deadlock-free.

## K12 — Bounded outbox recovery

The expired-lease recovery statement now selects at most the caller's validated batch limit and uses `FOR UPDATE SKIP LOCKED` before updating those rows. It no longer performs an unlimited blocking update ahead of the nonblocking pending-event claim. Both recovery and claiming have deterministic secondary ID ordering. The existing migration 028 processing-lease index supports recovery; no new schema migration is required for this change.

Completion fencing by attempt number and processing timestamp is unchanged. Recovery does not assert that a previously published event had no downstream effect: downstream delivery must remain idempotent. These are row-lock and batch bounds, not a promise of zero waiting on table locks or other infrastructure failures.

## K13 — Fail closed on uncertain in-memory writes

The arbitrary callbacks accepted by the development payment adapter do not share a transaction. A callback can change state and then fail, and an inverse callback can fail too. Reusing a compensated journal key or inventing a new payment/key is not a sound recovery mechanism.

The adapter now reserves the original recording intent and payment ID before its first side effect and places an obligation-scoped hold on incomplete operations. It records the current operation stage. On error or panic, the hold and original identity remain. Automatic inverse journal/callback compensation has been removed. Further recording, reversal, balance reconstruction and ordinary payment reads for that obligation return `ErrReconciliationRequired` rather than pretending the partial state is complete. A different idempotency key cannot bypass the obligation hold; reusing a failed key for a different intent is rejected. Successful operations publish their payment result and clear the temporary hold only after all required steps complete.

Nonempty allocation callback output must identify unique schedule items and sum exactly to the payment. Reversal checks that required inverse dependencies exist before starting. Successful recording and reversal remain available; the adapter has not been replaced with a blanket refusal.

**Important limitation:** this is fail-closed containment, not a retrofit of atomic storage into callbacks and not automatic repair of partial state. No generic clear/retry API is provided. Preserve the operation/journal/callback evidence and reconcile the affected development state explicitly. Restarting or replacing the adapter is not a financial recovery procedure. Other stores accessed outside the payment boundary are not made transactional by this hold. Real money and restart-safe records must use the existing PostgreSQL payment implementation, which remains selected by the database-backed runtime.

## K09 — Independent cleanup lifetime

`ledger.PostgresStore.post`, the five transaction-owning payment methods (`RecordContext`, `ReverseContext`, `GetContext`, `RebuildContext`, `ReadContext`) and `outbox.Store.Claim` now defer the existing shared `txcleanup.Finish` helper with their named return error. Cleanup is bounded independently of request cancellation, preserves rollback failures and rethrows original panics. Caller-owned `RecordTx` and journal transactions do not take over commit/rollback ownership. No transaction or uncertain-commit retry was added.

## D02-01 — Accounting dates are explicit intent

A nonzero requested effective date must match the recorded date after UTC/microsecond normalization. A zero date means the original recorded date is reused on replay; a default is assigned only for a genuinely new journal. Both shared ledger implementations now apply that rule, and the durable writer preserves whether the caller originally omitted the date before filling its insert value. Existing immutable journals and their dates are not rewritten. The payment-specific replay validator already enforces its supplied date at the same precision.

Callers retrying an explicit-date operation must retain that original date rather than supply a new `time.Now()` value. This is a deliberate conflict rule, not silent date correction.

## Verification and scope boundaries

The six modified originals were reconstructed from the previously retrieved complete source and verified byte-for-byte against their Git blob SHAs before editing. Changed Go source was formatted with `gofmt` and parsed with the standard-library Go parser. The source diff, preserved method signatures, cleanup ownership, lock sequence and money/replay semantics were reviewed. No application or financial operation was executed during these checks.

No tests were written or run. No test files or workflows were modified. Full Go compilation and database execution remain unverified: this environment has Go 1.23.2, while the repository requires 1.26.8, and its complete source/dependency tree is not available locally. Local network access could not retrieve the repository/dependencies. Syntax parsing is not semantic compilation or operational verification.

Batch 2 adds no migration. The combined repair branch still requires Batch 1 migration 204 and its documented rollout prerequisites. Do not mix old request-first order binaries with the new payment/order binaries and assume the repaired lock order is globally enforced. Coordinate the application/worker rollout. No merge, deployment, database mutation, or production request is part of this change.
