-- +goose Up
-- Balances, including reversals after capacity has been reused, remain truthful.
-- New draws are blocked by zero availability when a line is over its limit.
-- +goose StatementBegin
DO $$
DECLARE c record;
BEGIN
 FOR c IN SELECT conname FROM pg_constraint WHERE conrelid='app.trade_lines'::regclass AND contype='c' AND pg_get_constraintdef(oid) LIKE '%available_limit_kobo =%' LOOP
  EXECUTE format('ALTER TABLE app.trade_lines DROP CONSTRAINT %I',c.conname);
 END LOOP;
END $$;
-- +goose StatementEnd
ALTER TABLE app.trade_lines ADD CONSTRAINT trade_line_available_balance CHECK (available_limit_kobo=GREATEST(0,approved_limit_kobo-current_exposure_kobo-reserved_pending_kobo));

-- +goose StatementBegin
CREATE FUNCTION app.sync_drawdown_exposure() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE line_id uuid;
BEGIN
 IF NEW.outstanding_kobo=OLD.outstanding_kobo THEN RETURN NEW; END IF;
 SELECT trade_line_id INTO line_id FROM app.drawdowns WHERE obligation_id=NEW.id AND state='ACTIVATED';
 IF line_id IS NOT NULL THEN
  UPDATE app.trade_lines SET
   current_exposure_kobo=current_exposure_kobo+NEW.outstanding_kobo-OLD.outstanding_kobo,
   available_limit_kobo=GREATEST(0,approved_limit_kobo-(current_exposure_kobo+NEW.outstanding_kobo-OLD.outstanding_kobo)-reserved_pending_kobo),
   version=version+1,updated_at=now()
  WHERE id=line_id;
 END IF;
 RETURN NEW;
END $$;
-- +goose StatementEnd
CREATE TRIGGER obligation_drawdown_exposure AFTER UPDATE OF outstanding_kobo ON app.obligations FOR EACH ROW EXECUTE FUNCTION app.sync_drawdown_exposure();
-- Repair existing exposure using the authoritative obligations.
WITH balances AS (
 SELECT l.id,COALESCE(sum(o.outstanding_kobo),0) AS amount FROM app.trade_lines l
 LEFT JOIN app.drawdowns d ON d.trade_line_id=l.id AND d.state='ACTIVATED'
 LEFT JOIN app.obligations o ON o.id=d.obligation_id GROUP BY l.id
)
UPDATE app.trade_lines l SET current_exposure_kobo=b.amount,
 available_limit_kobo=GREATEST(0,l.approved_limit_kobo-b.amount-l.reserved_pending_kobo),version=l.version+1,updated_at=now()
FROM balances b WHERE l.id=b.id AND l.current_exposure_kobo<>b.amount;

ALTER TABLE app.notifications ADD COLUMN supplier_organization_id uuid REFERENCES app.organizations(id);
ALTER TABLE app.notifications ADD COLUMN priority text NOT NULL DEFAULT 'critical' CHECK(priority IN('critical','routine'));

-- +goose Down
-- Exposure repair is intentionally not reversed: prior values were incorrect.
ALTER TABLE app.notifications DROP COLUMN priority, DROP COLUMN supplier_organization_id;
DROP TRIGGER obligation_drawdown_exposure ON app.obligations;
DROP FUNCTION app.sync_drawdown_exposure();
-- Keep the nonnegative-availability constraint: reverting it would reject
-- legitimate reversal-created over-limit balances.
