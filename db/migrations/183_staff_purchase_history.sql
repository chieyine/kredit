-- +goose Up
-- Read-only access follows the buying business's current permission. It never
-- authorizes payment initiation, payment confirmation or dispute decisions.
-- +goose StatementBegin
CREATE FUNCTION app.purchase_obligation_read(debt_id uuid) RETURNS boolean
LANGUAGE sql STABLE SECURITY DEFINER SET search_path=pg_catalog,app AS $$
 SELECT COALESCE((SELECT app.can_purchase(buyer_business_id) FROM app.obligations WHERE id=debt_id),false);
$$;
REVOKE ALL ON FUNCTION app.purchase_obligation_read(uuid) FROM PUBLIC;
DO $$ DECLARE tab text; r text; BEGIN
 FOREACH tab IN ARRAY ARRAY['payments','fees','disputes','repayment_schedules'] LOOP
  EXECUTE format('CREATE POLICY delegated_purchase_history_read ON app.%I FOR SELECT USING(app.purchase_obligation_read(obligation_id::uuid))',tab);
 END LOOP;
 FOREACH r IN ARRAY ARRAY['kredit_app','kredit_worker'] LOOP
  IF EXISTS(SELECT 1 FROM pg_roles WHERE rolname=r) THEN EXECUTE format('GRANT EXECUTE ON FUNCTION app.purchase_obligation_read(uuid) TO %I',r); END IF;
 END LOOP;
END $$;
-- +goose StatementEnd
CREATE POLICY delegated_schedule_items_read ON app.schedule_items FOR SELECT
USING(EXISTS(SELECT 1 FROM app.repayment_schedules s WHERE s.id=schedule_id AND app.purchase_obligation_read(s.obligation_id)));
-- +goose Down
-- +goose StatementBegin
DO $$ BEGIN RAISE EXCEPTION 'Staff history scope requires forward recovery'; END $$;
-- +goose StatementEnd
