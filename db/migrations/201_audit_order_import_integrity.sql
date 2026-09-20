-- +goose Up
-- Forward-only repair of the new order/import boundaries. Stop order writers
-- during this migration and deploy the matching API before enabling writes:
-- shipment quantity/status transitions are now owned by database triggers.
-- Existing inconsistent evidence is deliberately NOT deleted or silently fixed.

ALTER TABLE app.credit_requests ADD CONSTRAINT audit_credit_supplier_identity UNIQUE(id,supplier_organization_id);
ALTER TABLE app.obligations ADD CONSTRAINT audit_obligation_order_identity UNIQUE(id,credit_request_id,supplier_organization_id);
ALTER TABLE app.order_line_items ADD CONSTRAINT audit_order_line_identity UNIQUE(id,order_id);
ALTER TABLE app.order_shipments ADD CONSTRAINT audit_shipment_order_identity UNIQUE(id,order_id);
ALTER TABLE app.order_shipments ADD CONSTRAINT audit_shipment_supplier_fk
  FOREIGN KEY(order_id,supplier_organization_id) REFERENCES app.credit_requests(id,supplier_organization_id);
ALTER TABLE app.order_credit_notes ADD CONSTRAINT audit_note_supplier_fk
  FOREIGN KEY(order_id,supplier_organization_id) REFERENCES app.credit_requests(id,supplier_organization_id);
ALTER TABLE app.order_credit_notes ADD CONSTRAINT audit_note_obligation_fk
  FOREIGN KEY(obligation_id,order_id,supplier_organization_id) REFERENCES app.obligations(id,credit_request_id,supplier_organization_id);
ALTER TABLE app.order_delivery_receipts ADD CONSTRAINT audit_receipt_order_fk
  FOREIGN KEY(shipment_id,order_id) REFERENCES app.order_shipments(id,order_id);
ALTER TABLE app.order_delivery_receipts ADD CONSTRAINT audit_receipt_once UNIQUE(shipment_id);
ALTER TABLE app.order_shipment_items ADD COLUMN order_id uuid;
UPDATE app.order_shipment_items i SET order_id=s.order_id FROM app.order_shipments s WHERE s.id=i.shipment_id;
ALTER TABLE app.order_shipment_items ALTER COLUMN order_id SET NOT NULL;
ALTER TABLE app.order_shipment_items ADD CONSTRAINT audit_shipment_item_parent_fk
  FOREIGN KEY(shipment_id,order_id) REFERENCES app.order_shipments(id,order_id);
ALTER TABLE app.order_shipment_items ADD CONSTRAINT audit_shipment_item_order_fk
  FOREIGN KEY(line_item_id,order_id) REFERENCES app.order_line_items(id,order_id);
CREATE INDEX audit_order_shipments_order_idx ON app.order_shipments(order_id,dispatched_at,id);
CREATE INDEX audit_order_credit_notes_order_idx ON app.order_credit_notes(order_id,created_at,id);
CREATE INDEX audit_order_line_items_order_idx ON app.order_line_items(order_id,created_at,id);
CREATE INDEX audit_order_shipment_items_line_idx ON app.order_shipment_items(line_item_id,order_id);

-- Ordinary invoker rights deliberately preserve parent RLS and branch limits.
-- +goose StatementBegin
CREATE FUNCTION app.order_supplier_authorized(order_ref uuid, allowed_roles text[]) RETURNS boolean
LANGUAGE sql STABLE SECURITY INVOKER SET search_path=pg_catalog,app,pg_temp AS $$
 SELECT EXISTS(SELECT 1 FROM app.credit_requests cr
 JOIN app.memberships m ON m.organization_id=cr.supplier_organization_id AND m.user_id=app.current_user_id()
 JOIN app.users u ON u.id=m.user_id JOIN app.organizations o ON o.id=m.organization_id
 WHERE cr.id=order_ref AND cr.supplier_organization_id=app.current_organization_id()
 AND m.status='active' AND m.role=ANY(allowed_roles) AND u.status='active' AND o.status<>'suspended');
$$;
CREATE FUNCTION app.order_evidence_visible(order_ref uuid) RETURNS boolean
LANGUAGE sql STABLE SECURITY INVOKER SET search_path=pg_catalog,app,pg_temp AS $$
 SELECT EXISTS(SELECT 1 FROM app.credit_requests cr WHERE cr.id=order_ref AND
 (app.order_supplier_authorized(cr.id,ARRAY['owner','administrator','sales','finance','viewer'])
 OR app.can_purchase(cr.buyer_business_id,'read')));
