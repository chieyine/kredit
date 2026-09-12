-- +goose Up
-- Purpose-specific admin projections preserve tenant isolation on source tables.
-- +goose StatementBegin
CREATE FUNCTION app.admin_user_directory(text,integer,uuid) RETURNS TABLE(id text,display_name text,identifier text,status text,organization_count bigint,last_authenticated_at timestamptz,created_at timestamptz,version bigint) LANGUAGE sql STABLE SECURITY DEFINER SET search_path=pg_catalog,app AS $$
 SELECT * FROM (SELECT u.id::text,COALESCE(NULLIF(u.display_name,''),'Kredit user'),COALESCE(u.normalized_email,u.normalized_phone,''),u.status,count(DISTINCT m.organization_id),u.last_authenticated_at,u.created_at,u.version FROM app.users u LEFT JOIN app.memberships m ON m.user_id=u.id AND m.status IN('active','invited','suspended') WHERE ($1='' OR u.id::text=$1 OR lower(COALESCE(u.normalized_email,''))=lower($1) OR COALESCE(u.normalized_phone,'')=$1 OR lower(COALESCE(u.display_name,'')) LIKE '%'||lower($1)||'%') GROUP BY u.id ORDER BY u.created_at DESC LIMIT greatest(1,least($2,200))) scoped WHERE $3=app.current_user_id() AND app.has_admin_role($3,ARRAY['platform_owner','platform_admin','access_administrator'])
$$;
-- +goose StatementEnd
REVOKE ALL ON FUNCTION app.admin_user_directory(text,integer,uuid) FROM PUBLIC;
-- +goose StatementBegin
CREATE FUNCTION app.admin_organization_directory(text,integer,uuid) RETURNS TABLE(id text,legal_name text,trading_name text,business_type text,industry text,status text,member_count bigint,open_sales bigint,outstanding_kobo numeric,version bigint,created_at timestamptz) LANGUAGE sql STABLE SECURITY DEFINER SET search_path=pg_catalog,app AS $$
 SELECT * FROM (SELECT o.id::text,o.legal_name,COALESCE(o.trading_name,''),o.business_type,o.industry,o.status,(SELECT count(*) FROM app.memberships m WHERE m.organization_id=o.id AND m.status='active'),(SELECT count(*) FROM app.obligations ob WHERE ob.supplier_organization_id=o.id AND ob.lifecycle_status='ACTIVE'),(SELECT COALESCE(sum(ob.outstanding_kobo),0) FROM app.obligations ob WHERE ob.supplier_organization_id=o.id AND ob.lifecycle_status='ACTIVE'),o.version,o.created_at FROM app.organizations o WHERE ($1='' OR o.id::text=$1 OR lower(o.legal_name) LIKE '%'||lower($1)||'%' OR lower(COALESCE(o.trading_name,'')) LIKE '%'||lower($1)||'%') ORDER BY o.created_at DESC LIMIT greatest(1,least($2,200))) scoped WHERE $3=app.current_user_id() AND app.has_admin_role($3,ARRAY['platform_owner','platform_admin','access_administrator'])
$$;
-- +goose StatementEnd
REVOKE ALL ON FUNCTION app.admin_organization_directory(text,integer,uuid) FROM PUBLIC;
-- +goose StatementBegin
CREATE FUNCTION app.admin_audit_directory(text,integer,uuid) RETURNS TABLE(id text,occurred_at timestamptz,actor_user_id text,organization_id text,action text,resource_type text,resource_id text,outcome text,severity text,request_id text,metadata jsonb) LANGUAGE sql STABLE SECURITY DEFINER SET search_path=pg_catalog,app AS $$
 SELECT * FROM (SELECT id::text,occurred_at,COALESCE(actor_user_id::text,''),COALESCE(organization_id::text,''),action,resource_type,COALESCE(resource_id,''),outcome,severity,COALESCE(request_id,''),metadata FROM app.audit_events WHERE ($1='' OR organization_id=NULLIF($1,'')::uuid) ORDER BY occurred_at DESC LIMIT greatest(1,least($2,200))) scoped WHERE $3=app.current_user_id() AND app.has_admin_role($3,ARRAY['platform_owner','platform_admin','compliance_reviewer'])
