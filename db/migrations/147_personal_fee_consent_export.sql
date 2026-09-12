-- +goose Up
-- Personal exports include the consent a person gave, not company-wide bills.
CREATE POLICY fee_authorization_consent_read ON app.fee_authorizations FOR SELECT USING(created_by=app.current_user_id());
-- +goose Down
DROP POLICY fee_authorization_consent_read ON app.fee_authorizations;
