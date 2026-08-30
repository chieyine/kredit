# Financial and trust interface copy

Copy version: `trust-copy-v1`

Locked: 29 August 2026

This is the baseline English copy for new financially material and trust flows.
Amounts, dates, references, and organization names are interpolated from
validated server data. Product interfaces must not expose raw domain states.
Legal disclosures remain disabled until approved versions are supplied.

## Financial confirmations

| Context | Heading | Primary action | Supporting text |
| --- | --- | --- | --- |
| buyer accepts credit | Review the exact credit terms | `Accept {amount} credit` | By accepting, you confirm the goods, amount, repayment schedule, grace period, fees, and collection authorization shown here. |
| buyer confirms drawdown | Confirm this purchase | `Confirm {amount} drawdown` | This reserves part of your approved trade line for the goods and repayment terms shown here. No obligation becomes active until release and receipt are recorded. |
| supplier releases goods | Confirm goods release | `Release goods worth {amount}` | Confirm only after the described goods have left your control. This action is recorded with your identity, time, and evidence. |
| buyer confirms receipt | Confirm what you received | `Confirm receipt of {amount} goods` | Choose this only if the goods were received without an issue. Your repayment obligation will become active. |
| buyer reports receipt issue | Tell us what went wrong | `Report an issue` | Collection for the affected amount will remain blocked while the issue is reviewed under the dispute process. |
| supplier records payment | Record confirmed payment | `Record {amount} payment` | Record only money you can verify. Kredit will allocate it to the obligation and preserve the source and reference. |
| operations financial adjustment | Review the financial effect | `{action} {amount}` | This action requires recent MFA, a reason, and creates an immutable audit record. |

## Status explanations

| Domain condition | User-facing copy |
| --- | --- |
| buyer confirmation required | Waiting for the buyer to confirm the exact terms. |
| safe to release | The buyer has accepted and the payment authorization is active. You may release the goods described in this agreement. |
| receipt required | The supplier recorded release. Waiting for the buyer to confirm receipt or report an issue. |
| activated | Receipt was confirmed and the repayment obligation is now active. |
| provider result unknown | The provider has not confirmed the final result. Do not retry yet; Kredit is reconciling it. |
| mandate inactive | The payment authorization is not active. New credit and releases are blocked until a fresh authorization is completed. |
| organization not ready | This organization cannot use this action yet. Complete the readiness steps shown below. |

## Account recovery

- Heading: `Recover access safely`
- Introduction: `We will ask for more than control of a phone number. This helps protect your organization and financial records from account takeover.`
- Submitted: `If the account can be recovered, we have started a protected review. For your security, we cannot confirm account details here.`
- Cooling off: `Recovery was approved and is in a security waiting period. Sensitive financial changes remain blocked until {date}.`
- Contact warning: `Did not request this? Cancel the recovery request from a signed-in device or contact support using reference {reference}.`
- Completion action: `Complete account recovery`

## Disputes and receipt issues

- Heading: `Report a specific issue`
- Scope: `Tell us which goods or amount is affected. Undisputed amounts remain payable.`
- Evidence: `Upload evidence that helps explain the issue. Do not include passwords, PINs, OTPs, or full bank credentials.`
- Submitted: `Your issue was recorded. Collection is blocked only for the affected amount while it is reviewed.`
- Decision: `Review the decision, financial adjustment, evidence considered, and appeal or support route below.`

## Privacy requests

- Heading: `Your data rights`
- Introduction: `Request access, correction, deletion, restriction, objection, consent withdrawal, or portability. Some financial records may need to be retained by law or to resolve an active claim.`
- Submitted: `We recorded your request as {reference}. You can return here to see its status and any clarification we need.`
- Identity check: `Before releasing or changing personal data, we need to confirm that this request belongs to you.`
- Retention conflict: `We cannot remove the records listed below yet because they are subject to a legal, financial, fraud-prevention, or dispute hold. The remaining approved actions will continue.`
- Export action: `Download protected data export`

## Operations actions

- Retry heading: `Retry this operation safely`
- Retry warning: `Kredit will reuse the original idempotency reference. Review the provider and ledger status before retrying.`
- Suspension heading: `Suspend access and new financial actions`
- Suspension warning: `Existing obligations and factual history remain. Review the affected actions and notifications before confirming.`
- Restore heading: `Restore permitted access`
- Risk hold warning: `A risk hold blocks only the actions and amount shown. It does not erase an existing obligation.`
- Reason label: `Reason for this action`
- MFA label: `Confirm with recent MFA`

## Copy safety rules

- Never say paid, settled, verified, delivered, or collected without the
  corresponding authoritative evidence.
- Never imply that a provider timeout is a failure.
- Never hide fees, grace periods, collection timing, or mandate ceiling.
- Never use blame-oriented collection or dispute language.
- Never put secrets, raw tokens, full bank details, or restricted evidence into
  notifications.
- Final financial buttons include the amount and action; `Continue` is not an
  acceptable final confirmation label.
