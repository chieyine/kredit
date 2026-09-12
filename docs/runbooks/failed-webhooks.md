# Failed webhooks

Owner: provider operations. Verify the provider signature, event ID, and
inbox status before retrying. Duplicate event IDs are safe to acknowledge;
unsupported event types are rejected before inbox persistence and receive a non-success response. Investigate provider delivery failures; there is no saved inbox entry to replay for an unsupported type.

Replay only with `retry_webhook` so the original provider/event identity,
uniqueness and idempotency checks run. Record a reason, verify the current
attempt/version, complete recent MFA, review the impact preview, and attach the
immutable command ID, redacted correlation ID and resulting state transition.
Never accept an unsigned payload or mutate a payment directly. If a replay
caused an incorrect state, use the documented reversal/compensation flow.

Mono dispute-initiated and reversal-completed notices are supported and saved. The worker correlates the original debit and checks Mono's authoritative result. If it conflicts with an already recognized payment, the job remains failed for financial review; receiving a notice does not reverse the ledger. Preserve the provider reference and use the approved payment-reversal workflow after checking actual bank movement. Never acknowledge a reversal as resolved solely because its webhook was received.
