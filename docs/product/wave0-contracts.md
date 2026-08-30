# Wave 0 shared contract registry

Contract version: `wave0-v1`

Locked: 29 August 2026

This registry fixes the vocabulary and minimum contracts for implementation
waves 1–3. Existing domain constants remain authoritative where they already
exist. New code must use these names exactly or amend this document and the API
contract in the same reviewed change.

## Naming rules

- Database/domain states use uppercase snake case.
- Audit and analytics names use lowercase dot-separated past-tense events.
- Notification event names use UpperCamelCase.
- Problem codes use lowercase snake case and describe the stable problem, not
  an implementation detail.
- Permission names use `resource:verb`.
- State changes occur in domain services, never directly in handlers.
- User-facing text comes from `docs/product/interface-copy.md`; internal state
  names must not be displayed verbatim.

## Wave 1 contract — trade-line drawdowns

### States

| Aggregate | States |
| --- | --- |
| drawdown | `PENDING_BUYER_CONFIRMATION`, `BUYER_CONFIRMED`, `GOODS_RELEASED`, `RECEIPT_ISSUE_REPORTED`, `ACTIVATED`, `CANCELLED`, `EXPIRED` |
| drawdown reservation | `PENDING`, `CONFIRMED`, `RELEASED_TO_SUPPLIER`, `CONVERTED`, `RELEASED`, `EXPIRED` |
| drawdown agreement | `CREATED`, `ACCEPTED`, `SUPERSEDED`, `VOIDED` |
| receipt evidence | `NO_ISSUE`, `ISSUE_REPORTED` |

`ACTIVATED` requires immutable accepted terms, valid release evidence,
`NO_ISSUE` receipt evidence, one internally created obligation, one schedule,
balanced activation postings, converted exposure, and committed outbox events.

### Planned API commands

| Command | Permission/actor | Idempotency | Success |
| --- | --- | --- | --- |
| reserve drawdown | `credit:create` | required | `201` |
| confirm exact terms | authenticated line buyer | required | `200` |
| release goods | `goods:release` plus recent MFA | required | `200` |
| confirm receipt/no issue | authenticated line buyer | required | `200` |
| report issue at receipt | authenticated line buyer | required | `200` |
| cancel pending drawdown | supplier `credit:create` or line buyer under defined state rules | required | `200` |

The current activation request accepting `obligation_id` is deprecated and must
be removed in Wave 1. The obligation is created inside the receipt/activation
transaction.

### Problem codes

`drawdown_not_found`, `drawdown_state_conflict`, `drawdown_terms_changed`,
`drawdown_reservation_expired`, `drawdown_limit_exceeded`,
`drawdown_release_evidence_required`, `drawdown_receipt_required`,
`drawdown_receipt_issue_open`, `drawdown_activation_failed`,
`trade_line_inactive`, `mandate_inactive`, `mandate_ceiling_insufficient`,
`idempotency_key_required`, `idempotency_conflict`, `step_up_required`.

### Audit actions

`trade_line.drawdown_reserved`, `trade_line.drawdown_confirmed`,
`trade_line.drawdown_goods_released`, `trade_line.drawdown_receipt_confirmed`,
`trade_line.drawdown_receipt_issue_reported`,
`trade_line.drawdown_obligation_activated`, `trade_line.drawdown_cancelled`, and
`trade_line.drawdown_expired`.

### Notification events

`TradeLineDrawdownConfirmationRequired`, `TradeLineDrawdownConfirmed`,
`TradeLineDrawdownSafeToRelease`, `TradeLineDrawdownGoodsReleased`,
`TradeLineDrawdownReceiptRequired`, `TradeLineDrawdownReceiptIssueReported`,
`TradeLineDrawdownActivated`, `TradeLineDrawdownCancelled`, and
`TradeLineDrawdownExpired`.

## Wave 2 contract — supplier onboarding

### States

