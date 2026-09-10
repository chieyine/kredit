-- +goose Up
-- Legacy invitations have no reliable membership identity. Preserve their
-- history without guessing a recipient from a role, timestamp or target hash.
ALTER TABLE app.organization_invitations ADD COLUMN membership_id uuid
    REFERENCES app.memberships(id) DEFERRABLE INITIALLY DEFERRED;
CREATE UNIQUE INDEX organization_invitation_membership_idx
    ON app.organization_invitations(membership_id) WHERE membership_id IS NOT NULL;

-- +goose StatementBegin
CREATE FUNCTION app.close_membership_invitation() RETURNS trigger
LANGUAGE plpgsql SET search_path=pg_catalog,app AS $$
BEGIN
    IF OLD.status='invited' AND NEW.status IN ('active','removed') THEN
        UPDATE app.organization_invitations
        SET status=CASE WHEN NEW.status='active' THEN 'accepted'
                        WHEN expires_at<=clock_timestamp() THEN 'expired'
                        ELSE 'revoked' END,
            accepted_at=CASE WHEN NEW.status='active' THEN NEW.accepted_at END
        WHERE membership_id=NEW.id AND organization_id=NEW.organization_id
          AND status='pending';
    END IF;
    RETURN NEW;
END;
$$;
-- +goose StatementEnd
CREATE TRIGGER membership_invitation_lifecycle AFTER UPDATE OF status
    ON app.memberships FOR EACH ROW EXECUTE FUNCTION app.close_membership_invitation();

-- +goose Down
-- +goose StatementBegin
DO $$ BEGIN RAISE EXCEPTION 'Invitation lifecycle requires a forward recovery migration'; END $$;
-- +goose StatementEnd
