-- +goose Up
-- Business metadata and review evidence follow current user, membership and
-- business authority independently of the HTTP session or selected workspace.
-- +goose StatementBegin
DO $$ DECLARE tab text; BEGIN
 FOREACH tab IN ARRAY ARRAY['business_credit_controls','business_credit_control_history','credit_offer_approvals','credit_reviewer_limits','credit_reviewer_limit_history','business_branches','partner_assignments','network_operation_history'] LOOP
  EXECUTE format('CREATE POLICY current_network_authority ON app.%I AS RESTRICTIVE FOR ALL USING(EXISTS(SELECT 1 FROM app.memberships m JOIN app.users u ON u.id=m.user_id JOIN app.organizations o ON o.id=m.organization_id WHERE m.organization_id=%I.organization_id AND m.user_id=app.current_user_id() AND m.status=''active'' AND u.status=''active'' AND o.status<>''suspended''))',tab,tab);
 END LOOP;
END $$;
-- +goose StatementEnd
-- +goose Down
-- +goose StatementBegin
DO $$ BEGIN RAISE EXCEPTION 'Current authority protections require forward recovery'; END $$;
-- +goose StatementEnd