| Aggregate | States |
| --- | --- |
| onboarding profile | `DRAFT`, `IN_REVIEW`, `ACTION_REQUIRED`, `READY`, `SUSPENDED` |
| settlement destination | `DRAFT`, `PENDING_VERIFICATION`, `VERIFIED`, `REJECTED`, `EXPIRED`, `DISABLED` |
| billing setup | `NOT_CONFIGURED`, `PENDING`, `ACTIVE`, `PAST_DUE`, `SUSPENDED` |
| KYB readiness | `NOT_STARTED`, `PENDING_PROVIDER`, `UNDER_REVIEW`, `VERIFIED`, `ACTION_REQUIRED`, `EXPIRED` |

Readiness is server-derived and requires current evidence for organization
identity, verified contacts, KYB, settlement, billing, consents, and applicable
owner/finance MFA. Clients cannot submit `READY`.

### Planned APIs and permissions

| Resource | Read | Mutate | Step-up |
| --- | --- | --- | --- |
| onboarding summary | organization member | owner/administrator as step permits | sensitive steps |
| settlement destination | `financial:read` | `financial:manage` | always |
| billing setup | `financial:read` | owner or `financial:manage` | always |
| default credit policy | organization member | `organization:manage` or `financial:manage` | always |
| consent versions | owner | owner | acceptance action |
| security readiness | owner/administrator | individual user for own MFA | enrollment/changes |

### Problem codes

`onboarding_step_incomplete`, `organization_not_ready`,
`settlement_verification_pending`, `settlement_change_failed`,
`billing_setup_required`, `billing_setup_failed`, `kyb_action_required`,
`kyb_expired`, `consent_version_outdated`, and `mfa_readiness_required`.

### Audit actions

`onboarding.profile_updated`, `onboarding.submitted`,
`onboarding.action_required`, `onboarding.ready`, `onboarding.suspended`,
`settlement.destination_submitted`, `settlement.destination_verified`,
`settlement.destination_changed`, `billing.setup_changed`,
`credit_policy.default_changed`, `legal.supplier_terms_accepted`, and
`legal.privacy_notice_acknowledged`.

### Notification events

`SupplierOnboardingSubmitted`, `SupplierOnboardingActionRequired`,
`SupplierReady`, `SupplierReadinessSuspended`,
`SettlementDestinationChanged`, `BillingSetupChanged`, and
`VerificationExpiring`.

## Wave 3 contracts — user control and privacy

### Notification preferences

- Preference groups: `TRANSACTIONAL_REQUIRED`, `SECURITY_REQUIRED`,
  `PAYMENT_REMINDERS`, `PRODUCT_UPDATES`.
- Channels: `email`, `sms`, `whatsapp`; timezone defaults to
  `Africa/Lagos`.
- Required transactional/security messages cannot be opted out of, but channel
  availability and safe fallback may be configured.
- Permission: authenticated users manage only their own preferences.
- Problems: `notification_preference_invalid`,
  `notification_channel_unavailable`, `required_notification_cannot_disable`.
- Audit: `notification.preferences_updated`.
- Notification: `NotificationPreferencesChanged` is a security/routine receipt
  and does not recursively obey optional-message suppression.

### Account recovery

| Aggregate | States |
| --- | --- |
| recovery request | `PENDING_VERIFICATION`, `PENDING_REVIEW`, `COOLING_OFF`, `APPROVED`, `REJECTED`, `CANCELLED`, `EXPIRED`, `COMPLETED` |
| recovery code | `ACTIVE`, `USED`, `REVOKED`, `EXPIRED` |

- Public submission returns enumeration-safe responses.
- Phone possession alone can never move a request to `APPROVED`.
- Completion revokes existing sessions and old recovery codes.
- Platform permission: `accounts:recover`; approval requires compliance review,
  recent MFA, separation from the requester, and a structured reason.
