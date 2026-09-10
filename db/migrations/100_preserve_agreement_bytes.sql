-- +goose Up
-- JSONB preserves JSON values, not the exact bytes a buyer accepted.
ALTER TABLE app.agreement_versions ADD COLUMN canonical_bytes bytea;
ALTER TABLE app.agreement_versions ADD CONSTRAINT agreement_canonical_bytes_match
 CHECK(canonical_bytes IS NULL OR (
   encode(public.digest(canonical_bytes,'sha256'),'hex')=document_hash
   AND convert_from(canonical_bytes,'UTF8')::jsonb=canonical_json));

-- +goose Down
-- Original accepted bytes are evidence and must not be discarded by rollback.
-- +goose StatementBegin
DO $$ BEGIN RAISE EXCEPTION 'accepted agreement bytes require a forward migration'; END $$;
-- +goose StatementEnd
