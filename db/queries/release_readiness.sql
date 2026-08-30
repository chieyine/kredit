-- name: RecordReleaseEvidence :one
INSERT INTO app.release_evidence (gate_name, reference, reviewed_by, reviewed_at, state, notes)
VALUES ($1, $2, $3, $4, $5, $6) RETURNING *;

-- name: CreatePilotLimitConfig :one
INSERT INTO app.pilot_limit_configs (version, max_supplier_organizations, max_buyer_businesses, max_principal_kobo, max_active_exposure_kobo, max_drawdowns_per_line_day, max_collection_retries, enhanced_review_kobo, allowed_provider_accounts, allowed_industries, enabled, approved_by, approved_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13) RETURNING *;

-- name: ListReleaseEvidence :many
SELECT * FROM app.release_evidence ORDER BY created_at DESC;
