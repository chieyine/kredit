-- +goose Up
-- Imported invitations retain a stable source identity and encrypted link so
-- refreshing or retrying the same roster never creates another invitation.
ALTER TABLE app.buyer_invitations ADD COLUMN source_reference text;
ALTER TABLE app.buyer_invitations ADD COLUMN source_fingerprint bytea;
ALTER TABLE app.buyer_invitations ADD COLUMN token_ciphertext bytea;
ALTER TABLE app.buyer_invitations ADD CONSTRAINT invitation_import_complete CHECK (
 (source_reference IS NULL AND source_fingerprint IS NULL AND token_ciphertext IS NULL)
 OR (source_reference IS NOT NULL AND source_fingerprint IS NOT NULL AND token_ciphertext IS NOT NULL AND length(source_reference) BETWEEN 1 AND 128 AND octet_length(source_fingerprint)=32 AND octet_length(token_ciphertext)>0)
);
CREATE UNIQUE INDEX invitation_import_reference ON app.buyer_invitations(organization_id,source_reference) WHERE source_reference IS NOT NULL;

-- +goose Down
-- Keep import identities: removing them would make a replay create duplicates.
SELECT 1;
