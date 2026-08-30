-- +goose Up
CREATE TABLE app.notification_templates (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  event_type text NOT NULL,
  channel text NOT NULL,
  version text NOT NULL,
  body text NOT NULL,
  active boolean NOT NULL DEFAULT true,
  created_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (event_type, channel, version)
);
CREATE TABLE app.notification_preferences (
  recipient_id uuid PRIMARY KEY REFERENCES app.users(id),
  preferred_channel text NOT NULL DEFAULT 'whatsapp',
  fallback_channel text NOT NULL DEFAULT 'email',
  opted_out boolean NOT NULL DEFAULT false,
  quiet_start_hour integer NOT NULL DEFAULT 22 CHECK (quiet_start_hour BETWEEN 0 AND 23),
  quiet_end_hour integer NOT NULL DEFAULT 7 CHECK (quiet_end_hour BETWEEN 0 AND 23),
  timezone text NOT NULL DEFAULT 'Africa/Lagos',
  updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE app.notifications (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  recipient_id uuid NOT NULL REFERENCES app.users(id),
  channel text NOT NULL,
  template text NOT NULL,
  template_version text NOT NULL,
  event_reference text NOT NULL,
  state text NOT NULL CHECK (state IN ('scheduled','sent','delivered','read','failed','suppressed')),
  provider_message_id text,
  body text NOT NULL,
  scheduled_at timestamptz,
  sent_at timestamptz,
  delivered_at timestamptz,
  read_at timestamptz,
  failed_at timestamptz,
  failure_reason text,
  UNIQUE (event_reference, channel)
);
CREATE TABLE app.messaging_events (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  provider text NOT NULL,
  provider_event_id text NOT NULL,
  sender text NOT NULL,
  payload_hash text NOT NULL,
  command_type text,
  processed_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (provider, provider_event_id)
);
ALTER TABLE app.notifications ENABLE ROW LEVEL SECURITY;
CREATE POLICY notification_recipient_access ON app.notifications USING (recipient_id = app.current_user_id());
ALTER TABLE app.notification_preferences ENABLE ROW LEVEL SECURITY;
CREATE POLICY notification_preference_access ON app.notification_preferences USING (recipient_id = app.current_user_id());
ALTER TABLE app.messaging_events ENABLE ROW LEVEL SECURITY;
CREATE POLICY messaging_support_access ON app.messaging_events USING (current_user = 'kredit_app');

-- +goose Down
DROP TABLE IF EXISTS app.messaging_events, app.notifications, app.notification_preferences, app.notification_templates;
