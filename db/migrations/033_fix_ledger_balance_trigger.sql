-- +goose Up

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION ledger.assert_balanced_transaction() RETURNS trigger
LANGUAGE plpgsql AS $$
DECLARE
    v_transaction_id uuid := COALESCE(NEW.transaction_id, OLD.transaction_id);
    v_debits bigint;
    v_credits bigint;
BEGIN
    SELECT COALESCE(SUM(p.debit_kobo), 0), COALESCE(SUM(p.credit_kobo), 0)
      INTO v_debits, v_credits
      FROM ledger.postings p
     WHERE p.transaction_id = v_transaction_id;
    IF v_debits = 0 OR v_debits <> v_credits THEN
        RAISE EXCEPTION 'unbalanced ledger transaction %: debits %, credits %', v_transaction_id, v_debits, v_credits;
    END IF;
    IF TG_OP = 'DELETE' THEN
        RETURN OLD;
    END IF;
    RETURN NEW;
END;
$$;
-- +goose StatementEnd

-- +goose Down
-- The corrected function is intentionally retained during rollback because
-- restoring the ambiguous implementation would break every financial write.
