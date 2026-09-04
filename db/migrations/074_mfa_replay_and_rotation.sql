-- +goose Up

-- A TOTP counter may be accepted at most once. The application locks this row
-- while verifying the code and rotating the authenticated session.
ALTER TABLE app.mfa_methods
    ADD COLUMN last_used_counter BIGINT;

-- +goose Down
ALTER TABLE app.mfa_methods
    DROP COLUMN IF EXISTS last_used_counter;
