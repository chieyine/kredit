-- +goose Up
-- Product analytics is an append-only, privacy-minimised projection. Domain
-- tables remain authoritative for every financial and agreement fact.
ALTER TABLE app.analytics_events
  ADD COLUMN schema_version INTEGER NOT NULL DEFAULT 1 CHECK (schema_version > 0),
  ADD COLUMN deduplication_key TEXT,
  ADD COLUMN organization_id_hash TEXT,
  ADD COLUMN source TEXT NOT NULL DEFAULT 'application',
  ADD COLUMN recorded_at TIMESTAMPTZ NOT NULL DEFAULT NOW();

UPDATE app.analytics_events
SET deduplication_key = 'legacy:' || id::text
WHERE deduplication_key IS NULL;

ALTER TABLE app.analytics_events
  ALTER COLUMN deduplication_key SET NOT NULL,
  ADD CONSTRAINT analytics_events_deduplication_key_unique UNIQUE (deduplication_key),
  ADD CONSTRAINT analytics_events_metadata_object CHECK (jsonb_typeof(metadata) = 'object'),
  ADD CONSTRAINT analytics_events_metadata_size CHECK (octet_length(metadata::text) <= 2048),
  ADD CONSTRAINT analytics_events_metadata_privacy CHECK (NOT (metadata ?| ARRAY[
    'phone','email','bvn','nin','invoice_reference','goods_description','bank_account',
    'provider_token','reason','notes','statement','body','name','address'
  ]));

CREATE INDEX analytics_events_name_occurred_idx ON app.analytics_events(name, occurred_at DESC);
CREATE INDEX analytics_events_org_occurred_idx ON app.analytics_events(organization_id_hash, occurred_at DESC);
CREATE INDEX analytics_events_recorded_idx ON app.analytics_events(recorded_at DESC);

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION app.record_product_event(
  p_name TEXT,
  p_subject_id UUID,
  p_organization_id UUID,
  p_purpose TEXT,
  p_occurred_at TIMESTAMPTZ,
  p_deduplication_key TEXT,
  p_metadata JSONB DEFAULT '{}'::jsonb
) RETURNS VOID
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = app, public, pg_catalog
AS $$
BEGIN
  IF p_name !~ '^[a-z][a-z0-9_]*(\.[a-z][a-z0-9_]*)+$' THEN
    RAISE EXCEPTION 'invalid product event name';
  END IF;
  IF p_purpose NOT IN ('product_improvement','pilot_measurement','operations_reliability') THEN
    RAISE EXCEPTION 'invalid product event purpose';
  END IF;
  INSERT INTO app.analytics_events(
    id, name, subject_id_hash, purpose, metadata, occurred_at,
    schema_version, deduplication_key, organization_id_hash, source, recorded_at
  ) VALUES (
    gen_random_uuid(), p_name,
    encode(digest(p_subject_id::text, 'sha256'), 'hex'),
    p_purpose, COALESCE(p_metadata, '{}'::jsonb), COALESCE(p_occurred_at, NOW()),
    1, p_deduplication_key,
    CASE WHEN p_organization_id IS NULL THEN NULL ELSE encode(digest(p_organization_id::text, 'sha256'), 'hex') END,
    'domain_trigger', NOW()
  ) ON CONFLICT (deduplication_key) DO NOTHING;
END
$$;
-- +goose StatementEnd

