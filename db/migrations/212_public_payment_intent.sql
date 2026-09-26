-- +goose Up
-- The HTTP endpoint validates a short-lived signed payment link before using
-- this exact-reference projection. Return only the fields that page displays,
-- from current normalized balances rather than a process-local cache.
-- +goose StatementBegin
CREATE FUNCTION app.public_payment_intent(request_id uuid) RETURNS jsonb
LANGUAGE sql STABLE SECURITY DEFINER SET search_path=pg_catalog,app,pg_temp SET row_security=off AS $$
 SELECT jsonb_build_object(
 'request',jsonb_build_object('id',c.id,'supplier_trading_name',COALESCE(NULLIF(o.trading_name,''),o.legal_name),'goods_description',c.goods_description),
 'obligation',jsonb_build_object('outstanding_kobo',d.outstanding_kobo,'currency',d.currency,'payment_status',d.payment_status))
 FROM app.credit_requests c JOIN app.obligations d ON d.credit_request_id=c.id
 JOIN app.organizations o ON o.id=c.supplier_organization_id
 WHERE c.id=request_id;
$$;
-- +goose StatementEnd
REVOKE ALL ON FUNCTION app.public_payment_intent(uuid) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION app.public_payment_intent(uuid) TO kredit_app;
-- +goose Down
SELECT 1;
