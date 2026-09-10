-- +goose Up
-- A verified receipt token authorizes this limited projection. It must not
-- require a signed-in tenant or grant access to the full payment row.
-- +goose StatementBegin
CREATE FUNCTION app.public_payment_receipt(payment_id uuid)
RETURNS TABLE(reference uuid, amount_kobo bigint, currency text, source_type text,
              state text, paid_at timestamptz, recognized_at timestamptz)
LANGUAGE sql STABLE SECURITY DEFINER SET search_path=pg_catalog,app AS $$
  SELECT p.id,p.amount_kobo,p.currency,p.source_type,p.state,p.paid_at,p.recognized_at
  FROM app.payments p WHERE p.id=payment_id;
$$;
-- +goose StatementEnd
REVOKE ALL ON FUNCTION app.public_payment_receipt(uuid) FROM PUBLIC;
-- +goose StatementBegin
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname='kredit_app') THEN
    GRANT EXECUTE ON FUNCTION app.public_payment_receipt(uuid) TO kredit_app;
  END IF;
END;
$$;
-- +goose StatementEnd

-- +goose Down
DROP FUNCTION app.public_payment_receipt(uuid);