REVOKE ALL ON FUNCTION app.record_product_event(TEXT,UUID,UUID,TEXT,TIMESTAMPTZ,TEXT,JSONB) FROM PUBLIC;

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION app.analytics_onboarding_event() RETURNS TRIGGER LANGUAGE plpgsql AS $$
BEGIN
  IF TG_OP = 'INSERT' THEN
    PERFORM app.record_product_event('onboarding.started', NEW.organization_id, NEW.organization_id, 'product_improvement', NEW.created_at, 'onboarding:'||NEW.organization_id||':started', '{}'::jsonb);
  END IF;
  IF NEW.readiness_state = 'pilot_ready' AND (TG_OP = 'INSERT' OR OLD.readiness_state IS DISTINCT FROM NEW.readiness_state) THEN
    PERFORM app.record_product_event('onboarding.ready', NEW.organization_id, NEW.organization_id, 'pilot_measurement', NEW.readiness_changed_at, 'onboarding:'||NEW.organization_id||':ready:'||NEW.version, jsonb_build_object('state','pilot_ready'));
  ELSIF TG_OP = 'UPDATE' AND OLD.version IS DISTINCT FROM NEW.version THEN
    PERFORM app.record_product_event('onboarding.step_completed', NEW.organization_id, NEW.organization_id, 'product_improvement', NEW.updated_at, 'onboarding:'||NEW.organization_id||':step:'||NEW.version, jsonb_build_object('readiness_state',NEW.readiness_state));
  END IF;
  RETURN NEW;
END $$;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION app.analytics_invitation_event() RETURNS TRIGGER LANGUAGE plpgsql AS $$
BEGIN
  IF TG_OP = 'INSERT' THEN
    PERFORM app.record_product_event('customer.invited', NEW.id, NEW.organization_id, 'product_improvement', NEW.created_at, 'buyer_invitation:'||NEW.id||':invited', jsonb_build_object('channel',NEW.target_type));
  ELSIF NEW.status = 'accepted' AND OLD.status IS DISTINCT FROM NEW.status THEN
    PERFORM app.record_product_event('customer.invitation_accepted', NEW.id, NEW.organization_id, 'product_improvement', NEW.accepted_at, 'buyer_invitation:'||NEW.id||':accepted', '{}'::jsonb);
  END IF;
  RETURN NEW;
END $$;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION app.analytics_verification_event() RETURNS TRIGGER LANGUAGE plpgsql AS $$
BEGIN
  IF NEW.state = 'verified' AND (TG_OP = 'INSERT' OR OLD.state IS DISTINCT FROM NEW.state) THEN
    PERFORM app.record_product_event('customer.verified', NEW.subject_id, NULL, 'pilot_measurement', COALESCE(NEW.completed_at,NOW()), 'verification_case:'||NEW.id||':verified', jsonb_build_object('subject_type',NEW.subject_type,'level',NEW.verification_level));
  END IF;
  RETURN NEW;
END $$;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION app.analytics_credit_event() RETURNS TRIGGER LANGUAGE plpgsql AS $$
DECLARE event_name TEXT;
BEGIN
  IF TG_OP = 'INSERT' THEN event_name := 'credit.drafted';
  ELSIF OLD.state IS DISTINCT FROM NEW.state THEN
    event_name := CASE NEW.state
      WHEN 'SENT' THEN 'credit.sent'
      WHEN 'BUYER_REVIEWING' THEN 'credit.viewed'
      WHEN 'DECLINED' THEN 'credit.declined'
      ELSE NULL END;
  END IF;
  IF event_name IS NOT NULL THEN
    PERFORM app.record_product_event(event_name, NEW.id, NEW.supplier_organization_id, 'pilot_measurement', CASE WHEN TG_OP='INSERT' THEN NEW.created_at ELSE NEW.updated_at END, 'credit_request:'||NEW.id||':'||event_name, jsonb_build_object('state',NEW.state));
  END IF;
  RETURN NEW;
END $$;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION app.analytics_credit_child_event() RETURNS TRIGGER LANGUAGE plpgsql AS $$
DECLARE event_name TEXT; event_at TIMESTAMPTZ; org_id UUID; row_data JSONB; request_id UUID; subject_id UUID;
BEGIN
  row_data := to_jsonb(NEW);
  request_id := (row_data->>'credit_request_id')::UUID;
  subject_id := (row_data->>'id')::UUID;
  IF TG_TABLE_NAME='agreement_acceptances' THEN
    event_name := 'credit.accepted'; event_at := (row_data->>'accepted_at')::TIMESTAMPTZ;
  ELSIF TG_TABLE_NAME='goods_releases' THEN
    event_name := 'goods.released'; event_at := (row_data->>'released_at')::TIMESTAMPTZ;
  ELSE
    event_name := CASE row_data->>'state' WHEN 'confirmed' THEN 'receipt.confirmed' ELSE 'receipt.issue_raised' END;
    event_at := (row_data->>'received_at')::TIMESTAMPTZ;
  END IF;
  SELECT supplier_organization_id INTO org_id FROM app.credit_requests WHERE id=request_id;
  PERFORM app.record_product_event(event_name, subject_id, org_id, 'pilot_measurement', event_at, TG_TABLE_NAME||':'||subject_id||':'||event_name, '{}'::jsonb);
  RETURN NEW;
