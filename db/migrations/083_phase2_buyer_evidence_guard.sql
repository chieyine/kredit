-- +goose Up

-- Table grants are deliberately broad in the legacy provisioning template.
-- Until that template is fully decomposed, keep the strongest invariant at the
-- database object itself: an autonomous worker can read evidence required for
-- collections, but it can never manufacture, rewrite, or delete buyer-originated
-- acceptance/receipt/claim/acknowledgement evidence.
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION app.guard_buyer_originated_evidence()
RETURNS trigger
LANGUAGE plpgsql
SET search_path = pg_catalog, app
AS $$
BEGIN
    IF current_user = 'kredit_worker' THEN
        RAISE EXCEPTION 'autonomous worker may not mutate buyer-originated evidence'
            USING ERRCODE = '42501';
    END IF;
    IF TG_OP = 'DELETE' THEN
        RETURN OLD;
    END IF;
    RETURN NEW;
END
$$;
-- +goose StatementEnd

DROP TRIGGER IF EXISTS agreement_acceptances_worker_guard ON app.agreement_acceptances;
CREATE TRIGGER agreement_acceptances_worker_guard
    BEFORE INSERT OR UPDATE OR DELETE ON app.agreement_acceptances
    FOR EACH ROW EXECUTE FUNCTION app.guard_buyer_originated_evidence();

DROP TRIGGER IF EXISTS receipt_confirmations_worker_guard ON app.receipt_confirmations;
CREATE TRIGGER receipt_confirmations_worker_guard
    BEFORE INSERT OR UPDATE OR DELETE ON app.receipt_confirmations
    FOR EACH ROW EXECUTE FUNCTION app.guard_buyer_originated_evidence();

DROP TRIGGER IF EXISTS payment_claims_worker_guard ON app.payment_claims;
CREATE TRIGGER payment_claims_worker_guard
    BEFORE INSERT OR UPDATE OR DELETE ON app.payment_claims
    FOR EACH ROW EXECUTE FUNCTION app.guard_buyer_originated_evidence();

DROP TRIGGER IF EXISTS collection_notice_ack_worker_guard ON app.collection_notice_acknowledgements;
CREATE TRIGGER collection_notice_ack_worker_guard
    BEFORE INSERT OR UPDATE OR DELETE ON app.collection_notice_acknowledgements
    FOR EACH ROW EXECUTE FUNCTION app.guard_buyer_originated_evidence();

-- +goose Down
DROP TRIGGER IF EXISTS collection_notice_ack_worker_guard ON app.collection_notice_acknowledgements;
DROP TRIGGER IF EXISTS payment_claims_worker_guard ON app.payment_claims;
DROP TRIGGER IF EXISTS receipt_confirmations_worker_guard ON app.receipt_confirmations;
DROP TRIGGER IF EXISTS agreement_acceptances_worker_guard ON app.agreement_acceptances;
DROP FUNCTION IF EXISTS app.guard_buyer_originated_evidence();