$$;
DO $$ DECLARE tab text; BEGIN
 FOREACH tab IN ARRAY ARRAY['order_line_items','order_shipments','order_shipment_items','order_delivery_receipts','order_credit_notes'] LOOP
  EXECUTE format('CREATE POLICY audit_order_parent_boundary ON app.%I AS RESTRICTIVE FOR ALL USING(app.order_evidence_visible(order_id)) WITH CHECK(app.order_evidence_visible(order_id))',tab);
 END LOOP;
END $$;
-- +goose StatementEnd

-- Replace broad ALL policies with distinct authorities for each mutation.
DROP POLICY line_items_write ON app.order_line_items;
CREATE POLICY audit_line_insert ON app.order_line_items FOR INSERT WITH CHECK(
 app.order_supplier_authorized(order_id,ARRAY['owner','administrator','sales']) AND fulfilled_quantity=0 AND returned_quantity=0);
DROP POLICY shipments_write ON app.order_shipments;
CREATE POLICY audit_shipment_insert ON app.order_shipments FOR INSERT WITH CHECK(
 app.order_supplier_authorized(order_id,ARRAY['owner','administrator','sales']) AND dispatched_by=app.current_user_id() AND status='in_transit');
DROP POLICY shipment_items_write ON app.order_shipment_items;
CREATE POLICY audit_shipment_item_insert ON app.order_shipment_items FOR INSERT WITH CHECK(
 app.order_supplier_authorized(order_id,ARRAY['owner','administrator','sales']));
DROP POLICY delivery_receipts_write ON app.order_delivery_receipts;
CREATE POLICY audit_receipt_insert ON app.order_delivery_receipts FOR INSERT WITH CHECK(
 received_by=app.current_user_id() AND signed_proof_hash ~ '^[0-9a-f]{64}$' AND EXISTS(
 SELECT 1 FROM app.credit_requests cr WHERE cr.id=order_id AND app.can_purchase(cr.buyer_business_id,'receive')));
DROP POLICY credit_notes_write ON app.order_credit_notes;
CREATE POLICY audit_note_insert ON app.order_credit_notes FOR INSERT WITH CHECK(
 app.order_supplier_authorized(order_id,ARRAY['owner','administrator','sales']) AND issued_by=app.current_user_id() AND status='draft');
CREATE POLICY audit_note_review ON app.order_credit_notes FOR UPDATE USING(
 app.order_supplier_authorized(order_id,ARRAY['owner','administrator','finance'])) WITH CHECK(
 app.order_supplier_authorized(order_id,ARRAY['owner','administrator','finance']));

-- Fulfilled quantity can move only through one successful shipment-item insert.
-- The guarded UPDATE locks the line and refuses both over-fulfilment and a
-- different order's line; a later failure rolls back the whole shipment.
-- +goose StatementBegin
CREATE FUNCTION app.apply_order_shipment_item() RETURNS trigger
LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,app,pg_temp AS $$
DECLARE changed bigint;
BEGIN
 PERFORM 1 FROM app.order_shipments WHERE id=NEW.shipment_id AND order_id=NEW.order_id AND status='in_transit' FOR UPDATE;
 IF NOT FOUND THEN RAISE EXCEPTION 'shipment is not available for dispatch' USING ERRCODE='23514'; END IF;
 UPDATE app.order_line_items SET fulfilled_quantity=fulfilled_quantity+NEW.quantity
 WHERE id=NEW.line_item_id AND order_id=NEW.order_id AND NEW.quantity<=quantity-fulfilled_quantity;
 GET DIAGNOSTICS changed=ROW_COUNT;
 IF changed<>1 THEN RAISE EXCEPTION 'shipment quantity or order is invalid' USING ERRCODE='23514'; END IF;
 RETURN NEW;
END $$;
CREATE FUNCTION app.complete_order_receipt() RETURNS trigger
LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,app,pg_temp AS $$
DECLARE profile uuid; changed bigint;
BEGIN
 IF NEW.received_by IS DISTINCT FROM app.current_user_id() THEN
  RAISE EXCEPTION 'receipt actor mismatch' USING ERRCODE='42501';
 END IF;
 SELECT buyer_business_id INTO profile FROM app.credit_requests WHERE id=NEW.order_id;
 IF NOT FOUND THEN RAISE EXCEPTION 'order is not available' USING ERRCODE='42501'; END IF;
 PERFORM app.lock_purchase_permission(profile,'receive',0);
 UPDATE app.order_shipments SET status='delivered'
 WHERE id=NEW.shipment_id AND order_id=NEW.order_id AND status='in_transit';
 GET DIAGNOSTICS changed=ROW_COUNT;
 IF changed<>1 THEN RAISE EXCEPTION 'shipment cannot receive this receipt' USING ERRCODE='23514'; END IF;
 RETURN NEW;