END $$;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION app.analytics_mandate_event() RETURNS TRIGGER LANGUAGE plpgsql AS $$
DECLARE event_name TEXT; org_id UUID;
BEGIN
  IF TG_OP='INSERT' THEN event_name := 'mandate.started';
  ELSIF OLD.status IS DISTINCT FROM NEW.status THEN
    event_name := CASE lower(NEW.status) WHEN 'active' THEN 'mandate.active' WHEN 'failed' THEN 'mandate.failed' WHEN 'cancelled' THEN 'mandate.cancelled' ELSE NULL END;
  END IF;
  IF event_name IS NOT NULL THEN
    SELECT supplier_organization_id INTO org_id FROM app.credit_requests WHERE id=NEW.credit_request_id;
    PERFORM app.record_product_event(event_name, NEW.id, org_id, 'operations_reliability', COALESCE(NEW.activated_at,NEW.created_at,NOW()), 'mandate:'||NEW.id||':'||event_name, jsonb_build_object('status',lower(NEW.status)));
  END IF;
  RETURN NEW;
END $$;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION app.analytics_obligation_event() RETURNS TRIGGER LANGUAGE plpgsql AS $$
BEGIN
  PERFORM app.record_product_event('obligation.activated', NEW.id, NEW.supplier_organization_id, 'pilot_measurement', NEW.activated_at, 'obligation:'||NEW.id||':activated', jsonb_build_object('currency',NEW.currency));
  RETURN NEW;
END $$;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION app.analytics_payment_event() RETURNS TRIGGER LANGUAGE plpgsql AS $$
BEGIN
  IF NEW.state='recognized' THEN
    PERFORM app.record_product_event('payment.confirmed', NEW.id, NEW.supplier_organization_id, 'pilot_measurement', NEW.recognized_at, 'payment:'||NEW.id||':confirmed', jsonb_build_object('source_type',NEW.source_type));
  END IF;
  RETURN NEW;
END $$;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION app.analytics_payment_claim_event() RETURNS TRIGGER LANGUAGE plpgsql AS $$
DECLARE event_name TEXT;
BEGIN
  IF TG_OP='INSERT' THEN event_name := 'payment.claimed';
  ELSIF OLD.state IS DISTINCT FROM NEW.state AND NEW.state='confirmed' THEN event_name := 'payment.claim_confirmed'; END IF;
  IF event_name IS NOT NULL THEN
    PERFORM app.record_product_event(event_name, NEW.id, NEW.supplier_organization_id, 'pilot_measurement', COALESCE(NEW.reviewed_at,NEW.created_at), 'payment_claim:'||NEW.id||':'||event_name, jsonb_build_object('state',NEW.state));
  END IF;
  RETURN NEW;
END $$;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION app.analytics_schedule_event() RETURNS TRIGGER LANGUAGE plpgsql AS $$
DECLARE event_name TEXT; obligation_id UUID; org_id UUID;
BEGIN
  IF TG_OP='UPDATE' AND OLD.state IS DISTINCT FROM NEW.state THEN
    event_name := CASE NEW.state WHEN 'IN_GRACE' THEN 'payment.due' WHEN 'OVERDUE' THEN 'payment.late' ELSE NULL END;
    IF event_name IS NOT NULL THEN
      SELECT rs.obligation_id,o.supplier_organization_id INTO obligation_id,org_id FROM app.repayment_schedules rs JOIN app.obligations o ON o.id=rs.obligation_id WHERE rs.id=NEW.schedule_id;
      PERFORM app.record_product_event(event_name, obligation_id, org_id, 'pilot_measurement', NOW(), 'schedule_item:'||NEW.id||':'||event_name, jsonb_build_object('state',NEW.state));
    END IF;
  END IF;
  RETURN NEW;
