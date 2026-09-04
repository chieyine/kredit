# Mono Sweep sandbox integration

Status: backend implementation and local contract/database tests; **not Mono sandbox certified**. No Mono credentials or Sweep access were available for this implementation. Production is explicitly rejected while this gate remains open.

## Integration boundary

`internal/providers/mono` owns Mono HTTP paths, request/response vocabulary, customer registration, mandate authorization, cancellation, and webhook authentication. Kredit's existing credit, collection, payment, dispute and ledger packages remain provider-neutral. Amounts are integer kobo.

The buyer may accept the exact trade-credit agreement before authorizing a bank mandate. Acceptance remains immutable and separate. Goods release and collections require an active mandate. `ready_to_debit=true` plus provider approval establishes activation; a browser redirect does not.

The first mandate can have a buyer-selected ceiling above a single obligation via `amount_ceiling_kobo` on the existing authorization endpoint. The buyer confirms that ceiling in Mono's hosted flow. A suitable variable mandate is reused for the same supplier and business, within its dates and remaining capacity. Reservations serialize both obligation balance and shared mandate capacity.

## Setup

1. Use a separate sandbox database and a Mono payments app with Sweep enabled. Partial Sweep requires separate provider access.
2. Set `COLLECTION_PROVIDER=mono-sweep`, `MONO_SECRET_KEY` to a sandbox `test_sk_` key, `MONO_WEBHOOK_SECRET`, and `MONO_REDIRECT_URL` in secret/environment management. Do not put credentials in the repository.
3. Apply migrations through 054. Migration 052 adds provider customer bindings, mandate metadata/supplier scope, shared capacity guards, manual-payment reservation guards, immutable collection events, and retry timestamps. Migration 053 preserves valid payment reversals and checks currency; 054 rechecks active obligation, due schedule, disputes, claims and holds under the reservation lock. Apply `infra/postgres/roles.sql` after migrations (including River) to install the restricted application/worker grants, including ledger, job queue and reconciliation lookups.
4. Set `MONO_SWEEP_ENABLED=true` outside production. `PARTIAL_SWEEP_ENABLED`, `AUTOMATIC_COLLECTION_ENABLED`, and `AUTOMATIC_RETRY_ENABLED` default to false; enable each intentionally for the sandbox scenario.
5. Run both API and worker. Register the HTTPS webhook URL `/api/v1/webhooks/mono` (alias `/webhooks/mono`) in Mono with the same webhook secret.
6. Register the buyer's provider customer using `POST /api/v1/buyer/businesses/{businessID}/repayment-customer`. Requires ownership, recent MFA, CSRF and an idempotency key. Supply the documented customer details, BVN and consent version. The BVN is transient and is not persisted, returned or logged. Customer registration does not assert successful identity verification.
7. Create/review/accept the supplier's credit request, then request its mandate using the existing buyer mandate endpoint. Open `mandate.authorization_url` as the buyer, complete provider authorization, and wait for server confirmation.
8. Once the obligation is active and due, invoke collection explicitly or enable the automatic worker. The engine always recalculates the outstanding collectible amount.

## Financial controls and recovery

- A committed attempt/reservation and reference precede the provider HTTP call. Worker interruption never authorizes a second submission. Reconciliation uses the original mandate and reference, including when no submission response arrived.
- Processing/unknown outcomes retain their reservations. Unknown status values never become a failed debit. Database lock contention returns a conflict instead of exhausting the connection pool.
- Dispute, compliance-hold and payment-claim lookup failures block collection.
- A recognized manual payment before collection reduces its amount. A manual payment conflicting with an unresolved debit is held back for reconciliation; it does not release the debit's reservation.
- Partial Sweep's individual debit notifications are retained as safe receipts, but do not independently post money. Server lookup of the aggregate result owns the payment effect. `collected_amount` is used for partial results. Five duplicate final events have the same effect as one.
- Payment idempotency is tied to the attempt, not the delivery/event ID. This also recovers a crash after the payment commits but before the collection snapshot commits.
- Retryable failures and partial results have a next-eligible time (24 hours) and a bounded per-chain attempt count. New request keys cannot bypass the retry workflow. Unclassified Mono failure codes are terminal until an approved retry classification is added; timeouts are always reconciled.
- `GRACE` uses accepted schedule/grace times. `IMMEDIATE` cannot specify a positive grace period. `STANDARD` requires separate primary-only and later recovery stages; Mono Sweep does not expose that staging, so this combination is rejected rather than silently changing the policy.
- Authenticated cancellation/expiry events immediately block further collection. Old ready/lookup events cannot reactivate a cancelled/expired mandate. A cancelled Mono mandate requires fresh buyer authorization.
- Collection transitions enter immutable `app.collection_events`. Internal notification requests enter the transactional outbox; the outbox dispatcher now queues durable notifications with actual buyer destinations; the notification worker delivers them. No live message delivery was tested in this review.
- The worker discovers due obligations, eligible retries and unresolved debits without frontend activity. Eligibility reads fresh persisted state after restart and after payments in other processes; completed reconciliation jobs do not suppress subsequent checks. Pending/active Mono mandates are also reconciled periodically. Turning off automatic collection or MONO_SWEEP_ENABLED does not disable reconciliation when the sandbox provider credentials remain configured. Discovery pages through all eligible records, and completed maintenance/reconciliation jobs can run again.

## Data and operational boundaries

