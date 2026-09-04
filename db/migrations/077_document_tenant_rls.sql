-- +goose Up

-- Document metadata is tenant data. The API now establishes transaction-local
-- identity for every insert, quota check, completion, lookup, and download;
-- autonomous scan jobs retain the distinct worker-only path.
DROP POLICY IF EXISTS document_runtime_access ON app.documents;
CREATE POLICY document_tenant_or_worker ON app.documents
  USING (
    current_user = 'kredit_worker'
    OR uploaded_by = app.current_user_id()
    OR organization_id::text = NULLIF(current_setting('app.current_organization_id', true), '')
  )
  WITH CHECK (
    current_user = 'kredit_worker'
    OR uploaded_by = app.current_user_id()
    OR organization_id::text = NULLIF(current_setting('app.current_organization_id', true), '')
  );

-- +goose Down
DROP POLICY IF EXISTS document_tenant_or_worker ON app.documents;
CREATE POLICY document_runtime_access ON app.documents
  USING (current_user IN ('kredit_app','kredit_worker'))
  WITH CHECK (current_user IN ('kredit_app','kredit_worker'));