END $$;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION app.analytics_collection_event() RETURNS TRIGGER LANGUAGE plpgsql AS $$
DECLARE event_name TEXT; org_id UUID;
BEGIN
  IF TG_OP='UPDATE' AND OLD.state IS DISTINCT FROM NEW.state THEN
    event_name := CASE NEW.state WHEN 'FAILED' THEN 'payment.collection_failed' WHEN 'SUCCEEDED' THEN CASE WHEN NEW.attempt_number>1 THEN 'payment.recovered' ELSE 'payment.collected' END WHEN 'PARTIAL' THEN 'payment.collected' ELSE NULL END;
    IF event_name IS NOT NULL THEN
      SELECT supplier_organization_id INTO org_id FROM app.obligations WHERE id=NEW.obligation_id;
      PERFORM app.record_product_event(event_name, NEW.id, org_id, 'operations_reliability', COALESCE(NEW.final_at,NOW()), 'collection_attempt:'||NEW.id||':'||event_name, jsonb_build_object('state',NEW.state,'attempt_number',NEW.attempt_number));
    END IF;
  END IF;
  RETURN NEW;
END $$;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION app.analytics_trade_line_event() RETURNS TRIGGER LANGUAGE plpgsql AS $$
DECLARE event_name TEXT;
BEGIN
  IF TG_OP='INSERT' THEN event_name := 'trade_line.created';
  ELSIF OLD.state IS DISTINCT FROM NEW.state THEN event_name := CASE NEW.state WHEN 'ACTIVE' THEN 'trade_line.activated' WHEN 'EXPIRED' THEN 'trade_line.expired' ELSE NULL END; END IF;
  IF event_name IS NOT NULL THEN
    PERFORM app.record_product_event(event_name, NEW.id, NEW.supplier_organization_id, 'pilot_measurement', COALESCE(NEW.updated_at,NEW.created_at), 'trade_line:'||NEW.id||':'||event_name, jsonb_build_object('state',NEW.state));
  END IF;
  RETURN NEW;
END $$;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION app.analytics_drawdown_event() RETURNS TRIGGER LANGUAGE plpgsql AS $$
DECLARE event_name TEXT; org_id UUID; subject_id UUID; event_time TIMESTAMPTZ;
BEGIN
  IF TG_TABLE_NAME='drawdown_reservations' THEN
    subject_id := NEW.id; event_time := NEW.created_at;
    IF TG_OP='INSERT' THEN event_name := 'drawdown.reserved';
    ELSIF OLD.state IS DISTINCT FROM NEW.state THEN event_name := CASE NEW.state WHEN 'CONFIRMED' THEN 'drawdown.confirmed' WHEN 'RELEASED' THEN 'drawdown.released' WHEN 'EXPIRED' THEN 'drawdown.expired' ELSE NULL END; END IF;
  ELSE
    subject_id := NEW.id; event_time := COALESCE(NEW.activated_at,NEW.buyer_confirmed_at,NEW.created_at);
    IF TG_OP='UPDATE' AND OLD.state IS DISTINCT FROM NEW.state THEN event_name := CASE NEW.state WHEN 'BUYER_CONFIRMED' THEN 'drawdown.confirmed' WHEN 'ACTIVATED' THEN 'drawdown.activated' ELSE NULL END; END IF;
  END IF;
  IF event_name IS NOT NULL THEN
    SELECT supplier_organization_id INTO org_id FROM app.trade_lines WHERE id=NEW.trade_line_id;
    PERFORM app.record_product_event(event_name, subject_id, org_id, 'pilot_measurement', event_time, TG_TABLE_NAME||':'||subject_id||':'||event_name, jsonb_build_object('state',NEW.state));
  END IF;
  RETURN NEW;
