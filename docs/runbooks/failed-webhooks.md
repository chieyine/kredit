# Failed webhooks

Owner: provider operations. Verify the provider signature, event ID, and
inbox status before retrying. Duplicate event IDs are safe to acknowledge;
unknown events remain quarantined until the provider contract is confirmed.

Replay only with `retry_webhook` so the original provider/event identity,
uniqueness and idempotency checks run. Record a reason, verify the current
attempt/version, complete recent MFA, review the impact preview, and attach the
immutable command ID, redacted correlation ID and resulting state transition.
Never accept an unsigned payload or mutate a payment directly. If a replay
caused an incorrect state, use the documented reversal/compensation flow.
