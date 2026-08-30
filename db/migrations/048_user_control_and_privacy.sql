-- +goose Up
ALTER TABLE app.notification_preferences
  ADD COLUMN payment_reminders_enabled BOOLEAN NOT NULL DEFAULT TRUE,
  ADD COLUMN product_updates_enabled BOOLEAN NOT NULL DEFAULT FALSE,
  ADD COLUMN version BIGINT NOT NULL DEFAULT 1 CHECK (version > 0);

CREATE TABLE app.account_recovery_codes (
  id UUID PRIMARY KEY DEFAULT uuidv7(),
  user_id UUID NOT NULL REFERENCES app.users(id) ON DELETE CASCADE,
  code_hash BYTEA NOT NULL,
  code_hint TEXT NOT NULL CHECK (length(code_hint) = 4),
  state TEXT NOT NULL DEFAULT 'ACTIVE' CHECK (state IN ('ACTIVE','USED','REVOKED','EXPIRED')),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  expires_at TIMESTAMPTZ NOT NULL,
  used_at TIMESTAMPTZ,
  UNIQUE(user_id, code_hash)
);
CREATE INDEX account_recovery_codes_active_idx ON app.account_recovery_codes(user_id, expires_at) WHERE state='ACTIVE';

CREATE TABLE app.account_recovery_rate_limits (
  fingerprint_hash BYTEA PRIMARY KEY,
  window_started_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  attempt_count INTEGER NOT NULL DEFAULT 1 CHECK (attempt_count > 0)
);

CREATE TABLE app.account_recovery_requests (
  id UUID PRIMARY KEY DEFAULT uuidv7(),
  target_user_id UUID NOT NULL REFERENCES app.users(id),
  state TEXT NOT NULL DEFAULT 'PENDING_VERIFICATION' CHECK (state IN ('PENDING_VERIFICATION','PENDING_REVIEW','COOLING_OFF','APPROVED','REJECTED','CANCELLED','EXPIRED','COMPLETED')),
  requested_channel TEXT NOT NULL CHECK (requested_channel IN ('email','phone')),
  request_fingerprint BYTEA NOT NULL,
  risk_facts JSONB NOT NULL DEFAULT '{}'::jsonb,
  attempt_count INTEGER NOT NULL DEFAULT 0 CHECK (attempt_count >= 0),
  independent_factor_count INTEGER NOT NULL DEFAULT 0 CHECK (independent_factor_count BETWEEN 0 AND 8),
  reviewer_user_id UUID REFERENCES app.users(id),
  review_reason TEXT,
  reviewed_at TIMESTAMPTZ,
  cooling_off_until TIMESTAMPTZ,
  completion_token_hash BYTEA,
  completed_at TIMESTAMPTZ,
  cancelled_at TIMESTAMPTZ,
  expires_at TIMESTAMPTZ NOT NULL DEFAULT (now() + interval '7 days'),
  version BIGINT NOT NULL DEFAULT 1 CHECK (version > 0),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CHECK (reviewer_user_id IS NULL OR reviewer_user_id <> target_user_id)
);
CREATE UNIQUE INDEX account_recovery_one_open_idx ON app.account_recovery_requests(target_user_id)
  WHERE state IN ('PENDING_VERIFICATION','PENDING_REVIEW','COOLING_OFF','APPROVED');
CREATE INDEX account_recovery_review_idx ON app.account_recovery_requests(state, created_at);

CREATE TABLE app.account_recovery_evidence (
  id UUID PRIMARY KEY DEFAULT uuidv7(),
  request_id UUID NOT NULL REFERENCES app.account_recovery_requests(id) ON DELETE CASCADE,
  factor_type TEXT NOT NULL CHECK (factor_type IN ('recovery_code','verified_email','verified_phone','existing_mfa','business_evidence','manual_identity')),
  evidence_hash BYTEA NOT NULL,
  verified_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE(request_id, factor_type)
);