END $$;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION app.analytics_dispute_event() RETURNS TRIGGER LANGUAGE plpgsql AS $$
DECLARE event_name TEXT;
BEGIN
  IF TG_OP='INSERT' THEN event_name := 'dispute.opened';
  ELSIF OLD.state IS DISTINCT FROM NEW.state AND NEW.state IN ('RESOLVED','WITHDRAWN') THEN event_name := 'dispute.resolved'; END IF;
  IF event_name IS NOT NULL THEN
    PERFORM app.record_product_event(event_name, NEW.id, NEW.supplier_organization_id, 'pilot_measurement', COALESCE(NEW.resolved_at,NEW.opened_at), 'dispute:'||NEW.id||':'||event_name, jsonb_build_object('state',NEW.state));
  END IF;
  RETURN NEW;
END $$;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION app.analytics_support_event() RETURNS TRIGGER LANGUAGE plpgsql AS $$
BEGIN
  IF TG_OP='INSERT' THEN
    PERFORM app.record_product_event('operations.intervention', NEW.id, NEW.organization_id, 'operations_reliability', NEW.created_at, 'support_case:'||NEW.id||':opened', jsonb_build_object('subject_type',NEW.subject_type));
  END IF;
  RETURN NEW;
END $$;
-- +goose StatementEnd

CREATE TRIGGER product_analytics_onboarding AFTER INSERT OR UPDATE ON app.supplier_onboarding_profiles FOR EACH ROW EXECUTE FUNCTION app.analytics_onboarding_event();
CREATE TRIGGER product_analytics_invitation AFTER INSERT OR UPDATE ON app.buyer_invitations FOR EACH ROW EXECUTE FUNCTION app.analytics_invitation_event();
CREATE TRIGGER product_analytics_verification AFTER INSERT OR UPDATE ON app.verification_cases FOR EACH ROW EXECUTE FUNCTION app.analytics_verification_event();
CREATE TRIGGER product_analytics_credit AFTER INSERT OR UPDATE ON app.credit_requests FOR EACH ROW EXECUTE FUNCTION app.analytics_credit_event();
CREATE TRIGGER product_analytics_acceptance AFTER INSERT ON app.agreement_acceptances FOR EACH ROW EXECUTE FUNCTION app.analytics_credit_child_event();
CREATE TRIGGER product_analytics_release AFTER INSERT ON app.goods_releases FOR EACH ROW EXECUTE FUNCTION app.analytics_credit_child_event();
CREATE TRIGGER product_analytics_receipt AFTER INSERT ON app.receipt_confirmations FOR EACH ROW EXECUTE FUNCTION app.analytics_credit_child_event();
CREATE TRIGGER product_analytics_mandate AFTER INSERT OR UPDATE ON app.mandates FOR EACH ROW EXECUTE FUNCTION app.analytics_mandate_event();
CREATE TRIGGER product_analytics_obligation AFTER INSERT ON app.obligations FOR EACH ROW EXECUTE FUNCTION app.analytics_obligation_event();
CREATE TRIGGER product_analytics_payment AFTER INSERT ON app.payments FOR EACH ROW EXECUTE FUNCTION app.analytics_payment_event();
CREATE TRIGGER product_analytics_payment_claim AFTER INSERT OR UPDATE ON app.payment_claims FOR EACH ROW EXECUTE FUNCTION app.analytics_payment_claim_event();
CREATE TRIGGER product_analytics_schedule AFTER UPDATE ON app.schedule_items FOR EACH ROW EXECUTE FUNCTION app.analytics_schedule_event();
CREATE TRIGGER product_analytics_collection AFTER UPDATE ON app.collection_attempts FOR EACH ROW EXECUTE FUNCTION app.analytics_collection_event();
CREATE TRIGGER product_analytics_trade_line AFTER INSERT OR UPDATE ON app.trade_lines FOR EACH ROW EXECUTE FUNCTION app.analytics_trade_line_event();
CREATE TRIGGER product_analytics_drawdown AFTER UPDATE ON app.drawdowns FOR EACH ROW EXECUTE FUNCTION app.analytics_drawdown_event();
CREATE TRIGGER product_analytics_drawdown_reservation AFTER INSERT OR UPDATE ON app.drawdown_reservations FOR EACH ROW EXECUTE FUNCTION app.analytics_drawdown_event();
CREATE TRIGGER product_analytics_dispute AFTER INSERT OR UPDATE ON app.disputes FOR EACH ROW EXECUTE FUNCTION app.analytics_dispute_event();
CREATE TRIGGER product_analytics_support AFTER INSERT ON app.support_cases FOR EACH ROW EXECUTE FUNCTION app.analytics_support_event();

