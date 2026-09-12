-- +goose Up
-- Narrow operational projection; source tables keep their tenant policies.
-- +goose StatementBegin
CREATE FUNCTION app.provider_work() RETURNS jsonb LANGUAGE plpgsql STABLE SECURITY DEFINER SET search_path=pg_catalog,app AS $$
DECLARE result jsonb;
BEGIN
 IF NOT app.has_admin_role(app.current_user_id(),ARRAY['platform_owner','platform_admin','compliance_reviewer']) THEN RAISE EXCEPTION 'provider operations authority required'; END IF;
 WITH work AS (
 SELECT 'collection'::text kind,ca.id::text id,ca.provider,ca.state,o.supplier_organization_id::text organization_id,ca.obligation_id::text target_id,ca.requested_at started_at,'Check the bank result before another debit.'::text next_action
 FROM app.collection_attempts ca JOIN app.obligations o ON o.id=ca.obligation_id WHERE ca.state IN ('UNKNOWN','SUBMITTED')
 UNION ALL SELECT 'verification',id::text,provider,state,''::text,subject_id::text,COALESCE(started_at,created_at),'Match the original provider request before allowing a retry.' FROM app.buyer_verification_intents WHERE state='STARTED'
 UNION ALL SELECT 'authorization',id::text,provider,state,COALESCE(supplier_organization_id::text,''),business_id::text,created_at,'Recover the original bank authorization; do not create a duplicate.' FROM app.mandate_authorization_intents WHERE state='STARTED'
 UNION ALL SELECT 'notification',id::text,channel,state,COALESCE(supplier_organization_id::text,''),recipient_id::text,created_at,'Review delivery evidence and provider settings. Accepted messages may still arrive.' FROM app.notifications WHERE state IN ('failed','sending') OR (state='sent' AND sent_at<now()-interval '30 minutes')
 UNION ALL SELECT 'document',id::text,'document_scanner',scan_state,COALESCE(organization_id::text,''),id::text,created_at,'Check the scanner result. Quarantined files must not be downloaded.' FROM app.documents WHERE scan_state='QUARANTINED' OR (scan_state='PENDING' AND created_at<now()-interval '10 minutes')
 ), counts AS (SELECT kind,count(*) total FROM work GROUP BY kind), limited AS (SELECT * FROM work ORDER BY started_at,id LIMIT 100)
 SELECT jsonb_build_object('counts',COALESCE((SELECT jsonb_object_agg(kind,total) FROM counts),'{}'::jsonb),'items',COALESCE((SELECT jsonb_agg(to_jsonb(limited) ORDER BY started_at,id) FROM limited),'[]'::jsonb),'limit',100,'as_of',now()) INTO result;
 RETURN result;
END $$;
-- +goose StatementEnd
REVOKE ALL ON FUNCTION app.provider_work() FROM PUBLIC;
-- +goose StatementBegin
DO $$ BEGIN IF EXISTS(SELECT 1 FROM pg_roles WHERE rolname='kredit_app') THEN GRANT EXECUTE ON FUNCTION app.provider_work() TO kredit_app; END IF; END $$;
-- +goose StatementEnd
-- +goose Down
DROP FUNCTION app.provider_work();
