-- +goose Up
ALTER TABLE app.privacy_requests ADD COLUMN completion_mode text NOT NULL DEFAULT 'delegated_team'
 CHECK (completion_mode IN ('delegated_team','solo_owner'));
ALTER TABLE app.privacy_requests ADD COLUMN completion_reason text NOT NULL DEFAULT '';
-- Replace only the historical rule requiring distinct completion actors.
-- +goose StatementBegin
DO $$ DECLARE c record; BEGIN
 FOR c IN SELECT conname FROM pg_constraint WHERE conrelid='app.privacy_requests'::regclass
 AND contype='c' AND pg_get_constraintdef(oid) LIKE '%second_approved_by%'
 LOOP EXECUTE format('ALTER TABLE app.privacy_requests DROP CONSTRAINT %I',c.conname); END LOOP;
END $$;
-- +goose StatementEnd
ALTER TABLE app.privacy_requests ADD CONSTRAINT privacy_completion_independence CHECK (
 second_approved_by IS NULL OR second_approved_by<>decided_by OR
 (completion_mode='solo_owner' AND length(btrim(completion_reason)) BETWEEN 8 AND 2000)
);
-- +goose StatementBegin
CREATE FUNCTION app.guard_solo_owner_privacy_completion() RETURNS trigger
LANGUAGE plpgsql AS $$ BEGIN
 IF NEW.completion_mode='solo_owner' AND NEW.state='COMPLETED' THEN
  PERFORM pg_advisory_xact_lock(746219830045::bigint);
  IF NEW.second_approved_by IS NULL OR NEW.second_approved_by<>NEW.decided_by
   OR NEW.requester_user_id=NEW.second_approved_by
   OR app.current_governance_mode()<>'solo_owner'
   OR NOT app.is_platform_owner(NEW.second_approved_by)
   OR length(btrim(NEW.completion_reason)) NOT BETWEEN 8 AND 2000 THEN
   RAISE EXCEPTION 'solo-owner completion requires the current owner, recorded reason and solo-owner governance';
  END IF;
 END IF;
 RETURN NEW;
END $$;
-- +goose StatementEnd
CREATE TRIGGER privacy_solo_owner_guard BEFORE INSERT OR UPDATE ON app.privacy_requests
FOR EACH ROW EXECUTE FUNCTION app.guard_solo_owner_privacy_completion();
-- +goose Down
-- +goose StatementBegin
DO $$ BEGIN
 RAISE EXCEPTION 'Owner completion history must be preserved; use a forward migration';
END $$;
-- +goose StatementEnd
