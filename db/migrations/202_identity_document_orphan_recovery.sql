-- +goose Up
-- CreateUpload uses both <organization>/<purpose>/<object> and
-- identity/<user>/<purpose>/<object>. The original orphan predicate only
-- recognized the former, retaining abandoned identity evidence indefinitely.
-- Only canonical generated paths qualify. The worker separately requires a
-- known object modification time at least seven days old before calling this.
-- Completed uploads (including quarantined evidence), unknown expiries and
-- reservations inside the seven-day grace period remain protected.
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION app.document_object_is_orphan(object_path text) RETURNS boolean
LANGUAGE sql STABLE SECURITY DEFINER SET search_path=pg_catalog,app,pg_temp AS $$
 SELECT object_path ~ '^(identity/)?[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}/[A-Za-z0-9_-]{1,64}/[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$'
 AND NOT EXISTS (
   SELECT 1 FROM app.documents d
   WHERE d.object_key=object_path
     AND (d.upload_completed_at IS NOT NULL
       OR d.upload_expires_at IS NULL
       OR d.upload_expires_at>now()-interval '7 days')
 );
$$;
-- +goose StatementEnd
REVOKE ALL ON FUNCTION app.document_object_is_orphan(text) FROM PUBLIC;
-- +goose StatementBegin
DO $$ BEGIN
 IF EXISTS(SELECT 1 FROM pg_roles WHERE rolname='kredit_worker') THEN
   GRANT EXECUTE ON FUNCTION app.document_object_is_orphan(text) TO kredit_worker;
 END IF;
END $$;
-- +goose StatementEnd

-- +goose Down
-- Restore the prior recognition rule without dropping the function, changing
-- its owner/ACL, modifying metadata, or deleting any object during migration.
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION app.document_object_is_orphan(object_path text) RETURNS boolean
LANGUAGE sql STABLE SECURITY DEFINER SET search_path=pg_catalog,app,pg_temp AS $$
 SELECT object_path ~ '^[0-9a-f-]{36}/[A-Za-z0-9_-]{1,64}/[0-9a-f-]{36}$'
 AND NOT EXISTS(SELECT 1 FROM app.documents d WHERE d.object_key=object_path
   AND (d.upload_completed_at IS NOT NULL OR d.upload_expires_at IS NULL OR d.upload_expires_at>now()-interval '7 days'));
$$;
-- +goose StatementEnd
REVOKE ALL ON FUNCTION app.document_object_is_orphan(text) FROM PUBLIC;
