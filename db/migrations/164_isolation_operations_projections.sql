-- +goose Up
-- Cross-tenant operations receive only counts, exact references, or the
-- recipient of a pending send they are currently authorized to resolve.
-- +goose StatementBegin
CREATE FUNCTION app.isolation_operations_counts()
RETURNS TABLE(open_disputes bigint,notification_backlog bigint)
LANGUAGE sql STABLE SECURITY DEFINER SET search_path=pg_catalog,app SET row_security=off AS $$
 SELECT (SELECT count(*) FROM app.disputes WHERE state IN('OPEN','UNDER_REVIEW','PARTIALLY_RESOLVED')),
 (SELECT count(*) FROM app.notifications WHERE state IN('scheduled','failed'))
 WHERE app.has_admin_role(app.current_user_id(),ARRAY['platform_owner','platform_admin','compliance_reviewer','dispute_reviewer','support_agent','access_administrator','finance_operator','policy_manager','approver']);
$$;
CREATE FUNCTION app.dispute_reference_lookup(reference text)
RETURNS TABLE(id uuid,organization_id uuid,state text)
LANGUAGE sql STABLE SECURITY DEFINER SET search_path=pg_catalog,app SET row_security=off AS $$
 SELECT d.id,d.supplier_organization_id,d.state FROM app.disputes d
 WHERE lower(d.id::text)=lower(reference)
 AND app.has_admin_role(app.current_user_id(),ARRAY['platform_owner','platform_admin','access_administrator']) LIMIT 1;
$$;
CREATE FUNCTION app.notification_recovery_subject(notification uuid) RETURNS uuid
LANGUAGE sql STABLE SECURITY DEFINER SET search_path=pg_catalog,app SET row_security=off AS $$
 SELECT n.recipient_id FROM app.notifications n WHERE n.id=notification
 AND EXISTS(SELECT 1 FROM app.message_submissions m WHERE m.notification_id=n.id AND m.state='STARTED' AND m.created_at<now()-interval '1 minute')
 AND app.has_admin_role(app.current_user_id(),ARRAY['platform_owner','platform_admin','compliance_reviewer']);
$$;
-- +goose StatementEnd
REVOKE ALL ON FUNCTION app.isolation_operations_counts(),app.dispute_reference_lookup(text),app.notification_recovery_subject(uuid) FROM PUBLIC;
-- +goose StatementBegin
DO $$ BEGIN IF EXISTS(SELECT 1 FROM pg_roles WHERE rolname='kredit_app') THEN
 GRANT EXECUTE ON FUNCTION app.isolation_operations_counts(),app.dispute_reference_lookup(text),app.notification_recovery_subject(uuid) TO kredit_app;
END IF; END $$;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DO $$ BEGIN RAISE EXCEPTION 'Operations isolation requires forward recovery'; END $$;
-- +goose StatementEnd
