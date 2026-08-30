-- +goose Up
CREATE TABLE IF NOT EXISTS app.platform_role_assignments (
  id uuid PRIMARY KEY DEFAULT uuidv7(),
  user_id uuid NOT NULL REFERENCES app.users(id),
  role text NOT NULL CHECK (role IN ('support_agent','compliance_reviewer','dispute_reviewer','platform_admin')),
  granted_by uuid NOT NULL REFERENCES app.users(id),
  reason text NOT NULL CHECK (length(trim(reason)) >= 8),
  granted_at timestamptz NOT NULL DEFAULT now(),
  expires_at timestamptz,
  revoked_at timestamptz,
  revoked_by uuid REFERENCES app.users(id),
  CHECK (expires_at IS NULL OR expires_at > granted_at)
);
CREATE UNIQUE INDEX IF NOT EXISTS platform_role_active_unique
  ON app.platform_role_assignments(user_id, role) WHERE revoked_at IS NULL;
CREATE INDEX IF NOT EXISTS platform_role_user_active_idx
  ON app.platform_role_assignments(user_id, expires_at) WHERE revoked_at IS NULL;
ALTER TABLE app.platform_role_assignments ENABLE ROW LEVEL SECURITY;
CREATE POLICY platform_role_runtime_access ON app.platform_role_assignments
  USING (current_user IN ('kredit_app','kredit_worker'))
  WITH CHECK (current_user IN ('kredit_app','kredit_worker'));

-- +goose Down
DROP TABLE IF EXISTS app.platform_role_assignments;
