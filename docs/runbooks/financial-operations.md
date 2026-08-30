# Financial and support operations runbook

This runbook covers the production incidents required by the README. Never
edit financial tables directly. Use an authenticated command, preserve the
reference and request ID, and append an audit event for every operator action.

## Immediate containment

1. Record severity, public reference, tenant, operator, time, and evidence
   location.
2. Disable the smallest affected capability (provider, collection retry,
   settlement change, or account) with the feature flag; do not disable the
   ledger or erase evidence.
3. Check idempotency, provider event, outbox, River dead-letter, and audit
   records before retrying anything.
4. Communicate only the factual status: submitted, pending, settled, reversed,
   or unknown. Never promise cancellation or receipt before confirmation.

## Duplicate debit or buyer says they already paid

- Freeze new collection attempts for the obligation.
- Compare payment references, provider events, reservations, and ledger
  postings.
- If the provider debit is unknown, reconcile by external reference before any
  retry.
- Record an approved reversal or adjustment command when evidence supports it;
  preserve the original event and notify both parties.

## Missing settlement or provider outage

- Keep collection state as pending/unknown until provider reconciliation.
- Inspect webhook backlog, signatures, provider latency, and settlement events.
- Retry only according to the provider retry class and attempt budget.
- Escalate settlement delays with the provider reference and safe payload hash.

## Mandate mismatch or cancellation

- Treat provider state as authoritative for the mandate.
- Block new credit/drawdowns when inactive or cancelled while owing.
- Reconcile out-of-order events by provider event ID and append a normalised
  mandate event.

## Ledger mismatch

- Stop affected financial actions.
- Run `cmd/reconcile` and capture the aggregate report.
- Compare the affected transaction postings, outbox event, and domain
  projection; use a forward correction command, never a destructive edit.
- Require finance/compliance approval for any adjustment.

## Queue backlog or stuck job

- Inspect the queue, retry budget, timeout, and dead-letter record.
- Do not replay a job until its idempotency key and provider reference are
  understood.
- Retry through the controlled admin command with a reason and audit event.

## Object-storage malware or document incident

- Quarantine the document and block downloads.
- Preserve checksum, scan result, uploader, purpose, and audit evidence.
- Notify the affected tenant with a factual status and replace only through a
  new upload/version.

## Account takeover, settlement fraud, or data breach

- Revoke sessions and provider credentials; place a risk hold.
- Preserve logs, audit events, traces, and access reasons without copying raw
  restricted values into tickets.
- Require dual approval for settlement changes and notify the incident owner,
  security, privacy, and legal contacts under the approved escalation plan.

## Privacy request and complaint

- Verify the requester without exposing unrelated tenant data.
- Locate source events and retention/legal holds before restricting or
  pseudonymising data.
- Record the decision, deadline, reviewer, and communication in the support
  case.
