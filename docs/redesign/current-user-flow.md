# Kredit user flow

The public website explains one network: manufacturers invite distributors, distributors sell to retailers, and retailers serve individual consumers. A business can buy and sell from the same workspace. Each trading relationship keeps its own terms, records and balances.

## Public entry

- Home, How it works, Manufacturers, Distributors, Retailers and Consumers explain the roles and next steps.
- The business demo demonstrates a trade; the separate consumer demo demonstrates a personal purchase. Both are illustrative and move no money.
- Pricing retains live verified fee disclosure. Help, security, guides and legal pages remain available.
- Open Kredit leads to one sign-in and account choice. A private invitation or purchase link preserves its intended destination.

## Account choice

`/start` offers Business workspace and Personal purchases. An individual consumer does not need a business profile. Shared security, privacy, messages and display settings live under `/account`.

## Business workspace

Five everyday destinations: Today, Sales, Purchases, Partners and Money. Find a record, Reports, Team, Business setup, Settings, Messages, Personal purchases and Help are in the menu. Section navigation and search retain access to specialist tools.

- Today: chosen business, separate receivables and supplier balances, required setup steps and outstanding actions.
- Sales: business sales, independent credit approvals and reviewer ceilings, consumer offers, credit limits, overdue balances and disputes.
- Purchases: a validated business purchasing context, staff permissions and verification setup, offers, balances, buying limits, payments, mandates, history and disputes.
- Partners: individual invitations, reviewed CSV imports, invitation progress, branches/territories, owner-managed staff branch access, account-manager assignments and business-specific customer statements.
- Money: incoming payments, collection results, outgoing supplier obligations, fees and reconciliation links.

The business identifier stays in navigation URLs. Purchasing lists and reports validate current ownership or explicit staff purchasing authority and narrow results on the server. Customer statements distinguish business identities even when one person owns multiple businesses. Notification links to a purchase resolve its business before rendering the purchase.

## Personal purchases

`/personal/purchases` lists the individual's purchases. Each purchase keeps accepted terms, payment progress, delivery and return/dispute actions together. Personal records are separate from business trade credit.

## Removed duplication

Old `/app`, `/buyer`, `/for-suppliers` and `/for-buyers` page trees are removed without compatibility redirects. Buyer-specific copies of account settings/messages are removed. Account preferences are shared; business settings remain in the workspace. Optional field-agent referral screens remain available only where relevant.

## Release boundary

This file describes the implemented user flow, not completion of every enterprise feature in the earlier blueprint. See `implementation-status.md` for remaining architectural work. No production deployment or live provider transaction is represented by local browser fixtures. Deployments must apply migrations through 194, update provider return paths, and reconcile already published website/legal content with the new route map.
