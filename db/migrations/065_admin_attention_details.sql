-- +goose Up
-- +goose StatementBegin
CREATE FUNCTION app.admin_attention_details() RETURNS jsonb LANGUAGE sql STABLE SECURITY DEFINER SET search_path=pg_catalog,app AS $$
 SELECT jsonb_build_object('mandates',COALESCE((SELECT jsonb_agg(jsonb_build_object('id',id,'expires_at',ends_at,'buyer_id',metadata->>'user_id','buyer',app.admin_actor_name(NULLIF(metadata->>'user_id','')::uuid)) ORDER BY ends_at) FROM (SELECT id,ends_at,metadata FROM app.payment_mandates WHERE state='active' AND ends_at>now() AND ends_at<=now()+make_interval(days=>COALESCE((app.business_policy()->>'mandate_notice_days')::int,7)) ORDER BY ends_at LIMIT 100)m),'[]'::jsonb),
 'notices',COALESCE((SELECT jsonb_agg(jsonb_build_object('id',id,'recipient',app.admin_actor_name(recipient_id),'template',template,'channel',channel,'failed_at',failed_at,'reason',failure_reason) ORDER BY failed_at DESC) FROM (SELECT * FROM app.notifications WHERE state='failed' ORDER BY failed_at DESC LIMIT 100)n),'[]'::jsonb),
 'debits',COALESCE((SELECT jsonb_agg(jsonb_build_object('id',id,'obligation_id',obligation_id,'state',state,'requested_at',requested_at) ORDER BY requested_at) FROM (SELECT * FROM app.collection_attempts WHERE state IN ('UNKNOWN','SUBMITTED','PENDING') ORDER BY requested_at LIMIT 100)a),'[]'::jsonb));
$$;
-- +goose StatementEnd
REVOKE ALL ON FUNCTION app.admin_attention_details() FROM PUBLIC;
-- +goose StatementBegin
DO $$ BEGIN IF EXISTS(SELECT 1 FROM pg_roles WHERE rolname='kredit_app') THEN GRANT EXECUTE ON FUNCTION app.admin_attention_details() TO kredit_app; END IF; END $$;
-- +goose StatementEnd

-- +goose Down
DROP FUNCTION app.admin_attention_details();
