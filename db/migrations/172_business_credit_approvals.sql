-- +goose Up
CREATE TABLE app.business_credit_controls (
 organization_id uuid PRIMARY KEY REFERENCES app.organizations(id),
 enabled boolean NOT NULL DEFAULT false,
 threshold_kobo bigint NOT NULL DEFAULT 0 CHECK(threshold_kobo BETWEEN 0 AND 9007199254740991),
 version bigint NOT NULL DEFAULT 1,
 updated_by uuid NOT NULL REFERENCES app.users(id),
 updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE app.credit_offer_approvals (
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
 organization_id uuid NOT NULL REFERENCES app.organizations(id),
 credit_request_id uuid NOT NULL REFERENCES app.credit_requests(id),
 request_version bigint NOT NULL,
 fingerprint bytea NOT NULL CHECK(octet_length(fingerprint)=32),
 proposal jsonb NOT NULL,
 requested_by uuid NOT NULL REFERENCES app.users(id),
 state text NOT NULL DEFAULT 'pending' CHECK(state IN ('pending','approved','rejected')),
 decided_by uuid REFERENCES app.users(id),
 reason text NOT NULL DEFAULT '',
 created_at timestamptz NOT NULL DEFAULT now(),
 decided_at timestamptz,
 CHECK((state='pending' AND decided_by IS NULL AND decided_at IS NULL) OR (state<>'pending' AND decided_by IS NOT NULL AND decided_at IS NOT NULL AND decided_by<>requested_by)),
 UNIQUE(organization_id,credit_request_id,request_version)
);
ALTER TABLE app.business_credit_controls ENABLE ROW LEVEL SECURITY;
ALTER TABLE app.business_credit_controls FORCE ROW LEVEL SECURITY;
ALTER TABLE app.credit_offer_approvals ENABLE ROW LEVEL SECURITY;
ALTER TABLE app.credit_offer_approvals FORCE ROW LEVEL SECURITY;
-- +goose StatementBegin
CREATE FUNCTION app.credit_offer_proposal(c app.credit_requests) RETURNS jsonb
LANGUAGE sql IMMUTABLE SET search_path=pg_catalog AS $$
 SELECT to_jsonb(c)-ARRAY['state','version','updated_at','agreement_version_id','mandate_id','acceptance_id','release_id','receipt_id','obligation_id'];
$$;
CREATE FUNCTION app.credit_offer_fingerprint(c app.credit_requests) RETURNS bytea
LANGUAGE sql IMMUTABLE SET search_path=pg_catalog,app AS $$
 SELECT sha256(convert_to(app.credit_offer_proposal(c)::text,'UTF8'));
$$;
-- +goose StatementEnd
CREATE POLICY credit_controls_read ON app.business_credit_controls FOR SELECT USING (
 organization_id=app.current_organization_id() AND EXISTS(SELECT 1 FROM app.memberships WHERE organization_id=business_credit_controls.organization_id AND user_id=app.current_user_id() AND status='active'));
CREATE POLICY credit_controls_insert ON app.business_credit_controls FOR INSERT WITH CHECK (
 organization_id=app.current_organization_id() AND updated_by=app.current_user_id() AND EXISTS(SELECT 1 FROM app.memberships WHERE organization_id=business_credit_controls.organization_id AND user_id=app.current_user_id() AND status='active' AND role='owner'));
CREATE POLICY credit_controls_update ON app.business_credit_controls FOR UPDATE USING (
 organization_id=app.current_organization_id() AND EXISTS(SELECT 1 FROM app.memberships WHERE organization_id=business_credit_controls.organization_id AND user_id=app.current_user_id() AND status='active' AND role='owner')) WITH CHECK(updated_by=app.current_user_id());
CREATE POLICY credit_approvals_read ON app.credit_offer_approvals FOR SELECT USING (
 organization_id=app.current_organization_id() AND EXISTS(SELECT 1 FROM app.memberships WHERE organization_id=credit_offer_approvals.organization_id AND user_id=app.current_user_id() AND status='active' AND role IN ('owner','administrator','finance','sales')));
CREATE POLICY credit_approvals_insert ON app.credit_offer_approvals FOR INSERT WITH CHECK (
 organization_id=app.current_organization_id() AND requested_by=app.current_user_id() AND state='pending' AND EXISTS(SELECT 1 FROM app.memberships WHERE organization_id=credit_offer_approvals.organization_id AND user_id=app.current_user_id() AND status='active' AND role IN ('owner','administrator','sales')));
CREATE POLICY credit_approvals_decide ON app.credit_offer_approvals FOR UPDATE USING (
 organization_id=app.current_organization_id() AND state='pending' AND requested_by<>app.current_user_id() AND EXISTS(SELECT 1 FROM app.memberships WHERE organization_id=credit_offer_approvals.organization_id AND user_id=app.current_user_id() AND status='active' AND role IN ('owner','administrator','finance')))
 WITH CHECK (decided_by=app.current_user_id() AND state IN ('approved','rejected'));
-- +goose StatementBegin
CREATE FUNCTION app.guard_credit_offer_approval() RETURNS trigger
LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,app SET row_security=off AS $$
DECLARE control app.business_credit_controls;
BEGIN
 IF TG_OP='INSERT' AND EXISTS(SELECT 1 FROM app.credit_requests WHERE id=NEW.id) THEN RETURN NEW; END IF;
 IF EXISTS(SELECT 1 FROM app.drawdowns WHERE id=NEW.id) THEN RETURN NEW; END IF;
 IF NEW.state IN ('DRAFT','CANCELLED','DECLINED','EXPIRED') THEN RETURN NEW; END IF;
 IF TG_OP='UPDATE' AND OLD.state<>'DRAFT' THEN RETURN NEW; END IF;
 PERFORM pg_advisory_xact_lock_shared(hashtextextended(NEW.supplier_organization_id::text,172));
 SELECT * INTO control FROM app.business_credit_controls WHERE organization_id=NEW.supplier_organization_id FOR SHARE;
 IF NOT FOUND OR NOT control.enabled OR NEW.principal_kobo<=control.threshold_kobo THEN RETURN NEW; END IF;
 PERFORM 1 FROM app.credit_offer_approvals a JOIN app.memberships m ON m.organization_id=a.organization_id AND m.user_id=a.decided_by JOIN app.users u ON u.id=m.user_id
 WHERE a.organization_id=NEW.supplier_organization_id AND a.credit_request_id=NEW.id
 AND a.request_version=CASE WHEN TG_OP='UPDATE' THEN OLD.version ELSE -1 END
 AND a.state='approved' AND a.fingerprint=app.credit_offer_fingerprint(NEW)
 AND a.decided_by<>NEW.created_by AND a.decided_by<>a.requested_by
 AND m.status='active' AND m.role IN ('owner','administrator','finance') AND u.status='active'
 FOR SHARE OF a,m,u;
 IF NOT FOUND THEN RAISE EXCEPTION 'an independent credit approval is required for this offer' USING ERRCODE='42501'; END IF;
 RETURN NEW;
END $$;
-- +goose StatementEnd
CREATE TRIGGER credit_offer_approval BEFORE INSERT OR UPDATE ON app.credit_requests FOR EACH ROW EXECUTE FUNCTION app.guard_credit_offer_approval();
-- +goose StatementBegin
DO $$ BEGIN IF EXISTS(SELECT 1 FROM pg_roles WHERE rolname='kredit_app') THEN
 GRANT SELECT,INSERT,UPDATE ON app.business_credit_controls,app.credit_offer_approvals TO kredit_app;
END IF; END $$;
-- +goose StatementEnd
-- +goose Down
-- +goose StatementBegin
DO $$ BEGIN RAISE EXCEPTION 'Credit approval evidence requires forward recovery'; END $$;
-- +goose StatementEnd
