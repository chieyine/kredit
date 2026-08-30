-- +goose Up
CREATE TABLE app.collection_reservations (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  obligation_id uuid NOT NULL REFERENCES app.obligations(id),
  schedule_item_id uuid REFERENCES app.schedule_items(id),
  outstanding_snapshot_version bigint NOT NULL,
  reserved_amount_kobo bigint NOT NULL CHECK (reserved_amount_kobo > 0),
  state text NOT NULL CHECK (state IN ('PROCESSING','RELEASED','COMPLETED','EXPIRED','CANCELLED')),
  expires_at timestamptz NOT NULL,
  idempotency_key text NOT NULL UNIQUE,
  created_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE app.collection_attempts (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  reservation_id uuid NOT NULL REFERENCES app.collection_reservations(id),
  obligation_id uuid NOT NULL REFERENCES app.obligations(id),
  provider text NOT NULL,
  provider_collection_id text,
  external_reference text NOT NULL UNIQUE,
  requested_amount_kobo bigint NOT NULL CHECK (requested_amount_kobo > 0),
  succeeded_amount_kobo bigint NOT NULL DEFAULT 0 CHECK (succeeded_amount_kobo >= 0),
  state text NOT NULL CHECK (state IN ('PENDING','SUBMITTED','UNKNOWN','SUCCEEDED','PARTIAL','FAILED','CANCELLED')),
  attempt_number integer NOT NULL DEFAULT 1,
  retry_classification text,
  failure_code text,
  requested_at timestamptz NOT NULL DEFAULT now(),
  final_at timestamptz
);
CREATE TABLE app.collection_provider_events (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  provider text NOT NULL,
  event_id text NOT NULL,
  external_reference text NOT NULL,
  payload_hash text NOT NULL,
  state text NOT NULL,
  processed_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (provider, event_id)
);
ALTER TABLE app.collection_reservations ENABLE ROW LEVEL SECURITY;
CREATE POLICY collection_reservation_supplier_access ON app.collection_reservations USING (obligation_id IN (SELECT id FROM app.obligations WHERE supplier_organization_id IN (SELECT organization_id FROM app.memberships WHERE user_id = app.current_user_id() AND status = 'active')));
ALTER TABLE app.collection_attempts ENABLE ROW LEVEL SECURITY;
CREATE POLICY collection_attempt_supplier_access ON app.collection_attempts USING (obligation_id IN (SELECT id FROM app.obligations WHERE supplier_organization_id IN (SELECT organization_id FROM app.memberships WHERE user_id = app.current_user_id() AND status = 'active')));
ALTER TABLE app.collection_provider_events ENABLE ROW LEVEL SECURITY;
CREATE POLICY collection_event_support_access ON app.collection_provider_events USING (current_user = 'kredit_app');

-- +goose Down
DROP TABLE IF EXISTS app.collection_provider_events, app.collection_attempts, app.collection_reservations;
