-- name: CreateOrganization :one
INSERT INTO app.organizations (legal_name, trading_name, business_type, registration_info, business_address, industry, default_timezone, default_currency)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING id, legal_name, trading_name, business_type, registration_info, business_address, industry, status, default_timezone, default_currency, created_at, updated_at, version;

-- name: CreateMembership :one
INSERT INTO app.memberships (organization_id, user_id, role, status, invited_by, invited_at, accepted_at)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING id, organization_id, user_id, role, status, invited_by, invited_at, accepted_at, created_at;

-- name: ListOrganizationsForUser :many
SELECT organization.id, organization.legal_name, organization.trading_name, organization.business_type, organization.registration_info, organization.business_address, organization.industry, organization.status, organization.default_timezone, organization.default_currency, organization.created_at, organization.updated_at, organization.version
FROM app.organizations organization
JOIN app.memberships membership ON membership.organization_id = organization.id
WHERE membership.user_id = $1
  AND membership.status IN ('invited', 'active', 'suspended')
ORDER BY organization.created_at DESC, organization.id DESC;

-- name: ListMembers :many
SELECT id, organization_id, user_id, role, status, invited_by, invited_at, accepted_at, created_at
FROM app.memberships
WHERE organization_id = $1
ORDER BY created_at ASC, id ASC;

-- name: ChangeMembershipRole :one
UPDATE app.memberships
SET role = $2
WHERE organization_id = $1
  AND user_id = $3
  AND status = 'active'
  AND role <> 'owner'
RETURNING id, organization_id, user_id, role, status, invited_by, invited_at, accepted_at, created_at;

