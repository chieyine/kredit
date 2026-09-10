-- +goose Up
INSERT INTO app.platform_settings(key,category,value,is_secret,description) VALUES
 ('features.system_acceptance','features','false',false,'Recognize eligible delivered sales after the waiting period using separate system evidence; first-time buyers and delivery issues still require a response'),
 ('automation.system_acceptance_hours','features','72',false,'Minimum hours after confirmed delivery of the goods notice, from 72 to 720')
 ON CONFLICT(key) DO NOTHING;
-- +goose StatementBegin
CREATE FUNCTION app.guard_system_acceptance_policy() RETURNS trigger
LANGUAGE plpgsql SET search_path=pg_catalog,app AS $$
DECLARE enabled jsonb; hours jsonb;
BEGIN
 SELECT value INTO enabled FROM app.platform_settings WHERE key='features.system_acceptance' FOR SHARE;
 SELECT value INTO hours FROM app.platform_settings WHERE key='automation.system_acceptance_hours' FOR SHARE;
 IF enabled IS DISTINCT FROM 'true'::jsonb OR hours IS NULL OR jsonb_typeof(hours)<>'number'
 OR (hours::text)::numeric NOT BETWEEN 72 AND 720 OR trunc((hours::text)::numeric)<>(hours::text)::numeric
 THEN RAISE EXCEPTION 'automatic recognition is disabled or its waiting period is invalid'; END IF;
 IF NEW.minimum_seconds<(hours::text)::bigint*3600 THEN RAISE EXCEPTION 'automatic recognition waiting period is too short'; END IF;
 RETURN NEW;
END;
$$;
-- +goose StatementEnd
CREATE TRIGGER system_acceptance_policy BEFORE INSERT ON app.system_acceptances
 FOR EACH ROW EXECUTE FUNCTION app.guard_system_acceptance_policy();
-- +goose Down
-- +goose StatementBegin
DO $$ BEGIN RAISE EXCEPTION 'Automatic recognition controls require forward recovery'; END $$;
-- +goose StatementEnd
