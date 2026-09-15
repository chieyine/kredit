-- +goose Up
-- Accepting a personal purchase cannot be done by the retailer or by an
-- account other than the exact verified contact to whom it was offered.
-- +goose StatementBegin
CREATE FUNCTION app.consumer_acceptance_guard() RETURNS trigger LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,app AS $$
BEGIN
 IF OLD.accepted_at IS NULL AND NEW.accepted_at IS NOT NULL THEN
  IF OLD.state<>'offered' OR NEW.state<>'active' OR NEW.buyer_user_id IS DISTINCT FROM app.current_user_id() OR NOT app.consumer_contact_matches(OLD.target_type,OLD.target_value) THEN RAISE EXCEPTION 'Customer acceptance required'; END IF;
  IF NOT EXISTS(SELECT 1 FROM app.consumer_settings c JOIN app.supplier_onboarding_profiles p USING(organization_id) WHERE c.organization_id=OLD.organization_id AND c.enabled AND p.readiness_state='pilot_ready') THEN RAISE EXCEPTION 'Retailer consumer sales are not active'; END IF;
 END IF;
 IF NEW.released_at IS DISTINCT FROM OLD.released_at AND NOT(app.consumer_seller_role(OLD.organization_id,ARRAY['owner','administrator','sales']) OR app.has_admin_role(app.current_user_id(),ARRAY['platform_owner'])) THEN RAISE EXCEPTION 'Retailer release permission required'; END IF;
 IF NEW.received_at IS DISTINCT FROM OLD.received_at AND OLD.buyer_user_id IS DISTINCT FROM app.current_user_id() THEN RAISE EXCEPTION 'Customer receipt confirmation required'; END IF;
 RETURN NEW;
END $$;
-- +goose StatementEnd
CREATE TRIGGER consumer_acceptance_guard BEFORE UPDATE ON app.consumer_sales FOR EACH ROW EXECUTE FUNCTION app.consumer_acceptance_guard();
REVOKE ALL ON FUNCTION app.consumer_acceptance_guard() FROM PUBLIC;
-- +goose StatementBegin
DO $$ BEGIN
 IF EXISTS(SELECT 1 FROM pg_roles WHERE rolname='kredit_worker') THEN GRANT EXECUTE ON FUNCTION app.has_admin_role(uuid,text[]) TO kredit_worker; END IF;
END $$;
-- +goose StatementEnd
-- +goose Down
-- +goose StatementBegin
DO $$ BEGIN RAISE EXCEPTION 'Consumer acceptance evidence requires forward recovery'; END $$;
-- +goose StatementEnd