END $$;
-- These policies support FORCE RLS when the migration owner is not superuser.
-- Runtime roles do not own the functions and cannot create arbitrary triggers.
CREATE POLICY audit_quantity_trigger ON app.order_line_items FOR UPDATE USING(
 pg_trigger_depth()>0 AND current_user=pg_get_userbyid((SELECT proowner FROM pg_proc WHERE oid='app.apply_order_shipment_item()'::regprocedure)))
 WITH CHECK(pg_trigger_depth()>0 AND current_user=pg_get_userbyid((SELECT proowner FROM pg_proc WHERE oid='app.apply_order_shipment_item()'::regprocedure)));
CREATE POLICY audit_shipment_transition_trigger ON app.order_shipments FOR UPDATE USING(
 pg_trigger_depth()>0 AND current_user=pg_get_userbyid((SELECT proowner FROM pg_proc WHERE oid='app.complete_order_receipt()'::regprocedure)))
 WITH CHECK(pg_trigger_depth()>0 AND current_user=pg_get_userbyid((SELECT proowner FROM pg_proc WHERE oid='app.complete_order_receipt()'::regprocedure)));
CREATE TRIGGER audit_apply_shipment_item AFTER INSERT ON app.order_shipment_items FOR EACH ROW EXECUTE FUNCTION app.apply_order_shipment_item();
CREATE TRIGGER audit_complete_receipt AFTER INSERT ON app.order_delivery_receipts FOR EACH ROW EXECUTE FUNCTION app.complete_order_receipt();

CREATE FUNCTION app.guard_order_credit_note() RETURNS trigger
LANGUAGE plpgsql SECURITY INVOKER SET search_path=pg_catalog,app,pg_temp AS $$
DECLARE reviewer uuid;
BEGIN
 IF TG_OP='INSERT' THEN
  IF NEW.status<>'draft' OR NEW.issued_by IS DISTINCT FROM app.current_user_id()
   OR NEW.approved_by IS NOT NULL OR NEW.approved_at IS NOT NULL THEN
   RAISE EXCEPTION 'credit note must start as an unapproved draft' USING ERRCODE='23514';
  END IF;
  RETURN NEW;
 END IF;
 IF (NEW.id,NEW.order_id,NEW.obligation_id,NEW.supplier_organization_id,NEW.amount_kobo,NEW.reason,NEW.issued_by,NEW.created_at)
 IS DISTINCT FROM (OLD.id,OLD.order_id,OLD.obligation_id,OLD.supplier_organization_id,OLD.amount_kobo,OLD.reason,OLD.issued_by,OLD.created_at) THEN
  RAISE EXCEPTION 'credit note intent is immutable' USING ERRCODE='23514';
 END IF;
 IF OLD.status<>'draft' OR NEW.status<>'approved' OR NEW.approved_by IS DISTINCT FROM app.current_user_id()
 OR NEW.approved_by=NEW.issued_by OR NEW.approved_at IS NULL THEN
  RAISE EXCEPTION 'independent credit note review required' USING ERRCODE='42501';
 END IF;
 -- Lock current reviewer membership against concurrent revocation.
 SELECT m.user_id INTO reviewer FROM app.memberships m JOIN app.users u ON u.id=m.user_id
 JOIN app.organizations o ON o.id=m.organization_id
 WHERE m.organization_id=NEW.supplier_organization_id AND m.user_id=app.current_user_id()
 AND m.status='active' AND m.role IN ('owner','administrator','finance') AND u.status='active' AND o.status<>'suspended'
 FOR SHARE OF m,u,o;
 IF NOT FOUND THEN RAISE EXCEPTION 'current credit reviewer required' USING ERRCODE='42501'; END IF;
 RETURN NEW;
END $$;
CREATE TRIGGER audit_credit_note_intent BEFORE INSERT OR UPDATE ON app.order_credit_notes FOR EACH ROW EXECUTE FUNCTION app.guard_order_credit_note();

