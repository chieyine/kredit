-- +goose Up
CREATE TABLE app.settlement_registrations(
 id text PRIMARY KEY CHECK(length(id)=64),
 organization_id uuid NOT NULL REFERENCES app.organizations(id),
 provider text NOT NULL,
 connection_identity text NOT NULL CHECK(length(connection_identity)=64),
 bank_code text NOT NULL CHECK(bank_code ~ '^[0-9]{3,6}$'),
 account_last4 text NOT NULL CHECK(account_last4 ~ '^[0-9]{4}$'),
 state text NOT NULL CHECK(state IN ('STARTED','REGISTERED','READY')),
 result jsonb CHECK(result IS NULL OR jsonb_typeof(result)='object'),
 CHECK((state='REGISTERED')=(result IS NOT NULL)),
 created_at timestamptz NOT NULL DEFAULT now(),updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX settlement_one_uncertain_registration ON app.settlement_registrations(organization_id) WHERE state='STARTED';
ALTER TABLE app.settlement_registrations ENABLE ROW LEVEL SECURITY;
CREATE POLICY settlement_registration_tenant ON app.settlement_registrations
 USING(organization_id=app.current_organization_id()) WITH CHECK(organization_id=app.current_organization_id());
-- +goose StatementBegin
DO $$ BEGIN
 IF EXISTS(SELECT 1 FROM pg_roles WHERE rolname='kredit_app') THEN GRANT SELECT,INSERT,UPDATE ON app.settlement_registrations TO kredit_app; REVOKE DELETE ON app.settlement_registrations FROM kredit_app; END IF;
END $$;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE FUNCTION app.settlement_registration_review() RETURNS jsonb LANGUAGE plpgsql STABLE SECURITY DEFINER SET search_path=pg_catalog,app AS $$
BEGIN
 IF NOT app.has_admin_role(app.current_user_id(),ARRAY['platform_owner']) THEN RAISE EXCEPTION 'super-admin authority required'; END IF;
 RETURN jsonb_build_object(
 'registrations',COALESCE((SELECT jsonb_agg(to_jsonb(r)) FROM (SELECT id,organization_id,provider,bank_code,account_last4,created_at FROM app.settlement_registrations WHERE state='STARTED' AND created_at<now()-interval '1 minute' ORDER BY created_at LIMIT 100) r),'[]'::jsonb),
 'destinations',COALESCE((SELECT jsonb_agg(to_jsonb(p)) FROM (SELECT organization_id,version,settlement_provider,settlement_provider_reference,settlement_bank_name,settlement_account_name,settlement_account_last4,settlement_changed_at FROM app.supplier_onboarding_profiles WHERE settlement_state IN ('pending_verification','provider_review') ORDER BY settlement_changed_at LIMIT 100) p),'[]'::jsonb));
END $$;
-- +goose StatementEnd
REVOKE ALL ON FUNCTION app.settlement_registration_review() FROM PUBLIC;
-- +goose StatementBegin
DO $$ BEGIN
 IF EXISTS(SELECT 1 FROM pg_roles WHERE rolname='kredit_app') THEN GRANT EXECUTE ON FUNCTION app.settlement_registration_review() TO kredit_app; END IF;
END $$;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DO $$ BEGIN RAISE EXCEPTION 'Bank registration evidence requires forward recovery'; END $$;
-- +goose StatementEnd
