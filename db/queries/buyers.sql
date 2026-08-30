-- name: CreateBuyerInvitation :one
INSERT INTO app.buyer_invitations (organization_id, target_type, target_hash, token_hash, proposed_legal_name, proposed_trading_name, proposed_business_type, proposed_address, proposed_industry, created_by, expires_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
RETURNING id, organization_id, target_type, proposed_legal_name, proposed_trading_name, proposed_business_type, proposed_address, proposed_industry, status, created_by, created_at, expires_at;

-- name: GetBuyerInvitationByTokenHash :one
SELECT id, organization_id, target_type, target_hash, token_hash, proposed_legal_name, proposed_trading_name, proposed_business_type, proposed_address, proposed_industry, status, lookup_attempts, created_by, created_at, expires_at, accepted_at, accepted_by_user_id
FROM app.buyer_invitations
WHERE token_hash = $1;

-- name: AcceptBuyerInvitation :one
UPDATE app.buyer_invitations
SET status = 'accepted', accepted_at = NOW(), accepted_by_user_id = $2
WHERE id = $1 AND status = 'pending' AND expires_at > NOW()
RETURNING id, organization_id, target_type, status, accepted_at, accepted_by_user_id;

-- name: CreatePerson :one
INSERT INTO app.persons (user_id, full_name, status)
VALUES ($1, $2, $3)
RETURNING id, user_id, full_name, status, created_at;

-- name: CreateBusiness :one
INSERT INTO app.businesses (owner_user_id, legal_name, trading_name, business_type, business_address, industry, status)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING id, owner_user_id, legal_name, trading_name, business_type, business_address, industry, status, created_at;

-- name: CreateRepresentative :one
INSERT INTO app.business_representatives (business_id, person_id, role_title, authority_type, authority_verification_status)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, business_id, person_id, role_title, authority_type, authority_verification_status, created_at;

-- name: ListVerificationCasesForSubject :many
SELECT id, subject_type, subject_id, provider, provider_reference, verification_level, state, reasons, safe_result, started_at, completed_at, expires_at
FROM app.verification_cases
WHERE subject_id = ANY($1::uuid[])
ORDER BY started_at ASC, id ASC;

