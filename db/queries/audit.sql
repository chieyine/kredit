-- name: AppendAuditEvent :one
INSERT INTO app.audit_events (actor_user_id, organization_id, action, resource_type, resource_id, outcome, severity, request_id, metadata)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING id, occurred_at, actor_user_id, organization_id, action, resource_type, resource_id, outcome, severity, request_id, metadata;

-- name: ListAuditEventsForOrganization :many
SELECT id, occurred_at, actor_user_id, organization_id, action, resource_type, resource_id, outcome, severity, request_id, metadata
FROM app.audit_events
WHERE organization_id = $1
ORDER BY occurred_at DESC, id DESC
LIMIT $2;

