# Bank collections without Mono

Kredit has native direct-debit adapters for Flutterwave, Paystack and Monnify. Mono remains optional. A custom connector is a separate integration type; entering a vendor's API address in the connector field does not install that vendor's API.

## Super-admin setup

1. Open **Launch setup → Bank authorization and collection**. Select the adapter and give the account a stable name, such as `flutterwave-main`.
2. Enter its secret key. Flutterwave also needs its webhook secret. Monnify needs its API key and contract code. Leave the native endpoint blank. Paystack also accepts its fixed official API address.
3. Record the provider's actual direct-debit approval and Kredit's collection limits before enabling collections. Live approval is separate from having ordinary payment API credentials.
4. In **Seller bank accounts**, use the same account name to register seller destinations through that native provider. Leave the connector endpoint/token blank. Seller subaccount approval may be a separate provider requirement.
5. Set the provider dashboard webhook to the public API address serving Kredit:
   - Paystack: `/api/v1/webhooks/paystack/{account-name}`.
   - Flutterwave or Monnify: `/api/v1/webhooks/bank/{account-name}`.
6. Check **Launch setup** to confirm both website and background worker loaded the saved settings. Complete the provider's sandbox certification and an explicitly authorized controlled live transaction before general launch.

`APP_BASE_URL` must be the real customer-facing origin, normally `https://kredit.ng`. API credentials must match the deployment environment. API host selection is fixed by the adapter; native credentials are never sent to an arbitrary connector address.

The new authorization storage requires migration **154**. Bank details are encrypted with `SETTINGS_ENCRYPTION_KEY`; include this key in the existing secure backup/key-management procedure.

## Customer authorization

- **Paystack:** redirects to Paystack's hosted direct-debit authorization. The customer needs an email on their Kredit account. Authorization codes stay on the server and never enter public mandate metadata.
- **Flutterwave:** collects the bank details on Kredit, then displays the provider's activation-transfer amount and destination. The fee must come from the nominated account. The screen shows its expiry and clearly distinguishes the activation fee from repayment. `APPROVED` is not collectible: the token must be `ACTIVE`.
- **Monnify:** creates a finite, variable-amount mandate using the documented API schema and directs the customer to the returned authorization link. Kredit checks the original account, contract, amount and mandate before a debit.

Customers can check activation and cancel permission from **Bank permissions**. Cancellation pauses new Kredit debits before contacting the bank. An already-submitted request may still complete. Bank cancellation failures remain visible and can be retried.

A timed-out authorization is held for review rather than submitted again. **Mandate recovery** lets a super admin attach a verified provider reference, or permit another attempt only after recording evidence that the provider created nothing. Flutterwave recovery also checks the unique Kredit request identifier carried in the provider narration. Version checks prevent a late response from overwriting a newer recovery decision. Paystack does not echo Kredit’s client reference or cap: its recovery verifies the original buyer’s active bank authorization and retains the agreed local limit, with the operator’s provider evidence recorded.

## Payment and settlement rules

- All internal amounts are integer kobo. Flutterwave/Monnify amounts are converted to exact decimal naira without floating-point arithmetic.
- Submission is not proof of payment. Authenticated status lookup must match the original reference, amount and bank permission before Kredit recognizes money.
- Native callbacks are authenticated and saved as minimal reconciliation notices. Raw bank details and authorization tokens do not enter the webhook inbox.
- Uncertain outcomes are reconciled against the original provider. They are never retried with another collector.
- Native collections enter **settlement pending**. Seller subaccount registration does not prove seller payout. These adapters use Kredit's existing **reviewed seller transfer** workflow; they do not claim automatic split-settlement confirmation. The admin must record the actual payout evidence.
- Native collector setup does not automatically provide the separate seller-fee debit product. Existing consolidated invoice billing remains available; authorized seller-fee debit remains available only through a provider implementing that specific contract.
- Monnify's public documentation says sandbox callbacks omit the production signature. Kredit rejects unsigned callbacks and uses authenticated reconciliation instead.

## Switching collectors

Save the current connection in **Saved collection accounts** before choosing another active collector. Select its correct native adapter and preserve its exact name, credentials and Monnify contract code. The old connection remains available for status checks and cancellation. It cannot issue new debits as a retained account.

A new collector needs a new customer authorization. Switching the active collector does not migrate mandates, move existing attempts, or authorize a second charge. Existing saved requests keep their original provider identity. Automatic cross-provider debit retry is deliberately not implemented.

## CreditChek and additional recovery

CreditChek RecovaPRO live debits are not enabled. Kredit still needs a confirmed per-debit status/reconciliation contract, a safe rule for ambiguous submissions, and confirmation of the bureau-reporting requirement for this trade-credit use case. The user confirmed on 2026-09-16 that these answers are not yet available. Do not infer a payment from a scheduled debit or from a mandate's cumulative collection total.

Ordinary direct debit covers the explicitly authorized account. It does not grant access to every account linked to a BVN. Additional-account recovery requires the provider's actual approved product and the customer's explicit consent.

## Official references checked on 2026-09-16

- [Paystack direct debit](https://paystack.com/docs/payments/direct-debit/), [customer authorization APIs](https://paystack.com/docs/api/customer/), [transaction APIs](https://paystack.com/docs/api/transaction/), [webhooks](https://paystack.com/docs/payments/webhooks/), [seller subaccounts](https://paystack.com/docs/api/subaccount/).
- [Flutterwave direct debit](https://developer.flutterwave.com/docs/direct-debit), [reference-based verification](https://developer.flutterwave.com/reference/verify-transaction-with-tx_ref), [v3 webhooks](https://developer.flutterwave.com/docs/webhooks), [collection subaccounts](https://developer.flutterwave.com/reference/create-a-sub-account).
- [Monnify direct debit](https://developers.monnify.com/docs/collections/recurring-payments/direct-debit), [published OpenAPI schema](https://developers.monnify.com/collection/monnify-collection.yml), [webhook signatures](https://developers.monnify.com/docs/webhooks).

Monnify's introductory guide differs from its published API schema. The adapter follows the schema: `mandateStartDate`/`mandateEndDate` for creation and `mandateCode`/`debitAmount`/`customerEmail` for debit submission. Confirm this contract during provider certification.

## Verification completed

- Native-provider contract checks cover money units, activation gates, mismatched payment identities, authenticated callbacks and pending submission outcomes.
- Collection routing and configuration checks passed, including masking saved native credentials and refusing endpoint overrides.
- Backend checks passed for disabled-provider fallback, webhook authentication and duplicate-request handling.
- An isolated PostgreSQL 18 database migrated successfully through version 154. Restricted-role checks passed for buyer isolation, encrypted bank details, duplicate submission fencing, late-response rejection and cancellation remaining paused after a bank timeout.
- Frontend validation passed with zero errors and warnings. The final changed authorization components also compiled without warnings.

These are local fixture and isolated-database checks. No live provider credentials, customer account or real payment was used. The application database was not migrated or deployed during this work.
