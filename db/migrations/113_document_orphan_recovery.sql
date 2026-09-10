-- +goose Up
-- Only expired, never-completed uploads or objects with no metadata qualify.
-- A seven-day delay exceeds every signed-upload lifetime and normal request.
-- +goose StatementBegin
CREATE FUNCTION app.document_object_is_orphan(object_path text) RETURNS boolean
LANGUAGE sql STABLE SECURITY DEFINER SET search_path=pg_catalog,app AS $$
 SELECT object_path ~ '^[0-9a-f-]{36}/[A-Za-z0-9_-]{1,64}/[0-9a-f-]{36}$'
 AND NOT EXISTS(SELECT 1 FROM app.documents d WHERE d.object_key=object_path
   AND (d.upload_completed_at IS NOT NULL OR d.upload_expires_at IS NULL OR d.upload_expires_at>now()-interval '7 days'));
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
DROP FUNCTION app.document_object_is_orphan(text);
