-- +goose Up
-- A historical purchasing owner cannot use a staff grant to reactivate the
-- legacy owner-only bank/payment capabilities after losing the owner role.
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION app.can_purchase(profile_id uuid,action_name text DEFAULT 'read',amount_kobo bigint DEFAULT 0) RETURNS boolean
LANGUAGE sql STABLE SECURITY DEFINER SET search_path=pg_catalog,app AS $$
 SELECT EXISTS(SELECT 1 FROM app.businesses b JOIN app.organizations o ON o.id=b.organization_id
 JOIN app.memberships m ON m.organization_id=o.id AND m.user_id=app.current_user_id()
 JOIN app.users u ON u.id=m.user_id
 LEFT JOIN app.purchasing_delegations d ON d.organization_id=o.id AND d.user_id=m.user_id
 WHERE b.id=profile_id AND m.status='active' AND u.status='active' AND o.status<>'suspended'
 AND ((m.role='owner' AND b.owner_user_id=m.user_id) OR
 (b.owner_user_id IS DISTINCT FROM m.user_id AND d.membership_id=m.id AND d.membership_authority_version=m.purchasing_authority_version
 AND d.expires_at>statement_timestamp() AND 'read'=ANY(d.actions) AND action_name=ANY(d.actions)
 AND (action_name<>'accept' OR amount_kobo BETWEEN 0 AND d.ceiling_kobo))));
$$;
-- +goose StatementEnd
-- +goose Down
-- +goose StatementBegin
DO $$ BEGIN RAISE EXCEPTION 'Purchasing owner role boundary requires forward recovery'; END $$;
-- +goose StatementEnd
