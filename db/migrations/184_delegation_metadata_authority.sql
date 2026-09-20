-- +goose Up
-- Permission administration and its history stop at current business authority.
-- +goose StatementBegin
DO $$ DECLARE tab text; BEGIN
 FOREACH tab IN ARRAY ARRAY['purchasing_delegations','purchasing_delegation_history'] LOOP
  EXECUTE format('CREATE POLICY current_delegation_authority ON app.%I AS RESTRICTIVE FOR ALL USING(EXISTS(SELECT 1 FROM app.memberships m JOIN app.users u ON u.id=m.user_id JOIN app.organizations o ON o.id=m.organization_id WHERE m.organization_id=%I.organization_id AND m.user_id=app.current_user_id() AND m.status=''active'' AND u.status=''active'' AND o.status<>''suspended''))',tab,tab);
 END LOOP;
END $$;
-- +goose StatementEnd
CREATE POLICY current_business_verification_authority ON app.verification_cases AS RESTRICTIVE FOR SELECT
USING(current_user='kredit_worker' OR subject_type<>'business' OR app.purchasing_authority_current(subject_id));
-- +goose Down
-- +goose StatementBegin
DO $$ BEGIN RAISE EXCEPTION 'Delegation metadata authority requires forward recovery'; END $$;
-- +goose StatementEnd
