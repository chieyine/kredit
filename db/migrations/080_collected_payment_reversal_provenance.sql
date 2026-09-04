-- +goose Up
-- A reversal restores an existing debt; it is not a newly collected payment.
-- Preserve strict matching to the original payment while allowing the controlled
-- reversal path after its collection attempt has already completed.
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION app.guard_collected_payment_provenance() RETURNS trigger
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = app, pg_catalog
AS $$
DECLARE
    attempt_id UUID;
BEGIN
    IF NEW.source_type <> 'kredit_collection' THEN
        RETURN NEW;
    END IF;
    IF NEW.reversal_of IS NOT NULL THEN
        IF NEW.state <> 'reversed' OR NOT EXISTS (
            SELECT 1 FROM app.payments p
            WHERE p.id = NEW.reversal_of
              AND p.reversal_of IS NULL AND p.state = 'reversed'
              AND p.source_type = NEW.source_type
              AND p.obligation_id = NEW.obligation_id
              AND p.buyer_user_id = NEW.buyer_user_id
              AND p.supplier_organization_id = NEW.supplier_organization_id
              AND p.amount_kobo = NEW.amount_kobo AND p.currency = NEW.currency
              AND p.provider IS NOT DISTINCT FROM NEW.provider
              AND p.collection_fee_kobo = NEW.collection_fee_kobo
              AND NEW.idempotency_key = 'reversal:' || p.id::text
        ) THEN
            RAISE EXCEPTION 'invalid collected payment reversal provenance';
        END IF;
        RETURN NEW;
    END IF;
    IF NEW.recorded_by_reference <> 'collection-worker'
       OR NEW.provider IS NULL
       OR NEW.provider_reference IS NULL
       OR NEW.idempotency_key !~ '^collection-attempt:[0-9a-fA-F-]{36}$' THEN
        RAISE EXCEPTION 'invalid collected payment provenance';
    END IF;
    attempt_id := substring(NEW.idempotency_key FROM 20)::uuid;
    IF NOT EXISTS (
        SELECT 1
        FROM app.collection_attempts a
        WHERE a.id = attempt_id
          AND a.obligation_id = NEW.obligation_id
          AND a.provider = NEW.provider
          AND (a.provider_collection_id IS NULL OR a.provider_collection_id = NEW.provider_reference)
          AND a.requested_amount_kobo >= NEW.amount_kobo
          AND a.state IN ('PENDING','SUBMITTED','UNKNOWN')
    ) THEN
        RAISE EXCEPTION 'collected payment is not backed by an active collection attempt';
    END IF;
    RETURN NEW;
END
$$;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION app.guard_collected_payment_provenance() RETURNS trigger
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = app, pg_catalog
AS $$
DECLARE
    attempt_id UUID;
BEGIN
    IF NEW.source_type <> 'kredit_collection' THEN
        RETURN NEW;
    END IF;
    IF NEW.recorded_by_reference <> 'collection-worker'
       OR NEW.provider IS NULL
       OR NEW.provider_reference IS NULL
       OR NEW.idempotency_key !~ '^collection-attempt:[0-9a-fA-F-]{36}$' THEN
        RAISE EXCEPTION 'invalid collected payment provenance';
    END IF;
    attempt_id := substring(NEW.idempotency_key FROM 20)::uuid;
    IF NOT EXISTS (
        SELECT 1
        FROM app.collection_attempts a
        WHERE a.id = attempt_id
          AND a.obligation_id = NEW.obligation_id
          AND a.provider = NEW.provider
          AND (a.provider_collection_id IS NULL OR a.provider_collection_id = NEW.provider_reference)
          AND a.requested_amount_kobo >= NEW.amount_kobo
          AND a.state IN ('PENDING','SUBMITTED','UNKNOWN')
    ) THEN
        RAISE EXCEPTION 'collected payment is not backed by an active collection attempt';
    END IF;
    RETURN NEW;
END
$$;
-- +goose StatementEnd
