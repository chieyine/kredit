-- +goose Up
ALTER TABLE app.buyer_invitations ADD COLUMN accepted_business_id uuid REFERENCES app.businesses(id);

-- +goose Down
-- Retain the exact accepted business so recovery never selects another account.
SELECT 1;
