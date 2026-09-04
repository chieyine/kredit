-- +goose Up

-- A recurring authorization is scoped to both sides of the trade relationship.
REVOKE ALL ON FUNCTION app.trade_line_mandate(UUID, UUID, UUID) FROM PUBLIC;
-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'kredit_app') THEN
        REVOKE EXECUTE ON FUNCTION app.trade_line_mandate(UUID, UUID, UUID) FROM kredit_app;
    END IF;
END
$$;
-- +goose StatementEnd
DROP FUNCTION app.trade_line_mandate(UUID, UUID, UUID);

-- +goose StatementBegin
CREATE FUNCTION app.trade_line_mandate(
    p_mandate_id UUID,
    p_buyer_user_id UUID,
    p_buyer_business_id UUID,
    p_supplier_organization_id UUID
)
RETURNS TABLE (
    id TEXT,
    provider TEXT,
    provider_mandate_id TEXT,
    buyer_business_id TEXT,
    buyer_user_id TEXT,
    supplier_organization_id TEXT,
    amount_ceiling_kobo BIGINT,
    state TEXT,
    created_at TIMESTAMPTZ
)
LANGUAGE sql
SECURITY DEFINER
STABLE
SET search_path = app, pg_catalog
AS $$
    SELECT m.id::text,
           m.provider,
           m.provider_mandate_id,
           m.buyer_subject_id::text,
           b.owner_user_id::text,
           m.supplier_organization_id::text,
           m.amount_ceiling_kobo,
           m.state,
           m.created_at
    FROM app.payment_mandates m
    JOIN app.businesses b ON b.id = m.buyer_subject_id
    WHERE m.id = p_mandate_id
      AND m.buyer_subject_type = 'business'
      AND m.buyer_subject_id = p_buyer_business_id
      AND b.owner_user_id = p_buyer_user_id
      AND m.supplier_organization_id = p_supplier_organization_id
$$;
-- +goose StatementEnd

REVOKE ALL ON FUNCTION app.trade_line_mandate(UUID, UUID, UUID, UUID) FROM PUBLIC;
-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'kredit_app') THEN
        GRANT EXECUTE ON FUNCTION app.trade_line_mandate(UUID, UUID, UUID, UUID) TO kredit_app;
    END IF;
END
$$;
-- +goose StatementEnd

-- Only a persisted collection attempt may originate a collected payment.
-- +goose StatementBegin
CREATE FUNCTION app.guard_collected_payment_provenance() RETURNS trigger
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

CREATE TRIGGER collected_payment_provenance
BEFORE INSERT ON app.payments
FOR EACH ROW EXECUTE FUNCTION app.guard_collected_payment_provenance();

-- Dispute effect is server-owned and one active dispute represents the total
-- contested amount for an obligation at any instant.
UPDATE app.disputes
SET collection_effect = 'CONTESTED_ONLY'
WHERE collection_effect <> 'CONTESTED_ONLY'
  AND state IN ('OPEN','UNDER_REVIEW','PARTIALLY_RESOLVED');

-- Multiple legacy active disputes cannot be merged automatically without
-- changing buyer evidence or adjudication state. Fail before DDL with an
-- actionable inventory instead of failing opaquely while creating the index.
-- +goose StatementBegin
DO $$
DECLARE duplicate_count BIGINT;
BEGIN
    SELECT count(*) INTO duplicate_count
    FROM (
        SELECT obligation_id
        FROM app.disputes
        WHERE state IN ('OPEN','UNDER_REVIEW','PARTIALLY_RESOLVED')
        GROUP BY obligation_id
        HAVING count(*) > 1
    ) duplicates;
    IF duplicate_count > 0 THEN
        RAISE EXCEPTION 'migration 073 requires manual consolidation of active disputes for % obligation(s)', duplicate_count;
    END IF;
END
$$;
-- +goose StatementEnd

CREATE UNIQUE INDEX disputes_one_active_per_obligation_idx
ON app.disputes(obligation_id)
WHERE state IN ('OPEN','UNDER_REVIEW','PARTIALLY_RESOLVED');

-- +goose StatementBegin
CREATE FUNCTION app.force_dispute_collection_effect() RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    NEW.collection_effect := 'CONTESTED_ONLY';
    RETURN NEW;
END
$$;
-- +goose StatementEnd

CREATE TRIGGER dispute_collection_effect_server_owned
BEFORE INSERT OR UPDATE OF collection_effect ON app.disputes
FOR EACH ROW EXECUTE FUNCTION app.force_dispute_collection_effect();

-- +goose Down
DROP TRIGGER IF EXISTS dispute_collection_effect_server_owned ON app.disputes;
DROP FUNCTION IF EXISTS app.force_dispute_collection_effect();
DROP INDEX IF EXISTS app.disputes_one_active_per_obligation_idx;
DROP TRIGGER IF EXISTS collected_payment_provenance ON app.payments;
DROP FUNCTION IF EXISTS app.guard_collected_payment_provenance();
DROP FUNCTION IF EXISTS app.trade_line_mandate(UUID, UUID, UUID, UUID);

-- +goose StatementBegin
CREATE FUNCTION app.trade_line_mandate(p_mandate_id UUID, p_buyer_user_id UUID, p_buyer_business_id UUID)
RETURNS TABLE (id TEXT,provider TEXT,provider_mandate_id TEXT,buyer_business_id TEXT,buyer_user_id TEXT,amount_ceiling_kobo BIGINT,state TEXT,created_at TIMESTAMPTZ)
LANGUAGE sql SECURITY DEFINER STABLE SET search_path = app, pg_catalog AS $$
 SELECT m.id::text,m.provider,m.provider_mandate_id,m.buyer_subject_id::text,b.owner_user_id::text,m.amount_ceiling_kobo,m.state,m.created_at
 FROM app.payment_mandates m JOIN app.businesses b ON b.id=m.buyer_subject_id
 WHERE m.id=p_mandate_id AND m.buyer_subject_type='business' AND m.buyer_subject_id=p_buyer_business_id AND b.owner_user_id=p_buyer_user_id
$$;
-- +goose StatementEnd
REVOKE ALL ON FUNCTION app.trade_line_mandate(UUID, UUID, UUID) FROM PUBLIC;
-- +goose StatementBegin
DO $$ BEGIN IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname='kredit_app') THEN GRANT EXECUTE ON FUNCTION app.trade_line_mandate(UUID, UUID, UUID) TO kredit_app; END IF; END $$;
-- +goose StatementEnd
