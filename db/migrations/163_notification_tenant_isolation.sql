-- +goose Up
DROP POLICY notification_runtime_access ON app.notifications;
DROP POLICY notification_preference_runtime_access ON app.notification_preferences;

-- Discovery is bounded and operation-specific. Actual claims and mutations
-- still use recipient-scoped transactions and their existing lease predicates.
-- +goose StatementBegin
CREATE FUNCTION app.notification_due_work(batch_size integer) RETURNS TABLE(id uuid)
LANGUAGE sql STABLE SECURITY DEFINER SET search_path=pg_catalog,app SET row_security=off AS $$
 SELECT n.id FROM app.notifications n WHERE delivery_attempts<8 AND (
 (state='scheduled' AND scheduled_at<=now()) OR
 (state='failed' AND COALESCE(next_attempt_at,now())<=now()) OR
 (state='sending' AND lease_expires_at<=now()))
 ORDER BY COALESCE(next_attempt_at,scheduled_at,lease_expires_at,updated_at),n.id
 LIMIT GREATEST(0,LEAST(batch_size,500));
$$;
CREATE FUNCTION app.notification_work_subject(notification uuid) RETURNS uuid
LANGUAGE sql STABLE SECURITY DEFINER SET search_path=pg_catalog,app SET row_security=off AS $$
 SELECT recipient_id FROM app.notifications WHERE id=notification;
$$;
CREATE FUNCTION app.notification_receipt_subject(input_channel text,input_event text,input_message text) RETURNS uuid
LANGUAGE sql STABLE SECURITY DEFINER SET search_path=pg_catalog,app SET row_security=off AS $$
 SELECT recipient_id FROM app.notifications WHERE channel=input_channel
 AND event_reference=input_event AND provider_message_id=input_message
 AND state IN ('sent','delivered','read','failed');
$$;
CREATE FUNCTION app.notification_receipt_work(input_channel text,batch_size integer)
RETURNS TABLE(id uuid,event_reference text,provider_message_id text,destination_ciphertext bytea,recipient_id uuid)
LANGUAGE sql STABLE SECURITY DEFINER SET search_path=pg_catalog,app SET row_security=off AS $$
 SELECT n.id,n.event_reference,n.provider_message_id,n.destination_ciphertext,n.recipient_id
 FROM app.notifications n WHERE n.channel=input_channel AND state='sent'
 AND (provider_checked_at IS NULL OR provider_checked_at<now()-interval '1 minute')
 ORDER BY provider_checked_at NULLS FIRST,created_at,n.id LIMIT GREATEST(0,LEAST(batch_size,50));
$$;
CREATE FUNCTION app.notification_meta_candidates(input_message text)
RETURNS TABLE(id uuid,event_reference text,destination_ciphertext bytea,recipient_id uuid)
LANGUAGE sql STABLE SECURITY DEFINER SET search_path=pg_catalog,app SET row_security=off AS $$
 SELECT n.id,n.event_reference,n.destination_ciphertext,n.recipient_id FROM app.notifications n
 WHERE channel='whatsapp' AND provider_message_id=input_message
 AND state IN ('sent','delivered','read','failed') ORDER BY n.id LIMIT 10;
$$;
-- +goose StatementEnd
REVOKE ALL ON FUNCTION app.notification_due_work(integer),app.notification_work_subject(uuid),app.notification_receipt_work(text,integer),app.notification_receipt_subject(text,text,text),app.notification_meta_candidates(text) FROM PUBLIC;
-- +goose StatementBegin
DO $$ BEGIN
IF EXISTS(SELECT 1 FROM pg_roles WHERE rolname='kredit_worker') THEN
GRANT EXECUTE ON FUNCTION app.notification_due_work(integer),app.notification_work_subject(uuid),app.notification_receipt_work(text,integer) TO kredit_worker;
END IF;
-- Authenticated connector callbacks must resolve their exact original send
-- before receipt validation can select a subject; no message body is exposed.
IF EXISTS(SELECT 1 FROM pg_roles WHERE rolname='kredit_app') THEN
GRANT EXECUTE ON FUNCTION app.notification_receipt_subject(text,text,text),app.notification_meta_candidates(text) TO kredit_app;
END IF;
IF EXISTS(SELECT 1 FROM pg_roles WHERE rolname='kredit_worker') THEN
GRANT EXECUTE ON FUNCTION app.notification_receipt_subject(text,text,text),app.notification_meta_candidates(text) TO kredit_worker;
END IF;
END $$;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DO $$ BEGIN RAISE EXCEPTION 'Notification isolation requires forward recovery'; END $$;
-- +goose StatementEnd
