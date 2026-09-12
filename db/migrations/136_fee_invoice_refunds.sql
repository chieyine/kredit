-- +goose Up
ALTER TABLE app.fee_invoice_receipts ADD COLUMN direction text NOT NULL DEFAULT 'received' CHECK(direction IN ('received','refunded'));
-- +goose Down
-- +goose StatementBegin
DO $$ BEGIN RAISE EXCEPTION 'Fee refund evidence requires forward recovery'; END $$;
-- +goose StatementEnd
