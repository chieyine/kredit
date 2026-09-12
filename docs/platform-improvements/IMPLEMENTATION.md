# Kredit implementation record

Scope: recommendations 1–7 and all existing features. Recommendation 8, reducing the launch scope, was excluded. No agents were used. Only essential affected-feature, permission and money-flow checks were run. Production is `https://kredit.ng`.

## Implemented

| Area | Result |
| --- | --- |
| Sale workflow | Buyer and seller next steps, evidence timeline, due-payment worklist, receipt confirmation, exact accepted fees and dates, and preserved agreement. |
| Bank collections | Debit requests, verified payments and actual seller bank receipts remain separate. Requests commit before submission; unknown results retain their original reference and account. Full Mono collections support frozen fixed seller splits; partial recoveries use reviewed bank settlement. |
| Seller fees | Accepted base and collection fees, split deductions, calendar invoices, bank receipts/refunds and separate seller-authorized bank debits. Fee money never reduces buyer debt. Waivers, reversals, lifetime debit ceilings and unsettled funds have distinct accounting. |
| Native verification | Mono phone OTP and CAC lookup, consent, exact identity matching, separate authority review, private scanned documents, renewal and reviewed appeals with preserved prior decisions. |
| Recovery | Saved customer registrations, verification, bank permissions, bank destinations and unknown messages are recoverable through admin. A provider change does not silently resubmit unresolved work. |
| Provider accounts | Native Mono, Meta WhatsApp, Mesaj SMS and Sendly email; replaceable connector contracts; saved original financial and identity accounts; encrypted write-only credentials and runtime configuration refresh. |
| Super admin | Launch setup, access and policy management, evidence review, fee arrangements, fee bank reconciliation, seller payouts, message recovery, original-provider work and privacy requests. Sensitive changes require current authority, recent MFA and recorded evidence. |
| Privacy and operations | Personal exports include identity decisions and fee consent without live links or authentication secrets. The field inventory, API contract, deployment permissions and operating notes cover the added records. |

## Financial behavior

- Split fees use the accepted agreement, not current pricing. Fees still unpaid at month-end receive a monthly bill; prior deductions are credited, and a later reversal restores only the amount no longer paid. A partial recovery can retain only fees supported by the amount actually received. Invoiced or previously collected fees are not charged again.
- Seller settlement records confirm completed bank movements. A provider callback or registered bank destination does not prove that the seller received funds.
- Separate fee e-mandates have a one-year lifetime ceiling. The seller gives bank permission and chooses the billing arrangement; super admin checks the original customer, mandate, reference, ceiling, dates and readiness before approval.
- Fee bills close weekly or monthly and are due seven days after issue. Debits use a saved reference, original account and durable submission fence. A failed debit leaves the invoice payable by bank transfer. Small bills below the supported bank minimum also remain payable by transfer.
- Provider-held fees are recorded separately from bank cash. Super admin records actual bank receipts/returns and completed fee-debit reversals. Dispute signals flag review; they do not independently reverse money.
- Pausing fee debits stops future requests. It does not erase an already submitted request or cancel the bank's permission. Resumption requires renewed review; changing provider requires that provider's own authorization.

## Verification and release boundary

The disposable PostgreSQL 18 database has migrations through 148. The runtime minimum is 148. Role grants were applied to this database. Frontend checking completed with zero errors and warnings. Focused financial, native-provider and permission checks passed; the final personal-export/fee checks also passed; evidence is recorded in [ESSENTIAL-CHECKS.md](ESSENTIAL-CHECKS.md).

No existing customer database was reset. No production migration, live message, live debit or deployment was performed. The user's `.gitignore` change was preserved.

The owner must supply real account credentials, obtain provider account/template approvals, complete domain and email records, verify bank receiving details and enter the required business/legal approval evidence. Supported settings and review actions are in super admin; provider dashboards and domain records remain external account administration.

Provider accounts must be live-approved before live activation. Local checks do not certify the behavior of an account that has not yet been connected.
