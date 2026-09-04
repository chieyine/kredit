-- +goose Up
ALTER TABLE app.platform_role_assignments DROP CONSTRAINT platform_role_assignments_role_check;
ALTER TABLE app.platform_role_assignments ADD CONSTRAINT platform_role_assignments_role_check CHECK(role IN ('support_agent','compliance_reviewer','dispute_reviewer','platform_admin','finance_operator','policy_manager','approver','access_administrator'));
-- Boolean authorization lookups expose no user details across tenant boundaries.
-- +goose StatementBegin
CREATE FUNCTION app.has_admin_role(actor uuid, roles text[]) RETURNS boolean LANGUAGE sql STABLE SECURITY DEFINER SET search_path=pg_catalog,app AS $$
 SELECT EXISTS(SELECT 1 FROM app.platform_role_assignments r JOIN app.users u ON u.id=r.user_id WHERE r.user_id=actor AND r.role=ANY(roles) AND r.revoked_at IS NULL AND (r.expires_at IS NULL OR r.expires_at>now()) AND u.status='active');
$$;
-- +goose StatementEnd
REVOKE ALL ON FUNCTION app.has_admin_role(uuid,text[]) FROM PUBLIC;
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION app.is_active_policy_admin(actor uuid) RETURNS boolean LANGUAGE sql STABLE SECURITY DEFINER SET search_path=pg_catalog,app AS $$
 SELECT app.has_admin_role(actor,ARRAY['platform_admin','policy_manager']);
