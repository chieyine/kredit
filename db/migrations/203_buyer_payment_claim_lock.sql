-- +goose Up
-- SELECT FOR UPDATE also needs an UPDATE RLS policy. A buyer may read its
-- obligation but must never receive permission to rewrite the supplier's debt.
-- This capability locks one verified obligation without changing any row or
-- switching the request into a supplier tenant.
-- +goose StatementBegin
CREATE FUNCTION app.lock_buyer_payment_claim(p_obligation_id uuid)
RETURNS TABLE(outstanding_kobo bigint,supplier_organization_id uuid,currency character(3))
LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,app AS $$
DECLARE actor uuid; profile uuid;
BEGIN
 actor:=app.current_user_id();
 IF actor IS NULL THEN
  RAISE EXCEPTION 'payment claim obligation is unavailable' USING ERRCODE='42501';
 END IF;
 -- Serialize actor suspension before acquiring the obligation lock.
 PERFORM 1 FROM app.users u WHERE u.id=actor AND u.status='active' FOR SHARE;
 IF NOT FOUND THEN
  RAISE EXCEPTION 'payment claim obligation is unavailable' USING ERRCODE='42501';
 END IF;
 SELECT o.buyer_business_id INTO profile
 FROM app.obligations o JOIN app.credit_requests c ON c.id=o.credit_request_id
 WHERE o.id=p_obligation_id AND c.buyer_user_id=actor
 AND c.buyer_business_id=o.buyer_business_id
 AND c.supplier_organization_id=o.supplier_organization_id;
 IF NOT FOUND OR NOT app.obligation_authority_current(p_obligation_id)
 OR NOT app.branch_obligation_access(p_obligation_id) THEN
  RAISE EXCEPTION 'payment claim obligation is unavailable' USING ERRCODE='42501';
 END IF;
 -- Preserve the existing current-owner/revocation rules, including legacy
 -- profiles not yet bound to a workspace. This is not delegated claim access.
 PERFORM app.lock_obligation_authority(p_obligation_id);
 RETURN QUERY
 SELECT o.outstanding_kobo,o.supplier_organization_id,o.currency
 FROM app.obligations o JOIN app.credit_requests c ON c.id=o.credit_request_id
 WHERE o.id=p_obligation_id AND c.buyer_user_id=actor
 AND c.buyer_business_id=profile AND o.buyer_business_id=profile
 AND c.supplier_organization_id=o.supplier_organization_id
 AND app.obligation_authority_current(o.id) AND app.branch_obligation_access(o.id)
 FOR UPDATE OF o;
 IF NOT FOUND THEN
  RAISE EXCEPTION 'payment claim obligation is unavailable' USING ERRCODE='42501';
 END IF;
END $$;
-- +goose StatementEnd
REVOKE ALL ON FUNCTION app.lock_buyer_payment_claim(uuid) FROM PUBLIC;
REVOKE ALL ON FUNCTION app.lock_buyer_payment_claim(uuid) FROM kredit_worker;
GRANT EXECUTE ON FUNCTION app.lock_buyer_payment_claim(uuid) TO kredit_app;

-- Support a non-superuser function owner even when FORCE RLS is enabled.
-- Only the trusted function owner can use this additional lock policy. Normal
-- app connections do not match it. WITH CHECK(false) grants no row mutation;
-- the existing restrictive branch and current-authority policies still apply.
CREATE POLICY claim_obligation_lock_owner ON app.obligations FOR UPDATE
USING (
 current_user=pg_get_userbyid((SELECT proowner FROM pg_proc
  WHERE oid='app.lock_buyer_payment_claim(uuid)'::regprocedure))
 AND EXISTS(SELECT 1 FROM app.credit_requests c
  WHERE c.id=obligations.credit_request_id AND c.buyer_user_id=app.current_user_id())
)
WITH CHECK(false);

-- +goose Down
-- +goose StatementBegin
DO $$ BEGIN RAISE EXCEPTION 'Payment claim locking requires forward recovery'; END $$;
-- +goose StatementEnd
