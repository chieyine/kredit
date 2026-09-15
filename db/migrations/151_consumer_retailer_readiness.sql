-- +goose Up
-- Customers can check and lock readiness without gaining write access to the
-- retailer's private onboarding or receiving-account settings.
-- +goose StatementBegin
CREATE FUNCTION app.consumer_retailer_ready(org uuid) RETURNS boolean LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,app AS $$
DECLARE ready boolean;
BEGIN
 SELECT c.enabled AND p.readiness_state='pilot_ready' AND p.kyb_state='approved'
 AND (p.kyb_expires_at IS NULL OR p.kyb_expires_at>now()) AND p.settlement_state='verified' AND p.billing_state='configured'
 INTO ready FROM app.consumer_settings c JOIN app.supplier_onboarding_profiles p USING(organization_id) WHERE c.organization_id=org FOR SHARE OF c,p;
 RETURN COALESCE(ready,false);
END $$;
CREATE OR REPLACE FUNCTION app.consumer_acceptance_guard() RETURNS trigger LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,app AS $$
BEGIN
 IF OLD.accepted_at IS NULL AND NEW.accepted_at IS NOT NULL THEN
  IF OLD.state<>'offered' OR NEW.state<>'active' OR NEW.buyer_user_id IS DISTINCT FROM app.current_user_id() OR NOT app.consumer_contact_matches(OLD.target_type,OLD.target_value) THEN RAISE EXCEPTION 'Customer acceptance required'; END IF;
  IF NOT app.consumer_retailer_ready(OLD.organization_id) THEN RAISE EXCEPTION 'Retailer consumer sales are not active'; END IF;
 END IF;
 IF NEW.released_at IS DISTINCT FROM OLD.released_at AND NOT(app.consumer_seller_role(OLD.organization_id,ARRAY['owner','administrator','sales']) OR app.has_admin_role(app.current_user_id(),ARRAY['platform_owner'])) THEN RAISE EXCEPTION 'Retailer release permission required'; END IF;
 IF NEW.received_at IS DISTINCT FROM OLD.received_at AND OLD.buyer_user_id IS DISTINCT FROM app.current_user_id() THEN RAISE EXCEPTION 'Customer receipt confirmation required'; END IF;
 RETURN NEW;
END $$;
-- +goose StatementEnd
REVOKE ALL ON FUNCTION app.consumer_retailer_ready(uuid) FROM PUBLIC;
-- +goose StatementBegin
DO $$ BEGIN IF EXISTS(SELECT 1 FROM pg_roles WHERE rolname='kredit_app') THEN GRANT EXECUTE ON FUNCTION app.consumer_retailer_ready(uuid) TO kredit_app; END IF; END $$;
-- +goose StatementEnd
-- +goose Down
-- +goose StatementBegin
DO $$ BEGIN RAISE EXCEPTION 'Consumer readiness guards require forward recovery'; END $$;
-- +goose StatementEnd
