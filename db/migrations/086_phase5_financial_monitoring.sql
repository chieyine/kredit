-- +goose Up

-- Platform monitoring must not silently count only the current tenant. Expose
-- fixed aggregate values, not row identifiers or a caller-supplied query. The
-- owner must have complete visibility; row_security=off raises an error rather
-- than returning a misleading partial result if that ownership contract breaks.
-- +goose StatementBegin
CREATE FUNCTION app.phase5_financial_metrics()
RETURNS TABLE(metric text, value double precision)
LANGUAGE sql STABLE SECURITY DEFINER
SET search_path = pg_catalog, app, jobs
SET row_security = off
AS $$
    SELECT 'ledger_discrepancies'::text, count(*)::double precision FROM app.financial_discrepancies WHERE kind='ledger'
    UNION ALL SELECT 'balance_discrepancies', count(*)::double precision FROM app.financial_discrepancies WHERE kind='balance'
    UNION ALL SELECT 'schedule_discrepancies', count(*)::double precision FROM app.financial_discrepancies WHERE kind='schedule'
    UNION ALL SELECT 'collection_payment_discrepancies', count(*)::double precision FROM app.financial_discrepancies WHERE kind='collection_payment'
    UNION ALL SELECT 'settlement_discrepancies', count(*)::double precision FROM app.financial_discrepancies WHERE kind IN ('settlement','settlement_missing','provider_reversal','settlement_without_payment')
    UNION ALL SELECT 'collection_unknown_states', count(*)::double precision FROM app.collection_attempts WHERE state='UNKNOWN'
    UNION ALL SELECT 'notification_dead_letters', count(*)::double precision FROM app.notifications WHERE state='failed' AND delivery_attempts>=8
    UNION ALL SELECT 'outbox_delivery_failures', count(*)::double precision FROM app.outbox_events WHERE state='failed' AND attempts>=8
    UNION ALL SELECT 'provider_webhook_oldest_unprocessed_seconds', COALESCE(GREATEST(0,extract(epoch FROM now()-min(received_at))),0)::double precision FROM app.provider_webhook_inbox WHERE state IN ('received','processing','failed')
    UNION ALL SELECT 'collection_oldest_unresolved_seconds', COALESCE(GREATEST(0,extract(epoch FROM now()-min(requested_at))),0)::double precision FROM app.collection_attempts WHERE state IN ('PENDING','SUBMITTED','UNKNOWN')
    UNION ALL SELECT 'river_pending_jobs', count(*)::double precision FROM jobs.river_job WHERE state IN ('available','pending','retryable','scheduled') AND scheduled_at<=now()
    UNION ALL SELECT 'river_discarded_jobs', count(*)::double precision FROM jobs.river_job WHERE state='discarded'
    UNION ALL SELECT 'active_obligations', count(*)::double precision FROM app.obligations WHERE lifecycle_status='ACTIVE' AND outstanding_kobo>0
    UNION ALL SELECT 'negative_outstanding_balances', count(*)::double precision FROM app.obligations WHERE outstanding_kobo<0;
$$;
-- +goose StatementEnd
REVOKE ALL ON FUNCTION app.phase5_financial_metrics() FROM PUBLIC;
-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname='kredit_app') THEN
        GRANT EXECUTE ON FUNCTION app.phase5_financial_metrics() TO kredit_app;
    END IF;
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname='kredit_worker') THEN
        GRANT EXECUTE ON FUNCTION app.phase5_financial_metrics() TO kredit_worker;
    END IF;
END
$$;
-- +goose StatementEnd

-- +goose Down
DROP FUNCTION IF EXISTS app.phase5_financial_metrics();
