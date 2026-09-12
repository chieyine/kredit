-- +goose Up
CREATE TABLE app.message_submissions(
 event_key text PRIMARY KEY CHECK(length(event_key)=64),channel text NOT NULL CHECK(channel IN ('sms','whatsapp')),
 payload_hash text NOT NULL CHECK(length(payload_hash)=64),state text NOT NULL CHECK(state IN ('STARTED','ACCEPTED','REJECTED')),
 notification_id uuid REFERENCES app.notifications(id),
 provider_reference text,created_at timestamptz NOT NULL DEFAULT now(),updated_at timestamptz NOT NULL DEFAULT now()
);
-- Operational infrastructure stores keyed fingerprints and provider references,
-- never message bodies, phone numbers, OTP values or provider access tokens.
-- +goose StatementBegin
DO $$ BEGIN
 IF EXISTS(SELECT 1 FROM pg_roles WHERE rolname='kredit_app') THEN GRANT SELECT,INSERT,UPDATE ON app.message_submissions TO kredit_app; END IF;
 IF EXISTS(SELECT 1 FROM pg_roles WHERE rolname='kredit_worker') THEN GRANT SELECT,INSERT,UPDATE ON app.message_submissions TO kredit_worker; END IF;
END $$;
-- +goose StatementEnd
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION app.provider_work() RETURNS jsonb LANGUAGE plpgsql STABLE SECURITY DEFINER SET search_path=pg_catalog,app AS $$
DECLARE result jsonb;
BEGIN
 IF NOT app.has_admin_role(app.current_user_id(),ARRAY['platform_owner','platform_admin','compliance_reviewer']) THEN RAISE EXCEPTION 'provider operations authority required'; END IF;
 WITH work AS (
 SELECT 'collection'::text kind,ca.id::text id,ca.provider,ca.state,o.supplier_organization_id::text organization_id,ca.obligation_id::text target_id,ca.requested_at started_at,'Check the bank result before another debit.'::text next_action
 FROM app.collection_attempts ca JOIN app.obligations o ON o.id=ca.obligation_id WHERE ca.state IN ('UNKNOWN','SUBMITTED')
 UNION ALL SELECT 'verification',id::text,provider,state,''::text,subject_id::text,COALESCE(started_at,created_at),'Match the original provider request before allowing a retry.' FROM app.buyer_verification_intents WHERE state='STARTED'
 UNION ALL SELECT 'authorization',id::text,provider,state,COALESCE(supplier_organization_id::text,''),business_id::text,created_at,'Recover the original bank authorization; do not create a duplicate.' FROM app.mandate_authorization_intents WHERE state='STARTED'
 UNION ALL SELECT 'notification',id::text,channel,state,COALESCE(supplier_organization_id::text,''),recipient_id::text,created_at,'Review delivery evidence and provider settings. Accepted messages may still arrive.' FROM app.notifications WHERE state IN ('failed','sending') OR (state='sent' AND sent_at<now()-interval '30 minutes')
 UNION ALL SELECT 'message_submission',event_key,channel,state,''::text,COALESCE(notification_id::text,event_key),created_at,'The send result is unknown. Match the original message in the provider records before any resend.' FROM app.message_submissions WHERE state='STARTED' AND created_at<now()-interval '30 seconds'
 UNION ALL SELECT 'document',id::text,'document_scanner',scan_state,COALESCE(organization_id::text,''),id::text,created_at,'Check the scanner result. Quarantined files must not be downloaded.' FROM app.documents WHERE scan_state='QUARANTINED' OR (scan_state='PENDING' AND created_at<now()-interval '10 minutes')
 ), counts AS (SELECT kind,count(*) total FROM work GROUP BY kind), limited AS (SELECT * FROM work ORDER BY started_at,id LIMIT 100)
 SELECT jsonb_build_object('counts',COALESCE((SELECT jsonb_object_agg(kind,total) FROM counts),'{}'::jsonb),'items',COALESCE((SELECT jsonb_agg(to_jsonb(limited) ORDER BY started_at,id) FROM limited),'[]'::jsonb),'limit',100,'as_of',now()) INTO result;
 RETURN result;
END $$;
-- +goose StatementEnd
-- +goose Down
-- +goose StatementBegin
DO $$ BEGIN RAISE EXCEPTION 'Message submission evidence requires forward recovery'; END $$;
-- +goose StatementEnd
