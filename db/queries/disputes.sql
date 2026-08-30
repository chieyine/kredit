-- name: CreateDispute :one
INSERT INTO app.disputes (obligation_id, supplier_organization_id, buyer_user_id, opened_by, total_disputed_kobo, remaining_disputed_kobo, reason, explanation, state, collection_effect) VALUES ($1,$2,$3,$4,$5,$5,$6,$7,'OPEN',$8) RETURNING *;

-- name: CreateDisputeEvidence :one
INSERT INTO app.dispute_evidence (dispute_id, submitted_by, document_id, statement) VALUES ($1,$2,$3,$4) RETURNING *;

-- name: CreateDisputeDecision :one
INSERT INTO app.dispute_decisions (dispute_id, reviewer_id, outcome, valid_principal_kobo, adjustment_kobo, remaining_disputed_kobo, reason) VALUES ($1,$2,$3,$4,$5,$6,$7) RETURNING *;

-- name: ListDisputesForSupplier :many
SELECT * FROM app.disputes WHERE supplier_organization_id = $1 ORDER BY opened_at DESC;
