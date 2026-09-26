-- +goose Up
-- Approval covers agreed terms. Reservation, release and receipt evidence is
-- subsequently appended by the lifecycle and must not invalidate that decision.
-- Quiesce API/worker writers before applying and deploy matching binaries.
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION app.drawdown_approval_proposal(d app.drawdowns) RETURNS jsonb
LANGUAGE sql IMMUTABLE SET search_path=pg_catalog,app,pg_temp AS $$
 SELECT to_jsonb(d)-ARRAY['state','reservation_id','obligation_id','buyer_confirmed_at',
 'release_actor_id','delivery_method','release_notes','release_evidence_reference','released_at',
 'receipt_at','receipt_state','receipt_actor_id','receipt_issue_reason','receipt_dispute_id','activated_at'];
$$;
-- Recompute from each stored proposal, never from today's potentially changed
-- drawdown. Historical approval of different terms remains different.
ALTER TABLE app.tradeline_drawdown_approvals DISABLE TRIGGER branch_financial_write;
WITH intents AS (
 SELECT id,proposal-ARRAY['state','reservation_id','obligation_id','buyer_confirmed_at',
 'release_actor_id','delivery_method','release_notes','release_evidence_reference','released_at',
 'receipt_at','receipt_state','receipt_actor_id','receipt_issue_reason','receipt_dispute_id','activated_at'] AS intent
 FROM app.tradeline_drawdown_approvals
)
UPDATE app.tradeline_drawdown_approvals a SET proposal=i.intent,
 fingerprint=sha256(convert_to(i.intent::text,'UTF8')) FROM intents i WHERE i.id=a.id;
ALTER TABLE app.tradeline_drawdown_approvals ENABLE TRIGGER branch_financial_write;

CREATE OR REPLACE FUNCTION app.guard_drawdown_approval() RETURNS trigger
LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,app,pg_temp SET row_security=off AS $$
DECLARE control app.business_credit_controls; supplier uuid; reviewer uuid; ceiling bigint;
BEGIN
 -- An upsert first fires INSERT triggers even for an existing row. Its UPDATE
 -- trigger checks the actual transition after PostgreSQL resolves the conflict.
 IF TG_OP='INSERT' AND EXISTS(SELECT 1 FROM app.drawdowns WHERE id=NEW.id) THEN RETURN NEW; END IF;
 IF NEW.state NOT IN ('GOODS_RELEASED','ACTIVATED') THEN RETURN NEW; END IF;
 -- Once goods have been released, later revocation cannot block receipt or
 -- reconciliation of the existing sale when the aggregate is persisted again.
 IF TG_OP='UPDATE' AND OLD.state IN ('GOODS_RELEASED','ACTIVATED') THEN RETURN NEW; END IF;
 SELECT supplier_organization_id INTO supplier FROM app.trade_lines WHERE id=NEW.trade_line_id;
 PERFORM pg_advisory_xact_lock_shared(hashtextextended(supplier::text,172));
 PERFORM pg_advisory_xact_lock_shared(hashtextextended(supplier::text,175));
 SELECT * INTO control FROM app.business_credit_controls WHERE organization_id=supplier FOR SHARE;
 IF NOT FOUND OR NOT control.enabled OR NEW.principal_kobo<=control.threshold_kobo THEN RETURN NEW; END IF;
 SELECT a.decided_by INTO reviewer FROM app.tradeline_drawdown_approvals a
 JOIN app.memberships m ON m.organization_id=a.organization_id AND m.user_id=a.decided_by
 JOIN app.users u ON u.id=m.user_id
 WHERE a.organization_id=supplier AND a.drawdown_id=NEW.id AND a.state='approved'
 AND a.fingerprint=app.drawdown_approval_fingerprint(NEW) AND a.decided_by<>a.requested_by
 AND m.status='active' AND m.role IN ('owner','administrator','finance') AND u.status='active'
 FOR SHARE OF a,m,u;
 IF NOT FOUND THEN RAISE EXCEPTION 'an independent drawdown approval is required' USING ERRCODE='KR001'; END IF;
 SELECT ceiling_kobo INTO ceiling FROM app.credit_reviewer_limits
 WHERE organization_id=supplier AND user_id=reviewer FOR SHARE;
 IF FOUND AND NEW.principal_kobo>ceiling THEN RAISE EXCEPTION 'reviewer approval ceiling exceeded' USING ERRCODE='KR002'; END IF;
 RETURN NEW;
