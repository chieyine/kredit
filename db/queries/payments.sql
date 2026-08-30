-- name: CreatePayment :one
INSERT INTO app.payments (obligation_id, buyer_user_id, supplier_organization_id, source_type, amount_kobo, currency, provider, provider_reference, state, paid_at, recorded_by, idempotency_key) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,'recognized',$9,$10,$11) RETURNING *;

-- name: CreatePaymentAllocation :one
INSERT INTO app.payment_allocations (payment_id, obligation_id, amount_kobo, allocation_order) VALUES ($1,$2,$3,$4) RETURNING *;

-- name: ListPaymentsForObligation :many
SELECT * FROM app.payments WHERE obligation_id = $1 ORDER BY recognized_at DESC;

-- name: ListPaymentAllocations :many
SELECT * FROM app.payment_allocations WHERE payment_id = ANY($1::uuid[]);

-- name: CreateCollectionFee :one
INSERT INTO app.fees (supplier_organization_id, obligation_id, payment_id, fee_type, basis_amount_kobo, rate_basis_points, amount_kobo, currency, state) VALUES ($1,$2,$3,'collection',$4,50,$5,$6,'accrued') RETURNING *;