$$;
-- +goose StatementEnd
REVOKE ALL ON FUNCTION app.admin_audit_directory(text,integer,uuid) FROM PUBLIC;
-- +goose StatementBegin
CREATE FUNCTION app.admin_team_directory(uuid) RETURNS TABLE(assignment_id text,user_id text,display_name text,identifier text,role text,granted_at timestamptz,expires_at timestamptz) LANGUAGE sql STABLE SECURITY DEFINER SET search_path=pg_catalog,app AS $$
 SELECT * FROM (SELECT pra.id::text,u.id::text,COALESCE(NULLIF(u.display_name,''),'Kredit administrator'),COALESCE(u.normalized_email,u.normalized_phone,''),pra.role,pra.granted_at,pra.expires_at FROM app.platform_role_assignments pra JOIN app.users u ON u.id=pra.user_id WHERE pra.revoked_at IS NULL AND (pra.expires_at IS NULL OR pra.expires_at>now()) ORDER BY pra.granted_at DESC LIMIT 200) scoped WHERE $1=app.current_user_id() AND app.has_admin_role($1,ARRAY['platform_owner','platform_admin','access_administrator'])
$$;
-- +goose StatementEnd
REVOKE ALL ON FUNCTION app.admin_team_directory(uuid) FROM PUBLIC;
-- +goose StatementBegin
CREATE FUNCTION app.admin_money_summary(uuid) RETURNS TABLE(received numeric,reversed numeric,requested numeric,succeeded numeric,outstanding numeric,payment_count bigint) LANGUAGE sql STABLE SECURITY DEFINER SET search_path=pg_catalog,app AS $$
 SELECT * FROM (SELECT COALESCE(sum(amount_kobo) FILTER(WHERE state='recognized'),0),COALESCE(sum(amount_kobo) FILTER(WHERE state='reversed' AND reversal_of IS NOT NULL),0),(SELECT COALESCE(sum(requested_amount_kobo),0) FROM app.collection_attempts),(SELECT COALESCE(sum(succeeded_amount_kobo),0) FROM app.collection_attempts),COALESCE((SELECT sum(outstanding_kobo) FROM app.obligations WHERE lifecycle_status='ACTIVE'),0),count(*) FROM app.payments) scoped WHERE $1=app.current_user_id() AND app.has_admin_role($1,ARRAY['platform_owner','platform_admin','finance_operator','approver','compliance_reviewer'])
$$;
-- +goose StatementEnd
REVOKE ALL ON FUNCTION app.admin_money_summary(uuid) FROM PUBLIC;
-- +goose StatementBegin
CREATE FUNCTION app.admin_money_activity(integer,uuid) RETURNS TABLE(kind text,id text,organization_id text,amount_kobo bigint,state text,reference text,occurred_at timestamptz) LANGUAGE sql STABLE SECURITY DEFINER SET search_path=pg_catalog,app AS $$
 SELECT * FROM (SELECT kind,id,organization_id,amount_kobo,state,reference,occurred_at FROM (SELECT 'payment' kind,p.id::text id,p.supplier_organization_id::text organization_id,p.amount_kobo,p.state,COALESCE(p.provider_reference,p.id::text) reference,p.paid_at occurred_at FROM app.payments p UNION ALL SELECT 'collection',ca.id::text,o.supplier_organization_id::text,ca.requested_amount_kobo,ca.state,ca.external_reference,ca.requested_at FROM app.collection_attempts ca JOIN app.obligations o ON o.id=ca.obligation_id) activity ORDER BY occurred_at DESC LIMIT greatest(1,least($1,200))) scoped WHERE $2=app.current_user_id() AND app.has_admin_role($2,ARRAY['platform_owner','platform_admin','finance_operator','approver','compliance_reviewer'])
$$;
-- +goose StatementEnd
REVOKE ALL ON FUNCTION app.admin_money_activity(integer,uuid) FROM PUBLIC;
-- +goose StatementBegin
DO $$ BEGIN IF EXISTS(SELECT 1 FROM pg_roles WHERE rolname='kredit_app') THEN GRANT EXECUTE ON FUNCTION app.admin_user_directory(text,integer,uuid),app.admin_organization_directory(text,integer,uuid),app.admin_audit_directory(text,integer,uuid),app.admin_team_directory(uuid),app.admin_money_summary(uuid),app.admin_money_activity(integer,uuid) TO kredit_app; END IF; END $$;
-- +goose StatementEnd
-- +goose Down
DROP FUNCTION app.admin_user_directory(text,integer,uuid);
DROP FUNCTION app.admin_organization_directory(text,integer,uuid);
DROP FUNCTION app.admin_audit_directory(text,integer,uuid);
DROP FUNCTION app.admin_team_directory(uuid);
DROP FUNCTION app.admin_money_summary(uuid);
DROP FUNCTION app.admin_money_activity(integer,uuid);