-- Review is a decision about immutable imported proposals, NOT debt creation.
CREATE FUNCTION app.guard_terms_import_review() RETURNS trigger
LANGUAGE plpgsql SECURITY INVOKER SET search_path=pg_catalog,app,pg_temp AS $$
DECLARE current_role text;
BEGIN
 SELECT m.role INTO current_role FROM app.memberships m JOIN app.users u ON u.id=m.user_id
 JOIN app.organizations o ON o.id=m.organization_id
 WHERE m.organization_id=NEW.organization_id AND m.user_id=app.current_user_id()
 AND m.status='active' AND u.status='active' AND o.status<>'suspended' FOR SHARE OF m,u,o;
 IF NOT FOUND OR NEW.organization_id IS DISTINCT FROM app.current_organization_id() OR NOT app.branch_scope_all(NEW.organization_id) THEN
  RAISE EXCEPTION 'current company-wide import authority required' USING ERRCODE='42501';
 END IF;
 IF TG_OP='INSERT' THEN
  IF current_role NOT IN ('owner','administrator','sales') OR NEW.uploaded_by IS DISTINCT FROM app.current_user_id()
  OR NEW.state<>'staged' OR NEW.approved_by IS NOT NULL OR NEW.cancelled_by IS NOT NULL
  OR NEW.valid_rows+NEW.invalid_rows<>NEW.total_rows OR NEW.valid_rows<0 OR NEW.invalid_rows<0 THEN
   RAISE EXCEPTION 'invalid staged import' USING ERRCODE='42501';
  END IF;
 ELSE
  IF (to_jsonb(NEW)-ARRAY['state','approved_by','approved_at','cancelled_by','cancelled_at'])
   IS DISTINCT FROM (to_jsonb(OLD)-ARRAY['state','approved_by','approved_at','cancelled_by','cancelled_at']) THEN
   RAISE EXCEPTION 'import source evidence is immutable' USING ERRCODE='23514';
  END IF;
  IF current_role NOT IN ('owner','administrator','finance') OR NEW.uploaded_by=app.current_user_id()
   OR OLD.state NOT IN ('staged','reviewing') OR NEW.state NOT IN ('approved','cancelled') THEN
   RAISE EXCEPTION 'independent import reviewer required' USING ERRCODE='42501';
  END IF;
  IF (NEW.state='approved' AND (NEW.approved_by IS DISTINCT FROM app.current_user_id() OR NEW.approved_at IS NULL OR NEW.cancelled_by IS NOT NULL))
   OR (NEW.state='cancelled' AND (NEW.cancelled_by IS DISTINCT FROM app.current_user_id() OR NEW.cancelled_at IS NULL OR NEW.approved_by IS NOT NULL)) THEN
   RAISE EXCEPTION 'review actor or evidence mismatch' USING ERRCODE='42501';
  END IF;
 END IF;
 RETURN NEW;
END $$;
CREATE TRIGGER audit_terms_review BEFORE INSERT OR UPDATE ON app.partner_terms_import_batches FOR EACH ROW EXECUTE FUNCTION app.guard_terms_import_review();
CREATE FUNCTION app.guard_terms_import_row() RETURNS trigger
LANGUAGE plpgsql SECURITY INVOKER SET search_path=pg_catalog,app,pg_temp AS $$
BEGIN
 PERFORM 1 FROM app.partner_terms_import_batches WHERE id=NEW.batch_id AND state='staged'
 AND uploaded_by=app.current_user_id() AND NEW.row_number<=total_rows FOR SHARE;
 IF NOT FOUND OR NEW.invitation_id IS NOT NULL THEN
  RAISE EXCEPTION 'import rows require the current uploader and an undecided batch' USING ERRCODE='42501';
 END IF;
 RETURN NEW;
END $$;
CREATE TRIGGER audit_terms_row_insert BEFORE INSERT ON app.partner_terms_import_rows FOR EACH ROW EXECUTE FUNCTION app.guard_terms_import_row();
-- +goose StatementEnd

REVOKE ALL ON FUNCTION app.order_supplier_authorized(uuid,text[]),app.order_evidence_visible(uuid),app.apply_order_shipment_item(),app.complete_order_receipt(),app.guard_order_credit_note(),app.guard_terms_import_review(),app.guard_terms_import_row() FROM PUBLIC;
GRANT EXECUTE ON FUNCTION app.order_supplier_authorized(uuid,text[]),app.order_evidence_visible(uuid) TO kredit_app;
REVOKE ALL ON app.order_line_items,app.order_shipments,app.order_shipment_items,app.order_delivery_receipts,app.order_credit_notes FROM kredit_app,kredit_worker;
GRANT SELECT,INSERT ON app.order_line_items,app.order_shipments,app.order_shipment_items,app.order_delivery_receipts,app.order_credit_notes TO kredit_app;
GRANT UPDATE(status,approved_by,approved_at) ON app.order_credit_notes TO kredit_app;
REVOKE ALL ON app.partner_terms_import_batches,app.partner_terms_import_rows FROM kredit_app,kredit_worker;
GRANT SELECT,INSERT ON app.partner_terms_import_batches,app.partner_terms_import_rows TO kredit_app;
GRANT UPDATE(state,approved_by,approved_at,cancelled_by,cancelled_at) ON app.partner_terms_import_batches TO kredit_app;

-- +goose Down
-- Reverting these guards would reopen financial evidence vulnerabilities and
-- requires coordinated application rollback, not a generic migrate down.
-- +goose StatementBegin
DO $$ BEGIN RAISE EXCEPTION '201 is forward-only: use an explicitly reviewed corrective migration'; END $$;
-- +goose StatementEnd
