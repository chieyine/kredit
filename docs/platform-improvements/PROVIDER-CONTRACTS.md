# Provider replacement

Kredit owns sale terms, consent, balances, permissions, schedules and audit records. Provider adapters own vendor requests and response interpretation. Replacing a vendor must not transfer its bank mandates, verification references or uncertain requests to another vendor.

## Messaging connector v1

An alternate provider can be installed as a Go adapter in `internal/notifications/adapters.go`, or operated behind an HTTPS connector implementing this contract. A provider's own send URL is not automatically a Kredit connector.

Select `adapter: "connector"` in the encrypted admin channel connection or `NOTIFICATION_EMAIL_ADAPTER`, `NOTIFICATION_SMS_ADAPTER`, or `NOTIFICATION_WHATSAPP_ADAPTER` in deployment configuration. The endpoint and bearer token must be configured together.

POST the configured endpoint with bearer authentication, JSON and `Idempotency-Key: <event_id>:<channel>`:

```json
{
  "channel": "sms",
  "destination": "+2348012345678",
  "event_id": "stable-event-reference",
  "template": "PaymentDueSoon",
  "template_version": "v1",
  "body": "The complete message prepared by Kredit",
  "secure_link": ""
}
```

Success is HTTP 2xx with `{"message_id":"provider-reference"}`. This means accepted, not delivered. The connector must atomically deduplicate the key, reject changed payloads, and return the original ID on an identical retry. It must reconcile an uncertain vendor response before sending again. Never implement fallback by blindly trying another vendor after a timeout.

GET the same endpoint with `?message_id=<encoded-reference>` and bearer authentication:

```json
{
  "id": "provider-reference",
  "channel": "sms",
  "to": ["+2348012345678"],
  "status": "delivered",
  "deliveredAt": "2026-09-12T10:00:00Z"
}
```

Use `sent` for pending delivery, `delivered` only for authenticated evidence, and `failed`, `bounced` or `complained` for definitive negative outcomes. Never report delivery just because the send request or callback succeeded. Kredit matches the original event, provider ID and recipient before creating delivery evidence.

Kredit saves an immutable encrypted connection per event before sending. Changing the selected provider affects new events. Retries and delivery lookups use the saved connection. Retain the original provider account and encryption root while its requests remain unresolved. Root-key rotation must include these route ciphertexts; changing the key alone is insufficient.

## Financial and identity adapters

Existing interfaces are `identity.IdentityProvider`, `mandates.Provider`, and `collections.Provider`, with separate optional capability, reference lookup, cancellation and health interfaces. Additional providers must implement these contracts against their own published APIs. Configuration alone is not proof of a capability or approval to debit.

Collection attempts already record their provider identity. Reconciliation, callbacks, cancellation and payment attribution now select that identity rather than the currently selected provider. Retained collection adapters can be registered at startup with `RegisterRetainedProvider`; do not reuse one provider name for a different provider account. Missing retained adapters hold the original work for review. A bank authorization issued by another provider cannot be passed to the current provider to start a new debit.

Retained financial connectors can be configured using `COLLECTION_RETAINED_PROVIDERS`, a JSON array of objects with `name`, `endpoint`, `token` and `webhook_secret`. Keep this value in the deployment secret store, never in source control. Names must match historical attempts and be different from the active provider. These connections are used only for existing attempts.

Generic retained collection accounts can now be managed in super admin, with hidden credentials and protections against reusing them at a changed address. Consolidated invoice billing and recorded bank receipts/refunds are implemented. Remaining work includes native account retention, identity/mandate routing across multiple configured accounts, payment-level settlement, split billing and authorised fee debits. These interfaces and protections do not establish production readiness by themselves.

## Mesaj documentation

