-- +goose Up
-- Track 3: Branch-scoped drawdown workflows and independent drawdown approvals.

ALTER TABLE app.drawdowns ADD COLUMN IF NOT EXISTS branch_id uuid REFERENCES app.business_branches(id);

CREATE TABLE IF NOT EXISTS app.tradeline_drawdown_approvals (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id uuid NOT NULL REFERENCES app.organizations(id),
  drawdown_id uuid NOT NULL REFERENCES app.drawdowns(id) ON DELETE CASCADE,
  drawdown_version bigint NOT NULL DEFAULT 1,
  fingerprint bytea NOT NULL CHECK(octet_length(fingerprint) = 32),
  proposal jsonb NOT NULL,
  requested_by uuid NOT NULL REFERENCES app.users(id),
  state text NOT NULL DEFAULT 'pending' CHECK(state IN ('pending', 'approved', 'rejected')),
  decided_by uuid REFERENCES app.users(id),
  reason text NOT NULL DEFAULT '',
  created_at timestamptz NOT NULL DEFAULT now(),
  decided_at timestamptz,
  CHECK((state = 'pending' AND decided_by IS NULL AND decided_at IS NULL) OR 
        (state <> 'pending' AND decided_by IS NOT NULL AND decided_at IS NOT NULL AND decided_by <> requested_by)),
  UNIQUE(organization_id, drawdown_id, drawdown_version)
);

ALTER TABLE app.tradeline_drawdown_approvals ENABLE ROW LEVEL SECURITY;
ALTER TABLE app.tradeline_drawdown_approvals FORCE ROW LEVEL SECURITY;

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION app.drawdown_approval_proposal(d app.drawdowns) RETURNS jsonb
LANGUAGE sql IMMUTABLE SET search_path=pg_catalog AS $$
  SELECT to_jsonb(d) - ARRAY['state', 'buyer_confirmed_at', 'activated_at', 'released_at', 'receipt_at', 'receipt_state', 'receipt_actor_id', 'receipt_issue_reason', 'receipt_dispute_id'];
$$;

CREATE OR REPLACE FUNCTION app.drawdown_approval_fingerprint(d app.drawdowns) RETURNS bytea
LANGUAGE sql IMMUTABLE SET search_path=pg_catalog,app AS $$
  SELECT sha256(convert_to(app.drawdown_approval_proposal(d)::text, 'UTF8'));
$$;
-- +goose StatementEnd

CREATE POLICY drawdown_approvals_read ON app.tradeline_drawdown_approvals FOR SELECT USING (
  organization_id = app.current_organization_id() AND EXISTS(
    SELECT 1 FROM app.memberships WHERE organization_id = tradeline_drawdown_approvals.organization_id AND user_id = app.current_user_id() AND status = 'active' AND role IN ('owner', 'administrator', 'finance', 'sales')
  )
);

CREATE POLICY drawdown_approvals_insert ON app.tradeline_drawdown_approvals FOR INSERT WITH CHECK (
  organization_id = app.current_organization_id() AND requested_by = app.current_user_id() AND state = 'pending' AND EXISTS(
    SELECT 1 FROM app.memberships WHERE organization_id = tradeline_drawdown_approvals.organization_id AND user_id = app.current_user_id() AND status = 'active' AND role IN ('owner', 'administrator', 'sales')
  )
);

CREATE POLICY drawdown_approvals_decide ON app.tradeline_drawdown_approvals FOR UPDATE USING (
  organization_id = app.current_organization_id() AND state = 'pending' AND requested_by <> app.current_user_id() AND EXISTS(
    SELECT 1 FROM app.memberships WHERE organization_id = tradeline_drawdown_approvals.organization_id AND user_id = app.current_user_id() AND status = 'active' AND role IN ('owner', 'administrator', 'finance')
  )
) WITH CHECK (decided_by = app.current_user_id() AND state IN ('approved', 'rejected'));

CREATE POLICY branch_boundary ON app.tradeline_drawdown_approvals AS RESTRICTIVE USING (
  app.branch_drawdown_access(drawdown_id)
);

