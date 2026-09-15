# Consumer sales — 15 September 2026

Retailers can sell to individuals separately from their wholesaler purchases. Consumer money goes directly to the retailer. No consumer payment is routed to, allocated against, or used to settle a wholesaler obligation.

## Screens

- Retailer: **Sales → Consumer sales** (`/app/consumer-sales`). Use **Use my registered receiving account** to enter the account already registered for the business. Kredit matches the full account to the original provider registration; no extra admin approval is needed. Create an offer and share its sign-in link with the exact customer email or WhatsApp number. Open each purchase to record receipts, delivery, returns and refunds.
- Customer: **Personal purchases** (`/buyer/purchases`). Sign in with the invited contact, review the complete agreement, provide a name and delivery address, and confirm adult status and consent. A business profile or CAC registration is not required.
- Super admin: **Consumer purchases** (`/admin/consumer-sales`). Inspect purchase history, resolve escalated returns and record completed financial corrections. Eligible retailers activate automatically after connecting their existing registered bank account. Restrict a retailer only when an exception requires it. Removing a restriction does not bypass eligibility checks, and reconnecting a bank cannot clear a restriction. Existing payment/refund records remain available.

## Agreement and payment behavior

The immutable agreement contains the item, quantity, total price, deposit, installment dates, delivery condition, stock reservation reference, delivery deadline, return arrangements, retailer identity and approved receiving bank account. It has a saved hash and version, with separate evidence of the customer's acceptance. A changed offer requires a new agreement; the old one can be declined or cancelled.

Schedules reuse the existing equal-payment generator: weekly, fortnightly or monthly, with month-end capping and exact kobo totals. One installment supports a single due-date purchase. Deposits have a separate earlier due date. There are no additional consumer interest or late fees in this flow.

Consumers transfer to the retailer's approved bank account, or pay the retailer in cash, then report the completed payment. Reports remain unconfirmed until an authorised retailer checks and records the bank credit or signed cash receipt. Confirmation must exactly match a selected report. A bank transfer reference cannot be recognised twice. An incorrect or reversed receipt has an explicit reversal record and reopens the correct remaining balance.

This flow does not create a consumer bank mandate or initiate a provider charge. Existing business direct-debit workflows are unchanged. Bank-account instructions are frozen in each agreement; later settings changes do not redirect an accepted purchase.

## Delivery, cancellation and refunds

- Delivery can follow acceptance, a chosen paid percentage, or full payment. Only confirmed net receipts count; unverified reports never release goods. A price reduction also reduces the relevant delivery threshold.
- The retailer confirms stock reservation when offering the sale and records dispatch/handover evidence when eligible. The delivery deadline starts when the recorded eligibility condition is met. Later payment corrections do not erase a delivery deadline after release. Receipt is confirmed explicitly by the customer; silence is not treated as delivery acceptance.
- Either party can cancel before release. The remaining scheduled price stops being payable and all confirmed funds still held become refundable.
- After release, the customer can request a return or report a delivery problem. The retailer records a reasoned decision and return/non-delivery evidence. A declined case can be escalated; only super admin can decide an escalated case.
- An approved full return removes the unpaid purchase balance and makes the remaining customer funds refundable. Price reductions support partial goodwill refunds without inventing a second debt. Refund recording requires evidence of an actual completed repayment to the customer; it does not send money.
- Consumer receipts, customer funds held by the retailer, delivered-goods receivables and their reversals use separate balanced ledger accounts. They do not touch trade-credit principal or supplier fees.

## Messages and access

Purchase changes use the durable notification queue, with links to the private purchase record. Daily due reminders respect notification preferences and optional-processing restrictions. Open payment reports and unresolved return cases suppress further payment prompts. Cancelled/paid purchases are not prompted; stale due amounts are checked again before entering delivery processing.

The API checks the current customer or retailer role on every action. Retailer and super-admin changes require recent MFA, CSRF protection and an idempotency key. Financial actions and permission checks run within a database transaction with a locked sale version. Money history and accepted terms are immutable. Lists support older-page navigation, and the purchase can be printed with its history.

Personal privacy exports include consumer agreements and purchase history. Consumer settings, agreements and events are classified as restricted financial data in the inventory. The runtime requires migrations through **152**, followed by the current role template.

## External activation

There is no additional per-retailer consumer approval. Existing business verification, current bank verification and billing readiness determine eligibility automatically. Super admin manages exceptions and platform-wide legal/operating requirements. This code does not certify regulatory status or provider-account approval. WhatsApp/email delivery still requires the previously configured live providers and approved templates.

The consumer-rights provisions concerning delivery, cancellation and refunds informed the separate lifecycle: [FCCPC consumer rights](https://fccpc.gov.ng/consumers/) and [FCCPA](https://fccpc.gov.ng/wp-content/uploads/2022/07/FCCPA-2018.pdf). Provider settlement is a separate process from recording an incoming retailer transfer: [Mono settlement documentation](https://docs.mono.co/docs/payments/settlements).
