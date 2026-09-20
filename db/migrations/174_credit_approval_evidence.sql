-- +goose Up
-- Approval evidence is captured from the locked draft, never supplied by a client.
-- +goose StatementBegin
CREATE FUNCTION app.guard_credit_approval_evidence() RETURNS trigger
LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,app SET row_security=off AS $$
DECLARE offer app.credit_requests;
BEGIN
 SELECT * INTO offer FROM app.credit_requests WHERE id=NEW.credit_request_id FOR UPDATE;
 IF NOT FOUND OR offer.supplier_organization_id<>NEW.organization_id THEN
  RAISE EXCEPTION 'approval must reference its own business offer' USING ERRCODE='23514';
 END IF;
 IF TG_OP='INSERT' THEN
  IF offer.state<>'DRAFT' OR NEW.state<>'pending' OR NEW.request_version<>offer.version
    OR NEW.fingerprint<>app.credit_offer_fingerprint(offer) OR NEW.proposal<>app.credit_offer_proposal(offer) THEN
   RAISE EXCEPTION 'approval must capture the current draft' USING ERRCODE='23514';
  END IF;
 ELSE
  IF (to_jsonb(NEW)-ARRAY['state','decided_by','reason','decided_at']) IS DISTINCT FROM
     (to_jsonb(OLD)-ARRAY['state','decided_by','reason','decided_at']) THEN
   RAISE EXCEPTION 'approval evidence cannot be rewritten' USING ERRCODE='23514';
  END IF;
  IF OLD.state<>'pending' OR NEW.state NOT IN ('approved','rejected') OR
    NEW.decided_by=offer.created_by OR NEW.decided_by=NEW.requested_by OR
    length(btrim(NEW.reason))<3 OR length(NEW.reason)>1000 OR
    offer.state<>'DRAFT' OR offer.version<>NEW.request_version OR
    app.credit_offer_fingerprint(offer)<>NEW.fingerprint THEN
   RAISE EXCEPTION 'approval requires an independent decision on the current draft' USING ERRCODE='23514';
  END IF;
 END IF;
 RETURN NEW;
END $$;
-- +goose StatementEnd
CREATE TRIGGER credit_approval_evidence BEFORE INSERT OR UPDATE ON app.credit_offer_approvals
 FOR EACH ROW EXECUTE FUNCTION app.guard_credit_approval_evidence();

-- +goose Down
-- +goose StatementBegin
DO $$ BEGIN RAISE EXCEPTION 'Credit approval evidence requires forward recovery'; END $$;
-- +goose StatementEnd
