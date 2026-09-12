-- +goose Up
-- Historical invalid references remain identifiable for review; every new
-- reference is enforced immediately, without deleting old evidence.
ALTER TABLE app.dispute_evidence ADD CONSTRAINT dispute_evidence_document_fk FOREIGN KEY(document_id) REFERENCES app.documents(id) NOT VALID;
-- +goose StatementBegin
CREATE FUNCTION app.guard_dispute_document_evidence() RETURNS trigger LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,app AS $$
DECLARE item app.disputes; document app.documents;
BEGIN
 SELECT * INTO item FROM app.disputes WHERE id=NEW.dispute_id FOR SHARE;
 IF NOT FOUND OR item.state IN ('RESOLVED','WITHDRAWN') OR NEW.submitted_by IS DISTINCT FROM app.current_user_id() THEN RAISE EXCEPTION 'dispute is not open to this submitter'; END IF;
 IF item.buyer_user_id<>NEW.submitted_by AND NOT EXISTS(SELECT 1 FROM app.memberships WHERE user_id=NEW.submitted_by AND organization_id=item.supplier_organization_id AND status='active' AND role IN ('owner','finance','collections')) THEN RAISE EXCEPTION 'submitter is not a party to this dispute'; END IF;
 IF NEW.document_id IS NULL THEN RETURN NEW; END IF;
 SELECT * INTO document FROM app.documents WHERE id=NEW.document_id FOR SHARE;
 IF NOT FOUND OR document.uploaded_by<>NEW.submitted_by OR document.organization_id IS DISTINCT FROM item.supplier_organization_id OR document.purpose<>'dispute_'||item.id::text OR document.upload_completed_at IS NULL OR document.scan_state<>'CLEAN' THEN RAISE EXCEPTION 'document does not belong to this dispute or has not passed its safety check'; END IF;
 RETURN NEW;
END $$;
-- +goose StatementEnd
CREATE TRIGGER dispute_document_evidence_guard BEFORE INSERT ON app.dispute_evidence FOR EACH ROW EXECUTE FUNCTION app.guard_dispute_document_evidence();
-- +goose Down
-- +goose StatementBegin
DO $$ BEGIN RAISE EXCEPTION 'Dispute evidence requires forward recovery'; END $$;
-- +goose StatementEnd
