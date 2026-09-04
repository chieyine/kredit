-- +goose Up
ALTER TABLE app.payment_mandates ADD COLUMN IF NOT EXISTS metadata JSONB NOT NULL DEFAULT '{}'::jsonb;
ALTER TABLE app.payment_mandates DROP CONSTRAINT IF EXISTS payment_mandates_state_check;
ALTER TABLE app.payment_mandates ADD CONSTRAINT payment_mandates_state_check CHECK(state IN ('pending','active','cancelled','expired','failed','suspended','paused'));
CREATE POLICY payment_mandate_worker_read ON app.payment_mandates FOR SELECT USING(current_user='kredit_worker');
CREATE TABLE app.provider_customer_bindings (
 provider TEXT NOT NULL,
 buyer_user_id UUID NOT NULL REFERENCES app.users(id),
 buyer_business_id UUID NOT NULL REFERENCES app.businesses(id),
 provider_customer_reference TEXT NOT NULL,
 consent_version TEXT NOT NULL,
 created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
 PRIMARY KEY(provider,buyer_business_id),
 UNIQUE(provider,provider_customer_reference)
);
ALTER TABLE app.provider_customer_bindings ENABLE ROW LEVEL SECURITY;
CREATE POLICY provider_customer_runtime ON app.provider_customer_bindings USING(current_user IN ('kredit_app','kredit_worker')) WITH CHECK(current_user IN ('kredit_app','kredit_worker'));
ALTER TABLE app.payment_mandates ADD COLUMN IF NOT EXISTS supplier_organization_id UUID REFERENCES app.organizations(id);
ALTER TABLE app.collection_reservations ADD COLUMN IF NOT EXISTS mandate_id UUID REFERENCES app.payment_mandates(id);
ALTER TABLE app.collection_attempts ADD COLUMN IF NOT EXISTS next_retry_at TIMESTAMPTZ;
CREATE INDEX collection_mandate_reservations_idx ON app.collection_reservations(mandate_id,state);

-- A debit reservation is checked under the same obligation lock used by
-- manual payments. A shared mandate lock serializes capacity across invoices.
-- +goose StatementBegin
CREATE FUNCTION app.guard_collection_reservation() RETURNS trigger LANGUAGE plpgsql SECURITY DEFINER SET search_path=app,pg_catalog AS $$
DECLARE remaining BIGINT; ceiling BIGINT; used BIGINT; held BIGINT; mid UUID; mstate TEXT; mstart TIMESTAMPTZ; mend TIMESTAMPTZ; buyer UUID; supplier UUID; mbuyer UUID; msupplier UUID;
BEGIN
 IF EXISTS(SELECT 1 FROM app.collection_reservations WHERE id=NEW.id) THEN RETURN NEW; END IF;
 SELECT o.outstanding_kobo,c.mandate_id,c.buyer_business_id,c.supplier_organization_id INTO remaining,mid,buyer,supplier FROM app.obligations o JOIN app.credit_requests c ON c.id=o.credit_request_id WHERE o.id=NEW.obligation_id FOR UPDATE OF o;
 IF NEW.reserved_amount_kobo > remaining THEN RAISE EXCEPTION 'collection exceeds authoritative outstanding balance'; END IF;
 IF EXISTS(SELECT 1 FROM app.collection_reservations WHERE obligation_id=NEW.obligation_id AND state IN ('PROCESSING','COMPLETED')) THEN RAISE EXCEPTION 'collection already reserved'; END IF;
 IF mid IS NOT NULL THEN
   SELECT amount_ceiling_kobo,state,starts_at,ends_at,buyer_subject_id,supplier_organization_id INTO ceiling,mstate,mstart,mend,mbuyer,msupplier FROM app.payment_mandates WHERE id=mid FOR UPDATE;
   IF ceiling IS NULL OR mbuyer<>buyer OR (msupplier IS NOT NULL AND msupplier<>supplier) THEN RAISE EXCEPTION 'mandate ownership mismatch'; END IF;
   IF mstate <> 'active' OR mstart>now() OR mend<=now() THEN RAISE EXCEPTION 'mandate is not active in validity period'; END IF;
   SELECT COALESCE(SUM(a.succeeded_amount_kobo),0) INTO used FROM app.collection_attempts a JOIN app.collection_reservations r ON r.id=a.reservation_id WHERE r.mandate_id=mid;
   SELECT COALESCE(SUM(reserved_amount_kobo),0) INTO held FROM app.collection_reservations WHERE mandate_id=mid AND state IN ('PROCESSING','COMPLETED');
   IF NEW.reserved_amount_kobo > ceiling-used-held THEN RAISE EXCEPTION 'mandate capacity exhausted'; END IF;
   NEW.mandate_id := mid;
 END IF;
 RETURN NEW;
END $$;
-- +goose StatementEnd
CREATE TRIGGER guard_collection_reservation BEFORE INSERT ON app.collection_reservations FOR EACH ROW EXECUTE FUNCTION app.guard_collection_reservation();

