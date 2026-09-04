-- +goose Up
CREATE VIEW app.financial_discrepancies WITH (security_invoker=true) AS
WITH forgiven AS (
 SELECT o.id,COALESCE(SUM(p.credit_kobo-p.debit_kobo),0) amount
 FROM app.obligations o
 JOIN ledger.transactions t ON (t.event_type='write_off' AND t.reference_type='obligation' AND t.reference_id=o.id::text)
 OR (t.event_type='dispute_adjustment' AND t.reference_type='dispute' AND EXISTS(SELECT 1 FROM app.disputes d WHERE d.id::text=t.reference_id AND d.obligation_id=o.id))
 JOIN ledger.postings p ON p.transaction_id=t.id JOIN ledger.accounts a ON a.id=p.account_id AND a.code='TRADE_RECEIVABLE_CONTROL' GROUP BY o.id
), paid AS (SELECT obligation_id,SUM(amount_kobo) amount FROM app.payments WHERE state='recognized' GROUP BY obligation_id)
SELECT 'ledger'::text kind,t.id::text target_id,COALESCE(SUM(p.debit_kobo),0) expected,COALESCE(SUM(p.credit_kobo),0) actual
 FROM ledger.transactions t LEFT JOIN ledger.postings p ON p.transaction_id=t.id GROUP BY t.id HAVING count(p.id)=0 OR SUM(p.debit_kobo)<>SUM(p.credit_kobo)
UNION ALL
SELECT 'balance',o.id::text,o.principal_kobo-COALESCE(p.amount,0)-COALESCE(f.amount,0),o.outstanding_kobo
 FROM app.obligations o LEFT JOIN paid p ON p.obligation_id=o.id LEFT JOIN forgiven f ON f.id=o.id
 WHERE o.outstanding_kobo<>o.principal_kobo-COALESCE(p.amount,0)-COALESCE(f.amount,0)
UNION ALL
SELECT 'schedule',o.id::text,o.outstanding_kobo,COALESCE(SUM(i.principal_due_kobo-i.allocated_kobo) FILTER(WHERE i.state<>'CANCELLED'),0)
 FROM app.obligations o LEFT JOIN app.repayment_schedules s ON s.obligation_id=o.id LEFT JOIN app.schedule_items i ON i.schedule_id=s.id
 GROUP BY o.id HAVING count(i.id)=0 OR o.outstanding_kobo<>COALESCE(SUM(i.principal_due_kobo-i.allocated_kobo) FILTER(WHERE i.state<>'CANCELLED'),0)
UNION ALL
SELECT 'collection_payment',a.id::text,a.succeeded_amount_kobo,COALESCE(p.amount_kobo,0)
 FROM app.collection_attempts a LEFT JOIN app.payments p ON p.idempotency_key='collection-attempt:'||a.id::text
 WHERE a.succeeded_amount_kobo<>COALESCE(p.amount_kobo,0)
 OR (p.id IS NOT NULL AND (p.obligation_id<>a.obligation_id OR p.provider<>a.provider OR p.provider_reference IS DISTINCT FROM a.provider_collection_id))
UNION ALL
SELECT 'settlement',s.id::text,s.gross_amount_kobo,s.fee_amount_kobo::numeric+s.net_amount_kobo
 FROM app.settlement_events s LEFT JOIN app.payments p ON p.id=s.payment_id
 WHERE s.gross_amount_kobo<>s.fee_amount_kobo::numeric+s.net_amount_kobo OR p.id IS NULL
 OR p.amount_kobo<>s.gross_amount_kobo OR p.supplier_organization_id<>s.supplier_organization_id OR p.provider<>s.provider
 OR (s.expected_at<now() AND s.actual_at IS NULL)
UNION ALL
SELECT 'settlement_missing',a.id::text,a.succeeded_amount_kobo,0
 FROM app.collection_attempts a WHERE a.succeeded_amount_kobo>0 AND lower(COALESCE(a.settlement_state,'')) IN ('settled','completed','success')
 AND NOT EXISTS(SELECT 1 FROM app.settlement_events s WHERE s.provider=a.provider AND s.provider_settlement_reference=a.settlement_reference AND s.actual_at IS NOT NULL);

CREATE TABLE app.financial_review_cases (
 id uuid PRIMARY KEY DEFAULT uuidv7(), kind text NOT NULL, target_id text NOT NULL,
 expected numeric NOT NULL, actual numeric NOT NULL,
 owner_id uuid REFERENCES app.users(id), state text NOT NULL DEFAULT 'OPEN' CHECK(state IN ('OPEN','RESOLVED')),
 first_seen_at timestamptz NOT NULL DEFAULT now(), last_seen_at timestamptz NOT NULL DEFAULT now(),
 resolved_at timestamptz, UNIQUE(kind,target_id)
);
CREATE TABLE app.financial_review_events (
 id uuid PRIMARY KEY DEFAULT uuidv7(), case_id uuid NOT NULL REFERENCES app.financial_review_cases(id),
 actor_id uuid REFERENCES app.users(id), action text NOT NULL, reason text NOT NULL,
 occurred_at timestamptz NOT NULL DEFAULT now()
);
ALTER TABLE app.financial_review_cases ENABLE ROW LEVEL SECURITY;
ALTER TABLE app.financial_review_events ENABLE ROW LEVEL SECURITY;
CREATE POLICY financial_review_cases_runtime ON app.financial_review_cases USING(current_user IN ('kredit_app','kredit_worker')) WITH CHECK(current_user IN ('kredit_app','kredit_worker'));
CREATE POLICY financial_review_events_runtime ON app.financial_review_events USING(current_user IN ('kredit_app','kredit_worker')) WITH CHECK(current_user IN ('kredit_app','kredit_worker'));
CREATE TRIGGER financial_review_events_immutable BEFORE UPDATE OR DELETE ON app.financial_review_events FOR EACH ROW EXECUTE FUNCTION app.reject_operations_command_mutation();
-- +goose Down
DROP TABLE app.financial_review_events,app.financial_review_cases;
DROP VIEW app.financial_discrepancies;
