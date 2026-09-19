-- +goose Up
-- Support lookup is exact-key and must not scan every session scope.
CREATE INDEX idempotency_records_recovery_key_idx ON app.idempotency_records(idempotency_key,created_at DESC,id);

-- Keep the existing metrics function unchanged for rolling deployments.
-- This separate aggregate-only function exposes no customer identifiers.
-- +goose StatementBegin
CREATE FUNCTION app.recovery_queue_metrics()
RETURNS TABLE(metric text, value double precision)
LANGUAGE plpgsql STABLE SECURITY DEFINER
SET search_path = pg_catalog, app, jobs
SET row_security = off
AS $$
BEGIN
    RETURN QUERY
    SELECT 'financial_reviews_open'::text, count(*)::double precision FROM app.financial_review_cases WHERE state='OPEN'
    UNION ALL SELECT 'financial_reviews_unassigned', count(*)::double precision FROM app.financial_review_cases WHERE state='OPEN' AND owner_id IS NULL
    UNION ALL SELECT 'financial_review_oldest_open_seconds', COALESCE(GREATEST(0,extract(epoch FROM now()-min(first_seen_at))),0)::double precision FROM app.financial_review_cases WHERE state='OPEN'
    UNION ALL SELECT 'drawdown_receipt_issues_open', count(*)::double precision FROM app.drawdown_receipt_disputes WHERE state='OPEN'
    UNION ALL SELECT 'drawdown_receipt_issue_oldest_open_seconds', COALESCE(GREATEST(0,extract(epoch FROM now()-min(created_at))),0)::double precision FROM app.drawdown_receipt_disputes WHERE state='OPEN';
END;
$$;
-- +goose StatementEnd

REVOKE ALL ON FUNCTION app.recovery_queue_metrics() FROM PUBLIC;
-- +goose StatementBegin
DO $$ BEGIN
 IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname='kredit_app') THEN
  GRANT EXECUTE ON FUNCTION app.recovery_queue_metrics() TO kredit_app;
 END IF;
 IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname='kredit_worker') THEN
  GRANT EXECUTE ON FUNCTION app.recovery_queue_metrics() TO kredit_worker;
 END IF;
END $$;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DO $$ BEGIN RAISE EXCEPTION 'Recovery monitoring requires a forward migration'; END $$;
-- +goose StatementEnd
