-- +goose Up
CREATE TABLE app.distributor_import_batches (
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
 organization_id uuid NOT NULL REFERENCES app.organizations(id),
 source_hash text NOT NULL CHECK(source_hash ~ '^[0-9a-f]{64}$'),
 payload_hash bytea NOT NULL CHECK(octet_length(payload_hash)=32),
 payload_ciphertext bytea NOT NULL,
 row_count integer NOT NULL CHECK(row_count BETWEEN 1 AND 200),
 state text NOT NULL DEFAULT 'draft' CHECK(state IN ('draft','approved','completed','cancelled')),
 created_by uuid NOT NULL REFERENCES app.users(id),
 approved_by uuid REFERENCES app.users(id),
 cancelled_by uuid REFERENCES app.users(id),
 created_at timestamptz NOT NULL DEFAULT now(),
 approved_at timestamptz,
 cancelled_at timestamptz,
 UNIQUE(organization_id,source_hash),
 CHECK((approved_by IS NULL)=(approved_at IS NULL)),
 CHECK((cancelled_by IS NULL)=(cancelled_at IS NULL)),
 CHECK(state NOT IN ('approved','completed') OR approved_by IS NOT NULL),
 CHECK(state <> 'cancelled' OR cancelled_by IS NOT NULL)
);
CREATE TABLE app.distributor_import_rows (
 batch_id uuid NOT NULL REFERENCES app.distributor_import_batches(id),
 row_number integer NOT NULL CHECK(row_number BETWEEN 1 AND 200),
 invitation_id uuid NOT NULL REFERENCES app.buyer_invitations(id),
 created_by uuid NOT NULL REFERENCES app.users(id),
 created_at timestamptz NOT NULL DEFAULT now(),
 PRIMARY KEY(batch_id,row_number)
);
ALTER TABLE app.distributor_import_batches ENABLE ROW LEVEL SECURITY;
ALTER TABLE app.distributor_import_batches FORCE ROW LEVEL SECURITY;
ALTER TABLE app.distributor_import_rows ENABLE ROW LEVEL SECURITY;
ALTER TABLE app.distributor_import_rows FORCE ROW LEVEL SECURITY;
CREATE POLICY distributor_import_batch_authority ON app.distributor_import_batches USING (
 organization_id=app.current_organization_id() AND EXISTS (
 SELECT 1 FROM app.memberships m JOIN app.users u ON u.id=m.user_id
 WHERE m.organization_id=distributor_import_batches.organization_id AND m.user_id=app.current_user_id()
 AND m.status='active' AND m.role IN ('owner','administrator','sales') AND u.status='active'
 ));
CREATE POLICY distributor_import_row_authority ON app.distributor_import_rows USING (
 EXISTS(SELECT 1 FROM app.distributor_import_batches b WHERE b.id=distributor_import_rows.batch_id));
-- +goose StatementBegin
DO $$ BEGIN IF EXISTS(SELECT 1 FROM pg_roles WHERE rolname='kredit_app') THEN
 GRANT SELECT,INSERT,UPDATE ON app.distributor_import_batches TO kredit_app;
 GRANT SELECT,INSERT ON app.distributor_import_rows TO kredit_app;
END IF; END $$;
-- +goose StatementEnd

-- +goose Down
-- Durable source identities must survive deployment rollback.
-- +goose StatementBegin
DO $$ BEGIN RAISE EXCEPTION 'Import batch history requires forward recovery'; END $$;
-- +goose StatementEnd