- Problems: `recovery_request_invalid`, `recovery_verification_incomplete`,
  `recovery_review_required`, `recovery_cooling_off`,
  `recovery_code_invalid`, `recovery_conflict`.
- Audit: `account.recovery_requested`, `account.recovery_verified`,
  `account.recovery_reviewed`, `account.recovery_cancelled`,
  `account.recovery_completed`, `account.recovery_codes_regenerated`.
- Notifications: `AccountRecoveryRequested`, `AccountRecoveryCoolingOff`,
  `AccountRecoveryCancelled`, `AccountRecoveryCompleted`.

### Privacy rights

| Aggregate | States |
| --- | --- |
| privacy request | `RECEIVED`, `IDENTITY_CHECK`, `IN_REVIEW`, `CLARIFICATION_REQUIRED`, `APPROVED`, `PARTIALLY_APPROVED`, `REJECTED`, `IN_PROGRESS`, `COMPLETED`, `CANCELLED` |
| request type | `ACCESS`, `CORRECTION`, `DELETION`, `RESTRICTION`, `OBJECTION`, `CONSENT_WITHDRAWAL`, `PORTABILITY` |

- Authenticated users submit for themselves; assisted requests require
  identity-bound evidence.
- Platform permission: `privacy:review`; destructive completion requires dual
  control and cannot remove records under a valid financial/legal hold.
- Problems: `privacy_request_invalid`, `privacy_identity_check_required`,
  `privacy_request_conflict`, `privacy_legal_hold`,
  `privacy_export_unavailable`, `privacy_decision_invalid`.
- Audit: `privacy.request_received`, `privacy.identity_verified`,
  `privacy.clarification_requested`, `privacy.request_decided`,
  `privacy.restriction_applied`, `privacy.export_generated`,
  `privacy.request_completed`.
- Notifications: `PrivacyRequestReceived`, `PrivacyClarificationRequired`,
  `PrivacyRequestDecided`, `PrivacyExportReady`, `PrivacyRequestCompleted`.

## Planned analytics event vocabulary

These events are reserved now and instrumented in Wave 6. They contain opaque
or privacy-hashed subject identifiers and structured allow-listed metadata only.

`onboarding.started`, `onboarding.step_completed`, `onboarding.ready`,
`customer.invited`, `customer.verified`, `credit.drafted`, `credit.sent`,
`credit.viewed`, `credit.accepted`, `credit.declined`, `mandate.started`,
`mandate.activated`, `mandate.failed`, `mandate.cancelled`,
`goods.released`, `receipt.confirmed`, `receipt.issue_reported`,
`obligation.activated`, `payment_link.created`, `payment.claimed`,
`payment.confirmed`, `payment.due`, `payment.late`, `collection.submitted`,
`collection.failed`, `collection.recovered`, `trade_line.created`,
`trade_line.drawdown_reserved`, `trade_line.drawdown_confirmed`,
`trade_line.drawdown_released`, `trade_line.drawdown_activated`,
`trade_line.drawdown_expired`, `dispute.opened`, `dispute.resolved`, and
`credit.repeat_sale`.

## Permission additions reserved by this contract

| Permission | Organization/platform roles |
| --- | --- |
| `settlement:manage` | owner, finance; recent MFA |
| `billing:manage` | owner, finance; recent MFA |
| `credit_policy:manage` | owner, administrator, finance; recent MFA |
| `providers:operate` | platform administrator/compliance reviewer under existing rules |
| `accounts:recover` | platform administrator/compliance reviewer with separation of duty |
| `privacy:review` | compliance reviewer/platform administrator with dual control where destructive |
| `risk_hold:manage` | platform administrator/compliance reviewer; recent MFA and reason |

## Change control

Changing a state, problem code, audit action, notification event, analytics
event, or permission requires, in the same change:

1. a rationale and compatibility assessment;
2. OpenAPI/state documentation updates;
3. migration strategy where persisted values change;
4. test and fixture updates;
5. interface-copy review where the user-visible meaning changes.