-- +goose StatementBegin
CREATE FUNCTION app.guard_manual_payment_reservations() RETURNS trigger LANGUAGE plpgsql SECURITY DEFINER SET search_path=app,pg_catalog AS $$
DECLARE remaining BIGINT; held BIGINT;
BEGIN
 IF NEW.source_type='kredit_collection' THEN RETURN NEW; END IF;
 IF EXISTS(SELECT 1 FROM app.payments WHERE idempotency_key=NEW.idempotency_key) THEN RETURN NEW; END IF;
 SELECT outstanding_kobo INTO remaining FROM app.obligations WHERE id=NEW.obligation_id FOR UPDATE;
 SELECT COALESCE(SUM(reserved_amount_kobo),0) INTO held FROM app.collection_reservations WHERE obligation_id=NEW.obligation_id AND state IN ('PROCESSING','COMPLETED');
 IF NEW.amount_kobo>remaining-held THEN RAISE EXCEPTION 'payment conflicts with an unresolved debit; reconcile first'; END IF;
 RETURN NEW;
END $$;
-- +goose StatementEnd
CREATE TRIGGER guard_manual_payment_reservations BEFORE INSERT ON app.payments FOR EACH ROW EXECUTE FUNCTION app.guard_manual_payment_reservations();

-- +goose StatementBegin
DO $$ BEGIN
 IF EXISTS(SELECT 1 FROM pg_roles WHERE rolname='kredit_app') THEN GRANT SELECT,INSERT ON app.provider_customer_bindings TO kredit_app; END IF;
 IF EXISTS(SELECT 1 FROM pg_roles WHERE rolname='kredit_worker') THEN
 GRANT SELECT ON app.provider_customer_bindings TO kredit_worker;
 GRANT EXECUTE ON FUNCTION app.payment_mandate_by_provider(TEXT,TEXT),app.credit_snapshot_by_id(TEXT),app.credit_snapshot_by_obligation(TEXT) TO kredit_worker;
 END IF;
END $$;
-- +goose StatementEnd

CREATE TABLE app.collection_events (
 id UUID PRIMARY KEY DEFAULT uuidv7(),
 attempt_id UUID NOT NULL REFERENCES app.collection_attempts(id),
 obligation_id UUID NOT NULL REFERENCES app.obligations(id),
 event_type TEXT NOT NULL,
 amount_kobo BIGINT NOT NULL CHECK(amount_kobo>=0),
 actor TEXT NOT NULL DEFAULT 'collection_engine',
 correlation_id TEXT NOT NULL,
 occurred_at TIMESTAMPTZ NOT NULL DEFAULT now(),
 idempotency_key TEXT NOT NULL UNIQUE
);
ALTER TABLE app.collection_events ENABLE ROW LEVEL SECURITY;
CREATE POLICY collection_event_runtime ON app.collection_events USING(current_user IN ('kredit_app','kredit_worker')) WITH CHECK(current_user IN ('kredit_app','kredit_worker'));
-- +goose StatementBegin
CREATE FUNCTION app.prevent_collection_event_mutation() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'financial events are immutable'; END $$;
-- +goose StatementEnd
CREATE TRIGGER collection_event_immutable BEFORE UPDATE OR DELETE ON app.collection_events FOR EACH ROW EXECUTE FUNCTION app.prevent_collection_event_mutation();
-- +goose StatementBegin
DO $$ BEGIN
 IF EXISTS(SELECT 1 FROM pg_roles WHERE rolname='kredit_app') THEN GRANT SELECT,INSERT ON app.collection_events TO kredit_app; END IF;
 IF EXISTS(SELECT 1 FROM pg_roles WHERE rolname='kredit_worker') THEN GRANT SELECT,INSERT ON app.collection_events TO kredit_worker; END IF;
END $$;
-- +goose StatementEnd

-- +goose Down
DROP POLICY IF EXISTS payment_mandate_worker_read ON app.payment_mandates;
DROP TABLE IF EXISTS app.collection_events;
DROP FUNCTION IF EXISTS app.prevent_collection_event_mutation();
DROP TRIGGER IF EXISTS guard_manual_payment_reservations ON app.payments;
DROP FUNCTION IF EXISTS app.guard_manual_payment_reservations();
DROP TRIGGER IF EXISTS guard_collection_reservation ON app.collection_reservations;
DROP FUNCTION IF EXISTS app.guard_collection_reservation();
DROP INDEX IF EXISTS app.collection_mandate_reservations_idx;
ALTER TABLE app.collection_reservations DROP COLUMN IF EXISTS mandate_id;
ALTER TABLE app.collection_attempts DROP COLUMN IF EXISTS next_retry_at;
ALTER TABLE app.payment_mandates DROP COLUMN IF EXISTS metadata, DROP COLUMN IF EXISTS supplier_organization_id;
DROP TABLE IF EXISTS app.provider_customer_bindings;
