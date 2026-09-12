-- +goose Up
-- The previous UI recorded a free-text reference as configured billing.
-- Preserve its evidence, but require actual provider confirmation for readiness.
WITH changed AS (
 UPDATE app.supplier_onboarding_profiles SET billing_state='pending_verification',version=version+1,readiness_state='incomplete',readiness_changed_at=now(),updated_at=now()
 WHERE billing_state='configured' RETURNING *
)
INSERT INTO app.supplier_onboarding_revisions(organization_id,profile_version,change_type,actor_reference,snapshot)
SELECT c.organization_id,c.version,'billing.verification_required','migration:128',to_jsonb(c) FROM changed c;
-- +goose Down
-- +goose StatementBegin
DO $$ BEGIN RAISE EXCEPTION 'Unverified billing cannot be restored as configured'; END $$;
-- +goose StatementEnd
