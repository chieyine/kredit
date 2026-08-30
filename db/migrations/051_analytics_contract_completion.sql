-- +goose Up
-- Align the implemented vocabulary with the Wave 0 contract. Existing event
-- IDs and deduplication keys remain stable; consumers can migrate by name.
UPDATE app.analytics_events SET name = CASE name
  WHEN 'mandate.active' THEN 'mandate.activated'
  WHEN 'receipt.issue_raised' THEN 'receipt.issue_reported'
  WHEN 'payment.link_created' THEN 'payment_link.created'
  WHEN 'payment.collection_failed' THEN 'collection.failed'
  WHEN 'payment.recovered' THEN 'collection.recovered'
  WHEN 'drawdown.reserved' THEN 'trade_line.drawdown_reserved'
  WHEN 'drawdown.confirmed' THEN 'trade_line.drawdown_confirmed'
  WHEN 'drawdown.released' THEN 'trade_line.drawdown_released'
  WHEN 'drawdown.activated' THEN 'trade_line.drawdown_activated'
  WHEN 'drawdown.expired' THEN 'trade_line.drawdown_expired'
  ELSE name END;

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
    event_name := CASE row_data->>'state' WHEN 'confirmed' THEN 'receipt.confirmed' ELSE 'receipt.issue_reported' END;
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
    event_name := CASE lower(NEW.status) WHEN 'active' THEN 'mandate.activated' WHEN 'failed' THEN 'mandate.failed' WHEN 'cancelled' THEN 'mandate.cancelled' ELSE NULL END;
  END IF;
  IF event_name IS NOT NULL THEN
    SELECT supplier_organization_id INTO org_id FROM app.credit_requests WHERE id=NEW.credit_request_id;
    PERFORM app.record_product_event(event_name, NEW.id, org_id, 'operations_reliability', COALESCE(NEW.activated_at,NEW.created_at,NOW()), 'mandate:'||NEW.id||':'||event_name, jsonb_build_object('status',lower(NEW.status)));
  END IF;
  RETURN NEW;
END $$;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION app.analytics_payment_mandate_event() RETURNS TRIGGER LANGUAGE plpgsql AS $$
DECLARE event_name TEXT; org_id UUID;
BEGIN
  IF TG_OP='INSERT' THEN
    event_name := CASE NEW.state WHEN 'active' THEN 'mandate.activated' WHEN 'failed' THEN 'mandate.failed' WHEN 'cancelled' THEN 'mandate.cancelled' ELSE 'mandate.started' END;
  ELSIF OLD.state IS DISTINCT FROM NEW.state THEN
    event_name := CASE NEW.state WHEN 'active' THEN 'mandate.activated' WHEN 'failed' THEN 'mandate.failed' WHEN 'cancelled' THEN 'mandate.cancelled' ELSE NULL END;
  END IF;
  IF event_name IS NOT NULL THEN
    SELECT supplier_organization_id INTO org_id FROM app.trade_lines WHERE mandate_id=NEW.id ORDER BY created_at LIMIT 1;
    PERFORM app.record_product_event(event_name, NEW.id, org_id, 'operations_reliability', COALESCE(NEW.provider_updated_at,NEW.created_at,NOW()), 'payment_mandate:'||NEW.id||':'||event_name, jsonb_build_object('status',NEW.state));
  END IF;
  RETURN NEW;
END $$;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION app.analytics_obligation_event() RETURNS TRIGGER LANGUAGE plpgsql AS $$
BEGIN
  PERFORM app.record_product_event('obligation.activated', NEW.id, NEW.supplier_organization_id, 'pilot_measurement', NEW.activated_at, 'obligation:'||NEW.id||':activated', jsonb_build_object('currency',NEW.currency));
  IF EXISTS (SELECT 1 FROM app.obligations o WHERE o.id<>NEW.id AND o.supplier_organization_id=NEW.supplier_organization_id AND o.buyer_business_id=NEW.buyer_business_id AND o.activated_at<=NEW.activated_at) THEN
    PERFORM app.record_product_event('credit.repeat_sale', NEW.id, NEW.supplier_organization_id, 'pilot_measurement', NEW.activated_at, 'obligation:'||NEW.id||':credit.repeat_sale', '{}'::jsonb);
  END IF;
  IF EXISTS (SELECT 1 FROM app.obligations o WHERE o.id<>NEW.id AND o.supplier_organization_id=NEW.supplier_organization_id AND o.activated_at<=NEW.activated_at) THEN
    PERFORM app.record_product_event('supplier.retained', NEW.supplier_organization_id, NEW.supplier_organization_id, 'pilot_measurement', NEW.activated_at, 'obligation:'||NEW.id||':supplier.retained', '{}'::jsonb);
  END IF;
  RETURN NEW;
