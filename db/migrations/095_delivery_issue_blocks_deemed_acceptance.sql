-- +goose Up
-- A buyer who reported a delivery issue has answered; silence must not later
-- override that answer. Explicit buyer confirmation remains available.
-- +goose StatementBegin
CREATE FUNCTION app.guard_deemed_acceptance_delivery_issue() RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
  IF NEW.issue_reason = 'deemed_acceptance_auto_activated'
     AND NOT EXISTS (SELECT 1 FROM app.receipt_confirmations WHERE id = NEW.id)
     AND EXISTS (
       SELECT 1 FROM app.receipt_confirmations
       WHERE credit_request_id = NEW.credit_request_id AND state = 'issue_raised'
     ) THEN
    RAISE EXCEPTION 'delivery issue requires explicit buyer confirmation';
  END IF;
  RETURN NEW;
END $$;
-- +goose StatementEnd

CREATE TRIGGER deemed_acceptance_delivery_issue_guard
  BEFORE INSERT ON app.receipt_confirmations
  FOR EACH ROW EXECUTE FUNCTION app.guard_deemed_acceptance_delivery_issue();

-- +goose Down
DROP TRIGGER IF EXISTS deemed_acceptance_delivery_issue_guard ON app.receipt_confirmations;
DROP FUNCTION IF EXISTS app.guard_deemed_acceptance_delivery_issue();
