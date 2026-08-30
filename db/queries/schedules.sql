-- name: CreateRepaymentSchedule :one
INSERT INTO app.repayment_schedules (obligation_id, schedule_type, timezone, allocation_policy, cadence, grace_hours, status) VALUES ($1,$2,$3,$4,$5,$6,'ACTIVE') RETURNING *;

-- name: CreateScheduleItem :one
INSERT INTO app.schedule_items (schedule_id, sequence, principal_due_kobo, due_at, grace_hours, collection_at, state) VALUES ($1,$2,$3,$4,$5,$6,'OPEN') RETURNING *;

-- name: ListScheduleItems :many
SELECT * FROM app.schedule_items WHERE schedule_id = $1 ORDER BY sequence;

-- name: ListCollectionEligibleItems :many
SELECT * FROM app.schedule_items WHERE state IN ('OVERDUE','PARTIALLY_PAID') AND collection_at <= $1 ORDER BY collection_at, sequence;