CREATE TABLE app.account_recovery_events (
  id UUID PRIMARY KEY DEFAULT uuidv7(),
  request_id UUID NOT NULL REFERENCES app.account_recovery_requests(id) ON DELETE CASCADE,
  event_type TEXT NOT NULL,
  actor_user_id UUID REFERENCES app.users(id),
  actor_reference TEXT NOT NULL,
  metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE app.privacy_requests (
  id UUID PRIMARY KEY DEFAULT uuidv7(),
  requester_user_id UUID NOT NULL REFERENCES app.users(id),
  organization_id UUID REFERENCES app.organizations(id),
  request_type TEXT NOT NULL CHECK (request_type IN ('ACCESS','CORRECTION','DELETION','RESTRICTION','OBJECTION','CONSENT_WITHDRAWAL','PORTABILITY')),
  state TEXT NOT NULL DEFAULT 'RECEIVED' CHECK (state IN ('RECEIVED','IDENTITY_CHECK','IN_REVIEW','CLARIFICATION_REQUIRED','APPROVED','PARTIALLY_APPROVED','REJECTED','IN_PROGRESS','COMPLETED','CANCELLED')),
  identity_verified_at TIMESTAMPTZ,
  due_at TIMESTAMPTZ NOT NULL DEFAULT (now() + interval '30 days'),
  details TEXT NOT NULL DEFAULT '',
  decision_reason TEXT,
  retention_outcome TEXT,
  legal_hold_applies BOOLEAN NOT NULL DEFAULT FALSE,
  decided_by UUID REFERENCES app.users(id),
  second_approved_by UUID REFERENCES app.users(id),
  decided_at TIMESTAMPTZ,
  completed_at TIMESTAMPTZ,
  version BIGINT NOT NULL DEFAULT 1 CHECK (version > 0),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CHECK (second_approved_by IS NULL OR second_approved_by <> decided_by)
);
CREATE INDEX privacy_requests_requester_idx ON app.privacy_requests(requester_user_id, created_at DESC);
CREATE INDEX privacy_requests_review_idx ON app.privacy_requests(state, due_at);

CREATE TABLE app.privacy_request_events (
  id UUID PRIMARY KEY DEFAULT uuidv7(),
  request_id UUID NOT NULL REFERENCES app.privacy_requests(id) ON DELETE CASCADE,
  event_type TEXT NOT NULL,
  actor_user_id UUID REFERENCES app.users(id),
  actor_reference TEXT NOT NULL,
  metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE app.privacy_exports (
  id UUID PRIMARY KEY DEFAULT uuidv7(),
  request_id UUID NOT NULL UNIQUE REFERENCES app.privacy_requests(id) ON DELETE CASCADE,
  object_reference TEXT NOT NULL,
  content_sha256 TEXT NOT NULL CHECK (length(content_sha256)=64),
  payload JSONB NOT NULL,
  expires_at TIMESTAMPTZ NOT NULL,
  downloaded_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE app.processing_restrictions (
  id UUID PRIMARY KEY DEFAULT uuidv7(),
  user_id UUID NOT NULL REFERENCES app.users(id),
  privacy_request_id UUID NOT NULL REFERENCES app.privacy_requests(id),
  scope TEXT NOT NULL,
  reason TEXT NOT NULL,
  active BOOLEAN NOT NULL DEFAULT TRUE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  lifted_at TIMESTAMPTZ
);

CREATE TABLE app.legal_holds (
  id UUID PRIMARY KEY DEFAULT uuidv7(),
  user_id UUID NOT NULL REFERENCES app.users(id),
  organization_id UUID REFERENCES app.organizations(id),
  scope TEXT NOT NULL,
  reason TEXT NOT NULL,
  active BOOLEAN NOT NULL DEFAULT TRUE,
  created_by UUID NOT NULL REFERENCES app.users(id),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  released_at TIMESTAMPTZ
);

ALTER TABLE app.account_recovery_codes ENABLE ROW LEVEL SECURITY;
ALTER TABLE app.account_recovery_rate_limits ENABLE ROW LEVEL SECURITY;
ALTER TABLE app.account_recovery_requests ENABLE ROW LEVEL SECURITY;
ALTER TABLE app.account_recovery_evidence ENABLE ROW LEVEL SECURITY;
ALTER TABLE app.account_recovery_events ENABLE ROW LEVEL SECURITY;
ALTER TABLE app.privacy_requests ENABLE ROW LEVEL SECURITY;
ALTER TABLE app.privacy_request_events ENABLE ROW LEVEL SECURITY;
ALTER TABLE app.privacy_exports ENABLE ROW LEVEL SECURITY;
ALTER TABLE app.processing_restrictions ENABLE ROW LEVEL SECURITY;
ALTER TABLE app.legal_holds ENABLE ROW LEVEL SECURITY;

CREATE POLICY recovery_code_owner ON app.account_recovery_codes USING (user_id=app.current_user_id());
CREATE POLICY recovery_rate_runtime ON app.account_recovery_rate_limits USING (current_user IN ('kredit_app','kredit_worker')) WITH CHECK (current_user IN ('kredit_app','kredit_worker'));
CREATE POLICY recovery_request_owner ON app.account_recovery_requests USING (target_user_id=app.current_user_id());
CREATE POLICY recovery_evidence_owner ON app.account_recovery_evidence USING (EXISTS (SELECT 1 FROM app.account_recovery_requests r WHERE r.id=request_id AND r.target_user_id=app.current_user_id()));
CREATE POLICY recovery_event_owner ON app.account_recovery_events USING (EXISTS (SELECT 1 FROM app.account_recovery_requests r WHERE r.id=request_id AND r.target_user_id=app.current_user_id()));
CREATE POLICY privacy_request_owner ON app.privacy_requests USING (requester_user_id=app.current_user_id());
CREATE POLICY privacy_event_owner ON app.privacy_request_events USING (EXISTS (SELECT 1 FROM app.privacy_requests r WHERE r.id=request_id AND r.requester_user_id=app.current_user_id()));
CREATE POLICY privacy_export_owner ON app.privacy_exports USING (EXISTS (SELECT 1 FROM app.privacy_requests r WHERE r.id=request_id AND r.requester_user_id=app.current_user_id()));
CREATE POLICY restriction_owner_read ON app.processing_restrictions USING (user_id=app.current_user_id());
CREATE POLICY legal_hold_runtime ON app.legal_holds USING (current_user IN ('kredit_app','kredit_worker')) WITH CHECK (current_user IN ('kredit_app','kredit_worker'));

CREATE POLICY recovery_code_runtime ON app.account_recovery_codes USING (current_user IN ('kredit_app','kredit_worker')) WITH CHECK (current_user IN ('kredit_app','kredit_worker'));
CREATE POLICY recovery_request_runtime ON app.account_recovery_requests USING (current_user IN ('kredit_app','kredit_worker')) WITH CHECK (current_user IN ('kredit_app','kredit_worker'));
CREATE POLICY recovery_evidence_runtime ON app.account_recovery_evidence USING (current_user IN ('kredit_app','kredit_worker')) WITH CHECK (current_user IN ('kredit_app','kredit_worker'));
CREATE POLICY recovery_event_runtime ON app.account_recovery_events USING (current_user IN ('kredit_app','kredit_worker')) WITH CHECK (current_user IN ('kredit_app','kredit_worker'));
CREATE POLICY privacy_request_runtime ON app.privacy_requests USING (current_user IN ('kredit_app','kredit_worker')) WITH CHECK (current_user IN ('kredit_app','kredit_worker'));
CREATE POLICY privacy_event_runtime ON app.privacy_request_events USING (current_user IN ('kredit_app','kredit_worker')) WITH CHECK (current_user IN ('kredit_app','kredit_worker'));
CREATE POLICY privacy_export_runtime ON app.privacy_exports USING (current_user IN ('kredit_app','kredit_worker')) WITH CHECK (current_user IN ('kredit_app','kredit_worker'));
CREATE POLICY restriction_runtime ON app.processing_restrictions USING (current_user IN ('kredit_app','kredit_worker')) WITH CHECK (current_user IN ('kredit_app','kredit_worker'));

-- Runtime services perform permission checks before accessing review queues.
-- +goose StatementBegin
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname='kredit_app') THEN
    GRANT SELECT,INSERT,UPDATE ON app.account_recovery_codes,app.account_recovery_rate_limits,app.account_recovery_requests,app.account_recovery_evidence,app.account_recovery_events,app.privacy_requests,app.privacy_request_events,app.privacy_exports,app.processing_restrictions,app.legal_holds TO kredit_app;
  END IF;
END $$;
-- +goose StatementEnd

-- +goose Down
DROP TABLE IF EXISTS app.legal_holds,app.processing_restrictions,app.privacy_exports,app.privacy_request_events,app.privacy_requests,app.account_recovery_events,app.account_recovery_evidence,app.account_recovery_requests,app.account_recovery_rate_limits,app.account_recovery_codes;
ALTER TABLE app.notification_preferences DROP COLUMN IF EXISTS version,DROP COLUMN IF EXISTS product_updates_enabled,DROP COLUMN IF EXISTS payment_reminders_enabled;
