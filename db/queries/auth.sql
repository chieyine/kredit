-- name: GetUserByEmail :one
SELECT id, normalized_email, normalized_phone, display_name, status, last_authenticated_at, created_at
FROM app.users
WHERE normalized_email = $1;

-- name: GetUserByPhone :one
SELECT id, normalized_email, normalized_phone, display_name, status, last_authenticated_at, created_at
FROM app.users
WHERE normalized_phone = $1;

-- name: GetSessionByTokenHash :one
SELECT id, user_id, device_label, ip_metadata, user_agent, authentication_level, created_at, expires_at, revoked_at
FROM app.sessions
WHERE token_hash = $1;

-- name: ConsumeOTPChallenge :one
UPDATE app.otp_challenges
SET consumed_at = NOW()
WHERE id = $1
  AND consumed_at IS NULL
  AND expires_at > NOW()
  AND attempt_count < 5
RETURNING id, target_type, target_hash, purpose, code_hmac, attempt_count, expires_at, consumed_at;

