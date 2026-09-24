-- +goose Up
-- Give the two credit-approval refusals their own SQLSTATE so the API can
-- recognise them by code. Both used 42501 (insufficient_privilege), which the
-- API could only tell apart by matching the message text. Function bodies are
-- otherwise unchanged apart from listing pg_temp last in search_path.
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION app.guard_credit_offer_approval() RETURNS trigger
LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,app,pg_temp SET row_security=off AS $$
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
 IF NOT FOUND THEN RAISE EXCEPTION 'an independent credit approval is required for this offer' USING ERRCODE='KR001'; END IF;
 RETURN NEW;
END $$;
-- +goose StatementEnd
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION app.enforce_credit_reviewer_limit() RETURNS trigger LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,app,pg_temp SET row_security=off AS $$
DECLARE amount bigint; ceiling bigint;
BEGIN
 IF NEW.state<>'approved' THEN RETURN NEW; END IF;
 PERFORM pg_advisory_xact_lock_shared(hashtextextended(NEW.organization_id::text,175));
 SELECT principal_kobo INTO amount FROM app.credit_requests WHERE id=NEW.credit_request_id;
 SELECT ceiling_kobo INTO ceiling FROM app.credit_reviewer_limits WHERE organization_id=NEW.organization_id AND user_id=NEW.decided_by FOR SHARE;
 IF FOUND AND amount>ceiling THEN RAISE EXCEPTION 'reviewer approval ceiling exceeded' USING ERRCODE='KR002'; END IF;
 RETURN NEW;
END $$;
-- +goose StatementEnd
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION app.enforce_offer_reviewer_limit() RETURNS trigger LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,app,pg_temp SET row_security=off AS $$
DECLARE ceiling bigint;
BEGIN
 IF TG_OP='INSERT' THEN RETURN NEW; END IF;
 PERFORM pg_advisory_xact_lock_shared(hashtextextended(NEW.supplier_organization_id::text,175));
 IF OLD.state<>'DRAFT' OR NEW.state IN ('DRAFT','CANCELLED','DECLINED','EXPIRED') THEN RETURN NEW; END IF;
 IF EXISTS(SELECT 1 FROM app.drawdowns WHERE id=NEW.id) THEN RETURN NEW; END IF;
 IF NOT EXISTS(SELECT 1 FROM app.business_credit_controls WHERE organization_id=NEW.supplier_organization_id AND enabled AND NEW.principal_kobo>threshold_kobo) THEN RETURN NEW; END IF;
 SELECT l.ceiling_kobo INTO ceiling FROM app.credit_offer_approvals a JOIN app.credit_reviewer_limits l ON l.organization_id=a.organization_id AND l.user_id=a.decided_by WHERE a.credit_request_id=NEW.id AND a.request_version=OLD.version AND a.state='approved' FOR SHARE OF l;
 IF FOUND AND NEW.principal_kobo>ceiling THEN RAISE EXCEPTION 'reviewer approval ceiling exceeded' USING ERRCODE='KR002'; END IF;
 RETURN NEW;
END $$;
-- +goose StatementEnd

-- Money columns that had no range check. Every writer already keeps these
-- values in range; the constraints make the database refuse anything else.
ALTER TABLE app.obligations
  ADD CONSTRAINT obligations_principal_kobo_range CHECK (principal_kobo > 0 AND principal_kobo <= 9007199254740991),
  ADD CONSTRAINT obligations_outstanding_kobo_range CHECK (outstanding_kobo >= 0 AND outstanding_kobo <= principal_kobo),
  ADD CONSTRAINT obligations_base_fee_kobo_range CHECK (base_fee_kobo >= 0 AND base_fee_kobo <= principal_kobo);
ALTER TABLE app.business_credit_control_history
  ADD CONSTRAINT business_credit_control_history_threshold_kobo_range CHECK (threshold_kobo >= 0 AND threshold_kobo <= 9007199254740991);
ALTER TABLE app.dsa_referrals
  ADD CONSTRAINT dsa_referrals_activation_baseline_kobo_range CHECK (activation_baseline_kobo >= 0),
  ADD CONSTRAINT dsa_referrals_fees_kobo_range CHECK (fees_kobo >= 0);
ALTER TABLE app.provider_reconciliation_events
  ADD CONSTRAINT provider_reconciliation_events_amount_kobo_range CHECK (amount_kobo IS NULL OR amount_kobo >= 0);

-- +goose Down
ALTER TABLE app.provider_reconciliation_events DROP CONSTRAINT provider_reconciliation_events_amount_kobo_range;
ALTER TABLE app.dsa_referrals
  DROP CONSTRAINT dsa_referrals_fees_kobo_range,
  DROP CONSTRAINT dsa_referrals_activation_baseline_kobo_range;
ALTER TABLE app.business_credit_control_history DROP CONSTRAINT business_credit_control_history_threshold_kobo_range;
ALTER TABLE app.obligations
  DROP CONSTRAINT obligations_base_fee_kobo_range,
  DROP CONSTRAINT obligations_outstanding_kobo_range,
  DROP CONSTRAINT obligations_principal_kobo_range;
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION app.guard_credit_offer_approval() RETURNS trigger
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
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION app.enforce_credit_reviewer_limit() RETURNS trigger LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,app SET row_security=off AS $$
DECLARE amount bigint; ceiling bigint;
BEGIN
 IF NEW.state<>'approved' THEN RETURN NEW; END IF;
 PERFORM pg_advisory_xact_lock_shared(hashtextextended(NEW.organization_id::text,175));
 SELECT principal_kobo INTO amount FROM app.credit_requests WHERE id=NEW.credit_request_id;
 SELECT ceiling_kobo INTO ceiling FROM app.credit_reviewer_limits WHERE organization_id=NEW.organization_id AND user_id=NEW.decided_by FOR SHARE;
 IF FOUND AND amount>ceiling THEN RAISE EXCEPTION 'reviewer approval ceiling exceeded' USING ERRCODE='42501'; END IF;
 RETURN NEW;
END $$;
-- +goose StatementEnd
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION app.enforce_offer_reviewer_limit() RETURNS trigger LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,app SET row_security=off AS $$
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
