-- name: UpsertNotificationPreference :one
INSERT INTO app.notification_preferences (recipient_id, preferred_channel, fallback_channel, opted_out, quiet_start_hour, quiet_end_hour, timezone) VALUES ($1,$2,$3,$4,$5,$6,$7) ON CONFLICT (recipient_id) DO UPDATE SET preferred_channel=EXCLUDED.preferred_channel, fallback_channel=EXCLUDED.fallback_channel, opted_out=EXCLUDED.opted_out, quiet_start_hour=EXCLUDED.quiet_start_hour, quiet_end_hour=EXCLUDED.quiet_end_hour, timezone=EXCLUDED.timezone, updated_at=now() RETURNING *;

-- name: CreateNotification :one
INSERT INTO app.notifications (recipient_id, channel, template, template_version, event_reference, state, body) VALUES ($1,$2,$3,$4,$5,$6,$7) RETURNING *;

-- name: ListNotificationsForUser :many
SELECT * FROM app.notifications WHERE recipient_id = $1 ORDER BY COALESCE(sent_at, scheduled_at, now()) DESC;

-- name: RecordMessagingEvent :one
INSERT INTO app.messaging_events (provider, provider_event_id, sender, payload_hash, command_type) VALUES ($1,$2,$3,$4,$5) RETURNING *;
