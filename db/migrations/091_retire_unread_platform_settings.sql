-- +goose Up
-- Migration 087 seeded 41 platform settings. Three of them are read by the
-- application: features.trade_lines, features.drawdowns and features.disputes.
-- The rest were switches an owner could move with no effect anywhere — a launch
-- banner nothing rendered, session and login limits nothing enforced, a late
-- penalty fee the fee engine does not charge, and API keys for Paystack, Termii
-- and Resend, none of which has an adapter in this codebase.
--
-- Storing them was not neutral. The console showed an owner a control panel of
-- levers that did nothing, and it invited real provider secrets into a table
-- whose values are only ever read back to that same screen. Keys whose feature
-- needs credentials to exist at all (WhatsApp, Mono) belong in configuration,
-- where a missing value fails at boot instead of failing silently at runtime.
--
-- The retirement itself is recorded: platform_settings_history is append-only,
-- so each removed key gets a 'delete' row before it goes.

-- +goose StatementBegin
DO $$
DECLARE retired text[] := ARRAY[
  'launch.mode', 'launch.banner_enabled', 'launch.banner_text', 'launch.waitlist_enabled',
  'features.repayment_extensions', 'features.early_settlement_discounts',
  'features.notifications_whatsapp', 'features.mono_direct_debit',
  'integrations.mono.enabled', 'integrations.mono.app_id', 'integrations.mono.secret_key',
  'integrations.mono.public_key', 'integrations.mono.status',
  'integrations.paystack.enabled', 'integrations.paystack.secret_key',
  'integrations.paystack.public_key', 'integrations.paystack.status',
  'integrations.termii.enabled', 'integrations.termii.api_key',
  'integrations.termii.sender_id', 'integrations.termii.status',
  'integrations.resend.enabled', 'integrations.resend.api_key',
  'integrations.resend.from_email', 'integrations.resend.status',
  'security.mfa_enforced', 'security.session_idle_minutes',
  'security.max_login_attempts', 'security.ip_allowlist_enabled',
  'kyc.tier1_max_kobo', 'kyc.tier2_bvn_required', 'kyc.tier3_cac_required',
  'fees.supplier_rate_bps', 'fees.late_fee_rate_bps', 'fees.grace_period_days',
  'notifications.channels', 'notifications.pre_debit_reminder_days'
];
BEGIN
  INSERT INTO app.platform_settings_history (key, old_value, new_value, version, action, actor_id, reason)
  SELECT key, value, 'null'::jsonb, version + 1, 'delete', NULL,
         'Retired: no code reads this setting. See migration 091.'
  FROM app.platform_settings
  WHERE key = ANY(retired) OR category NOT IN ('features', 'governance');

  DELETE FROM app.platform_settings WHERE key = ANY(retired) OR category NOT IN ('features', 'governance');
END $$;
-- +goose StatementEnd

-- A settings table that can hold a key the application does not know is a
-- settings table that will hold one again. The application refuses to write an
-- unknown key; this refuses to keep one.
ALTER TABLE app.platform_settings
  DROP CONSTRAINT IF EXISTS platform_settings_category_known;
ALTER TABLE app.platform_settings
  ADD CONSTRAINT platform_settings_category_known CHECK (category IN ('features', 'governance'));

-- +goose Down
-- Rolling back application code does not make dead settings meaningful again,
-- and their history rows are already written. Leave the table as it is.
SELECT 1;