-- Backfill only facts that can be reconstructed exactly. The same keys used by
-- triggers make migration replay and concurrent writes harmless.
SELECT app.record_product_event('onboarding.started',p.organization_id,p.organization_id,'product_improvement',p.created_at,'onboarding:'||p.organization_id||':started','{}') FROM app.supplier_onboarding_profiles p;
SELECT app.record_product_event('onboarding.ready',p.organization_id,p.organization_id,'pilot_measurement',p.readiness_changed_at,'onboarding:'||p.organization_id||':ready:'||p.version,jsonb_build_object('state','pilot_ready')) FROM app.supplier_onboarding_profiles p WHERE p.readiness_state='pilot_ready';
SELECT app.record_product_event('customer.invited',i.id,i.organization_id,'product_improvement',i.created_at,'buyer_invitation:'||i.id||':invited',jsonb_build_object('channel',i.target_type)) FROM app.buyer_invitations i;
SELECT app.record_product_event('customer.verified',v.subject_id,NULL,'pilot_measurement',COALESCE(v.completed_at,v.started_at),'verification_case:'||v.id||':verified',jsonb_build_object('subject_type',v.subject_type,'level',v.verification_level)) FROM app.verification_cases v WHERE v.state='verified';
SELECT app.record_product_event('credit.drafted',c.id,c.supplier_organization_id,'pilot_measurement',c.created_at,'credit_request:'||c.id||':credit.drafted',jsonb_build_object('state','DRAFT')) FROM app.credit_requests c;
SELECT app.record_product_event('credit.accepted',a.id,c.supplier_organization_id,'pilot_measurement',a.accepted_at,'agreement_acceptances:'||a.id||':credit.accepted','{}') FROM app.agreement_acceptances a JOIN app.credit_requests c ON c.id=a.credit_request_id;
SELECT app.record_product_event('goods.released',g.id,c.supplier_organization_id,'pilot_measurement',g.released_at,'goods_releases:'||g.id||':goods.released','{}') FROM app.goods_releases g JOIN app.credit_requests c ON c.id=g.credit_request_id;
SELECT app.record_product_event(CASE r.state WHEN 'confirmed' THEN 'receipt.confirmed' ELSE 'receipt.issue_raised' END,r.id,c.supplier_organization_id,'pilot_measurement',r.received_at,'receipt_confirmations:'||r.id||':'||CASE r.state WHEN 'confirmed' THEN 'receipt.confirmed' ELSE 'receipt.issue_raised' END,'{}') FROM app.receipt_confirmations r JOIN app.credit_requests c ON c.id=r.credit_request_id;
SELECT app.record_product_event('obligation.activated',o.id,o.supplier_organization_id,'pilot_measurement',o.activated_at,'obligation:'||o.id||':activated',jsonb_build_object('currency',o.currency)) FROM app.obligations o;
SELECT app.record_product_event('payment.confirmed',p.id,p.supplier_organization_id,'pilot_measurement',p.recognized_at,'payment:'||p.id||':confirmed',jsonb_build_object('source_type',p.source_type)) FROM app.payments p WHERE p.state='recognized';
SELECT app.record_product_event('trade_line.created',t.id,t.supplier_organization_id,'pilot_measurement',t.created_at,'trade_line:'||t.id||':trade_line.created',jsonb_build_object('state',t.state)) FROM app.trade_lines t;
SELECT app.record_product_event('dispute.opened',d.id,d.supplier_organization_id,'pilot_measurement',d.opened_at,'dispute:'||d.id||':dispute.opened',jsonb_build_object('state',d.state)) FROM app.disputes d;
SELECT app.record_product_event('operations.intervention',s.id,s.organization_id,'operations_reliability',s.created_at,'support_case:'||s.id||':opened',jsonb_build_object('subject_type',s.subject_type)) FROM app.support_cases s;