The owner-supplied [public documentation](https://mesaj.cloud/docs.html#bulk-sms) specifies `/client/sms/send/bulk`, a `data` wrapper, `sender_id`, digits-only international recipient numbers, and a response containing `message_id`, `accepted` and `rejected`. This differs from the [Swagger contract](https://api.mesaj.cloud:25274/client-docs#/) for `/client/sms/send`. The unused legacy adapter was removed before launch. Mesaj setup uses only the bulk endpoint.

Bulk documentation does not publish authenticated status lookup or webhook signing. That adapter can send but cannot establish evidence needed by delivery-dependent collection timers. Another conforming connector can supply the complete capability without changing Kredit's business logic. Both documents still identify the API subdomain on port 25274 as the API host; HTTPS on the documentation host does not replace API-host certificate validation.

No live messages, provider mutations, tests, builds or migrations were run while preparing these changes.


## Seller bank registration connector v1

Configure the native provider name `mono-sweep`, or an alternate named HTTPS connector, through **Launch setup → Seller bank accounts**. This connection registers destinations; it does not attest to payment settlement.

An alternate connector implements:

- `GET <endpoint>/v1/banks` with bearer authentication, returning `{"banks":[{"code":"000014","name":"Bank name"}]}`. Bank codes must be unique digit strings, three to six characters.
- `POST <endpoint>/v1/destinations` with bearer authentication and `Idempotency-Key: <reference>`. Input is `reference`, `organization_id`, `bank_code`, and ten-digit `account_number`.
- A successful response echoes `reference` and `organization_id` and supplies `destination` with `provider_reference`, `bank_code`, `account_name`, and `account_last4`. Kredit checks these against the original request. The connector must validate the account with the actual provider, never echo an unverified account name supplied by a customer.

Kredit persists a keyed request fingerprint before the provider call, retains only masked account details, and holds unknown outcomes. An identical retry uses a recorded result. The super admin can release a held request only after the provider confirms non-creation; this permission is audited and does not itself make another request.

The super-admin ownership review requires current business verification, an unchanged profile version and matching saved provider evidence. This review is separate from payment reconciliation. A successful destination registration must never be reported as a seller having received funds.

Native Meta setup uses the channel adapter `meta`, an explicitly versioned Graph phone-number endpoint, app secret, subscription verification token and three approved templates (authentication, utility and marketing). Configure `/api/v1/webhooks/meta` as the callback. Preserve historical route credentials while their messages are unresolved.

### Phone lookup completion

Mono's public [OTP verification reference](https://docs.mono.co/api/phone/verify) was retrieved over verified HTTPS. It documents POST `/v3/lookup/phone/verify` with `reference` and `otp`. The response contains NIN identity information including first/last/middle names and phone number. This is sensitive identity data; the implemented native adapter minimises stored results, binds the original user, limits OTP attempts, and keeps consent and expiry evidence.

### Mono native lookup and settlement continuation

- Phone OTP uses `/v3/lookup/phone/initiate` and `/v3/lookup/phone/verify`; local sessions persist consent and submission state, not the phone/NIN/OTP payload. A mismatched verified name requires review. Representative authority is separately reviewed against current person/business evidence and a clean private document.
- CAC lookup uses `/v3/lookup/cac?search={registration}&exact=true`. Exactly one active, registration-approved, matching RC record is required; exact name checks are done locally because the provider's `exact` flag is prefix matching. Supplier KYB additionally requires reviewed evidence.
- Variable e-mandates can also be initiated through `/v2/payments/initiate`, with `type=recurring-debit`, `method=mandate`, `mandate_type=emandate`, `debit_type=variable`; the authorized amount is the lifetime ceiling, not a per-debit allowance. [Official variable mandate guide](https://docs.mono.co/docs/payments/direct-debit/mandate-setup-variable), read 2026-09-12. The separate fee-debit workflow is implemented.
- Full debit seller distributions use `split.type=fixed`, `fee_bearer=business`, and original sub-account ID plus the exact kobo amount. Frozen collection destination records do not themselves prove a payout. Batch settlement callbacks are not assigned to individual payments by guesswork. [Official split guide](https://docs.mono.co/docs/payments/split-payments).

### Completed seller fee billing contract

Native seller fee setup uses Mono customer creation followed by `/v2/payments/initiate` with `mandate_type=emandate` and `debit_type=variable`. This is separate from buyer Sweep permissions. The amount is a lifetime ceiling. Business customer first/last fields contain the bank-registered business name, split into two parts; the submitted BVN belongs to a consenting shareholder. The application retains a keyed fingerprint, not the BVN. [Mono customer guide](https://docs.mono.co/docs/payments/direct-debit/integration-guide-create-customers).

Before approval and each debit, the original account's GET `/v3/payments/mandates/{id}` must match customer, reference, ceiling and calendar validity, and report approved/ready status. Provider date boundaries are compared as Lagos calendar dates; actual provider timestamps still gate readiness. Fee debits use `fee_bearer=business` so provider transaction costs are not added to the seller's requested debit. The supported Kredit fee debit range is ₦200–₦25 million. Mono documents larger limits for some corporate accounts; those are not inferred from a customer's business label. [Current debit guide](https://docs.mono.co/docs/payments/direct-debit/debit-an-account), [debit reference](https://docs.mono.co/api/direct-debit/account/debit-account).

Partial Sweep is separately approved by Mono and can be enabled at mandate initiation with `allow_partial_sweep=true`. The documented existing-mandate toggle is PATCH `/v3/payments/mandates/{id}/partial-sweep/toggle`; changing Kredit's setting does not silently mutate old mandates. [Partial Sweep guide](https://docs.mono.co/docs/payments/direct-debit/mono-sweep/partial-sweep).

A replacement fee connector uses the saved financial account's HTTPS endpoint and bearer token. It must implement these exact Kredit adapter endpoints:

- POST `/fee-customers`: business_name, email, phone, address and transient bvn; returns id. GET `/fee-customers/{id}` returns matching id and transient bvn for reviewed reconciliation.
- POST `/fee-authorizations`: reference, customer, ceiling_kobo, starts_at and ends_at; returns matching values plus id and an HTTPS authorization_url. GET `/fee-authorizations/{id}` returns those identity/limit fields and ready.
- POST `/fee-debits`: mandate, reference and amount_kobo. GET `/fee-debits/{reference}` returns mandate, reference, requested_amount_kobo, state and amount_kobo actually collected. Terminal supported states are succeeded (full exact amount) or failed (no debit); uncertain/processing results remain pending.

Connector implementations must preserve original references and not retry a mutation after an ambiguous response. A connector is an integration boundary, not a claim that an arbitrary vendor already implements these routes. Native Mono is implemented directly.
