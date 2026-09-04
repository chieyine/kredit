-- +goose Up
-- Reversal restores debt; it is not a second incoming payment.
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION app.guard_manual_payment_reservations() RETURNS trigger LANGUAGE plpgsql SECURITY DEFINER SET search_path=app,pg_catalog AS $$
DECLARE remaining BIGINT; held BIGINT; obligation_currency TEXT;
BEGIN
 IF NEW.reversal_of IS NOT NULL THEN
   IF NOT EXISTS(SELECT 1 FROM app.payments p WHERE p.id=NEW.reversal_of AND p.reversal_of IS NULL AND p.state='reversed' AND p.obligation_id=NEW.obligation_id AND p.amount_kobo=NEW.amount_kobo AND p.currency=NEW.currency AND NEW.state='reversed') THEN RAISE EXCEPTION 'invalid payment reversal'; END IF;
   RETURN NEW;
 END IF;
 IF EXISTS(SELECT 1 FROM app.payments WHERE idempotency_key=NEW.idempotency_key) THEN RETURN NEW; END IF;
 SELECT outstanding_kobo,currency INTO remaining,obligation_currency FROM app.obligations WHERE id=NEW.obligation_id FOR UPDATE;
 IF NEW.currency<>obligation_currency THEN RAISE EXCEPTION 'payment currency differs from obligation'; END IF;
 IF NEW.source_type='kredit_collection' THEN RETURN NEW; END IF;
 SELECT COALESCE(SUM(reserved_amount_kobo),0) INTO held FROM app.collection_reservations WHERE obligation_id=NEW.obligation_id AND state IN ('PROCESSING','COMPLETED');
 IF NEW.amount_kobo>remaining-held THEN RAISE EXCEPTION 'payment conflicts with an unresolved debit; reconcile first'; END IF;
 RETURN NEW;
END $$;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION app.guard_manual_payment_reservations() RETURNS trigger LANGUAGE plpgsql SECURITY DEFINER SET search_path=app,pg_catalog AS $$
DECLARE remaining BIGINT; held BIGINT;
BEGIN
 IF NEW.source_type='kredit_collection' THEN RETURN NEW; END IF;
 IF EXISTS(SELECT 1 FROM app.payments WHERE idempotency_key=NEW.idempotency_key) THEN RETURN NEW; END IF;
 SELECT outstanding_kobo INTO remaining FROM app.obligations WHERE id=NEW.obligation_id FOR UPDATE;
 SELECT COALESCE(SUM(reserved_amount_kobo),0) INTO held FROM app.collection_reservations WHERE obligation_id=NEW.obligation_id AND state IN ('PROCESSING','COMPLETED');
 IF NEW.amount_kobo>remaining-held THEN RAISE EXCEPTION 'payment conflicts with an unresolved debit; reconcile first'; END IF;
 RETURN NEW;
END $$;
-- +goose StatementEnd
