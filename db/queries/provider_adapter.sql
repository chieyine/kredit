-- name: CreateProviderApproval :one
INSERT INTO app.provider_approvals (provider_name, written_reference, approved_by, approved_at, allowed_capabilities, pilot_limit_kobo, state)
VALUES ($1, $2, $3, $4, $5, $6, 'approved') RETURNING *;

-- name: RecordProviderEventFromAdapter :one
INSERT INTO app.provider_events (provider_name, provider_event_id, external_reference, event_type, payload_hash, state, settlement_state)
VALUES ($1, $2, $3, $4, $5, $6, $7)
ON CONFLICT (provider_name, provider_event_id) DO NOTHING RETURNING *;

-- name: RecordProviderReconciliation :one
INSERT INTO app.provider_reconciliation_events (provider_name, provider_collection_id, state, settlement_reference, amount_kobo)
VALUES ($1, $2, $3, $4, $5) RETURNING *;
