-- +goose Up
-- Resolve authorized business IDs before scanning credit/history, so delegated
-- purchasing grows with that person's memberships rather than the whole network.
CREATE INDEX IF NOT EXISTS credit_requests_purchasing_business_idx ON app.credit_requests(buyer_business_id,updated_at DESC,id);
CREATE INDEX IF NOT EXISTS credit_requests_purchasing_owner_idx ON app.credit_requests(buyer_user_id,updated_at DESC,id);
CREATE INDEX IF NOT EXISTS obligations_purchasing_business_idx ON app.obligations(buyer_business_id,activated_at,id);
-- +goose Down
DROP INDEX IF EXISTS app.obligations_purchasing_business_idx;
DROP INDEX IF EXISTS app.credit_requests_purchasing_owner_idx;
DROP INDEX IF EXISTS app.credit_requests_purchasing_business_idx;
