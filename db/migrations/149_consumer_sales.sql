-- +goose Up
CREATE TABLE app.consumer_settings(
 organization_id uuid PRIMARY KEY REFERENCES app.organizations(id),enabled boolean NOT NULL DEFAULT false,
 bank_name text NOT NULL,account_name text NOT NULL,account_number text NOT NULL CHECK(account_number ~ '^[0-9]{10}$'),
 review_evidence text NOT NULL CHECK(length(review_evidence) BETWEEN 20 AND 2000),
 updated_by uuid NOT NULL REFERENCES app.users(id),updated_at timestamptz NOT NULL DEFAULT now()
);
ALTER TABLE app.consumer_settings ENABLE ROW LEVEL SECURITY;
CREATE POLICY consumer_settings_owner ON app.consumer_settings USING(app.has_admin_role(app.current_user_id(),ARRAY['platform_owner']));
CREATE POLICY consumer_settings_read ON app.consumer_settings FOR SELECT USING(organization_id=app.current_organization_id());
CREATE TABLE app.consumer_sales(
 id uuid PRIMARY KEY DEFAULT uuidv7(), organization_id uuid NOT NULL REFERENCES app.organizations(id),
 created_by uuid NOT NULL REFERENCES app.users(id), buyer_user_id uuid REFERENCES app.users(id),
 target_type text NOT NULL CHECK(target_type IN ('email','phone')), target_value text NOT NULL,
 terms jsonb NOT NULL, agreement_hash text NOT NULL, state text NOT NULL DEFAULT 'offered' CHECK(state IN ('offered','active','cancelled','completed')),
 customer_name text NOT NULL DEFAULT '', delivery_address text NOT NULL DEFAULT '',
 accepted_at timestamptz, released_at timestamptz, received_at timestamptz,
 case_state text NOT NULL DEFAULT '' CHECK(case_state IN ('','requested','rejected','escalated','approved')),
 version bigint NOT NULL DEFAULT 1, created_at timestamptz NOT NULL DEFAULT now(), updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX consumer_sales_business ON app.consumer_sales(organization_id,created_at DESC);
CREATE INDEX consumer_sales_buyer ON app.consumer_sales(buyer_user_id,created_at DESC);
CREATE INDEX consumer_sales_target ON app.consumer_sales(target_type,target_value) WHERE buyer_user_id IS NULL;
CREATE TABLE app.consumer_events(
 id uuid PRIMARY KEY DEFAULT uuidv7(), sale_id uuid NOT NULL REFERENCES app.consumer_sales(id),
 actor_id uuid NOT NULL REFERENCES app.users(id), action text NOT NULL, amount_kobo bigint NOT NULL DEFAULT 0 CHECK(amount_kobo>=0),
 reference text NOT NULL DEFAULT '', related_id uuid REFERENCES app.consumer_events(id), note text NOT NULL DEFAULT '',
 occurred_at timestamptz NOT NULL DEFAULT now(), created_at timestamptz NOT NULL DEFAULT now(),
 UNIQUE(sale_id,action,reference)
);
CREATE UNIQUE INDEX consumer_receipt_reference ON app.consumer_events(reference) WHERE action IN ('payment','refund') AND reference<>'';
CREATE UNIQUE INDEX consumer_claim_decision ON app.consumer_events(related_id) WHERE action IN ('payment','reject_claim');
CREATE UNIQUE INDEX consumer_payment_reversal ON app.consumer_events(related_id) WHERE action='reverse_payment';
-- +goose StatementBegin
CREATE FUNCTION app.consumer_contact_matches(kind text,value text) RETURNS boolean LANGUAGE sql STABLE SECURITY DEFINER SET search_path=pg_catalog,app AS $$
 SELECT EXISTS(SELECT 1 FROM app.users WHERE id=app.current_user_id() AND status='active' AND CASE WHEN kind='email' THEN normalized_email=value ELSE normalized_phone=value END)
$$;
CREATE FUNCTION app.consumer_seller_role(org uuid, roles text[]) RETURNS boolean LANGUAGE sql STABLE SECURITY DEFINER SET search_path=pg_catalog,app AS $$
 SELECT EXISTS(SELECT 1 FROM app.memberships m JOIN app.users u ON u.id=m.user_id WHERE m.organization_id=org AND m.user_id=app.current_user_id() AND m.status='active' AND u.status='active' AND m.role=ANY(roles))
$$;
-- +goose StatementEnd
REVOKE ALL ON FUNCTION app.consumer_contact_matches(text,text),app.consumer_seller_role(uuid,text[]) FROM PUBLIC;
ALTER TABLE app.consumer_sales ENABLE ROW LEVEL SECURITY;
ALTER TABLE app.consumer_events ENABLE ROW LEVEL SECURITY;
CREATE POLICY consumer_sale_read ON app.consumer_sales FOR SELECT USING(
 buyer_user_id=app.current_user_id() OR (buyer_user_id IS NULL AND app.consumer_contact_matches(target_type,target_value)) OR
 app.consumer_seller_role(organization_id,ARRAY['owner','administrator','finance','sales','collections','viewer']) OR app.has_admin_role(app.current_user_id(),ARRAY['platform_owner'])
);
CREATE POLICY consumer_sale_insert ON app.consumer_sales FOR INSERT WITH CHECK(created_by=app.current_user_id() AND app.consumer_seller_role(organization_id,ARRAY['owner','administrator','sales']));
CREATE POLICY consumer_sale_update ON app.consumer_sales FOR UPDATE USING(
 buyer_user_id=app.current_user_id() OR (buyer_user_id IS NULL AND app.consumer_contact_matches(target_type,target_value)) OR app.consumer_seller_role(organization_id,ARRAY['owner','administrator','finance','sales','collections']) OR app.has_admin_role(app.current_user_id(),ARRAY['platform_owner'])
) WITH CHECK(buyer_user_id=app.current_user_id() OR (buyer_user_id IS NULL AND app.consumer_contact_matches(target_type,target_value)) OR app.consumer_seller_role(organization_id,ARRAY['owner','administrator','finance','sales','collections']) OR app.has_admin_role(app.current_user_id(),ARRAY['platform_owner']));
CREATE POLICY consumer_event_read ON app.consumer_events FOR SELECT USING(EXISTS(SELECT 1 FROM app.consumer_sales s WHERE s.id=sale_id));
CREATE POLICY consumer_event_insert ON app.consumer_events FOR INSERT WITH CHECK(actor_id=app.current_user_id() AND EXISTS(SELECT 1 FROM app.consumer_sales s WHERE s.id=sale_id));
-- +goose StatementBegin
CREATE FUNCTION app.consumer_terms_immutable() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN
 IF (NEW.organization_id,NEW.created_by,NEW.target_type,NEW.target_value,NEW.terms,NEW.agreement_hash,NEW.created_at) IS DISTINCT FROM (OLD.organization_id,OLD.created_by,OLD.target_type,OLD.target_value,OLD.terms,OLD.agreement_hash,OLD.created_at) THEN RAISE EXCEPTION 'Consumer agreement is immutable'; END IF;
 IF OLD.buyer_user_id IS NOT NULL AND (NEW.buyer_user_id,NEW.customer_name,NEW.delivery_address,NEW.accepted_at) IS DISTINCT FROM (OLD.buyer_user_id,OLD.customer_name,OLD.delivery_address,OLD.accepted_at) THEN RAISE EXCEPTION 'Consumer acceptance is immutable'; END IF;
 RETURN NEW;
END $$;
CREATE FUNCTION app.consumer_event_immutable() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'Consumer history is immutable'; END $$;
-- +goose StatementEnd
CREATE TRIGGER consumer_terms_guard BEFORE UPDATE ON app.consumer_sales FOR EACH ROW EXECUTE FUNCTION app.consumer_terms_immutable();
CREATE TRIGGER consumer_event_guard BEFORE UPDATE OR DELETE ON app.consumer_events FOR EACH ROW EXECUTE FUNCTION app.consumer_event_immutable();
INSERT INTO ledger.accounts(code,name,normal_balance) VALUES
 ('CONSUMER_PAYMENT_CONTROL','Consumer payments received by retailer','debit'),
 ('CONSUMER_CUSTOMER_FUNDS','Consumer funds held by retailer','credit'),
 ('CONSUMER_RECEIVABLE','Consumer delivered-goods receivable','debit'),
 ('CONSUMER_SALES_CONTROL','Consumer delivered goods control','credit') ON CONFLICT(code) DO NOTHING;
-- +goose StatementBegin
DO $$ BEGIN
 IF EXISTS(SELECT 1 FROM pg_roles WHERE rolname='kredit_app') THEN
 GRANT SELECT,INSERT,UPDATE ON app.consumer_sales,app.consumer_settings TO kredit_app;
 GRANT SELECT,INSERT ON app.consumer_events TO kredit_app;
 GRANT EXECUTE ON FUNCTION app.consumer_contact_matches(text,text),app.consumer_seller_role(uuid,text[]) TO kredit_app;
 END IF;
 IF EXISTS(SELECT 1 FROM pg_roles WHERE rolname='kredit_worker') THEN
 GRANT SELECT ON app.consumer_sales,app.consumer_events TO kredit_worker;
 GRANT EXECUTE ON FUNCTION app.consumer_contact_matches(text,text),app.consumer_seller_role(uuid,text[]) TO kredit_worker;
 END IF;
END $$;
-- +goose StatementEnd
-- +goose StatementBegin
CREATE FUNCTION app.consumer_reminder_work() RETURNS TABLE(sale_id uuid,organization_id uuid,buyer_user_id uuid) LANGUAGE sql STABLE SECURITY DEFINER SET search_path=pg_catalog,app AS $$
 SELECT id,organization_id,buyer_user_id FROM app.consumer_sales WHERE state='active' AND accepted_at IS NOT NULL AND case_state NOT IN ('requested','escalated') ORDER BY id
$$;
CREATE FUNCTION app.consumer_event_permission() RETURNS trigger LANGUAGE plpgsql SET search_path=pg_catalog,app AS $$
DECLARE org uuid; buyer uuid;
BEGIN
 SELECT organization_id,buyer_user_id INTO org,buyer FROM app.consumer_sales WHERE id=NEW.sale_id;
 IF NEW.related_id IS NOT NULL AND NOT EXISTS(SELECT 1 FROM app.consumer_events WHERE id=NEW.related_id AND sale_id=NEW.sale_id) THEN RAISE EXCEPTION 'Related event belongs to another sale'; END IF;
 IF NEW.action IN ('payment','refund','reverse_payment','reduce_price','approve_return','reject_return','reject_claim') AND NOT(app.consumer_seller_role(org,ARRAY['owner','administrator','finance','collections']) OR app.has_admin_role(app.current_user_id(),ARRAY['platform_owner'])) THEN RAISE EXCEPTION 'Financial permission required'; END IF;
 IF NEW.action='release' AND NOT(app.consumer_seller_role(org,ARRAY['owner','administrator','sales']) OR app.has_admin_role(app.current_user_id(),ARRAY['platform_owner'])) THEN RAISE EXCEPTION 'Release permission required'; END IF;
 IF NEW.action IN ('claim','received','request_return','escalate') AND buyer IS DISTINCT FROM app.current_user_id() THEN RAISE EXCEPTION 'Customer permission required'; END IF;
 RETURN NEW;
END $$;
-- +goose StatementEnd
CREATE TRIGGER consumer_event_permission_guard BEFORE INSERT ON app.consumer_events FOR EACH ROW EXECUTE FUNCTION app.consumer_event_permission();
REVOKE ALL ON FUNCTION app.consumer_reminder_work() FROM PUBLIC;
-- +goose StatementBegin
DO $$ BEGIN IF EXISTS(SELECT 1 FROM pg_roles WHERE rolname='kredit_worker') THEN GRANT EXECUTE ON FUNCTION app.consumer_reminder_work() TO kredit_worker; END IF; END $$;
-- +goose StatementEnd
-- +goose Down
-- +goose StatementBegin
DO $$ BEGIN RAISE EXCEPTION 'Consumer financial history requires forward recovery'; END $$;
-- +goose StatementEnd
