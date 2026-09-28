# Kredit user flow

Current source: 27 September 2026. This describes the product, not a deployment certification.

## Product rule

**One person manages each business.** There are no business staff invitations, roles to configure, delegated purchasing or internal sale approvals. One person can own more than one business; each business keeps its own records. Kredit's platform administration is a separate permission boundary.

Kredit records supplier-funded trade credit: a seller offers goods on agreed terms, the customer accepts, goods are handed over and payments stay attached to the sale. Retailer-to-person purchases have their own acceptance, delivery, receipts and refunds. A business can buy and sell.

## Everyday use

Three destinations—**Who owes me**, **Customers**, **Money in**—and one action, **Give goods on credit**. Business settings, purchases from suppliers, account and help stay in the menu. Business scope travels with record links.

- **Who owes me:** balances, overdue amounts, offers awaiting acceptance and the next action on payments, deliveries, disputes and refunds. Missing financial data is shown as unavailable. A failed unrelated read does not hide actions from successful reads.
- **Give goods on credit:** choose a business or person, goods, amount and date. Optional invoice and payment-plan details stay collapsed. Business sales open a saved draft showing exact terms and fees before Send. Personal offers show delivery/return terms and require stock confirmation before producing a shareable link. This is the sole sale-entry form; there are no old-form redirects or previous-format device drafts.
- **Customers:** business balances come from a scoped database projection over obligations; personal balances come from a complete summary projection without event histories. An individual's row opens all their purchases. Missing or malformed balances never become zero.
- **Money in:** business receipts and reported transfers, plus a separate personal-purchase summary after refunds and reversals. Each reported payment links to its authorized confirmation screen.

The owner can review and apply a credit note without another employee. Draft intent, current-owner authorization, balance limits, collection reservations, ledger posting and audit evidence remain enforced.

## Setup and records

Settings links to business details, receiving accounts, Kredit fees, default terms, purchasing setup, personal-sale bank setup and existing customer credit limits. Owner identity, bank and sign-in checks remain. Readiness checks cover the owner; there is no finance-staff setup.

Sale, purchase, dispute, mandate and credit-limit records have direct routes. Staff pages, duplicated sale forms and hidden alternative screens are deleted. There is no expanded-workspace switch, compatibility redirect table, staff API or historical-access screen. Unshipped staff/delegation/branch/approval modules and proposed-terms imports are removed.

The platform has not launched; the only account is the owner's testing account. Migration 213 removes unused staff tables, history and policy machinery. It preserves business ownership, customer invitations and financial records. No live database cleanup or account reset has been performed.

See [current pages](current-pages.md) for the route map and [implementation record](../audits/2026-09-27-platform-audit/IMPLEMENTATION.md) for changes and checks. The longer implementation-status document is historical.

## Operational boundary

The backend requires migrations through **213**. That migration rejects ambiguous/missing active ownership before removing the unused staff schema; it does not choose an owner automatically. Follow the [stopped-writer cutover procedure](../operations/production-cutover-runbook.md).

Each new financial projection reads one database snapshot. The business and personal domains are still separate requests; a combined screen is not a single cross-domain accounting snapshot. No tests, browser journeys, database changes, migration execution or deployment were performed for this implementation.
