-- +goose Up

-- Existing policies in Milestones 3-9 use app.current_user_id(). Keep the
-- helper in the latest migration as well so upgrades from an older database
-- repair the function before any application traffic reaches the schema.
CREATE OR REPLACE FUNCTION app.current_user_id() RETURNS uuid
LANGUAGE sql STABLE
AS $$ SELECT NULLIF(current_setting('app.current_user_id', true), '')::uuid $$;

CREATE OR REPLACE FUNCTION app.current_organization_id() RETURNS uuid
LANGUAGE sql STABLE
AS $$ SELECT NULLIF(current_setting('app.current_organization_id', true), '')::uuid $$;

-- Runtime identifiers are opaque strings (for example payment-...); the
-- ledger must not force them through a UUID cast.
ALTER TABLE ledger.transactions
    ALTER COLUMN reference_id TYPE TEXT USING reference_id::text;

INSERT INTO ledger.accounts (code, name, normal_balance) VALUES
    ('TRADE_RECEIVABLE_CONTROL', 'Trade receivable control', 'debit'),
    ('PRINCIPAL_ORIGINATED_CONTROL', 'Principal originated control', 'credit'),
    ('SUPPLIER_FEE_RECEIVABLE', 'Supplier fee receivable', 'debit'),
    ('PLATFORM_SERVICE_REVENUE', 'Platform service revenue', 'credit'),
    ('VOLUNTARY_SETTLEMENT_CONTROL', 'Voluntary settlement control', 'debit'),
    ('COLLECTION_SETTLEMENT_CONTROL', 'Collection settlement control', 'debit'),
    ('PLATFORM_COLLECTION_REVENUE', 'Platform collection revenue', 'credit'),
    ('RETURNS_ADJUSTMENT_CONTROL', 'Returns adjustment control', 'debit'),
    ('WRITE_OFF_CONTROL', 'Write-off control', 'debit')
ON CONFLICT (code) DO NOTHING;

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION ledger.assert_balanced_transaction() RETURNS trigger
LANGUAGE plpgsql AS $$
DECLARE
    transaction_id uuid := COALESCE(NEW.transaction_id, OLD.transaction_id);
    debits bigint;
    credits bigint;
BEGIN
    SELECT COALESCE(SUM(debit_kobo), 0), COALESCE(SUM(credit_kobo), 0)
      INTO debits, credits
      FROM ledger.postings
     WHERE ledger.postings.transaction_id = transaction_id;
    IF debits = 0 OR debits <> credits THEN
        RAISE EXCEPTION 'unbalanced ledger transaction %: debits %, credits %', transaction_id, debits, credits;
    END IF;
    IF TG_OP = 'DELETE' THEN
        RETURN OLD;
    END IF;
    RETURN NEW;
END;
$$;
-- +goose StatementEnd

DROP TRIGGER IF EXISTS postings_balanced ON ledger.postings;
CREATE CONSTRAINT TRIGGER postings_balanced
AFTER INSERT OR UPDATE OR DELETE ON ledger.postings
DEFERRABLE INITIALLY DEFERRED
FOR EACH ROW EXECUTE FUNCTION ledger.assert_balanced_transaction();

CREATE TABLE IF NOT EXISTS app.outbox_events (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    aggregate_type text NOT NULL,
    aggregate_id text NOT NULL,
    event_type text NOT NULL,
    payload jsonb NOT NULL,
    idempotency_key text NOT NULL UNIQUE,
    state text NOT NULL DEFAULT 'pending' CHECK (state IN ('pending', 'processing', 'published', 'failed')),
    attempts integer NOT NULL DEFAULT 0 CHECK (attempts >= 0),
    available_at timestamptz NOT NULL DEFAULT now(),
    last_error text,
    created_at timestamptz NOT NULL DEFAULT now(),
    published_at timestamptz
);
CREATE INDEX IF NOT EXISTS outbox_pending_idx ON app.outbox_events (available_at, created_at) WHERE state IN ('pending', 'failed');

-- +goose Down
DROP TABLE IF EXISTS app.outbox_events;
DROP TRIGGER IF EXISTS postings_balanced ON ledger.postings;
DROP FUNCTION IF EXISTS ledger.assert_balanced_transaction();