$$;
-- +goose StatementEnd
CREATE TABLE app.admin_change_requests (
 id uuid PRIMARY KEY, kind text NOT NULL CHECK(kind IN ('write_off','fee_waiver','schedule_amendment')),
 obligation_id uuid NOT NULL REFERENCES app.obligations(id), organization_id uuid NOT NULL REFERENCES app.organizations(id), buyer_id uuid NOT NULL REFERENCES app.users(id),
 proposed_by uuid NOT NULL REFERENCES app.users(id), reason text NOT NULL CHECK(length(trim(reason)) BETWEEN 8 AND 2000),
 before_values jsonb NOT NULL, proposed_values jsonb NOT NULL, created_at timestamptz NOT NULL DEFAULT now(), expires_at timestamptz NOT NULL,
 state text NOT NULL DEFAULT 'pending' CHECK(state IN ('pending','awaiting_buyer','applied','rejected','cancelled')),
 approved_by uuid REFERENCES app.users(id), buyer_decided_by uuid REFERENCES app.users(id), decided_at timestamptz,
 CHECK(expires_at>created_at), CHECK(approved_by IS NULL OR approved_by<>proposed_by),
 CHECK(state NOT IN ('awaiting_buyer','applied') OR approved_by IS NOT NULL),
 CHECK(kind<>'schedule_amendment' OR state<>'applied' OR buyer_decided_by=buyer_id)
);
CREATE UNIQUE INDEX admin_change_one_open ON app.admin_change_requests(obligation_id) WHERE state IN ('pending','awaiting_buyer');
CREATE TABLE app.admin_change_events (
 id uuid PRIMARY KEY DEFAULT uuidv7(), change_id uuid NOT NULL REFERENCES app.admin_change_requests(id), actor_id uuid NOT NULL REFERENCES app.users(id),
 action text NOT NULL, reason text NOT NULL, occurred_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE app.admin_review_assignments (
 kind text NOT NULL, resource_id uuid NOT NULL, owner_id uuid REFERENCES app.users(id), due_at timestamptz NOT NULL, updated_at timestamptz NOT NULL DEFAULT now(), PRIMARY KEY(kind,resource_id)
);
CREATE TABLE app.admin_assignment_events (
 id uuid PRIMARY KEY DEFAULT uuidv7(), kind text NOT NULL, resource_id uuid NOT NULL, actor_id uuid NOT NULL REFERENCES app.users(id), before_values jsonb NOT NULL, after_values jsonb NOT NULL, reason text NOT NULL, occurred_at timestamptz NOT NULL DEFAULT now()
);
-- +goose StatementBegin
CREATE FUNCTION app.guard_admin_change() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 IF TG_OP='DELETE' THEN RAISE EXCEPTION 'approval history is immutable'; END IF;
 IF (to_jsonb(NEW)-ARRAY['state','approved_by','buyer_decided_by','decided_at']) IS DISTINCT FROM (to_jsonb(OLD)-ARRAY['state','approved_by','buyer_decided_by','decided_at']) THEN RAISE EXCEPTION 'proposed financial change is immutable'; END IF;
 IF NOT ((OLD.state='pending' AND NEW.state IN ('awaiting_buyer','applied','rejected','cancelled')) OR (OLD.state='awaiting_buyer' AND NEW.state IN ('applied','rejected','cancelled'))) THEN RAISE EXCEPTION 'invalid approval transition'; END IF;
 IF OLD.approved_by IS NOT NULL AND NEW.approved_by IS DISTINCT FROM OLD.approved_by THEN RAISE EXCEPTION 'approval evidence is immutable'; END IF;
 RETURN NEW;
END $$;
-- +goose StatementEnd
CREATE TRIGGER admin_change_guard BEFORE UPDATE OR DELETE ON app.admin_change_requests FOR EACH ROW EXECUTE FUNCTION app.guard_admin_change();
CREATE TRIGGER admin_change_events_immutable BEFORE UPDATE OR DELETE ON app.admin_change_events FOR EACH ROW EXECUTE FUNCTION app.reject_operations_command_mutation();
CREATE TRIGGER admin_assignment_events_immutable BEFORE UPDATE OR DELETE ON app.admin_assignment_events FOR EACH ROW EXECUTE FUNCTION app.reject_operations_command_mutation();
-- +goose StatementBegin
DO $$ DECLARE t text; BEGIN
 FOREACH t IN ARRAY ARRAY['admin_change_requests','admin_change_events','admin_review_assignments','admin_assignment_events'] LOOP
 EXECUTE format('ALTER TABLE app.%I ENABLE ROW LEVEL SECURITY',t);
 EXECUTE format('CREATE POLICY admin_workflow_runtime ON app.%I USING(current_user IN (''kredit_app'',''kredit_worker'')) WITH CHECK(current_user=''kredit_app'')',t);
 IF EXISTS(SELECT 1 FROM pg_roles WHERE rolname='kredit_app') THEN EXECUTE format('GRANT SELECT,INSERT,UPDATE ON app.%I TO kredit_app',t); END IF;
 IF EXISTS(SELECT 1 FROM pg_roles WHERE rolname='kredit_worker') THEN EXECUTE format('GRANT SELECT ON app.%I TO kredit_worker',t); END IF;
 END LOOP;
 IF EXISTS(SELECT 1 FROM pg_roles WHERE rolname='kredit_app') THEN GRANT EXECUTE ON FUNCTION app.has_admin_role(uuid,text[]) TO kredit_app; END IF;
END $$;
-- +goose StatementEnd
-- +goose StatementBegin
CREATE FUNCTION app.admin_actor_name(actor uuid) RETURNS text LANGUAGE sql STABLE SECURITY DEFINER SET search_path=pg_catalog,app AS $$
 SELECT COALESCE((SELECT NULLIF(display_name,'') FROM app.users WHERE id=actor),'Administrator');
$$;
-- +goose StatementEnd
REVOKE ALL ON FUNCTION app.admin_actor_name(uuid) FROM PUBLIC;
CREATE VIEW app.admin_review_queue AS
 SELECT 'policy'::text kind,id,state,reason title,proposed_by author_id,created_at,effective_at due_at,'/admin/settings'::text href FROM app.business_policy_changes WHERE state='pending' OR(state='approved' AND effective_at>now())
 UNION ALL SELECT 'financial_change',id,state,replace(kind,'_',' ')||': '||reason,proposed_by,created_at,expires_at,'/admin/approvals?change='||id FROM app.admin_change_requests WHERE state IN ('pending','awaiting_buyer')
 UNION ALL SELECT 'dispute',id,state,reason,opened_by,opened_at,opened_at+interval '2 days','/admin/disputes/'||id FROM app.disputes WHERE state IN ('OPEN','UNDER_REVIEW','PARTIALLY_RESOLVED')
 UNION ALL SELECT 'financial_review',id,state,kind||' discrepancy',NULL,first_seen_at,first_seen_at+interval '1 day','/admin/reconciliation' FROM app.financial_review_cases WHERE state='OPEN'
 UNION ALL SELECT 'recovery',id,state,'Account recovery',target_user_id,created_at,expires_at,'/admin/recovery' FROM app.account_recovery_requests WHERE state IN ('PENDING_REVIEW','COOLING_OFF','APPROVED')
 UNION ALL SELECT 'privacy',id,state,request_type||' request',requester_user_id,created_at,due_at,'/admin/privacy' FROM app.privacy_requests WHERE state NOT IN ('COMPLETED','CANCELLED','REJECTED');
CREATE VIEW app.admin_change_history AS
 SELECT 'policy'::text kind,c.id,c.created_at,c.state,c.reason,c.proposed_by,app.admin_actor_name(c.proposed_by) proposer,c.decided_by approved_by,CASE WHEN c.decided_by IS NOT NULL THEN app.admin_actor_name(c.decided_by) END approver,
 COALESCE((SELECT values FROM app.business_policy_changes b WHERE b.revision=c.base_revision),(SELECT values FROM app.business_policy_defaults WHERE singleton)) before_values,c.values after_values,
 COALESCE((SELECT jsonb_agg(to_jsonb(e)||jsonb_build_object('actor_name',app.admin_actor_name(e.actor_id)) ORDER BY e.occurred_at,e.id) FROM app.business_policy_events e WHERE e.change_id=c.id),'[]'::jsonb) events
 FROM app.business_policy_changes c
 UNION ALL SELECT c.kind,c.id,c.created_at,c.state,c.reason,c.proposed_by,app.admin_actor_name(c.proposed_by),c.approved_by,CASE WHEN c.approved_by IS NOT NULL THEN app.admin_actor_name(c.approved_by) END,c.before_values,c.proposed_values,
 COALESCE((SELECT jsonb_agg(to_jsonb(e)||jsonb_build_object('actor_name',app.admin_actor_name(e.actor_id)) ORDER BY e.occurred_at,e.id) FROM app.admin_change_events e WHERE e.change_id=c.id),'[]'::jsonb) FROM app.admin_change_requests c
 UNION ALL SELECT 'assignment',id,occurred_at,'recorded',reason,actor_id,app.admin_actor_name(actor_id),NULL,NULL,before_values,after_values,'[]'::jsonb FROM app.admin_assignment_events;
-- +goose StatementBegin
DO $$ BEGIN
 IF EXISTS(SELECT 1 FROM pg_roles WHERE rolname='kredit_app') THEN
 GRANT EXECUTE ON FUNCTION app.admin_actor_name(uuid) TO kredit_app;
 GRANT SELECT ON app.admin_review_queue,app.admin_change_history TO kredit_app;
 END IF;
END $$;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE FUNCTION app.admin_policy_impact(v jsonb) RETURNS jsonb LANGUAGE sql STABLE SECURITY DEFINER SET search_path=pg_catalog,app AS $$
 SELECT jsonb_build_object(
 'active_obligations',(SELECT count(*) FROM app.obligations WHERE lifecycle_status='ACTIVE' AND outstanding_kobo>0),
 'due_instalments',(SELECT count(*) FROM app.schedule_items WHERE state NOT IN ('PAID','CANCELLED') AND principal_due_kobo>allocated_kobo AND collection_at<=now()),
 'upcoming_reminders',(SELECT count(*) FROM app.schedule_items WHERE state NOT IN ('PAID','CANCELLED') AND principal_due_kobo>allocated_kobo AND due_at>now() AND due_at<=now()+make_interval(days=>(v->>'upcoming_notice_days')::int)),
 'expiring_mandates',(SELECT count(*) FROM app.payment_mandates WHERE state='active' AND ends_at>now() AND ends_at<=now()+make_interval(days=>(v->>'mandate_notice_days')::int)),
 'existing_offers_above_new_principal_cap',(SELECT count(*) FROM app.credit_requests WHERE (v->>'max_principal_kobo')::bigint>0 AND principal_kobo>(v->>'max_principal_kobo')::bigint),
 'failed_attempts_at_new_limit',(SELECT count(*) FROM app.collection_attempts WHERE state='FAILED' AND attempt_number>=(v->>'max_retries')::int)
 );
$$;
-- +goose StatementEnd
-- +goose StatementBegin
CREATE FUNCTION app.admin_attention(actor uuid,kinds text[]) RETURNS jsonb LANGUAGE sql STABLE SECURITY DEFINER SET search_path=pg_catalog,app AS $$
 SELECT jsonb_agg(jsonb_build_object('label',label,'count',amount,'href',href,'action',action)) FROM (
 SELECT 'Overdue reviews' label,count(*) amount,'/admin/inbox' href,'Assign an owner or review the decision' action FROM app.admin_review_queue q LEFT JOIN app.admin_review_assignments a ON a.kind=q.kind AND a.resource_id=q.id WHERE q.kind=ANY(kinds) AND COALESCE(a.due_at,q.due_at)<now()
 UNION ALL SELECT 'Unresolved settlements',count(*),'/admin/reconciliation','Review provider settlement evidence' FROM app.financial_review_cases WHERE state='OPEN' AND kind IN ('settlement','settlement_missing') AND 'financial_review'=ANY(kinds)
 UNION ALL SELECT 'Uncertain debits',count(*),'/admin/controls','Look up the original debit before deciding what to do' FROM app.collection_attempts WHERE state IN ('UNKNOWN','SUBMITTED','PENDING') AND 'financial_review'=ANY(kinds)
 UNION ALL SELECT 'Failed notices',count(*),'/admin/diagnostics','Review delivery failures and processing queues' FROM app.notifications WHERE state='failed' AND 'financial_review'=ANY(kinds)
 UNION ALL SELECT 'Mandates approaching expiry',count(*),'/admin/attention','Review mandates needing renewed buyer authorization' FROM app.payment_mandates WHERE state='active' AND ends_at>now() AND ends_at<=now()+make_interval(days=>COALESCE((app.business_policy()->>'mandate_notice_days')::int,7)) AND 'financial_review'=ANY(kinds)
 )m;
$$;
-- +goose StatementEnd
REVOKE ALL ON FUNCTION app.admin_policy_impact(jsonb),app.admin_attention(uuid,text[]) FROM PUBLIC;
-- +goose StatementBegin
DO $$ BEGIN IF EXISTS(SELECT 1 FROM pg_roles WHERE rolname='kredit_app') THEN GRANT EXECUTE ON FUNCTION app.admin_policy_impact(jsonb),app.admin_attention(uuid,text[]) TO kredit_app; END IF; END $$;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DO $$ BEGIN RAISE EXCEPTION 'Admin approval and consent history must be retained; use a forward migration'; END $$;
-- +goose StatementEnd
