-- +goose Up
CREATE TABLE app.purchasing_delegations (
 organization_id uuid NOT NULL REFERENCES app.organizations(id),
 user_id uuid NOT NULL REFERENCES app.users(id),
 actions text[] NOT NULL DEFAULT '{}' CHECK(actions <@ ARRAY['read','review','accept','receive']::text[]),
 ceiling_kobo bigint NOT NULL CHECK(ceiling_kobo BETWEEN 0 AND 9007199254740991),
 expires_at timestamptz NOT NULL,
 version bigint NOT NULL DEFAULT 1 CHECK(version>0),
 updated_by uuid NOT NULL REFERENCES app.users(id),
 updated_at timestamptz NOT NULL DEFAULT now(),
 PRIMARY KEY(organization_id,user_id)
);
CREATE TABLE app.purchasing_delegation_history (
 organization_id uuid NOT NULL REFERENCES app.organizations(id),
 user_id uuid NOT NULL REFERENCES app.users(id),
 version bigint NOT NULL, actor_id uuid NOT NULL REFERENCES app.users(id),
 recorded_at timestamptz NOT NULL DEFAULT now(), evidence jsonb NOT NULL,
 PRIMARY KEY(organization_id,user_id,version)
);
-- +goose StatementBegin
CREATE FUNCTION app.can_purchase(profile_id uuid, action_name text DEFAULT 'read', amount_kobo bigint DEFAULT 0) RETURNS boolean
LANGUAGE sql STABLE SECURITY DEFINER SET search_path=pg_catalog,app SET row_security=off AS $$
 SELECT EXISTS(SELECT 1 FROM app.businesses b JOIN app.organizations o ON o.id=b.organization_id
 JOIN app.memberships m ON m.organization_id=o.id AND m.user_id=app.current_user_id()
 JOIN app.users u ON u.id=m.user_id
 LEFT JOIN app.purchasing_delegations d ON d.organization_id=o.id AND d.user_id=m.user_id
 WHERE b.id=profile_id AND m.status='active' AND u.status='active' AND o.status<>'suspended'
 AND ((m.role='owner' AND b.owner_user_id=m.user_id) OR
 (d.expires_at>statement_timestamp() AND 'read'=ANY(d.actions) AND action_name=ANY(d.actions)
 AND (action_name<>'accept' OR amount_kobo BETWEEN 0 AND d.ceiling_kobo))));
$$;
CREATE FUNCTION app.purchase_request_read(request_id uuid) RETURNS boolean
LANGUAGE sql STABLE SECURITY DEFINER SET search_path=pg_catalog,app SET row_security=off AS $$
 SELECT COALESCE((SELECT c.state<>'DRAFT' AND app.can_purchase(c.buyer_business_id) FROM app.credit_requests c WHERE c.id=request_id),false);
$$;
CREATE FUNCTION app.lock_purchase_permission(profile_id uuid,action_name text,amount_kobo bigint) RETURNS void
LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,app SET row_security=off AS $$
DECLARE workspace uuid;
BEGIN
 SELECT organization_id INTO workspace FROM app.businesses WHERE id=profile_id;
 PERFORM 1 FROM app.users WHERE id=app.current_user_id() AND status='active' FOR SHARE;
 PERFORM 1 FROM app.organizations WHERE id=workspace AND status<>'suspended' FOR SHARE;
 PERFORM 1 FROM app.memberships WHERE organization_id=workspace AND user_id=app.current_user_id() AND status='active' FOR SHARE;
 PERFORM 1 FROM app.purchasing_delegations WHERE organization_id=workspace AND user_id=app.current_user_id() FOR SHARE;
 IF NOT app.can_purchase(profile_id,action_name,amount_kobo) THEN RAISE EXCEPTION 'current purchasing permission required' USING ERRCODE='42501'; END IF;
END $$;
CREATE FUNCTION app.record_purchasing_delegation() RETURNS trigger
LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,app SET row_security=off AS $$
BEGIN
 IF NEW.updated_by IS DISTINCT FROM app.current_user_id() THEN RAISE EXCEPTION 'delegation actor mismatch' USING ERRCODE='42501'; END IF;
 PERFORM 1 FROM app.memberships m JOIN app.users u ON u.id=m.user_id JOIN app.organizations o ON o.id=m.organization_id WHERE m.organization_id=NEW.organization_id AND m.user_id=NEW.updated_by AND m.role='owner' AND m.status='active' AND u.status='active' AND o.status<>'suspended' FOR SHARE OF m,u,o;
 IF NOT FOUND THEN RAISE EXCEPTION 'current owner required' USING ERRCODE='42501'; END IF;
 IF TG_OP='UPDATE' AND ((NEW.organization_id,NEW.user_id) IS DISTINCT FROM (OLD.organization_id,OLD.user_id) OR NEW.version<>OLD.version+1) THEN RAISE EXCEPTION 'delegation changed' USING ERRCODE='23514'; END IF;
 IF TG_OP='INSERT' AND NEW.version<>1 THEN RAISE EXCEPTION 'initial version must be one' USING ERRCODE='23514'; END IF;
 IF cardinality(NEW.actions)>0 THEN
  IF NOT 'read'=ANY(NEW.actions) OR NEW.expires_at<=statement_timestamp() OR NEW.expires_at>statement_timestamp()+interval '366 days' THEN RAISE EXCEPTION 'permission requires read access and expiry within one year' USING ERRCODE='23514'; END IF;
  PERFORM 1 FROM app.memberships m JOIN app.users u ON u.id=m.user_id WHERE m.organization_id=NEW.organization_id AND m.user_id=NEW.user_id AND m.status='active' AND u.status='active' FOR SHARE OF m,u;
  IF NOT FOUND THEN RAISE EXCEPTION 'delegate must be current staff' USING ERRCODE='23514'; END IF;
 END IF;
 INSERT INTO app.purchasing_delegation_history(organization_id,user_id,version,actor_id,evidence) VALUES(NEW.organization_id,NEW.user_id,NEW.version,NEW.updated_by,to_jsonb(NEW));
 RETURN NEW;
