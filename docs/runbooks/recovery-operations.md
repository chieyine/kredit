# Recovery operations

Status: implemented in source; runtime and provider exercises have not been performed for this change. Owner roles below are responsibilities, not evidence that a person has been assigned. Assign named primary and backup operators before rollout.

## An unconfirmed customer or operator request

1. Keep the original details. The shared mutation client retains the original random identity and payload digest across refreshes. It never stores payment evidence in its recovery record. Do not clear browser storage, switch accounts or invent a fresh request to bypass recovery.
2. Within the supported replay window, retry only the exact original action. After fifteen minutes, use record review; a retained request is not silently expired by the browser. A 409 or timeout is not proof that the action failed.
3. Ask for the support reference, approximate time, affected sale or payment reference and account identity through the authenticated support process. Never ask for passwords, OTPs, bank authorization links or a raw network export.
4. An authorized operator can paste the complete `kredit-…` reference into **Find a reference** (`/admin/search`). Lookup reveals only request-receipt metadata; it does not reveal saved request/response bodies or grant permission to execute the request.
5. `RESULT_UNCONFIRMED` means the receipt has no completed response. `HTTP_RESPONSE_RECORDED_200` (or another status) means an HTTP response was recorded, not that a bank debit or settlement succeeded. Several receipts may exist across sessions. Missing receipts can have expired; absence never proves failure.
6. Compare authoritative payment, collection, settlement, journal and provider records. Use existing domain recovery commands once the actual outcome is known. Do not delete idempotency rows to permit a retry. Do not instruct a second bank transfer to resolve a missing application response.
7. Record the investigation and verified outcome in the support case. If domain evidence remains inconclusive, retain the hold and escalate to financial operations; the reference lookup deliberately cannot reset requests.

## Financial differences and handover

- Open `/admin/reconciliation`, complete identity confirmation, filter unassigned cases and claim a review with investigation notes.
- The assigned reviewer may release a case for handover, recording why and what remains. A platform owner can take over an assigned case when its reviewer is unavailable or loses access. Ordinary reviewers cannot seize another person's case.
- Claim, release, takeover and resolution use current authority checks and the same transaction/lifecycle locks. The immutable event trail records the actor, time, reason and previous owner for handovers.
- Assignment does not change the financial records. Resolution still requires current source records to agree. Fix records through existing approved financial commands; never replace a balance solely to remove an alert.
- Initial alerts: an unassigned case persisting fifteen minutes is critical; a case older than one hour is critical. These are escalation targets, not promises that investigation can finish in that time. Reopened cases retain their original first-seen time.

## Delivery issues

A reported drawdown issue retains reserved capacity and creates no debt until buyer receipt confirmation. The buyer can confirm corrected delivery; the seller can accept a return and cancel the purchase. These actions close the separate receipt dispute and preserve its original reason. Do not confirm receipt on the buyer's behalf or cancel released goods merely to free capacity.

An issue older than one day raises a warning for support follow-up. Use the drawdown and line references to coordinate a resolution. A bank mandate or other independent financial restriction may still block activation; resolve that restriction without bypassing it.

## Mandate cancellation

A saved local cancellation block prevents new eligible submissions even when a provider lookup or cancellation call fails. A provider outage does not undo that local block. Reconcile the original provider reference before claiming bank-side cancellation is complete. A debit already submitted may still complete.

## Escalation record

Record case/reference, environment, first observed time, named owner and backup, containment, verified evidence locations, next action and review deadline. Keep personal and bank data in the restricted case system. Release owners must supply alert recipients and verify delivery; this repository does not send incident messages automatically.
