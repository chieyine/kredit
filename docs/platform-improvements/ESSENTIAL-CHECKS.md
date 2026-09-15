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

## Consumer purchases — 15 September 2026

Implemented and checked the independent retailer-to-person flow described in [CONSUMER-SALES.md](CONSUMER-SALES.md). No agents, real bank transfers, provider messages or production migrations were used.

Passed:

- Fresh disposable PostgreSQL migrations through **151** and the final application/worker role template.
- Consumer agreement totals and unchanged agreement hash after payments; confirmed-payment delivery thresholds; personal acceptance; cancellation and return escalation rules.
- Restricted database roles: exact invited customer access, outsider denial, viewer write denial, customer receipt-forgery denial, immutable financial history and unchanged accepted identity.
- Concurrent receipt attempts recognise money once. Customer reports do not count as receipts. Deposit reversal restores the outstanding balance.
- Price reduction, partial goodwill refund, full return and remaining refund leave the correct customer balance. Every affected ledger account closes to zero after the fully refunded purchase; no trade-credit or supplier-fee account is touched.
- Worker due-reminder discovery, active-purchase access and durable deduplication; buyer/seller/admin list queries and older-page navigation.
- Personal export integration, including the new consumer agreement and history sections; focused email/WhatsApp OTP routing regression.
- Final frontend diagnostics: **0 errors, 0 warnings**. OpenAPI YAML parses and references resolve. Patch whitespace is clean.
- Inventory regenerated with **1,354 fields**.

The checks exposed and corrected the buyer audit transaction's organization scope and an overly broad readiness lock that required retailer settings-update permission. The final readiness lookup returns only a boolean, locks the approved configuration and checks current verification. An initial test-only ledger join used a UUID against a text reference and was corrected. Fresh frontend route types were regenerated before the final diagnostics.

Evidence: `/private/tmp/kredit-consumer-final-money.log`, `/private/tmp/kredit-consumer-essential.log`, `/private/tmp/kredit-consumer-web-final.log`, `/private/tmp/kredit-consumer-migration.log`, `/private/tmp/kredit-consumer-migration150.log`, `/private/tmp/kredit-consumer-migration151.log`, `/private/tmp/kredit-consumer-final-roles.log` and `/private/tmp/kredit-consumer-inventory.log`. The web package compiled during the focused backend check; no broad web test suite was run.

## Automatic consumer activation — 15 September 2026

The separate per-retailer consumer approval has been removed. Existing business verification, registered bank and billing requirements remain; Super Admin manages exceptional restrictions.

- Migration **152** and the final role template applied to the disposable database. Inventory refreshed with **1,360 fields**.
- Essential consumer checks passed: a retailer without platform-admin authority can connect its registered account and create an offer; an account with matching last four digits but a different full number is rejected.
- Unauthorized restriction changes, restricted offer creation and restricted acceptance are rejected. Reconnecting the bank cannot remove a restriction.
- Existing consumer payment, reversal, delivery, return, refund and reminder checks passed. Settlement and web packages compiled during the focused check.
- Frontend diagnostics: **0 errors, 0 warnings**. API YAML parses; patch whitespace is clean.

Evidence: `/private/tmp/kredit-auto-consumer-tests.log`, `/private/tmp/kredit-auto-consumer-migrate.log` and `/private/tmp/kredit-auto-consumer-web.log`. No live provider calls or production migrations ran.

## DSA referral programme — 15 September 2026

- Fresh disposable database migrations through **153** passed. Final scoped SQL functions and immutable attribution/agent guards were applied and checked. Application/worker role grants were applied; production was not changed.
- Focused DSA integration checks passed: owner-only attribution and replay, no reward without CAC evidence, isolated agent views, verified onboarding, accepted sales alone earning no activation reward, activation and fee sharing, concurrent refreshes, seven-day holds, duplicate payout prevention, frozen pending payout destinations, refunds, negative balances and commission-ledger reconciliation.
- Provider-held fees earned nothing; independently reconciled bank receipts qualified; disputed debit evidence and returned provider cash reduced rewards. Duplicate CAC identities were restricted, onboarding limits were enforced, and increasing a limit released the waiting reward.
- The affected privacy/recovery integration check passed, including the four DSA export sections. The web package compiled during the focused backend check; no broad web test suite ran.
- Final frontend diagnostics: **0 errors, 0 warnings**. OpenAPI YAML parses, local references resolve and operation IDs are unique. Patch whitespace is clean.
- Inventory regenerated with **1,424 fields**.

Checks exposed and corrected an enrolment read lock, a referral SQL alias conflict, scoped agent-status access for payout preparation and a test-only ledger-reference cast. No agents, live provider calls, real messages or real transfers were used.

Evidence: `/private/tmp/kredit-dsa-fresh-migrations.log`, `/private/tmp/kredit-dsa-functions.log`, `/private/tmp/kredit-dsa-roles.log`, `/private/tmp/kredit-dsa-essential.log`, `/private/tmp/kredit-dsa-privacy.log`, `/private/tmp/kredit-dsa-web.log` and `/private/tmp/kredit-dsa-inventory.log`.