END $$;
DO $$ DECLARE tab text; BEGIN
 FOREACH tab IN ARRAY ARRAY['purchasing_delegations','purchasing_delegation_history'] LOOP
  EXECUTE format('ALTER TABLE app.%I ENABLE ROW LEVEL SECURITY',tab);
  EXECUTE format('ALTER TABLE app.%I FORCE ROW LEVEL SECURITY',tab);
  EXECUTE format('CREATE POLICY purchasing_delegation_read ON app.%I FOR SELECT USING(organization_id=app.current_organization_id() AND EXISTS(SELECT 1 FROM app.memberships m JOIN app.users u ON u.id=m.user_id WHERE m.organization_id=%I.organization_id AND m.user_id=app.current_user_id() AND m.status=''active'' AND u.status=''active'' AND (m.role=''owner'' OR %I.user_id=m.user_id)))',tab,tab,tab);
 END LOOP;
END $$;
-- +goose StatementEnd
CREATE POLICY purchasing_delegation_insert ON app.purchasing_delegations FOR INSERT WITH CHECK(organization_id=app.current_organization_id() AND updated_by=app.current_user_id());
CREATE POLICY purchasing_delegation_update ON app.purchasing_delegations FOR UPDATE USING(organization_id=app.current_organization_id()) WITH CHECK(organization_id=app.current_organization_id() AND updated_by=app.current_user_id());
CREATE TRIGGER purchasing_delegation_history AFTER INSERT OR UPDATE ON app.purchasing_delegations FOR EACH ROW EXECUTE FUNCTION app.record_purchasing_delegation();
CREATE POLICY delegated_business_read ON app.businesses FOR SELECT USING(app.can_purchase(id));
CREATE POLICY delegated_credit_read ON app.credit_requests FOR SELECT USING(state<>'DRAFT' AND app.can_purchase(buyer_business_id));
CREATE POLICY delegated_obligation_read ON app.obligations FOR SELECT USING(app.can_purchase(buyer_business_id));
-- +goose StatementBegin
DO $$ DECLARE tab text; BEGIN
 FOREACH tab IN ARRAY ARRAY['credit_aggregate_snapshots','agreement_versions','agreement_acceptances','goods_releases','receipt_confirmations'] LOOP
  EXECUTE format('CREATE POLICY delegated_purchase_read ON app.%I FOR SELECT USING(app.purchase_request_read(credit_request_id::uuid))',tab);
 END LOOP;
END $$;
CREATE OR REPLACE FUNCTION app.credit_snapshot_by_id(p_request_id text) RETURNS jsonb LANGUAGE sql SECURITY DEFINER STABLE SET search_path=pg_catalog,app AS $$
 SELECT aggregate FROM app.credit_aggregate_snapshots WHERE credit_request_id=p_request_id AND
 (supplier_organization_id=NULLIF(current_setting('app.current_organization_id',true),'') OR buyer_user_id=NULLIF(current_setting('app.current_user_id',true),'') OR app.purchase_request_read(credit_request_id::uuid));
$$;
CREATE OR REPLACE FUNCTION app.credit_snapshot_by_obligation(p_obligation_id text) RETURNS TABLE(credit_request_id text,aggregate jsonb) LANGUAGE sql SECURITY DEFINER STABLE SET search_path=pg_catalog,app AS $$
 SELECT credit_request_id,aggregate FROM app.credit_aggregate_snapshots WHERE aggregate->'obligation'->>'id'=p_obligation_id AND
 (supplier_organization_id=NULLIF(current_setting('app.current_organization_id',true),'') OR buyer_user_id=NULLIF(current_setting('app.current_user_id',true),'') OR app.purchase_request_read(credit_request_id::uuid)) LIMIT 1;
$$;
REVOKE ALL ON FUNCTION app.can_purchase(uuid,text,bigint),app.purchase_request_read(uuid),app.lock_purchase_permission(uuid,text,bigint),app.record_purchasing_delegation() FROM PUBLIC;
DO $$ DECLARE r text; BEGIN
 FOREACH r IN ARRAY ARRAY['kredit_app','kredit_worker'] LOOP
 IF EXISTS(SELECT 1 FROM pg_roles WHERE rolname=r) THEN EXECUTE format('GRANT EXECUTE ON FUNCTION app.can_purchase(uuid,text,bigint),app.purchase_request_read(uuid),app.lock_purchase_permission(uuid,text,bigint) TO %I',r); END IF;
 END LOOP;
 IF EXISTS(SELECT 1 FROM pg_roles WHERE rolname='kredit_app') THEN
 GRANT SELECT,INSERT,UPDATE ON app.purchasing_delegations TO kredit_app;
 GRANT SELECT ON app.purchasing_delegation_history TO kredit_app;
 END IF;
END $$;
-- +goose StatementEnd
-- +goose Down
-- +goose StatementBegin
DO $$ BEGIN RAISE EXCEPTION 'Purchasing delegation requires forward recovery'; END $$;
-- +goose StatementEnd
