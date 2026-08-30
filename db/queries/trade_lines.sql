-- name: CreateTradeLine :one
INSERT INTO app.trade_lines (supplier_organization_id, buyer_user_id, buyer_business_id, approved_limit_kobo, available_limit_kobo, cadence, default_grace_hours, start_at, end_at, state, mandate_id, mandate_active, terms_version) VALUES ($1,$2,$3,$4,$4,$5,$6,$7,$8,$9,$10,$11,$12) RETURNING *;

-- name: GetTradeLine :one
SELECT * FROM app.trade_lines WHERE id = $1;

-- name: ListTradeLinesForSupplier :many
SELECT * FROM app.trade_lines WHERE supplier_organization_id = $1 ORDER BY created_at DESC;

-- name: CreateDrawdownReservation :one
INSERT INTO app.drawdown_reservations (trade_line_id, drawdown_id, amount_kobo, state, expires_at, idempotency_key) VALUES ($1,$2,$3,'PENDING',$4,$5) RETURNING *;

-- name: ListDrawdowns :many
SELECT * FROM app.drawdowns WHERE trade_line_id = $1 ORDER BY created_at DESC;
