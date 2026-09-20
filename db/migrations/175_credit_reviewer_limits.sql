-- +goose Up
CREATE TABLE app.credit_reviewer_limits (
 organization_id uuid NOT NULL REFERENCES app.organizations(id),
 user_id uuid NOT NULL REFERENCES app.users(id),
 ceiling_kobo bigint NOT NULL CHECK(ceiling_kobo BETWEEN 0 AND 9007199254740991),
 version bigint NOT NULL DEFAULT 1 CHECK(version>0),
 updated_by uuid NOT NULL REFERENCES app.users(id),
 updated_at timestamptz NOT NULL DEFAULT now(),
 PRIMARY KEY(organization_id,user_id)
);
CREATE TABLE app.credit_reviewer_limit_history (LIKE app.credit_reviewer_limits INCLUDING DEFAULTS INCLUDING CONSTRAINTS);
ALTER TABLE app.credit_reviewer_limit_history ADD PRIMARY KEY(organization_id,user_id,version);
ALTER TABLE app.credit_reviewer_limits ENABLE ROW LEVEL SECURITY;
ALTER TABLE app.credit_reviewer_limits FORCE ROW LEVEL SECURITY;
ALTER TABLE app.credit_reviewer_limit_history ENABLE ROW LEVEL SECURITY;
ALTER TABLE app.credit_reviewer_limit_history FORCE ROW LEVEL SECURITY;
CREATE POLICY reviewer_limits_read ON app.credit_reviewer_limits FOR SELECT USING(organization_id=app.current_organization_id() AND EXISTS(SELECT 1 FROM app.memberships m WHERE m.organization_id=credit_reviewer_limits.organization_id AND m.user_id=app.current_user_id() AND m.status='active'));
CREATE POLICY reviewer_limits_create ON app.credit_reviewer_limits FOR INSERT WITH CHECK(organization_id=app.current_organization_id() AND updated_by=app.current_user_id() AND EXISTS(SELECT 1 FROM app.memberships m WHERE m.organization_id=credit_reviewer_limits.organization_id AND m.user_id=app.current_user_id() AND m.status='active' AND m.role='owner'));
CREATE POLICY reviewer_limits_change ON app.credit_reviewer_limits FOR UPDATE USING(organization_id=app.current_organization_id() AND EXISTS(SELECT 1 FROM app.memberships m WHERE m.organization_id=credit_reviewer_limits.organization_id AND m.user_id=app.current_user_id() AND m.status='active' AND m.role='owner')) WITH CHECK(updated_by=app.current_user_id() AND organization_id=app.current_organization_id());
CREATE POLICY reviewer_limit_history_read ON app.credit_reviewer_limit_history FOR SELECT USING(organization_id=app.current_organization_id() AND EXISTS(SELECT 1 FROM app.memberships m WHERE m.organization_id=credit_reviewer_limit_history.organization_id AND m.user_id=app.current_user_id() AND m.status='active' AND m.role IN ('owner','administrator','finance')));
-- +goose StatementBegin
CREATE FUNCTION app.record_credit_reviewer_limit() RETURNS trigger LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,app SET row_security=off AS $$
BEGIN
 PERFORM pg_advisory_xact_lock(hashtextextended(NEW.organization_id::text,175));
 IF TG_OP='UPDATE' AND (NEW.organization_id<>OLD.organization_id OR NEW.user_id<>OLD.user_id OR NEW.version<>OLD.version+1) THEN RAISE EXCEPTION 'reviewer limit identity and versions are immutable' USING ERRCODE='23514'; END IF;
 IF TG_OP='INSERT' AND NEW.version<>1 THEN RAISE EXCEPTION 'initial reviewer limit version must be one' USING ERRCODE='23514'; END IF;
 INSERT INTO app.credit_reviewer_limit_history SELECT NEW.*;
 RETURN NEW;
END $$;
CREATE FUNCTION app.enforce_credit_reviewer_limit() RETURNS trigger LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,app SET row_security=off AS $$
DECLARE amount bigint; ceiling bigint;
BEGIN
 IF NEW.state<>'approved' THEN RETURN NEW; END IF;
 PERFORM pg_advisory_xact_lock_shared(hashtextextended(NEW.organization_id::text,175));
 SELECT principal_kobo INTO amount FROM app.credit_requests WHERE id=NEW.credit_request_id;
 SELECT ceiling_kobo INTO ceiling FROM app.credit_reviewer_limits WHERE organization_id=NEW.organization_id AND user_id=NEW.decided_by FOR SHARE;
 IF FOUND AND amount>ceiling THEN RAISE EXCEPTION 'reviewer approval ceiling exceeded' USING ERRCODE='42501'; END IF;
 RETURN NEW;
END $$;
CREATE FUNCTION app.enforce_offer_reviewer_limit() RETURNS trigger LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,app SET row_security=off AS $$
DECLARE ceiling bigint;
BEGIN
 IF TG_OP='INSERT' THEN RETURN NEW; END IF;
 PERFORM pg_advisory_xact_lock_shared(hashtextextended(NEW.supplier_organization_id::text,175));
 IF OLD.state<>'DRAFT' OR NEW.state IN ('DRAFT','CANCELLED','DECLINED','EXPIRED') THEN RETURN NEW; END IF;
 IF EXISTS(SELECT 1 FROM app.drawdowns WHERE id=NEW.id) THEN RETURN NEW; END IF;
 IF NOT EXISTS(SELECT 1 FROM app.business_credit_controls WHERE organization_id=NEW.supplier_organization_id AND enabled AND NEW.principal_kobo>threshold_kobo) THEN RETURN NEW; END IF;
 SELECT l.ceiling_kobo INTO ceiling FROM app.credit_offer_approvals a JOIN app.credit_reviewer_limits l ON l.organization_id=a.organization_id AND l.user_id=a.decided_by WHERE a.credit_request_id=NEW.id AND a.request_version=OLD.version AND a.state='approved' FOR SHARE OF l;
 IF FOUND AND NEW.principal_kobo>ceiling THEN RAISE EXCEPTION 'reviewer approval ceiling exceeded' USING ERRCODE='42501'; END IF;
 RETURN NEW;
END $$;
-- +goose StatementEnd
CREATE TRIGGER reviewer_limit_history AFTER INSERT OR UPDATE ON app.credit_reviewer_limits FOR EACH ROW EXECUTE FUNCTION app.record_credit_reviewer_limit();
CREATE TRIGGER reviewer_limit_decision BEFORE INSERT OR UPDATE ON app.credit_offer_approvals FOR EACH ROW EXECUTE FUNCTION app.enforce_credit_reviewer_limit();
CREATE TRIGGER reviewer_limit_send BEFORE INSERT OR UPDATE ON app.credit_requests FOR EACH ROW EXECUTE FUNCTION app.enforce_offer_reviewer_limit();
-- +goose StatementBegin
DO $$ BEGIN IF EXISTS(SELECT 1 FROM pg_roles WHERE rolname='kredit_app') THEN
 GRANT SELECT,INSERT,UPDATE ON app.credit_reviewer_limits TO kredit_app;
 GRANT SELECT ON app.credit_reviewer_limit_history TO kredit_app;
END IF; END $$;
-- +goose StatementEnd
-- +goose Down
-- +goose StatementBegin
DO $$ BEGIN RAISE EXCEPTION 'Reviewer limits require forward recovery'; END $$;
-- +goose StatementEnd
