# Failed jobs

Owner: on-call operations. Escalate to engineering when a job is financial,
non-idempotent, or fails three times.

Before acting, confirm the job ID, queue, attempt/version, and terminal error in
the operations console. Do not edit a River row or replay a financial job by
hand. Use `retry_job`, record a structured reason, complete recent MFA, and
confirm its impact preview. The command preserves the original job identity,
idempotency boundary, immutable command/audit history, and affected-user
notification. If replay is unsafe, quarantine the job and open an incident.
Rollback is the compensating domain operation, never deleting the original job
or ledger event. Attach command ID, redacted correlation ID, before/after state,
and retry outcome.
