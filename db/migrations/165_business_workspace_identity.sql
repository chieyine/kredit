-- +goose Up
-- A purchasing profile belongs to a business workspace. Keeping verification
-- evidence separate does not create a second selling identity.
CREATE UNIQUE INDEX business_workspace_profile_unique ON app.businesses(organization_id) WHERE organization_id IS NOT NULL;

-- +goose StatementBegin
CREATE FUNCTION app.ensure_business_workspace(profile_id uuid) RETURNS uuid
LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,app SET row_security=off AS $$
DECLARE profile app.businesses; workspace uuid; actor uuid;
BEGIN
 actor := app.current_user_id();
 IF actor IS NULL THEN RAISE EXCEPTION 'business authority required' USING ERRCODE='42501'; END IF;
 SELECT * INTO profile FROM app.businesses WHERE id=profile_id AND owner_user_id=actor FOR UPDATE;
 IF NOT FOUND THEN RAISE EXCEPTION 'business authority required' USING ERRCODE='42501'; END IF;
 IF profile.status IN ('suspended','closed') THEN RAISE EXCEPTION 'business is restricted' USING ERRCODE='42501'; END IF;
 IF profile.organization_id IS NOT NULL THEN
  IF NOT EXISTS(SELECT 1 FROM app.memberships WHERE organization_id=profile.organization_id AND user_id=actor AND status='active' AND role='owner') THEN
   RAISE EXCEPTION 'business authority required' USING ERRCODE='42501';
  END IF;
  RETURN profile.organization_id;
 END IF;
 workspace := profile.id;
 -- Never attach a colliding identifier to an unrelated existing workspace.
 INSERT INTO app.organizations(id,legal_name,trading_name,business_type,registration_info,business_address,industry,status)
 VALUES(workspace,profile.legal_name,profile.trading_name,profile.business_type,profile.registration_info,profile.business_address,profile.industry,'onboarding');
 INSERT INTO app.memberships(organization_id,user_id,role,status,accepted_at) VALUES(workspace,actor,'owner','active',now());
 UPDATE app.businesses SET organization_id=workspace WHERE id=profile_id;
 RETURN workspace;
END $$;
-- +goose StatementEnd
REVOKE ALL ON FUNCTION app.ensure_business_workspace(uuid) FROM PUBLIC;
-- +goose StatementBegin
DO $$ BEGIN
 IF EXISTS(SELECT 1 FROM pg_roles WHERE rolname='kredit_app') THEN
  GRANT EXECUTE ON FUNCTION app.ensure_business_workspace(uuid) TO kredit_app;
 END IF;
END $$;
-- +goose StatementEnd

-- +goose Down
DROP FUNCTION IF EXISTS app.ensure_business_workspace(uuid);
DROP INDEX IF EXISTS app.business_workspace_profile_unique;
