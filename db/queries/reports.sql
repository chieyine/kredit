-- name: ListReceivablesForOrganization :many
SELECT o.id AS obligation_id, o.supplier_organization_id, o.buyer_business_id,
       o.principal_kobo, o.outstanding_kobo, o.base_fee_kobo,
       o.payment_status, o.activated_at, cr.due_date, cr.grace_hours
FROM app.obligations o
JOIN app.credit_requests cr ON cr.obligation_id = o.id
WHERE o.supplier_organization_id = $1
ORDER BY o.activated_at ASC;

-- name: ListCorrectionRequestsForOrganization :many
SELECT * FROM app.correction_requests
WHERE organization_id = $1
ORDER BY created_at DESC;

-- name: CreateCorrectionRequest :one
INSERT INTO app.correction_requests (organization_id, subject_type, subject_id, source_event_id, requested_by, reason, evidence, state)
VALUES ($1, $2, $3, $4, $5, $6, $7, 'OPEN') RETURNING *;

-- name: CreateAnalyticsEvent :one
INSERT INTO app.analytics_events (name, subject_id_hash, purpose, metadata)
VALUES ($1, $2, $3, $4) RETURNING *;
