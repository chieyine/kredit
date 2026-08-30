-- name: CreateCollectionReservation :one
INSERT INTO app.collection_reservations (obligation_id, schedule_item_id, outstanding_snapshot_version, reserved_amount_kobo, state, expires_at, idempotency_key) VALUES ($1,$2,$3,$4,'PROCESSING',$5,$6) RETURNING *;

-- name: CreateCollectionAttempt :one
INSERT INTO app.collection_attempts (reservation_id, obligation_id, provider, external_reference, requested_amount_kobo, state, attempt_number) VALUES ($1,$2,$3,$4,$5,'PENDING',$6) RETURNING *;

-- name: ListCollectionAttempts :many
SELECT * FROM app.collection_attempts WHERE obligation_id = $1 ORDER BY requested_at DESC;

-- name: RecordProviderEvent :one
INSERT INTO app.collection_provider_events (provider, event_id, external_reference, payload_hash, state) VALUES ($1,$2,$3,$4,$5) RETURNING *;
