-- +goose Up
-- Durable baseline activity does not depend on a later HTTP logging or external
-- message call. Never copy row contents, documents, credentials or free text.
-- +goose StatementBegin
CREATE FUNCTION app.record_domain_activity() RETURNS trigger
LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,app AS $$
DECLARE row_data jsonb; before_data jsonb; org uuid; subject_user uuid; actor uuid;
 event_id uuid; record_id text; notify boolean; path text;
BEGIN
 IF TG_OP='DELETE' THEN row_data:=to_jsonb(OLD); ELSE row_data:=to_jsonb(NEW); END IF;
 IF TG_OP='UPDATE' THEN
  before_data:=to_jsonb(OLD);
  IF before_data=row_data THEN RETURN NEW; END IF;
 END IF;
 record_id:=COALESCE(row_data->>'id',row_data->>'organization_id',row_data->>'key');
 org:=NULLIF(COALESCE(row_data->>'supplier_organization_id',row_data->>'organization_id'),'')::uuid;
 subject_user:=NULLIF(COALESCE(row_data->>'buyer_user_id',row_data->>'requester_user_id',row_data->>'user_id',row_data->>'requested_by',row_data->>'opened_by',row_data->>'owner_user_id'),'')::uuid;
 IF TG_TABLE_NAME='organizations' THEN org:=(row_data->>'id')::uuid; END IF;
 IF org IS NULL AND row_data ? 'credit_request_id' THEN
  SELECT supplier_organization_id,buyer_user_id INTO org,subject_user FROM app.credit_requests WHERE id=(row_data->>'credit_request_id')::uuid;
 END IF;
 IF TG_TABLE_NAME='drawdowns' THEN
  SELECT supplier_organization_id,buyer_user_id INTO org,subject_user FROM app.trade_lines WHERE id=(row_data->>'trade_line_id')::uuid;
 END IF;
 IF TG_TABLE_NAME='correction_decisions' THEN
  SELECT organization_id,requested_by INTO org,subject_user FROM app.correction_requests WHERE id=(row_data->>'request_id')::uuid;
 END IF;
 IF TG_TABLE_NAME='support_case_events' THEN
  SELECT organization_id,opened_by INTO org,subject_user FROM app.support_cases WHERE id=(row_data->>'case_id')::uuid;
 END IF;
 IF TG_TABLE_NAME IN ('privacy_requests','processing_restrictions','support_cases','support_case_events') THEN org:=NULL; END IF;
 SELECT id INTO actor FROM app.users WHERE id=app.current_user_id();
 -- Deleted organizations/users cannot remain foreign-key destinations.
 IF NOT EXISTS(SELECT 1 FROM app.organizations WHERE id=org) THEN org:=NULL; END IF;
 IF NOT EXISTS(SELECT 1 FROM app.users WHERE id=subject_user) THEN subject_user:=NULL; END IF;
 INSERT INTO app.audit_events(actor_user_id,organization_id,action,resource_type,resource_id,outcome,metadata)
 VALUES(actor,org,'record.'||lower(TG_OP),TG_TABLE_NAME,record_id,'success',
 jsonb_build_object('operation',TG_OP,'state',COALESCE(row_data->>'state',row_data->>'status',''),'version',COALESCE(row_data->>'version','')))
 RETURNING id INTO event_id;
 notify:=TG_OP='INSERT' OR TG_OP='DELETE' OR
  COALESCE(before_data->>'state',before_data->>'status','') IS DISTINCT FROM COALESCE(row_data->>'state',row_data->>'status','') OR
  before_data->>'role' IS DISTINCT FROM row_data->>'role';
 -- Root customer-facing workflows get an in-app notice. Child bookkeeping
 -- records are audited without generating a notice for every ledger row.
 IF notify AND TG_TABLE_NAME=ANY(ARRAY['memberships','businesses','credit_requests','trade_lines','drawdowns','payments','payment_claims','disputes','correction_decisions','privacy_requests','support_cases','support_case_events','system_acceptances']) THEN
  INSERT INTO app.notifications(recipient_id,channel,template,template_version,event_reference,state,body,delivered_at)
  SELECT audience.id,'in_app','RecordUpdated','v1','activity:'||event_id::text||':'||audience.id::text,'delivered',
   'A '||replace(TG_TABLE_NAME,'_',' ')||' record was updated. Open the relevant sale, account request or business activity in Kredit to review it.',now()
  FROM (SELECT subject_user AS id WHERE subject_user IS NOT NULL
    UNION SELECT m.user_id FROM app.memberships m JOIN app.users u ON u.id=m.user_id
      WHERE TG_TABLE_NAME IN ('memberships','businesses','credit_requests','trade_lines','drawdowns','payments','payment_claims','disputes','system_acceptances')
      AND m.organization_id=org AND m.status='active' AND u.status='active'
      AND m.role IN ('owner','administrator','finance','collections')) audience
  ON CONFLICT(event_reference,channel) DO NOTHING;
 END IF;
 IF TG_OP='DELETE' THEN RETURN OLD; END IF;
 RETURN NEW;
END;
$$;
-- +goose StatementEnd
REVOKE ALL ON FUNCTION app.record_domain_activity() FROM PUBLIC;
-- +goose StatementBegin
DO $$ DECLARE name text; BEGIN
 FOREACH name IN ARRAY ARRAY['organizations','memberships','organization_invitations','businesses','business_representatives','buyer_invitations','relationship_consents','credit_requests','agreement_acceptances','goods_releases','receipt_confirmations','system_acceptances','obligations','trade_lines','drawdowns','payments','payment_claims','disputes','dispute_decisions','correction_requests','correction_decisions','support_cases','support_case_events','privacy_requests','processing_restrictions','supplier_onboarding_profiles','platform_role_assignments','platform_suspensions','platform_settings','business_policy_changes','operations_commands','documents'] LOOP
  EXECUTE format('CREATE TRIGGER domain_activity AFTER INSERT OR UPDATE OR DELETE ON app.%I FOR EACH ROW EXECUTE FUNCTION app.record_domain_activity()',name);
 END LOOP;
END $$;
-- +goose StatementEnd
-- +goose Down
-- +goose StatementBegin
DO $$ BEGIN RAISE EXCEPTION 'Durable domain activity requires forward recovery'; END $$;
-- +goose StatementEnd
