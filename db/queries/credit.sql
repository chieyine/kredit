-- name: CreateCreditRequest :one
INSERT INTO app.credit_requests (supplier_organization_id, buyer_user_id, buyer_business_id, principal_kobo, currency, goods_description, invoice_reference, invoice_document_hash, due_date, grace_hours, collection_at, created_by) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12) RETURNING *;

-- name: GetCreditRequest :one
SELECT * FROM app.credit_requests WHERE id = $1;

-- name: ListCreditRequestsForSupplier :many
SELECT * FROM app.credit_requests WHERE supplier_organization_id = $1 ORDER BY created_at DESC;

-- name: CreateAgreementVersion :one
INSERT INTO app.agreement_versions (credit_request_id, version, canonical_json, document_hash, terms_version, privacy_version, created_by) VALUES ($1,$2,$3,$4,$5,$6,$7) RETURNING *;

-- name: CreateObligation :one
INSERT INTO app.obligations (credit_request_id, agreement_version_id, supplier_organization_id, buyer_business_id, principal_kobo, currency, lifecycle_status, payment_status, outstanding_kobo, base_fee_kobo, ledger_transaction_id, activated_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12) RETURNING *;
