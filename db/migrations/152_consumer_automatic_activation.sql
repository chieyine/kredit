-- +goose Up
ALTER TABLE app.consumer_settings ADD COLUMN registration_id text REFERENCES app.settlement_registrations(id);
CREATE TABLE app.consumer_restrictions(
 organization_id uuid PRIMARY KEY REFERENCES app.organizations(id),blocked boolean NOT NULL DEFAULT false,
 reason text NOT NULL CHECK(length(reason) BETWEEN 20 AND 2000),updated_by uuid NOT NULL REFERENCES app.users(id),updated_at timestamptz NOT NULL DEFAULT now()
);
ALTER TABLE app.consumer_restrictions ENABLE ROW LEVEL SECURITY;
CREATE POLICY consumer_restriction_owner ON app.consumer_restrictions USING(app.has_admin_role(app.current_user_id(),ARRAY['platform_owner']));
CREATE POLICY consumer_restriction_read ON app.consumer_restrictions FOR SELECT USING(organization_id=app.current_organization_id());
INSERT INTO app.consumer_restrictions(organization_id,blocked,reason,updated_by) SELECT organization_id,true,review_evidence,updated_by FROM app.consumer_settings WHERE NOT enabled;
CREATE POLICY consumer_settings_self_service ON app.consumer_settings FOR ALL USING(app.consumer_seller_role(organization_id,ARRAY['owner','administrator','finance','collections'])) WITH CHECK(app.consumer_seller_role(organization_id,ARRAY['owner','administrator','finance','collections']));
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION app.consumer_retailer_ready(org uuid) RETURNS boolean LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,app AS $$
DECLARE ready boolean;
BEGIN
 -- Serialize exceptional restrictions with setup, offers and acceptance.
 PERFORM pg_advisory_xact_lock(hashtextextended('consumer-eligibility:'||org::text,0));
 IF EXISTS(SELECT 1 FROM app.consumer_restrictions WHERE organization_id=org AND blocked) THEN RETURN false; END IF;
 SELECT p.readiness_state='pilot_ready' AND p.kyb_state='approved'
 AND (p.kyb_expires_at IS NULL OR p.kyb_expires_at>now()) AND p.settlement_state='verified' AND p.billing_state='configured'
 AND r.state='REGISTERED' AND r.provider=p.settlement_provider AND r.result->>'provider_reference'=p.settlement_provider_reference
 AND r.account_last4=right(c.account_number,4) AND r.result->>'account_name'=c.account_name
 INTO ready FROM app.consumer_settings c JOIN app.supplier_onboarding_profiles p USING(organization_id)
 JOIN app.settlement_registrations r ON r.id=c.registration_id AND r.organization_id=org
 WHERE c.organization_id=org FOR SHARE OF c,p,r;
 RETURN COALESCE(ready,false);
END $$;
-- +goose StatementEnd
-- +goose StatementBegin
DO $$ BEGIN IF EXISTS(SELECT 1 FROM pg_roles WHERE rolname='kredit_app') THEN GRANT SELECT,INSERT,UPDATE ON app.consumer_restrictions TO kredit_app; END IF; END $$;
-- +goose StatementEnd
-- +goose Down
-- +goose StatementBegin
DO $$ BEGIN RAISE EXCEPTION 'Consumer activation changes require forward recovery'; END $$;
-- +goose StatementEnd
