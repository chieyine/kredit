-- +goose Up
-- Extend delegated purchasing to trade-line drawdowns, payment claims, disputes, and amendments.
-- Bank authorization remains strictly owner-controlled; staff delegates cannot authorize bank debits.

ALTER TABLE app.purchasing_delegations DROP CONSTRAINT IF EXISTS purchasing_delegations_actions_check;
ALTER TABLE app.purchasing_delegations ADD CONSTRAINT purchasing_delegations_actions_check
  CHECK(actions <@ ARRAY['read','review','accept','receive','drawdown','dispute','claim','amend']::text[]);

ALTER TABLE app.purchasing_delegations ADD COLUMN IF NOT EXISTS drawdown_ceiling_kobo bigint NOT NULL DEFAULT 0 CHECK(drawdown_ceiling_kobo BETWEEN 0 AND 9007199254740991);

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION app.can_purchase(profile_id uuid, action_name text DEFAULT 'read'::text, amount_kobo bigint DEFAULT 0)
RETURNS boolean
LANGUAGE sql
STABLE SECURITY DEFINER
SET search_path TO 'pg_catalog', 'app'
AS $function$
 SELECT EXISTS(SELECT 1 FROM app.businesses b JOIN app.organizations o ON o.id=b.organization_id
 JOIN app.memberships m ON m.organization_id=o.id AND m.user_id=app.current_user_id()
 JOIN app.users u ON u.id=m.user_id
 LEFT JOIN app.purchasing_delegations d ON d.organization_id=o.id AND d.user_id=m.user_id
 WHERE b.id=profile_id AND m.status='active' AND u.status='active' AND o.status<>'suspended'
 AND ((m.role='owner' AND b.owner_user_id=m.user_id) OR
 (b.owner_user_id IS DISTINCT FROM m.user_id AND d.membership_id=m.id AND d.membership_authority_version=m.purchasing_authority_version
 AND d.expires_at>statement_timestamp() AND 'read'=ANY(d.actions) AND action_name=ANY(d.actions)
 AND (
   (action_name NOT IN ('accept', 'drawdown')) OR
   (action_name='accept' AND amount_kobo BETWEEN 0 AND d.ceiling_kobo) OR
   (action_name='drawdown' AND amount_kobo BETWEEN 0 AND d.drawdown_ceiling_kobo)
 ))));
$function$;
-- +goose StatementEnd

DROP POLICY IF EXISTS trade_line_buyer_access ON app.trade_lines;
CREATE POLICY trade_line_buyer_access ON app.trade_lines
  USING (buyer_user_id = app.current_user_id() OR app.can_purchase(buyer_business_id, 'read'));

DROP POLICY IF EXISTS delegated_drawdown_read ON app.drawdowns;
CREATE POLICY delegated_drawdown_read ON app.drawdowns FOR SELECT
  USING (EXISTS (SELECT 1 FROM app.trade_lines tl WHERE tl.id = drawdowns.trade_line_id AND app.can_purchase(tl.buyer_business_id, 'read')));

-- +goose Down
DROP POLICY IF EXISTS delegated_drawdown_read ON app.drawdowns;
DROP POLICY IF EXISTS trade_line_buyer_access ON app.trade_lines;
CREATE POLICY trade_line_buyer_access ON app.trade_lines
  USING (buyer_user_id = app.current_user_id());

ALTER TABLE app.purchasing_delegations DROP COLUMN IF EXISTS drawdown_ceiling_kobo;
ALTER TABLE app.purchasing_delegations DROP CONSTRAINT IF EXISTS purchasing_delegations_actions_check;
ALTER TABLE app.purchasing_delegations ADD CONSTRAINT purchasing_delegations_actions_check
  CHECK(actions <@ ARRAY['read'::text, 'review'::text, 'accept'::text, 'receive'::text]);
