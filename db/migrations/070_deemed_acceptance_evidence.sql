-- +goose Up
-- Deemed acceptance is the only path in this system where buyer silence creates
-- an activated, collectable obligation. Application code decides when to try it;
-- this trigger decides whether the evidence exists, so a bug, a direct database
-- change, or a future caller cannot convert silence into a debit without it.
--
-- It mirrors app.guard_collection_notice (migration 057): a real delivery
-- receipt for a notice we can name, not merely a queued or accepted send.
--
-- Three conditions must hold, and all three are checked here:
--   1. goods were actually released;
--   2. this buyer business has answered one of our notices before, so the
--      obligation is never the buyer's first;
--   3. a goods-release notice was delivered and has been with the buyer for the
--      full waiting period.
--
-- Explicit receipt confirmations are untouched: a buyer who answers is always
-- allowed to answer.

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION app.guard_deemed_acceptance() RETURNS trigger
LANGUAGE plpgsql
AS $$
DECLARE
  minimum_seconds bigint;
  release_id uuid;
  buyer_business uuid;
BEGIN
  IF NEW.issue_reason IS DISTINCT FROM 'deemed_acceptance_auto_activated' THEN
    RETURN NEW;
  END IF;

  -- The aggregate is re-persisted on every later mutation, replaying its
  -- receipts through ON CONFLICT DO NOTHING. Re-validating an activation that
  -- already happened would let a subsequent change to the waiting period block
  -- unrelated writes on a historic sale. Only genuinely new rows are judged.
  IF EXISTS (SELECT 1 FROM app.receipt_confirmations WHERE id = NEW.id) THEN
    RETURN NEW;
  END IF;

  -- Unset means the safe default, not "disabled". A deployment that has not
  -- thought about this setting gets the protective behaviour.
  minimum_seconds := COALESCE(NULLIF(current_setting('app.deemed_acceptance_min_seconds', true), ''), '259200')::bigint;

  SELECT buyer_business_id INTO buyer_business
  FROM app.credit_requests
  WHERE id = NEW.credit_request_id;
  IF buyer_business IS NULL THEN
    RAISE EXCEPTION 'deemed acceptance requires an identified buyer business';
  END IF;

  SELECT id INTO release_id
  FROM app.goods_releases
  WHERE credit_request_id = NEW.credit_request_id
  ORDER BY released_at DESC
  LIMIT 1;
  IF release_id IS NULL THEN
    RAISE EXCEPTION 'deemed acceptance requires a recorded goods release';
  END IF;

  IF NOT EXISTS (
    SELECT 1
    FROM app.receipt_confirmations rc
    JOIN app.credit_requests c ON c.id = rc.credit_request_id
    WHERE c.buyer_business_id = buyer_business
      AND rc.credit_request_id <> NEW.credit_request_id
      AND rc.issue_reason IS DISTINCT FROM 'deemed_acceptance_auto_activated'
  ) THEN
    RAISE EXCEPTION 'deemed acceptance is not available for a buyer''s first trade credit';
  END IF;

  IF minimum_seconds > 0 AND NOT EXISTS (
    SELECT 1
    FROM app.outbox_events e
    JOIN app.notifications n ON n.event_reference = 'outbox:' || e.id::text
    JOIN app.notification_delivery_receipts receipt ON receipt.notification_id = n.id
    WHERE e.idempotency_key = 'goods-release-notification:' || release_id::text
      AND n.state IN ('delivered','read')
      AND receipt.received_at <= now() - make_interval(secs => minimum_seconds::double precision)
  ) THEN
    RAISE EXCEPTION 'deemed acceptance requires a confirmed goods-release notice delivered for the full waiting period';
  END IF;

  RETURN NEW;
END $$;
-- +goose StatementEnd

CREATE TRIGGER deemed_acceptance_guard
    BEFORE INSERT ON app.receipt_confirmations
    FOR EACH ROW EXECUTE FUNCTION app.guard_deemed_acceptance();

-- +goose Down
DROP TRIGGER IF EXISTS deemed_acceptance_guard ON app.receipt_confirmations;
DROP FUNCTION IF EXISTS app.guard_deemed_acceptance();
