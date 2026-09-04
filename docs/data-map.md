# Kredit data map

| Data class | Examples | Primary store | Access notes |
| --- | --- | --- | --- |
| identity | phone, email, verification state | PostgreSQL | masked outside authorised flows |
| business/KYB | legal name, registration, address, industry | PostgreSQL | tenant and compliance scoped |
| authority/consent | representative, acceptance, consent version | PostgreSQL + immutable evidence | append-only history |
| financial | principal, kobo amounts, payments, ledger | PostgreSQL | ledger-only financial writes |
| mandate/provider | provider reference, status, capability | PostgreSQL | no raw credentials or PINs |
| documents | invoices, evidence, generated summaries | S3-compatible storage | signed access, retention class, scan state |
| operations | audit, support cases, jobs, webhooks | PostgreSQL | case-bound sensitive access |
| buyer identity | person, business, representative, authority state | PostgreSQL | separate facts; tenant/relationship scoped |
| verification | provider reference, level, state, safe result | PostgreSQL | no raw BVN, biometrics, credentials, or OTP |
| invitation | hashed public token, target hash, expiry, status | PostgreSQL | raw token is transient and never logged |
| consent | consent type/version/timestamp/evidence hash | PostgreSQL | append-only history |

The authoritative per-field register is `docs/compliance/data-inventory.tsv`;
its catalog check currently covers all 943 persisted production fields.
Retention and lawful-basis values remain marked pending until legal/compliance
approval, which keeps the production release gate fail-closed.

## Mono Sweep extension

`provider_customer_bindings` retains the buyer/business provider reference,
consent version and creation time. Customer BVN is transient during registration;
registration does not confer verification status. Existing person/business/
authority verification cases remain separate from profiles and retain safe
provider evidence. Sweep adds no credit score or new credit-history provider.

Mandate metadata holds authorization URL, approved ceiling, dates and
capabilities; supplier credit views omit the URL. Collection reservations link
to mandates; attempts retain retry time; immutable collection events retain
amount, actor and correlation reference. Webhook jobs/inbox retain an allowlisted
notice plus payload hash, not raw bank/identity fields.

Verification safe results allow separate `bvn_status`, `nin_status`,
`cac_status`, `bank_account_status`, `directors_status`, `shareholders_status`
and `credit_history_status` facts, constrained to status values. The provider
reference and case timestamps identify supporting evidence; identity consents
retain the buyer consent basis. Unrecognized provider result fields are
discarded before persistence. Credit-history checks remain inactive until an
approved provider supplies evidence.

## Financial delivery and review evidence

Migrations 055–060 add authenticated notification delivery receipts and financial review cases/events. These contain restricted delivery references, monetary differences, reviewer identities and investigation notes. Runtime workers and authorized platform operations can access them; they are not tenant-facing analytics. Receipt and review events are append-only. The data inventory records financial/legal-hold treatment and leaves the retention period and legal basis pending the applicable approvals. No provider secrets or destination contact data are added to these tables.

## Business policies and offer fee terms

`app.business_policy_defaults` retains the initial deployment policy. `app.business_policy_changes` retains complete proposed values, their base revision, proposer, effective date and decision state; `app.business_policy_events` is immutable actor/reason history. Platform administrators manage these records through the protected policy workflow; workers read effective settings. Retention follows financial/audit holds pending the approved retention register.

`app.credit_requests.fee_terms` and `app.drawdowns.fee_terms` preserve the quoted policy revision and rates. These are immutable offer terms, copied into agreement evidence and used for financial calculations. Global fee policy changes never rewrite these records.


## Admin approvals, buyer date consent and review ownership

Migrations 064–066 add immutable financial proposals and decision events (`admin_change_requests`, `admin_change_events`), review assignments and immutable assignment history. Proposals retain exact previous balances/schedule and intended changes, proposer/approver identities, expiry and buyer decision identity. The buyer API exposes only their own amendment dates and outcomes. Admin reads are filtered by active specialist roles; worker reads are read-only. Named-actor/impact helpers expose bounded administrator display names and aggregate/detail projections only through authorized endpoints.

The change-history and review-queue views retain no separate copies of sensitive source records. CSV exports are role scoped, paginated, audited and neutralize spreadsheet formulas. Approval evidence follows financial/audit retention holds; numeric retention periods and lawful bases still need the applicable approved register. The inventory includes 1,021 base-table fields after migration 066.

## Cross-supplier trade history

Trade history (README section 8.9) is the one flow that discloses one business's
payment behaviour to a different business, and several of its fields are adverse
inferences rather than neutral facts. It is assessed separately in
`docs/compliance/dpia-trade-history-sharing.md`, which records the lawful-basis
and minimisation decisions still outstanding.

Consent is per buyer/supplier relationship, versioned and evidenced in
`app.relationship_consents`, and is only recordable where an invited or active
trade relationship already exists. Until the assessment's decisions are signed,
the disclosed field set is not treated as settled and sharing is not enabled for
a real buyer.
