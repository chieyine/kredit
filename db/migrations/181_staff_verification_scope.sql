-- +goose Up
-- Staff may reuse their business's verification evidence, but cannot write its
-- KYB result or read another representative's personal identity evidence.
CREATE POLICY delegated_business_verification_read ON app.verification_cases FOR SELECT
USING(subject_type='business' AND app.can_purchase(subject_id));
-- +goose Down
DROP POLICY delegated_business_verification_read ON app.verification_cases;
