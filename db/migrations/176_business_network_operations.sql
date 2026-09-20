-- +goose Up
CREATE TABLE app.business_branches (
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
 organization_id uuid NOT NULL REFERENCES app.organizations(id),
 name text NOT NULL CHECK(length(btrim(name)) BETWEEN 1 AND 100),
 territory text NOT NULL DEFAULT '' CHECK(length(territory)<=160),
 active boolean NOT NULL DEFAULT true,
 version bigint NOT NULL DEFAULT 1 CHECK(version>0),
 updated_by uuid NOT NULL REFERENCES app.users(id),
 updated_at timestamptz NOT NULL DEFAULT now(),
 UNIQUE(organization_id,id), UNIQUE(organization_id,name)
);
CREATE TABLE app.partner_assignments (
 organization_id uuid NOT NULL REFERENCES app.organizations(id),
 buyer_business_id uuid NOT NULL REFERENCES app.businesses(id),
 branch_id uuid,
 manager_user_id uuid REFERENCES app.users(id),
 version bigint NOT NULL DEFAULT 1 CHECK(version>0),
 updated_by uuid NOT NULL REFERENCES app.users(id),
 updated_at timestamptz NOT NULL DEFAULT now(),
 PRIMARY KEY(organization_id,buyer_business_id),
 FOREIGN KEY(organization_id,branch_id) REFERENCES app.business_branches(organization_id,id)
);
CREATE TABLE app.network_operation_history (
 id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
 organization_id uuid NOT NULL REFERENCES app.organizations(id),
 record_type text NOT NULL,
 record_id uuid NOT NULL,
 version bigint NOT NULL,
 actor_id uuid NOT NULL REFERENCES app.users(id),
 recorded_at timestamptz NOT NULL DEFAULT now(),
 evidence jsonb NOT NULL,
 UNIQUE(organization_id,record_type,record_id,version)
);
-- +goose StatementBegin
DO $$ DECLARE tab text; BEGIN
 FOREACH tab IN ARRAY ARRAY['business_branches','partner_assignments','network_operation_history'] LOOP
  EXECUTE format('ALTER TABLE app.%I ENABLE ROW LEVEL SECURITY',tab);
  EXECUTE format('ALTER TABLE app.%I FORCE ROW LEVEL SECURITY',tab);
  EXECUTE format('CREATE POLICY network_read ON app.%I FOR SELECT USING(organization_id=app.current_organization_id() AND EXISTS(SELECT 1 FROM app.memberships m JOIN app.users u ON u.id=m.user_id WHERE m.organization_id=%I.organization_id AND m.user_id=app.current_user_id() AND m.status=''active'' AND u.status=''active''))',tab,tab);
  IF tab<>'network_operation_history' THEN
   EXECUTE format('CREATE POLICY network_insert ON app.%I FOR INSERT WITH CHECK(organization_id=app.current_organization_id() AND updated_by=app.current_user_id() AND EXISTS(SELECT 1 FROM app.memberships m WHERE m.organization_id=%I.organization_id AND m.user_id=app.current_user_id() AND m.status=''active'' AND m.role IN (''owner'',''administrator'')))',tab,tab);
   EXECUTE format('CREATE POLICY network_update ON app.%I FOR UPDATE USING(organization_id=app.current_organization_id() AND EXISTS(SELECT 1 FROM app.memberships m WHERE m.organization_id=%I.organization_id AND m.user_id=app.current_user_id() AND m.status=''active'' AND m.role IN (''owner'',''administrator''))) WITH CHECK(organization_id=app.current_organization_id() AND updated_by=app.current_user_id())',tab,tab);
  END IF;
 END LOOP;
END $$;
CREATE FUNCTION app.record_network_operation() RETURNS trigger LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,app SET row_security=off AS $$
DECLARE record_id uuid;
BEGIN
 IF TG_OP='UPDATE' AND (NEW.organization_id<>OLD.organization_id OR NEW.version<>OLD.version+1) THEN RAISE EXCEPTION 'network operation requires next version' USING ERRCODE='23514'; END IF;
 IF TG_OP='INSERT' AND NEW.version<>1 THEN RAISE EXCEPTION 'network operation initial version must be one' USING ERRCODE='23514'; END IF;
 IF TG_TABLE_NAME='business_branches' THEN
  record_id:=NEW.id;
  IF TG_OP='UPDATE' AND NEW.id<>OLD.id THEN RAISE EXCEPTION 'branch identity cannot change'; END IF;
 ELSE
  record_id:=NEW.buyer_business_id;
  IF TG_OP='UPDATE' AND NEW.buyer_business_id<>OLD.buyer_business_id THEN RAISE EXCEPTION 'partner identity cannot change'; END IF;
  IF NOT EXISTS(SELECT 1 FROM app.trade_relationships WHERE supplier_organization_id=NEW.organization_id AND buyer_business_id=NEW.buyer_business_id) THEN RAISE EXCEPTION 'partner must belong to supplier network' USING ERRCODE='23514'; END IF;
  IF NEW.manager_user_id IS NOT NULL THEN
   PERFORM 1 FROM app.memberships m JOIN app.users u ON u.id=m.user_id WHERE m.organization_id=NEW.organization_id AND m.user_id=NEW.manager_user_id AND m.status='active' AND u.status='active' AND m.role IN ('owner','administrator','sales') FOR SHARE OF m,u;
   IF NOT FOUND THEN RAISE EXCEPTION 'account manager must be active supplier staff' USING ERRCODE='23514'; END IF;
  END IF;
  IF NEW.branch_id IS NOT NULL THEN
   PERFORM 1 FROM app.business_branches WHERE organization_id=NEW.organization_id AND id=NEW.branch_id AND active FOR SHARE;
   IF NOT FOUND THEN RAISE EXCEPTION 'branch must be active' USING ERRCODE='23514'; END IF;
  END IF;
 END IF;
 INSERT INTO app.network_operation_history(organization_id,record_type,record_id,version,actor_id,evidence) VALUES(NEW.organization_id,TG_TABLE_NAME,record_id,NEW.version,NEW.updated_by,to_jsonb(NEW));
 RETURN NEW;
END $$;
-- +goose StatementEnd
CREATE TRIGGER branch_history AFTER INSERT OR UPDATE ON app.business_branches FOR EACH ROW EXECUTE FUNCTION app.record_network_operation();
CREATE TRIGGER partner_assignment_history AFTER INSERT OR UPDATE ON app.partner_assignments FOR EACH ROW EXECUTE FUNCTION app.record_network_operation();
-- +goose StatementBegin
DO $$ BEGIN IF EXISTS(SELECT 1 FROM pg_roles WHERE rolname='kredit_app') THEN
 GRANT SELECT,INSERT,UPDATE ON app.business_branches,app.partner_assignments TO kredit_app;
 GRANT SELECT ON app.network_operation_history TO kredit_app;
END IF; END $$;
-- +goose StatementEnd
-- +goose Down
-- +goose StatementBegin
DO $$ BEGIN RAISE EXCEPTION 'Network operations require forward recovery'; END $$;
-- +goose StatementEnd