-- +goose StatementBegin
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname='kredit_app') THEN
    GRANT EXECUTE ON FUNCTION app.record_product_event(TEXT,UUID,UUID,TEXT,TIMESTAMPTZ,TEXT,JSONB) TO kredit_app;
    GRANT SELECT,INSERT ON app.analytics_events TO kredit_app;
  END IF;
  IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname='kredit_worker') THEN
    GRANT EXECUTE ON FUNCTION app.record_product_event(TEXT,UUID,UUID,TEXT,TIMESTAMPTZ,TEXT,JSONB) TO kredit_worker;
    GRANT SELECT,INSERT ON app.analytics_events TO kredit_worker;
  END IF;
END $$;
-- +goose StatementEnd

-- +goose Down
DROP TRIGGER IF EXISTS product_analytics_support ON app.support_cases;
DROP TRIGGER IF EXISTS product_analytics_dispute ON app.disputes;
DROP TRIGGER IF EXISTS product_analytics_drawdown_reservation ON app.drawdown_reservations;
DROP TRIGGER IF EXISTS product_analytics_drawdown ON app.drawdowns;
DROP TRIGGER IF EXISTS product_analytics_trade_line ON app.trade_lines;
DROP TRIGGER IF EXISTS product_analytics_collection ON app.collection_attempts;
DROP TRIGGER IF EXISTS product_analytics_schedule ON app.schedule_items;
DROP TRIGGER IF EXISTS product_analytics_payment_claim ON app.payment_claims;
DROP TRIGGER IF EXISTS product_analytics_payment ON app.payments;
DROP TRIGGER IF EXISTS product_analytics_obligation ON app.obligations;
DROP TRIGGER IF EXISTS product_analytics_mandate ON app.mandates;
DROP TRIGGER IF EXISTS product_analytics_receipt ON app.receipt_confirmations;
DROP TRIGGER IF EXISTS product_analytics_release ON app.goods_releases;
DROP TRIGGER IF EXISTS product_analytics_acceptance ON app.agreement_acceptances;
DROP TRIGGER IF EXISTS product_analytics_credit ON app.credit_requests;
DROP TRIGGER IF EXISTS product_analytics_verification ON app.verification_cases;
DROP TRIGGER IF EXISTS product_analytics_invitation ON app.buyer_invitations;
DROP TRIGGER IF EXISTS product_analytics_onboarding ON app.supplier_onboarding_profiles;
DROP FUNCTION IF EXISTS app.analytics_support_event(),app.analytics_dispute_event(),app.analytics_drawdown_event(),app.analytics_trade_line_event(),app.analytics_collection_event(),app.analytics_schedule_event(),app.analytics_payment_claim_event(),app.analytics_payment_event(),app.analytics_obligation_event(),app.analytics_mandate_event(),app.analytics_credit_child_event(),app.analytics_credit_event(),app.analytics_verification_event(),app.analytics_invitation_event(),app.analytics_onboarding_event();
DROP FUNCTION IF EXISTS app.record_product_event(TEXT,UUID,UUID,TEXT,TIMESTAMPTZ,TEXT,JSONB);
DROP INDEX IF EXISTS app.analytics_events_recorded_idx,app.analytics_events_org_occurred_idx,app.analytics_events_name_occurred_idx;
ALTER TABLE app.analytics_events DROP CONSTRAINT IF EXISTS analytics_events_metadata_privacy,DROP CONSTRAINT IF EXISTS analytics_events_metadata_size,DROP CONSTRAINT IF EXISTS analytics_events_metadata_object,DROP CONSTRAINT IF EXISTS analytics_events_deduplication_key_unique;
ALTER TABLE app.analytics_events DROP COLUMN IF EXISTS recorded_at,DROP COLUMN IF EXISTS source,DROP COLUMN IF EXISTS organization_id_hash,DROP COLUMN IF EXISTS deduplication_key,DROP COLUMN IF EXISTS schema_version;
