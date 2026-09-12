# Essential checks — 12 September 2026

Only focused checks were run, with no agents, live provider calls, customer messages or deployment. Database checks used a newly created disposable PostgreSQL 18 database. Existing databases were not reset or migrated.

## Passed

- Fresh database migrations through 148, including notification timestamps, bank registration, fee invoices and refund evidence.
- Affected backend packages: web handlers, onboarding, operations, payments, ledger and reports.
- Money flows: concurrent payment replay, collection restart safety, payment reversal, invoice issuance replay, partial fee receipts, fee waivers, recognised collection reversals and completed fee refunds. Fee receipts leave buyer principal unchanged.
- Permissions: ownership transfer, restricted invoice access, current owner authority, immutable bills and receipts, worker restrictions and bank-registration restrictions.
- Provider boundaries: native Mono bank/NIP routing and account-response matching; stored notification route rotation; retained collection credential redaction and refusal to carry credentials to a changed endpoint.
- Frontend: zero errors and zero warnings after correcting the affected types and saved-account validation pattern.
- Patch whitespace check.

## Failures corrected

- Reports compilation and missing request context in drawdown document helpers.
- Missing notification creation timestamp in migration 118.
- Outdated consent, notification-adapter, billing-readiness and production-configuration test fixtures.
- Invoice receipt locking attempted to use an UPDATE permission on immutable invoice rows; transaction-level locking now preserves their read/insert-only access.
- Invalid audit outcome values in invoice payment recording and message/bank recovery.
- Base fees could be counted twice when calculating a waiver. Waivers now use real fee lines and the corresponding revenue account. Reversals cancel only unwaived collection fees.
- Admin-triggered onboarding outcome notices now target the business's authorised financial users.

A provider reversal intentionally opens financial review; it does not by itself rewrite a recognised payment. The refund test includes the required confirmed-reversal step.

These checks cover affected paths, not live-provider certification. The implementation previously listed as unfinished has now been completed.

## Final continuation

- Native Mono phone/CAC request contracts, consent, OTP fences, minimal retained identity data, cross-user access, private evidence scanning, and reviewed appeals preserving the previous decision passed under restricted application permissions.
- Frozen seller destinations, exact split fee allocation, duplicate-payment protection, bank receipt/return limits and payment reversal passed. Seller fees and payouts leave buyer principal accounting separate.
- Separate native Mono fee customer/e-mandate request contracts passed. The native adapter sends the legal business name, uses a variable e-mandate and charges provider transaction costs to the business wallet.
- Authorized fee billing passed under the worker role: submission fence, original-account reconciliation after switching accounts, no duplicate debit, blocked conflicting receipt, lifetime ceiling, provider-held funds versus bank cash, callback review flags and completed reversal.
- Personal export and recovery controls passed. New identity/history/fee-consent sections exclude fingerprints, live authorization links and authentication secrets.
- Existing invoice/waiver/receipt and supplier onboarding MFA/role checks passed alongside the new checks.
- Full frontend diagnostics: **0 errors, 0 warnings**. API YAML parses, references resolve and operation IDs are unique. Patch whitespace is clean.
- PostgreSQL migrations through **148** applied only to the disposable database; the production role template applied successfully. The inventory contains **1,318 fields**.

Local evidence logs: `/private/tmp/kredit-final-essential.log`, `/private/tmp/kredit-final-privacy-fees.log`, `/private/tmp/kredit-final-web.log`, `/private/tmp/kredit-final-migrations.log`, `/private/tmp/kredit-privacy-migration.log`, `/private/tmp/kredit-final-roles.log` and `/private/tmp/kredit-inventory.log`.

During these checks, incomplete synthetic bank details, a reused synthetic mandate ID and incorrect frontend response-decoder calls were corrected. Restricted worker access and the final fee lock/ceiling behavior were then checked successfully. No broad end-to-end suite, production data reset, live provider message/debit or deployment ran.

The final billing-edge check also passed: unpaid split fees receive one monthly bill, prior partial deductions are not charged twice, and a reversal after billing restores the correct unpaid base fee. Logs: `/private/tmp/kredit-final-split-invoices.log` and `/private/tmp/kredit-split-fallback-migration.log`.

After the final billing UI and customer-validation changes, frontend diagnostics again finished with **0 errors and 0 warnings**, and the native Mono fee-customer/mandate check passed. Final logs: `/private/tmp/kredit-final-billing-web.log` and `/private/tmp/kredit-final-mono-fees.log`.
