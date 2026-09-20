-- +goose Up
-- Track 5: Reviewed proposed credit terms and opening-balance imports.
-- Staged terms and opening balances create NO debt, active credit limit, or bank debit mandate.
-- Independent review/approval is enforced before any invitation with staged terms is dispatched.

CREATE TABLE IF NOT EXISTS app.partner_terms_import_batches (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id uuid NOT NULL REFERENCES app.organizations(id),
  source_hash text NOT NULL CHECK(source_hash ~ '^[0-9a-f]{64}$'),
  payload_hash bytea NOT NULL CHECK(octet_length(payload_hash) = 32),
  payload_ciphertext bytea NOT NULL,
  total_rows integer NOT NULL CHECK(total_rows BETWEEN 1 AND 500),
  valid_rows integer NOT NULL DEFAULT 0,
  invalid_rows integer NOT NULL DEFAULT 0,
  state text NOT NULL DEFAULT 'staged' CHECK(state IN ('staged', 'reviewing', 'approved', 'completed', 'cancelled')),
  uploaded_by uuid NOT NULL REFERENCES app.users(id),
  approved_by uuid REFERENCES app.users(id),
  cancelled_by uuid REFERENCES app.users(id),
  created_at timestamptz NOT NULL DEFAULT now(),
  approved_at timestamptz,
  cancelled_at timestamptz,
  UNIQUE(organization_id, source_hash),
  CHECK((approved_by IS NULL) = (approved_at IS NULL)),
  CHECK((cancelled_by IS NULL) = (cancelled_at IS NULL)),
  CHECK(state NOT IN ('approved', 'completed') OR (approved_by IS NOT NULL AND approved_by <> uploaded_by)),
  CHECK(state <> 'cancelled' OR cancelled_by IS NOT NULL)
);

CREATE TABLE IF NOT EXISTS app.partner_terms_import_rows (
  id uuid DEFAULT gen_random_uuid() UNIQUE,
  batch_id uuid NOT NULL REFERENCES app.partner_terms_import_batches(id) ON DELETE CASCADE,
  row_number integer NOT NULL CHECK(row_number BETWEEN 1 AND 500),
  customer_name text NOT NULL,
  contact_email text NOT NULL DEFAULT '',
  contact_phone text NOT NULL DEFAULT '',
  proposed_credit_limit_kobo bigint NOT NULL DEFAULT 0 CHECK(proposed_credit_limit_kobo >= 0),
  proposed_grace_hours integer NOT NULL DEFAULT 0 CHECK(proposed_grace_hours >= 0 AND proposed_grace_hours <= 720),
  opening_balance_kobo bigint NOT NULL DEFAULT 0 CHECK(opening_balance_kobo >= 0),
  opening_balance_reference text NOT NULL DEFAULT '',
  validation_status text NOT NULL DEFAULT 'valid' CHECK(validation_status IN ('valid', 'invalid', 'duplicate')),
  validation_error text NOT NULL DEFAULT '',
  invitation_id uuid REFERENCES app.buyer_invitations(id),
  created_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY(batch_id, row_number)
);

ALTER TABLE app.partner_terms_import_batches ENABLE ROW LEVEL SECURITY;
ALTER TABLE app.partner_terms_import_batches FORCE ROW LEVEL SECURITY;
ALTER TABLE app.partner_terms_import_rows ENABLE ROW LEVEL SECURITY;
ALTER TABLE app.partner_terms_import_rows FORCE ROW LEVEL SECURITY;

CREATE POLICY partner_terms_batch_authority ON app.partner_terms_import_batches USING (
  organization_id = app.current_organization_id() AND EXISTS (
    SELECT 1 FROM app.memberships m JOIN app.users u ON u.id = m.user_id
    WHERE m.organization_id = partner_terms_import_batches.organization_id AND m.user_id = app.current_user_id()
    AND m.status = 'active' AND m.role IN ('owner', 'administrator', 'sales', 'finance') AND u.status = 'active'
  )
);

CREATE POLICY partner_terms_row_authority ON app.partner_terms_import_rows USING (
  EXISTS(SELECT 1 FROM app.partner_terms_import_batches b WHERE b.id = partner_terms_import_rows.batch_id)
);

CREATE POLICY branch_boundary ON app.partner_terms_import_batches AS RESTRICTIVE FOR ALL USING(
  app.branch_scope_all(organization_id)
);

GRANT SELECT, INSERT, UPDATE ON app.partner_terms_import_batches TO kredit_app, kredit_worker;
GRANT SELECT, INSERT, UPDATE ON app.partner_terms_import_rows TO kredit_app, kredit_worker;

-- +goose Down
DROP TABLE IF EXISTS app.partner_terms_import_rows;
DROP TABLE IF EXISTS app.partner_terms_import_batches;
