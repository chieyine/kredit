# Product analytics event catalog

Wave 6 events are first-party, append-only projections of authoritative domain
changes. They are not a financial source of truth and cannot change an
agreement, balance, payment, mandate, dispute, trade-line exposure or provider
state.

## Envelope

Every event contains a dotted `name`, `schema_version`, SHA-256
`subject_id_hash`, explicit `purpose`, server `occurred_at` and `recorded_at`,
deterministic `deduplication_key`, `source`, optional hashed organisation ID,
and a bounded metadata object. The current schema version is `1`.

Allowed purposes are `product_improvement`, `pilot_measurement`, and
`operations_reliability`. Metadata is capped at 16 fields in application code
and 2 KiB in PostgreSQL. Phone, email, BVN, NIN, names, addresses, invoice or
goods text, bank data, provider tokens, notes, reasons, statements and message
bodies are forbidden. Financial amounts are calculated from access-controlled
domain tables, not copied into generic event metadata.

## Events

| Journey | Version 1 events | Purpose | Allowed metadata |
| --- | --- | --- | --- |
| Supplier readiness | `onboarding.started`, `onboarding.step_completed`, `onboarding.ready` | improvement / pilot | readiness state only |
| Customer onboarding | `customer.invited`, `customer.invitation_accepted`, `customer.verified` | improvement / pilot | channel, subject type, verification level |
| One-time credit | `credit.drafted`, `credit.sent`, `credit.viewed`, `credit.accepted`, `credit.declined` | pilot | state only |
| Mandate | `mandate.started`, `mandate.activated`, `mandate.failed`, `mandate.cancelled` | reliability | status only |
| Fulfilment | `goods.released`, `receipt.confirmed`, `receipt.issue_reported`, `obligation.activated` | pilot | state or currency only |
| Payment | `payment_link.created`, `payment.claimed`, `payment.claim_confirmed`, `payment.confirmed`, `payment.due`, `payment.late`, `payment.collected`, `collection.submitted`, `collection.failed`, `collection.recovered` | improvement / pilot / reliability | source type, state, attempt number, surface |
| Recurring credit | `trade_line.created`, `trade_line.activated`, `trade_line.expired`, `trade_line.drawdown_reserved`, `trade_line.drawdown_confirmed`, `trade_line.drawdown_released`, `trade_line.drawdown_activated`, `trade_line.drawdown_expired` | pilot | state only |
| Disputes and support | `dispute.opened`, `dispute.resolved`, `operations.intervention` | pilot / reliability | state or subject type only |
| Retention | `credit.repeat_sale`, `supplier.retained` | pilot | no metadata |
| Product clarity | `feedback.clarity_submitted` | improvement | seller or buyer area, overview screen, and yes/partly/no answer only |

Database triggers emit reconstructable lifecycle facts in the same transaction
as their source writes. Command replay, webhook replay and migration backfill
use the same unique deduplication key. A payment-link event is emitted after a
link is successfully issued; the global command idempotency boundary prevents
request replay from invoking that producer twice.

Migration 051 aligns the stored names with the Wave 0 vocabulary. It preserves
event IDs and deduplication keys for compatibility and adds coverage for the
authoritative `payment_mandates` store, collection submission, repeat sales and
supplier retention.

The clarity event is submitted by an authenticated seller or customer from
their overview page. It deliberately excludes comments and contact details.
Seller answers require an active membership of the selected organisation. The
normal API idempotency boundary prevents retry duplicates, while a hashed
monthly person/business/page key prevents one person from inflating the result
by answering from several devices.

## Retention and consent boundary

These operational and pilot-measurement events are essential first-party
telemetry. Optional marketing/product-update consent never changes essential
transaction processing. The approved retention period and lawful basis remain
an external Data Protection Lead gate in the environment retention register;
production remains fail-closed until approved. Deletion or restriction follows
the privacy-request workflow and any applicable financial or legal hold.
