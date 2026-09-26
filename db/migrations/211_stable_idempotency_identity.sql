-- +goose Up
-- Keep historical authorization scopes and outcomes intact. Index the stable
-- command identity separately; pre-existing duplicates require reconciliation.
-- Quiesce old writers and deploy the matching API after applying this migration.
-- +goose StatementBegin
CREATE FUNCTION app.idempotency_command_scope(value text) RETURNS text
LANGUAGE sql IMMUTABLE STRICT SET search_path=pg_catalog AS $$
 SELECT split_part(value,' session:',1);
$$;
-- +goose StatementEnd
CREATE INDEX idempotency_command_identity ON app.idempotency_records(app.idempotency_command_scope(scope),idempotency_key);
-- +goose StatementBegin
CREATE FUNCTION app.guard_idempotency_command() RETURNS trigger
LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,app,pg_temp AS $$
BEGIN
 PERFORM pg_advisory_xact_lock(hashtextextended(app.idempotency_command_scope(NEW.scope)||chr(10)||NEW.idempotency_key,211));
 IF EXISTS(SELECT 1 FROM app.idempotency_records r
 WHERE app.idempotency_command_scope(r.scope)=app.idempotency_command_scope(NEW.scope)
 AND r.idempotency_key=NEW.idempotency_key AND r.scope<>NEW.scope) THEN
  RAISE EXCEPTION 'command already reserved under another authorization scope' USING ERRCODE='23505';
 END IF;
 RETURN NEW;
END $$;
CREATE OR REPLACE FUNCTION app.delete_expired_idempotency_record(p_scope text,p_idempotency_key text)
RETURNS boolean LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,app,pg_temp AS $$
DECLARE removed bigint;
BEGIN
 PERFORM pg_advisory_xact_lock(hashtextextended(app.idempotency_command_scope(p_scope)||chr(10)||p_idempotency_key,211));
 DELETE FROM app.idempotency_records
 WHERE app.idempotency_command_scope(scope)=app.idempotency_command_scope(p_scope)
 AND idempotency_key=p_idempotency_key AND expires_at<=now()
 AND completed_at IS NOT NULL AND response_status>=200 AND response_status<500;
 GET DIAGNOSTICS removed=ROW_COUNT;
 RETURN removed>0;
END $$;
-- A digest only: no membership, delegation or branch information is exposed.
-- Membership lifecycle versions prevent removal/rejoining from restoring an
-- old response. Expiry and branch reassignment also invalidate replay authority.
CREATE FUNCTION app.idempotency_authority_version(actor uuid) RETURNS text
LANGUAGE sql STABLE SECURITY DEFINER SET search_path=pg_catalog,app,pg_temp SET row_security=off AS $$
 SELECT encode(sha256(convert_to(jsonb_build_object(
 'memberships',(SELECT jsonb_agg(jsonb_build_array(m.id,m.organization_id,m.role,m.status,m.purchasing_authority_version,o.status,
 s.version,s.membership_id,s.membership_version,s.mode,s.branch_ids,
 d.version,d.membership_id,d.membership_authority_version,d.expires_at>statement_timestamp(),
 (SELECT COALESCE(max(h.id),0) FROM app.network_operation_history h WHERE h.organization_id=m.organization_id)) ORDER BY m.id)
 FROM app.memberships m JOIN app.organizations o ON o.id=m.organization_id
 LEFT JOIN app.member_branch_scopes s ON s.organization_id=m.organization_id AND s.user_id=m.user_id
 LEFT JOIN app.purchasing_delegations d ON d.organization_id=m.organization_id AND d.user_id=m.user_id
 WHERE m.user_id=actor),
 'platform',(SELECT jsonb_agg(jsonb_build_array(a.id,a.role,a.revoked_at,a.expires_at IS NULL OR a.expires_at>statement_timestamp()) ORDER BY a.id)
 FROM app.platform_role_assignments a WHERE a.user_id=actor)
 )::text,'UTF8')),'hex');
$$;
-- +goose StatementEnd
CREATE TRIGGER idempotency_command_guard BEFORE INSERT ON app.idempotency_records
 FOR EACH ROW EXECUTE FUNCTION app.guard_idempotency_command();
REVOKE ALL ON FUNCTION app.idempotency_command_scope(text),app.guard_idempotency_command(),app.idempotency_authority_version(uuid),app.delete_expired_idempotency_record(text,text) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION app.idempotency_command_scope(text),app.idempotency_authority_version(uuid) TO kredit_app;
-- +goose Down
-- Removing the command boundary would permit a replay to execute again.
SELECT 1;
