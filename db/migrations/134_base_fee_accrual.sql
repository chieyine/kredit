-- +goose Up
-- A base fee belongs to one activated obligation, independently of repayments.
CREATE UNIQUE INDEX fee_once_per_obligation ON app.fees(obligation_id,fee_type) WHERE fee_type='base_service';
-- +goose Down
DROP INDEX app.fee_once_per_obligation;
