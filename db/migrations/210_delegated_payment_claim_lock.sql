-- +goose Up
-- Match the delegated 'claim' permission offered by the API. This capability
-- only locks debt; it grants no UPDATE authority over financial values.
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION app.lock_buyer_payment_claim(p_obligation_id uuid)
RETURNS TABLE(outstanding_kobo bigint,supplier_organization_id uuid,currency character(3))
LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,app,pg_temp AS $$
DECLARE actor uuid:=app.current_user_id(); profile uuid; owner_id uuid;
BEGIN
 PERFORM 1 FROM app.users WHERE id=actor AND status='active' FOR SHARE;
 IF NOT FOUND THEN RAISE EXCEPTION 'payment claim obligation is unavailable' USING ERRCODE='42501'; END IF;
 SELECT o.buyer_business_id,c.buyer_user_id INTO profile,owner_id
 FROM app.obligations o JOIN app.credit_requests c ON c.id=o.credit_request_id
 WHERE o.id=p_obligation_id AND c.buyer_business_id=o.buyer_business_id
 AND c.supplier_organization_id=o.supplier_organization_id;
 IF NOT FOUND OR NOT app.obligation_authority_current(p_obligation_id)
 OR NOT app.branch_obligation_access(p_obligation_id) THEN
  RAISE EXCEPTION 'payment claim obligation is unavailable' USING ERRCODE='42501';
 END IF;
 IF owner_id=actor THEN
  PERFORM app.lock_obligation_authority(p_obligation_id);
 ELSE
  PERFORM app.lock_purchase_permission(profile,'claim',0);
 END IF;
 RETURN QUERY SELECT o.outstanding_kobo,o.supplier_organization_id,o.currency
 FROM app.obligations o JOIN app.credit_requests c ON c.id=o.credit_request_id
 WHERE o.id=p_obligation_id AND o.buyer_business_id=profile AND c.buyer_business_id=profile
 AND c.supplier_organization_id=o.supplier_organization_id
 AND (c.buyer_user_id=actor OR app.can_purchase(profile,'claim',0))
 AND app.obligation_authority_current(o.id) AND app.branch_obligation_access(o.id)
 FOR UPDATE OF o;
 IF NOT FOUND THEN RAISE EXCEPTION 'payment claim obligation is unavailable' USING ERRCODE='42501'; END IF;
END $$;
-- +goose StatementEnd
ALTER POLICY claim_obligation_lock_owner ON app.obligations
USING (
 current_user=pg_get_userbyid((SELECT proowner FROM pg_proc WHERE oid='app.lock_buyer_payment_claim(uuid)'::regprocedure))
 AND EXISTS(SELECT 1 FROM app.credit_requests c WHERE c.id=obligations.credit_request_id
 AND (c.buyer_user_id=app.current_user_id() OR app.can_purchase(c.buyer_business_id,'claim',0)))
)
WITH CHECK(false);
REVOKE ALL ON FUNCTION app.lock_buyer_payment_claim(uuid) FROM PUBLIC,kredit_worker;
GRANT EXECUTE ON FUNCTION app.lock_buyer_payment_claim(uuid) TO kredit_app;

-- +goose Down
SELECT 1;