END $$;

CREATE FUNCTION app.guard_drawdown_approval_evidence() RETURNS trigger
LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,app,pg_temp SET row_security=off AS $$
DECLARE d app.drawdowns; supplier uuid; ceiling bigint; actor uuid:=app.current_user_id();
BEGIN
 SELECT * INTO d FROM app.drawdowns WHERE id=NEW.drawdown_id FOR UPDATE;
 SELECT supplier_organization_id INTO supplier FROM app.trade_lines WHERE id=d.trade_line_id;
 IF supplier IS DISTINCT FROM NEW.organization_id OR d.state NOT IN ('PENDING_BUYER_CONFIRMATION','BUYER_CONFIRMED')
 OR NEW.fingerprint IS DISTINCT FROM app.drawdown_approval_fingerprint(d)
 OR NEW.proposal IS DISTINCT FROM app.drawdown_approval_proposal(d) THEN
  RAISE EXCEPTION 'approval must reference current drawdown terms' USING ERRCODE='23514';
 END IF;
 IF TG_OP='INSERT' THEN
  IF NEW.state<>'pending' OR NEW.requested_by IS DISTINCT FROM actor OR NEW.drawdown_version<>1 THEN
   RAISE EXCEPTION 'approval must begin with its current requester' USING ERRCODE='42501';
  END IF;
 ELSE
  IF (to_jsonb(NEW)-ARRAY['state','decided_by','reason','decided_at']) IS DISTINCT FROM
     (to_jsonb(OLD)-ARRAY['state','decided_by','reason','decided_at']) THEN
   RAISE EXCEPTION 'drawdown approval intent is immutable' USING ERRCODE='23514';
  END IF;
  IF OLD.state<>'pending' OR NEW.state NOT IN ('approved','rejected') OR NEW.decided_by IS DISTINCT FROM actor
  OR NEW.decided_by=NEW.requested_by OR length(btrim(NEW.reason))<3 OR length(NEW.reason)>1000 THEN
   RAISE EXCEPTION 'independent drawdown decision required' USING ERRCODE='42501';
  END IF;
 END IF;
 PERFORM 1 FROM app.memberships m JOIN app.users u ON u.id=m.user_id JOIN app.organizations o ON o.id=m.organization_id
 WHERE m.organization_id=supplier AND m.user_id=actor AND m.status='active' AND u.status='active' AND o.status<>'suspended'
 AND m.role=ANY(CASE WHEN TG_OP='INSERT' THEN ARRAY['owner','administrator','sales'] ELSE ARRAY['owner','administrator','finance'] END)
 FOR SHARE OF m,u,o;
 IF NOT FOUND THEN RAISE EXCEPTION 'current drawdown reviewer authority required' USING ERRCODE='42501'; END IF;
 IF NEW.state='approved' THEN
  PERFORM pg_advisory_xact_lock_shared(hashtextextended(supplier::text,175));
  SELECT ceiling_kobo INTO ceiling FROM app.credit_reviewer_limits WHERE organization_id=supplier AND user_id=actor FOR SHARE;
  IF FOUND AND d.principal_kobo>ceiling THEN RAISE EXCEPTION 'reviewer approval ceiling exceeded' USING ERRCODE='KR002'; END IF;
 END IF;
 RETURN NEW;
END $$;
-- +goose StatementEnd
DROP TRIGGER drawdown_approval_guard ON app.drawdowns;
CREATE TRIGGER drawdown_approval_guard BEFORE INSERT OR UPDATE ON app.drawdowns
 FOR EACH ROW EXECUTE FUNCTION app.guard_drawdown_approval();
CREATE TRIGGER drawdown_approval_evidence BEFORE INSERT OR UPDATE ON app.tradeline_drawdown_approvals
 FOR EACH ROW EXECUTE FUNCTION app.guard_drawdown_approval_evidence();
REVOKE ALL ON FUNCTION app.guard_drawdown_approval_evidence(),app.guard_drawdown_approval() FROM PUBLIC;

-- +goose Down
-- Keep immutable approval evidence and the release boundary on binary rollback.
SELECT 1;
