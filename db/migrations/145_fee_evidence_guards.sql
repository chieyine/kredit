-- +goose Up
-- +goose StatementBegin
CREATE FUNCTION app.guard_fee_authorization() RETURNS trigger LANGUAGE plpgsql SET search_path=pg_catalog,app AS $$
BEGIN
 IF TG_OP='DELETE' THEN RAISE EXCEPTION 'fee permission history is immutable';END IF;
 IF (to_jsonb(NEW)-ARRAY['state','customer_reference','mandate_reference','authorization_url','approved_by','approved_at','updated_at']) IS DISTINCT FROM (to_jsonb(OLD)-ARRAY['state','customer_reference','mandate_reference','authorization_url','approved_by','approved_at','updated_at']) THEN RAISE EXCEPTION 'fee permission consent and account are immutable';END IF;
 IF OLD.customer_reference<>'' AND NEW.customer_reference IS DISTINCT FROM OLD.customer_reference THEN RAISE EXCEPTION 'fee customer is immutable';END IF;
 IF OLD.mandate_reference<>'' AND NEW.mandate_reference IS DISTINCT FROM OLD.mandate_reference THEN RAISE EXCEPTION 'fee mandate is immutable';END IF;
 IF (NEW.approved_by IS DISTINCT FROM OLD.approved_by OR NEW.approved_at IS DISTINCT FROM OLD.approved_at) AND NEW.approved_by IS NOT NULL AND NOT app.has_admin_role(app.current_user_id(),ARRAY['platform_owner']) THEN RAISE EXCEPTION 'super-admin fee approval required';END IF;
 RETURN NEW;
END $$;
-- +goose StatementEnd
CREATE TRIGGER fee_authorization_immutable BEFORE UPDATE OR DELETE ON app.fee_authorizations FOR EACH ROW EXECUTE FUNCTION app.guard_fee_authorization();
-- +goose StatementBegin
CREATE FUNCTION app.guard_fee_debit() RETURNS trigger LANGUAGE plpgsql SET search_path=pg_catalog,app AS $$
BEGIN
 IF TG_OP='DELETE' THEN RAISE EXCEPTION 'fee debit history is immutable';END IF;
 IF (to_jsonb(NEW)-ARRAY['state','checked_at','review_required']) IS DISTINCT FROM (to_jsonb(OLD)-ARRAY['state','checked_at','review_required']) THEN RAISE EXCEPTION 'fee debit request is immutable';END IF;
 IF OLD.state<>NEW.state AND NOT (OLD.state='pending' AND NEW.state IN ('succeeded','failed') OR OLD.state='succeeded' AND NEW.state='reversed' AND app.has_admin_role(app.current_user_id(),ARRAY['platform_owner'])) THEN RAISE EXCEPTION 'invalid fee debit transition';END IF;
 RETURN NEW;
END $$;
-- +goose StatementEnd
CREATE TRIGGER fee_debit_immutable BEFORE UPDATE OR DELETE ON app.fee_debits FOR EACH ROW EXECUTE FUNCTION app.guard_fee_debit();
-- +goose StatementBegin
DO $$ BEGIN
 IF EXISTS(SELECT 1 FROM pg_roles WHERE rolname='kredit_app') THEN
 REVOKE UPDATE,DELETE ON app.split_fee_allocations,app.fee_bank_receipts,app.collection_settlement_routes,app.seller_settlement_receipts FROM kredit_app;
 REVOKE INSERT,UPDATE,DELETE ON app.native_identity_history FROM kredit_app;
 REVOKE DELETE ON app.fee_authorizations,app.fee_debits FROM kredit_app;
 END IF;
 IF EXISTS(SELECT 1 FROM pg_roles WHERE rolname='kredit_worker') THEN
 REVOKE UPDATE,DELETE ON app.split_fee_allocations,app.collection_settlement_routes FROM kredit_worker;
 REVOKE INSERT,UPDATE,DELETE ON app.fee_bank_receipts,app.seller_settlement_receipts,app.native_identity_history FROM kredit_worker;
 REVOKE INSERT,UPDATE,DELETE ON app.fee_authorizations FROM kredit_worker;
 GRANT UPDATE(approved_at) ON app.fee_authorizations TO kredit_worker;
 REVOKE DELETE ON app.fee_debits FROM kredit_worker;
 END IF;
END $$;
-- +goose StatementEnd
-- +goose Down
-- +goose StatementBegin
DO $$ BEGIN RAISE EXCEPTION 'Fee evidence guards require forward recovery';END $$;
-- +goose StatementEnd
