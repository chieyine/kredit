-- +goose Up
ALTER TABLE app.credit_requests ADD COLUMN invoice_document_id uuid REFERENCES app.documents(id);
-- +goose StatementBegin
CREATE FUNCTION app.guard_credit_invoice() RETURNS trigger LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,app AS $$
DECLARE document app.documents;
BEGIN
 IF TG_OP='UPDATE' AND OLD.state<>'DRAFT' THEN
  IF NEW.invoice_document_id IS DISTINCT FROM OLD.invoice_document_id OR NEW.invoice_document_hash IS DISTINCT FROM OLD.invoice_document_hash THEN RAISE EXCEPTION 'accepted invoice is immutable'; END IF;
  RETURN NEW;
 END IF;
 IF NEW.invoice_document_id IS NULL THEN RETURN NEW; END IF;
 SELECT * INTO document FROM app.documents WHERE id=NEW.invoice_document_id FOR SHARE;
 IF NOT FOUND OR document.organization_id IS DISTINCT FROM NEW.supplier_organization_id OR document.purpose<>'credit_invoice' OR document.sha256 IS DISTINCT FROM NEW.invoice_document_hash OR document.upload_completed_at IS NULL OR document.scan_state IN ('REJECTED','QUARANTINED') THEN RAISE EXCEPTION 'invoice is outside this business or is incomplete'; END IF;
 IF NEW.state<>'DRAFT' AND document.scan_state<>'CLEAN' THEN RAISE EXCEPTION 'wait for the invoice safety check before sending'; END IF;
 RETURN NEW;
END $$;
-- +goose StatementEnd
CREATE TRIGGER credit_invoice_evidence BEFORE INSERT OR UPDATE ON app.credit_requests FOR EACH ROW EXECUTE FUNCTION app.guard_credit_invoice();

-- A supplier may inspect capacity only for a mandate attached to its obligation.
-- The aggregate includes all uses of that mandate without exposing those rows.
-- +goose StatementBegin
CREATE FUNCTION app.collection_mandate_capacity(p_obligation uuid) RETURNS bigint
LANGUAGE sql STABLE SECURITY DEFINER SET search_path=pg_catalog,app AS $$
 SELECT GREATEST(0,m.amount_ceiling_kobo-COALESCE((SELECT SUM(a.succeeded_amount_kobo) FROM app.collection_attempts a JOIN app.collection_reservations r ON r.id=a.reservation_id WHERE r.mandate_id=m.id),0)-COALESCE((SELECT SUM(r.reserved_amount_kobo) FROM app.collection_reservations r WHERE r.mandate_id=m.id AND r.obligation_id<>o.id AND r.state IN ('PROCESSING','COMPLETED')),0))::bigint
 FROM app.credit_requests c JOIN app.obligations o ON o.credit_request_id=c.id JOIN app.payment_mandates m ON m.id=c.mandate_id
 WHERE o.id=p_obligation AND o.supplier_organization_id=NULLIF(current_setting('app.current_organization_id',true),'')::uuid
 AND m.buyer_subject_type='business' AND m.buyer_subject_id=c.buyer_business_id
 AND (m.supplier_organization_id IS NULL OR m.supplier_organization_id=o.supplier_organization_id)
$$;
-- +goose StatementEnd
REVOKE ALL ON FUNCTION app.collection_mandate_capacity(uuid) FROM PUBLIC;
-- +goose StatementBegin
DO $$ BEGIN
 IF EXISTS(SELECT 1 FROM pg_roles WHERE rolname='kredit_app') THEN GRANT EXECUTE ON FUNCTION app.collection_mandate_capacity(uuid) TO kredit_app; END IF;
 IF EXISTS(SELECT 1 FROM pg_roles WHERE rolname='kredit_worker') THEN GRANT EXECUTE ON FUNCTION app.collection_mandate_capacity(uuid) TO kredit_worker; END IF;
END $$;
-- +goose StatementEnd
-- +goose Down
-- +goose StatementBegin
DO $$ BEGIN RAISE EXCEPTION 'Invoice evidence requires forward recovery'; END $$;
-- +goose StatementEnd
