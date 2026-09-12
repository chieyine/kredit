# Provider contract review — 12 September 2026

**Result: not ready for live payments or production email.** This compares the supplied code with current official documentation. No provider requests that create customers, mandates, messages or payments were made. Configuration presence and synthetic fixtures are not provider certification. Finding IDs link to FINDINGS.md.

## Mono Sweep and direct debit

| Area | Current implementation | Assessment |
| --- | --- | --- |
| Hosted mandate | `mono.go:140` posts `/v2/payments/initiate`, with recurring-debit, mandate, sweep, variable, customer identity, dates and a stable reference | Matches the documented hosted variable-mandate shape. Its amount is a lifetime ceiling, not a per-payment entitlement. [Initiation reference](https://docs.mono.co/api/direct-debit/mandate/initiate-mandate-authorisation) |
| Authorization link | `hosted_url.go` restricts the destination to the expected HTTPS Mono host | Useful redirect protection. A returned link is pending authorization, not permission to debit. The real customer journey remains unverified. [Sweep integration guide](https://docs.mono.co/docs/payments/direct-debit/mono-sweep/integration-guide) |
| Mandate read/cancel | GET `/v3/payments/mandates/{id}`; PATCH `.../{id}/cancel` | Methods and paths match the published references. Cancellation must be confirmed; an in-flight debit still needs reconciliation. [Read](https://docs.mono.co/api/direct-debit/mandate/retrieve-a-mandate), [cancel](https://docs.mono.co/api/direct-debit/mandate/cancel-mandate) |
| Debit | POST `.../{mandate}/debit`, amount in kobo, distinct reference | Basic request shape matches. F009: the code does not enforce the documented ₦200 minimum. Published maxima distinguish personal and corporate accounts. [Debit reference](https://docs.mono.co/api/direct-debit/account/debit-account) |
| Polling | GET `.../{mandate}/debit/{reference}` | The reference heading uses singular `debit`, but its example uses plural `debits`. Obtain Mono confirmation and a captured sandbox response; changing the path based on one conflicting example is unsafe. [Retrieve debit](https://docs.mono.co/api/direct-debit/account/retrieve-a-debit) |
| Partial Sweep | Sends `allow_partial_sweep`; parses collected/pending amounts; reconciles the aggregate result | Correct direction: do not post each individual attempt and then post the aggregate again. Partial Sweep requires account enablement and variable mandates. The toggle for an existing mandate is a distinct provider operation; a local flag does not alter a previously created permission. [Partial Sweep](https://docs.mono.co/docs/payments/direct-debit/mono-sweep/partial-sweep) |
| Failed debit retries | Every parsed result has `Retryable: false` | F008: zero-recovery failures cannot be retried. Introduce only documented, safely retryable terminal codes, with fresh debt/mandate/notice checks. Unknown outcomes must be reconciled first. The reference limits daily insufficient-funds and other-failure retries separately. [Debit reference](https://docs.mono.co/api/direct-debit/account/debit-account) |
| Webhooks | Shared-secret authentication, allowlist, safe payload, durable inbox and reconciliation | F026: the documented nested dispute/reversal events are rejected. Preserve authentication and durable deduplication while adding them. Some official examples differ in envelope shape; support evidenced formats rather than weakening validation. [Webhook events, updated 24 August](https://docs.mono.co/docs/payments/direct-debit/webhook-events) |
| Unknown creation | Provider call occurs before a local mandate commits | F010: no durable mandate-creation recovery. Customer-registration recovery does not cover this separate operation. |
| Debt and ledger | Tenant-aware stores exist, but production collection interfaces lose that context | F021 prevents eligibility reads; F007 prevents recognizing an external success. These are application wiring defects, independent of whether Mono returns a correct response. |
| Seller settlement | Settlement setup accepts references; no production verification completes it | F018/F019: the platform does not establish a working verified seller destination and billing arrangement. Mono settlement to a contracted account is not proof of payout to each Kredit seller. |

Mono documents next-business-day settlement at 19:00 WAT; weekend transactions settle on Monday under that schedule. Do not confuse debit success, seller receipt and settlement. Record the contracted destination, fee treatment and any separately enabled instant-settlement feature. [Settlement documentation](https://docs.mono.co/docs/payments/settlements).

Before enabling Sweep, retain written evidence for the actual Mono account: live access, multi-account/partial capability, customer/mandate consent, covered banks, fee funding, beneficiary routing, callback URL and support escalation. Documentation describes a product; it does not establish Kredit's account entitlements. Use `https://kredit.ng` for customer returns and the intended `api.kredit.ng` endpoint for provider callbacks, consistent with the final deployment.

The CBN direct-debit framework includes mandate evidence, advance notification, failed-debit notification and record retention. Map the agreed customer notice period and retry rules to the actual mandate and provider contract; do not assume a generic reminder automatically satisfies them. This audit does not certify legal applicability or approval. [CBN direct-debit regulation](https://www.cbn.gov.ng/out/2018/bpsd/regulation%20for%20direct%20debit%20scheme%20in%20nigeria%202018%20(revised).pdf).

## Sendly email

F001 is a confirmed protocol mismatch. The current generic connector cannot be pointed directly at Sendly.

Use POST `https://api.sendlyai.com/v1/messages`, bearer authentication and a stable `Idempotency-Key`. Translate the recipient into `to` and provide email subject/content and the approved sender. Correlate the returned `id`; inspect `accepted` and `skipped`. HTTP 202 with `queued` is not delivery, and a suppressed recipient must not trigger a new-key retry.

Authenticate raw webhook bytes using `sendly-signature`, `sendly-timestamp` and the endpoint secret over `{timestamp}.{body}`; enforce the documented replay window. Persist event/message correlation before acknowledging successful handling. Configure and verify the `kredit.ng` sending domain. The reviewed documentation describes SMS and WhatsApp as forthcoming, so Sendly email does not supply those channels. [Sendly official documentation](https://developer.sendlyai.com/).

## Other integrations

| Service | What the repository supplies | Required launch decision/evidence |
| --- | --- | --- |
| Identity / business / authority checks | Kredit-specific HTTP connector and development implementations | Select the actual contracted service and implement/verify its adapter. Pending verification must survive restart and retries (F020). |
| Alternate collections and mandates | Generic collection/mandate connector contracts | Name the deployed translator or implement the chosen provider. Similar direct-debit products are not interchangeable integrations (F005). |
| SMS and WhatsApp | Generic outbound notification connector; separate inbound WhatsApp contract | Demonstrate the actual vendor adapter, sender/template approval, callback verification and delivery correlation. OTP readiness must match working channels. |
| Documents | S3 SDK storage plus a Kredit-specific scanner endpoint | Correct the unsupported R2 encryption request header (F038); connect a real scanner. A configured scanner URL is not malware-scanning evidence. [R2 compatibility](https://developers.cloudflare.com/r2/api/s3/api/) |
| Observability | OpenTelemetry exporter and monitoring definitions | Supply a reachable production collector and real alert destination; the example localhost default is insufficient (F002). |

Fincra's official documentation was reviewed as a comparison, not an implemented integration. Its direct-debit product does not establish portability of Mono mandates or authorize switching providers without the necessary new permission. [Fincra direct debit](https://docs.fincra.com/docs/direct-debit).

## Dependency research boundary

The locked versions reviewed are outside the affected ranges of these specific official advisories:

| Dependency | Locked | Advisory / fixed version |
| --- | --- | --- |
| pgx | 5.10.0 | [SQL placeholder parsing](https://github.com/jackc/pgx/security/advisories/GHSA-j88v-2chj-qfwx), fixed 5.9.2 |
| SvelteKit | 2.70.2 | [Body-size enforcement](https://github.com/sveltejs/kit/security/advisories/GHSA-2crg-3p73-43xp), fixed 2.57.1 |
| SvelteKit | 2.70.2 | [Large remote forms](https://github.com/sveltejs/kit/security/advisories/GHSA-wqjv-9729-c5q2) and [form prototype pollution](https://github.com/sveltejs/kit/security/advisories/GHSA-866w-xmhq-wj7x), fixed 2.69.1 |
| SvelteKit | 2.70.2 | [Regular-expression denial of service](https://github.com/sveltejs/kit/security/advisories/GHSA-29g2-3rmr-qm68), fixed 2.70.2 |
| Vercel adapter | 6.3.4 | [Cache poisoning](https://github.com/sveltejs/kit/security/advisories/GHSA-9pq4-5hcf-288c), fixed 6.3.2 |

This is targeted advisory research, not a complete dependency vulnerability scan or third-party source audit. No dependency scanner, installation or update ran.
