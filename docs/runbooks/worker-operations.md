# Worker operations

## Queue contract

River workers are separated into `critical-financial`, `provider-webhooks`,
`collections`, `reconciliation`, `notifications`, `documents`, `reports`, and
`maintenance` queues. Each job class has a stable kind, unique arguments, and a
bounded retry budget.

## Triage

1. Check queue depth and oldest scheduled job in the River dashboard/database.
2. Inspect `app.job_dead_letters` for terminal failures.
3. Check `app.provider_webhook_inbox` for `failed` or long-running `received`
   events.
4. For financial failures, stop new collection activity and run the ledger
   reconciliation job before retrying.
5. Replay only after confirming the operation is idempotent and the underlying
   provider state is known.

## Provider webhook recovery

Webhook signatures are validated before the event is accepted. The unique
`(provider, event_id)` key prevents duplicate processing. A failed event remains
visible in the inbox with its last error and can be retried after the provider
state is reconciled.

## Dead-letter recovery

Dead letters are operational records, not automatic permission to retry. The
operator must record the incident/case reference, verify the resource state,
and retry through a controlled admin action. Never edit ledger rows directly.