Only provider customer references and consent versions are retained from customer registration. Webhook jobs/inbox retain an allowlisted event notice and SHA-256 hash, not raw BVNs, account numbers, account names, narration or credentials. The current authentication mechanism is constant-time comparison of `mono-webhook-secret`; it is not a provider HMAC signature. Internal normalized collection events use a separate adapter-side authentication step.

Provider references and buyer authorization URLs are restricted data. Use encrypted database volumes/backups, TLS, tenant access controls and secret rotation as required by the existing production security contract. Supplier credit views omit authorization URLs. Do not export these fields into analytics or logs. Raw bank-account inventories are left with Mono because Kredit does not need them to authorize or reconcile a collection.

A dead-lettered webhook, provider/local financial mismatch, or uncertain customer registration requires operator review. Do not create replacement debits to clear a timeout. Do not enable production until actual provider sandbox evidence, approvals, access review and environment controls pass.

## Current official sources (checked 2 September 2026)

- [Sweep integration](https://docs.mono.co/docs/payments/direct-debit/mono-sweep/integration-guide)
- [Variable mandate setup and lifetime ceiling](https://docs.mono.co/docs/payments/direct-debit/mandate-setup-variable)
- [Mandate initiation](https://docs.mono.co/api/direct-debit/mandate/initiate-mandate-authorisation)
- [Retrieve mandate](https://docs.mono.co/api/direct-debit/mandate/retrieve-a-mandate)
- [Cancel mandate](https://docs.mono.co/api/direct-debit/mandate/cancel-mandate)
- [Debit account](https://docs.mono.co/api/direct-debit/account/debit-account)
- [Retrieve debit](https://docs.mono.co/api/direct-debit/account/retrieve-a-debit)
- [Partial Sweep](https://docs.mono.co/docs/payments/direct-debit/mono-sweep/partial-sweep)
- [Direct-debit webhook events](https://docs.mono.co/docs/payments/direct-debit/webhook-events)
- [Webhook authentication](https://docs.mono.co/docs/webhooks)
- [Customer registration](https://docs.mono.co/docs/payments/direct-debit/integration-guide-create-customers)

The retrieve-debit reference currently disagrees with itself: the operation URL says `/debit/{reference}`, while the cURL example says `/debits/{reference}`. The adapter follows the operation URL. Confirm this and final partial-debit retrieval fields in the actual sandbox before certification. No alternate URL or undocumented fallback is attempted.

## Direct code review follow-up

See [the code review and verification record](../testing/code-review-2026-09-02.md). Payment-claim confirmation now commits the recognized payment and claim decision together. Uncertain debit reservations also block write-offs/dispute reductions that would consume reserved principal. Accepted schedules cannot be overwritten by the supplier. Caller-supplied second-approver identifiers are rejected; corrections reaching a cumulative NGN 10,000 per obligation remain unavailable until a real independent approval workflow exists.

The generic connector's normalized webhook authentication now covers **every** field, including provider collection ID, failure classification and retryability. Both sender and verifier must use the new format in the same release: HMAC-SHA256 over Go `encoding/json` serialization of `collections.Webhook` with `Signature` set to the empty string, in its declared field order (all fields included); transmit lowercase hexadecimal. The simulator uses the same signer. This does not change Mono's raw `mono-webhook-secret` callback protocol.

Local tests establish the notice gate with synthetic delivery receipts. They do not establish real prior-debit notification delivery, provider fees/net margin, settlement/refund behaviour, regulatory status, or buyer authorization. These remain launch evidence requirements; never add undocumented buyer charges to cover provider costs.

## Durable notices and reconciliation (migrations 055–060)

`COLLECTION_NOTICE_MIN_HOURS` defaults to 24 and must be between 1 and 720 for real collection/Sweep enablement. Confirm the value against approved terms and provider requirements. A first collection may be delayed until delivery evidence and this waiting period exist. The guard covers fresh debits and retries; changing a schedule's collection date or principal requires new notice evidence. The worker queues notices for active unpaid instalments within the next 31 days when a real collection provider or Mono Sweep is enabled.

An approved internal notification connector must POST a delivery receipt to `/api/v1/webhooks/notifications/{channel}` (`email`, `sms` or `whatsapp`). Send `event_id` (unique connector receipt ID), `notification_event_id` (the send payload's `event_id`), `message_id` (the send response's `message_id`) and `delivered_at` (RFC3339). Authenticate the exact request bytes with lowercase hexadecimal HMAC-SHA256 in `X-Notification-Signature`, using that channel's configured connector token. A sent-message lookup race returns 409 and must be retried with the same receipt. Changed evidence under an existing receipt ID is rejected. A successful send response alone must never be represented as delivery evidence.

Operators use `/admin/reconciliation` to claim and resolve detected differences. No case action changes a balance. A discrepancy must first disappear from authoritative records before its owner can resolve the case. Current checks compare available settlement facts; they do not establish supplier receipt without actual settlement evidence.

For Prometheus, configure a separate random `METRICS_SCRAPE_TOKEN` of at least 32 characters and send it as a bearer credential to `/api/v1/ops/metrics/prometheus`. Keep it in secret storage. It provides no case-management access. Without it, the existing authenticated platform-role/MFA path applies. Never label these current-state counts as cumulative counters; alerts use the gauge values directly.

See [the completion record](../testing/code-completion-2026-09-02.md) for tests and remaining provider/approval boundaries.