END $$;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION app.analytics_collection_event() RETURNS TRIGGER LANGUAGE plpgsql AS $$
DECLARE event_name TEXT; org_id UUID;
BEGIN
  IF TG_OP='INSERT' THEN event_name := 'collection.submitted';
  ELSIF OLD.state IS DISTINCT FROM NEW.state THEN
    event_name := CASE NEW.state WHEN 'FAILED' THEN 'collection.failed' WHEN 'SUCCEEDED' THEN CASE WHEN NEW.attempt_number>1 THEN 'collection.recovered' ELSE 'payment.collected' END WHEN 'PARTIAL' THEN 'payment.collected' ELSE NULL END;
  END IF;
  IF event_name IS NOT NULL THEN
    SELECT supplier_organization_id INTO org_id FROM app.obligations WHERE id=NEW.obligation_id;
    PERFORM app.record_product_event(event_name, NEW.id, org_id, 'operations_reliability', COALESCE(NEW.final_at,NEW.requested_at,NOW()), 'collection_attempt:'||NEW.id||':'||event_name, jsonb_build_object('state',NEW.state,'attempt_number',NEW.attempt_number));
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
    IF TG_OP='INSERT' THEN event_name := 'trade_line.drawdown_reserved';
    ELSIF OLD.state IS DISTINCT FROM NEW.state THEN event_name := CASE NEW.state WHEN 'CONFIRMED' THEN 'trade_line.drawdown_confirmed' WHEN 'RELEASED' THEN 'trade_line.drawdown_released' WHEN 'EXPIRED' THEN 'trade_line.drawdown_expired' ELSE NULL END; END IF;
  ELSE
    subject_id := NEW.id; event_time := COALESCE(NEW.activated_at,NEW.buyer_confirmed_at,NEW.created_at);
    IF TG_OP='UPDATE' AND OLD.state IS DISTINCT FROM NEW.state THEN event_name := CASE NEW.state WHEN 'BUYER_CONFIRMED' THEN 'trade_line.drawdown_confirmed' WHEN 'ACTIVATED' THEN 'trade_line.drawdown_activated' ELSE NULL END; END IF;
  END IF;
  IF event_name IS NOT NULL THEN
    SELECT supplier_organization_id INTO org_id FROM app.trade_lines WHERE id=NEW.trade_line_id;
    PERFORM app.record_product_event(event_name, subject_id, org_id, 'pilot_measurement', event_time, TG_TABLE_NAME||':'||subject_id||':'||event_name, jsonb_build_object('state',NEW.state));
  END IF;
  RETURN NEW;
END $$;
-- +goose StatementEnd

DROP TRIGGER IF EXISTS product_analytics_payment_mandate ON app.payment_mandates;
CREATE TRIGGER product_analytics_payment_mandate AFTER INSERT OR UPDATE ON app.payment_mandates FOR EACH ROW EXECUTE FUNCTION app.analytics_payment_mandate_event();
DROP TRIGGER IF EXISTS product_analytics_collection ON app.collection_attempts;
CREATE TRIGGER product_analytics_collection AFTER INSERT OR UPDATE ON app.collection_attempts FOR EACH ROW EXECUTE FUNCTION app.analytics_collection_event();

-- Backfill newly introduced authoritative events. The per-source keys remain
-- deterministic, so concurrent migration/application activity is harmless.
SELECT app.record_product_event(CASE m.state WHEN 'active' THEN 'mandate.activated' WHEN 'failed' THEN 'mandate.failed' WHEN 'cancelled' THEN 'mandate.cancelled' ELSE 'mandate.started' END,m.id,NULL,'operations_reliability',COALESCE(m.provider_updated_at,m.created_at),'payment_mandate:'||m.id||':'||CASE m.state WHEN 'active' THEN 'mandate.activated' WHEN 'failed' THEN 'mandate.failed' WHEN 'cancelled' THEN 'mandate.cancelled' ELSE 'mandate.started' END,jsonb_build_object('status',m.state)) FROM app.payment_mandates m;
SELECT app.record_product_event('collection.submitted',c.id,o.supplier_organization_id,'operations_reliability',c.requested_at,'collection_attempt:'||c.id||':collection.submitted',jsonb_build_object('state',c.state,'attempt_number',c.attempt_number)) FROM app.collection_attempts c JOIN app.obligations o ON o.id=c.obligation_id;
SELECT app.record_product_event('credit.repeat_sale',o.id,o.supplier_organization_id,'pilot_measurement',o.activated_at,'obligation:'||o.id||':credit.repeat_sale','{}') FROM app.obligations o WHERE EXISTS(SELECT 1 FROM app.obligations prior WHERE prior.id<>o.id AND prior.supplier_organization_id=o.supplier_organization_id AND prior.buyer_business_id=o.buyer_business_id AND (prior.activated_at,prior.id)<(o.activated_at,o.id));
SELECT app.record_product_event('supplier.retained',o.supplier_organization_id,o.supplier_organization_id,'pilot_measurement',o.activated_at,'obligation:'||o.id||':supplier.retained','{}') FROM app.obligations o WHERE EXISTS(SELECT 1 FROM app.obligations prior WHERE prior.id<>o.id AND prior.supplier_organization_id=o.supplier_organization_id AND (prior.activated_at,prior.id)<(o.activated_at,o.id));

-- +goose Down
DROP TRIGGER IF EXISTS product_analytics_payment_mandate ON app.payment_mandates;
DROP FUNCTION IF EXISTS app.analytics_payment_mandate_event();
-- Event rows are append-only evidence and are intentionally not deleted by a
-- development rollback. Migration 050 functions remain compatible.