DROP TRIGGER IF EXISTS branch_financial_write ON app.tradeline_drawdown_approvals;
CREATE TRIGGER branch_financial_write BEFORE INSERT OR UPDATE ON app.tradeline_drawdown_approvals
  FOR EACH ROW EXECUTE FUNCTION app.guard_branch_financial_write();

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION app.guard_drawdown_approval() RETURNS trigger
LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,app SET row_security=off AS $$
DECLARE
  ctrl app.business_credit_controls;
  supp_org uuid;
  fp bytea;
  decider_id uuid;
  decider_role text;
  rev_limit bigint;
BEGIN
  IF NEW.state NOT IN ('ACTIVATED', 'GOODS_RELEASED') THEN
    RETURN NEW;
  END IF;

  SELECT supplier_organization_id INTO supp_org FROM app.trade_lines WHERE id = NEW.trade_line_id;
  SELECT * INTO ctrl FROM app.business_credit_controls WHERE organization_id = supp_org;
  
  IF NOT FOUND OR NOT ctrl.enabled OR NEW.principal_kobo <= ctrl.threshold_kobo THEN
    RETURN NEW;
  END IF;

  fp := app.drawdown_approval_fingerprint(NEW);
  SELECT decided_by INTO decider_id FROM app.tradeline_drawdown_approvals
  WHERE organization_id = supp_org AND drawdown_id = NEW.id AND state = 'approved' AND fingerprint = fp;

  IF decider_id IS NULL THEN
    RAISE EXCEPTION 'drawdown exceeds approval threshold and requires independent review' USING ERRCODE = '42501';
  END IF;

  SELECT m.role INTO decider_role FROM app.memberships m
  JOIN app.users u ON u.id = m.user_id
  WHERE m.organization_id = supp_org AND m.user_id = decider_id AND m.status = 'active' AND u.status = 'active';

  IF decider_role IS NULL OR decider_role NOT IN ('owner', 'administrator', 'finance') THEN
    RAISE EXCEPTION 'drawdown reviewer does not hold active approval authority' USING ERRCODE = '42501';
  END IF;

  SELECT ceiling_kobo INTO rev_limit FROM app.credit_reviewer_limits
  WHERE organization_id = supp_org AND user_id = decider_id;

  IF FOUND AND rev_limit IS NOT NULL AND NEW.principal_kobo > rev_limit THEN
    RAISE EXCEPTION 'drawdown principal % exceeds reviewer limit %', NEW.principal_kobo, rev_limit USING ERRCODE = '42501';
  END IF;

  RETURN NEW;
END $$;
-- +goose StatementEnd

DROP TRIGGER IF EXISTS drawdown_approval_guard ON app.drawdowns;
CREATE TRIGGER drawdown_approval_guard BEFORE UPDATE OF state ON app.drawdowns
  FOR EACH ROW EXECUTE FUNCTION app.guard_drawdown_approval();

GRANT SELECT, INSERT, UPDATE ON app.tradeline_drawdown_approvals TO kredit_app, kredit_worker;
GRANT EXECUTE ON FUNCTION app.drawdown_approval_proposal(app.drawdowns), app.drawdown_approval_fingerprint(app.drawdowns) TO kredit_app, kredit_worker;

-- +goose Down
DROP TRIGGER IF EXISTS drawdown_approval_guard ON app.drawdowns;
DROP FUNCTION IF EXISTS app.guard_drawdown_approval();
DROP TRIGGER IF EXISTS branch_financial_write ON app.tradeline_drawdown_approvals;
DROP POLICY IF EXISTS branch_boundary ON app.tradeline_drawdown_approvals;
DROP POLICY IF EXISTS drawdown_approvals_decide ON app.tradeline_drawdown_approvals;
DROP POLICY IF EXISTS drawdown_approvals_insert ON app.tradeline_drawdown_approvals;
DROP POLICY IF EXISTS drawdown_approvals_read ON app.tradeline_drawdown_approvals;
DROP FUNCTION IF EXISTS app.drawdown_approval_fingerprint(app.drawdowns);
DROP FUNCTION IF EXISTS app.drawdown_approval_proposal(app.drawdowns);
DROP TABLE IF EXISTS app.tradeline_drawdown_approvals;
ALTER TABLE app.drawdowns DROP COLUMN IF EXISTS branch_id;
