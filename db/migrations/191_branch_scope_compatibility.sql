-- +goose Up
-- Branch restrictions supplement existing tenant/authority policies. A purchaser
-- or staff member with no branch restriction retains those original boundaries.
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION app.branch_scope_all(p_org uuid) RETURNS boolean LANGUAGE sql STABLE SECURITY DEFINER SET search_path=pg_catalog,app AS $$
 SELECT CASE WHEN (current_setting('role',true)='kredit_worker' OR (current_setting('role',true)='none' AND pg_has_role(session_user,'kredit_worker','member') AND NOT pg_has_role(session_user,'kredit_app','member'))) THEN true
 WHEN app.current_user_id() IS NULL THEN NOT EXISTS(SELECT 1 FROM app.member_branch_scopes WHERE organization_id=p_org AND mode='branches')
 WHEN NOT EXISTS(SELECT 1 FROM app.member_branch_scopes WHERE organization_id=p_org AND user_id=app.current_user_id()) THEN true
 ELSE EXISTS(SELECT 1 FROM app.memberships m JOIN app.users u ON u.id=m.user_id JOIN app.organizations o ON o.id=m.organization_id LEFT JOIN app.member_branch_scopes s ON s.organization_id=m.organization_id AND s.user_id=m.user_id
 WHERE m.organization_id=p_org AND m.user_id=app.current_user_id() AND m.status='active' AND u.status='active' AND o.status<>'suspended'
 AND (m.role IN ('owner','administrator') OR s.user_id IS NULL OR (s.mode='all' AND s.membership_id=m.id AND s.membership_version=m.purchasing_authority_version))) END;
$$;

CREATE POLICY branch_scope_history_current ON app.member_branch_scope_history AS RESTRICTIVE FOR SELECT USING(EXISTS(SELECT 1 FROM app.memberships m JOIN app.users u ON u.id=m.user_id JOIN app.organizations o ON o.id=m.organization_id WHERE m.organization_id=member_branch_scope_history.organization_id AND m.user_id=app.current_user_id() AND m.status='active' AND u.status='active' AND o.status<>'suspended'));
-- +goose StatementEnd
-- +goose Down
-- +goose StatementBegin
DO $$ BEGIN RAISE EXCEPTION 'Branch access requires forward recovery'; END $$;
-- +goose StatementEnd
