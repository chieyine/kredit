-- +goose Up
-- A representative row is historical evidence, not a permanent access grant.
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION app.purchasing_authority_current(profile_id uuid,supplier_id uuid DEFAULT NULL) RETURNS boolean
LANGUAGE sql STABLE SECURITY DEFINER SET search_path=pg_catalog,app AS $$
 SELECT NOT EXISTS(SELECT 1 FROM app.businesses b WHERE b.id=profile_id AND b.organization_id IS NOT NULL
 AND (b.owner_user_id=app.current_user_id()
 OR EXISTS(SELECT 1 FROM app.purchasing_delegations d WHERE d.organization_id=b.organization_id AND d.user_id=app.current_user_id())
 OR EXISTS(SELECT 1 FROM app.business_representatives r JOIN app.persons p ON p.id=r.person_id WHERE r.business_id=b.id AND p.user_id=app.current_user_id()))
 AND NOT app.can_purchase(b.id)
 AND NOT EXISTS(SELECT 1 FROM app.memberships m JOIN app.users u ON u.id=m.user_id JOIN app.organizations o ON o.id=m.organization_id WHERE m.organization_id=supplier_id AND supplier_id=app.current_organization_id() AND m.user_id=app.current_user_id() AND m.status='active' AND u.status='active' AND o.status<>'suspended'));
$$;
-- +goose StatementEnd
CREATE POLICY purchasing_profile_owner_write ON app.businesses AS RESTRICTIVE FOR UPDATE
USING(current_user='kredit_worker' OR owner_user_id=app.current_user_id())
WITH CHECK(current_user='kredit_worker' OR owner_user_id=app.current_user_id());
-- +goose Down
-- +goose StatementBegin
DO $$ BEGIN RAISE EXCEPTION 'Delegated revocation requires forward recovery'; END $$;
-- +goose StatementEnd
