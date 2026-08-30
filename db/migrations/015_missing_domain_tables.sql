-- +goose Up
CREATE TABLE IF NOT EXISTS app.relationship_consents (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    buyer_user_id uuid NOT NULL REFERENCES app.users(id),
    supplier_organization_id uuid NOT NULL REFERENCES app.organizations(id),
    consent_type text NOT NULL,
    version text NOT NULL,
    evidence_hash text NOT NULL,
    granted boolean NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS app.documents (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    organization_id uuid REFERENCES app.organizations(id),
    uploaded_by uuid NOT NULL REFERENCES app.users(id),
    purpose text NOT NULL,
    object_key text NOT NULL UNIQUE,
    file_name text NOT NULL,
    content_type text NOT NULL,
    size_bytes bigint NOT NULL CHECK (size_bytes > 0),
    sha256 text NOT NULL,
    scan_state text NOT NULL CHECK (scan_state IN ('PENDING','CLEAN','REJECTED','QUARANTINED')),
    retention_class text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    scanned_at timestamptz
);

CREATE TABLE IF NOT EXISTS app.support_cases (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    organization_id uuid REFERENCES app.organizations(id),
    subject_type text NOT NULL,
    subject_id text NOT NULL,
    opened_by uuid NOT NULL REFERENCES app.users(id),
    state text NOT NULL CHECK (state IN ('OPEN','IN_PROGRESS','RESOLVED','CLOSED')),
    break_glass boolean NOT NULL DEFAULT false,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS app.support_case_events (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    case_id uuid NOT NULL REFERENCES app.support_cases(id),
    actor_id uuid NOT NULL REFERENCES app.users(id),
    action text NOT NULL,
    note text,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS app.idempotency_records (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    scope text NOT NULL,
    idempotency_key text NOT NULL,
    request_hash text NOT NULL,
    response_status integer,
    response_body jsonb,
    created_at timestamptz NOT NULL DEFAULT now(),
    completed_at timestamptz,
    expires_at timestamptz NOT NULL DEFAULT (now() + interval '24 hours'),
    UNIQUE (scope, idempotency_key)
);
CREATE INDEX IF NOT EXISTS idempotency_records_expiry_idx ON app.idempotency_records (expires_at);

ALTER TABLE app.relationship_consents ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS relationship_consent_buyer_access ON app.relationship_consents;
CREATE POLICY relationship_consent_buyer_access ON app.relationship_consents
    USING (buyer_user_id = app.current_user_id())
    WITH CHECK (buyer_user_id = app.current_user_id());
DROP POLICY IF EXISTS relationship_consent_supplier_access ON app.relationship_consents;
CREATE POLICY relationship_consent_supplier_access ON app.relationship_consents
    USING (supplier_organization_id IN (SELECT organization_id FROM app.memberships WHERE user_id = app.current_user_id() AND status = 'active'))
    WITH CHECK (supplier_organization_id IN (SELECT organization_id FROM app.memberships WHERE user_id = app.current_user_id() AND status = 'active'));
ALTER TABLE app.documents ENABLE ROW LEVEL SECURITY;
ALTER TABLE app.support_cases ENABLE ROW LEVEL SECURITY;
ALTER TABLE app.support_case_events ENABLE ROW LEVEL SECURITY;
ALTER TABLE app.idempotency_records ENABLE ROW LEVEL SECURITY;

-- +goose Down
DROP TABLE IF EXISTS app.idempotency_records;
DROP TABLE IF EXISTS app.support_case_events;
DROP TABLE IF EXISTS app.support_cases;
DROP TABLE IF EXISTS app.documents;
DROP TABLE IF EXISTS app.relationship_consents;
