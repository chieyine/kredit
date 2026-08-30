# Protected operations controls

Owner: platform operations. Risk holds additionally require compliance-reviewer
or platform-administrator authority. Escalate financial ambiguity to the
payments/provider incident commander; escalate suspected account compromise to
security and compliance.

## Universal prerequisites

Use only the Operations **Controls**, **Jobs**, or **Provider events** screen.
Never update application, River, inbox, collection, user, or organization rows
by hand. Before applying any command:

1. Confirm the exact target and current version from the protected screen.
2. Record a case-specific reason of at least eight characters; do not include
   credentials, raw provider payloads, or unnecessary personal data.
3. Complete MFA within 15 minutes and review the displayed impact preview.
4. Keep the generated command ID and redacted correlation ID in the incident.
5. Confirm the affected-user notification and immutable audit event exist.

All commands require an idempotency key. Retrying the same request with the same
key returns its original result. A stale version or changed payload must be
investigated rather than forced.

## Command procedures

| Situation | Command | Expected result | Rollback or mitigation | Evidence |
| --- | --- | --- | --- | --- |
| Terminal job failure | `retry_job` | the same River job becomes available within its existing retry/idempotency boundary | quarantine and escalate if it fails again; compensate through the owning domain, never delete ledger facts | command ID, job ID/state/attempt, error class |
| Verified failed webhook | `retry_webhook` | the existing provider/event identity becomes receivable and its terminal job is requeued | stop replay and reconcile with the provider; never submit a replacement event ID | provider, event ID, signature status, before/after state |
| Compromised user | `suspend_user` | login blocked and active sessions revoked | after investigation use `restore_user` with the active suspension ID; prior status is restored | user ID, case, notification, suspension ID |
| Unsafe supplier workspace | `suspend_organization` | sensitive organization mutations blocked | use `restore_organization`; prior status is restored | organization/suspension IDs, affected scopes |
| Buyer/supplier risk | `place_risk_hold` | selected credit/release/collection/settlement scope blocked until expiry | use `lift_risk_hold` after documented review; expiry is automatic | target, scope, expiry, reviewer reason |
| Provider ambiguity | `request_reconciliation` | tracked reconciliation case created | resolve from authoritative provider evidence; do not infer success from timeout | case/attempt/provider references and age |
| Unknown collection | `resolve_unknown_submission` | provider is queried and only its authoritative idempotent result is applied | if still unknown, leave quarantined and escalate to provider | old/new state and provider reference |
| Retryable collection failure | `retry_collection` | one bounded retry is submitted; payment provider-reference uniqueness remains active | reconcile before any further retry; reverse only through the payment reversal runbook | attempt IDs, amounts, retry classification |
| Pending provider collection | `cancel_collection` | local cancellation occurs only after a reversal-capable provider confirms it | if not confirmed, state stays pending/unknown and must be reconciled | capability snapshot and provider confirmation |

## Diagnostics and escalation

Use **Diagnostics** with a bounded 5–1,440 minute window. Attach provider
latency, errors/timeouts, duplicate/out-of-order counts, oldest webhook and
queue age, unknown/reconciliation counts, drift signals, dead letters,
notification/scanner backlog, and mandate/settlement mismatch. Correlation IDs
must remain redacted in tickets and chat.

Escalate immediately for ledger drift, a duplicate financial effect, settlement
mismatch after its expected time, cancellation without provider confirmation,
or any command/audit discrepancy. Preserve all records; mitigation is a new
domain event or explicit lift/restore command, never deletion.
