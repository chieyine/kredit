-- +goose Up
-- Permissive policies combine with OR. The service-wide policy from 058
-- would otherwise bypass the tenant policy installed in 081.
DROP POLICY IF EXISTS settlement_runtime_access ON app.settlement_events;

-- Aggregate callers install the buyer or organization transaction context.
-- Run their lookups with that caller's RLS, rather than the migration owner's
-- authority. Global worker discovery has separate identifier-only functions.
ALTER FUNCTION app.credit_snapshot_by_id(TEXT) SECURITY INVOKER;
ALTER FUNCTION app.credit_snapshot_by_obligation(TEXT) SECURITY INVOKER;
REVOKE ALL ON FUNCTION app.credit_snapshot_by_id(TEXT) FROM PUBLIC;
REVOKE ALL ON FUNCTION app.credit_snapshot_by_obligation(TEXT) FROM PUBLIC;

-- Governance checks are internal helpers, not public database entry points.
REVOKE ALL ON FUNCTION app.is_platform_owner(UUID) FROM PUBLIC;
REVOKE ALL ON FUNCTION app.current_governance_mode() FROM PUBLIC;

-- +goose Down
-- Do not reopen cross-tenant access when rolling back application code.
SELECT 1;
