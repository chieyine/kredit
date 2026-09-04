-- +goose Up
ALTER VIEW app.financial_discrepancies RENAME TO financial_discrepancies_base;
CREATE VIEW app.financial_discrepancies WITH (security_invoker=true) AS
 SELECT * FROM app.financial_discrepancies_base
 UNION ALL
 SELECT 'provider_reversal',a.id::text,0::numeric,p.amount_kobo::numeric
 FROM app.collection_attempts a JOIN app.payments p ON p.idempotency_key='collection-attempt:'||a.id::text
 WHERE lower(COALESCE(a.settlement_state,''))='reversed' AND p.state='recognized'
 UNION ALL
 SELECT 'settlement_without_payment',a.id::text,a.requested_amount_kobo::numeric,0::numeric
 FROM app.collection_attempts a WHERE lower(COALESCE(a.settlement_state,'')) IN ('settled','completed','success') AND a.succeeded_amount_kobo=0;
-- +goose Down
DROP VIEW app.financial_discrepancies;
ALTER VIEW app.financial_discrepancies_base RENAME TO financial_discrepancies;
