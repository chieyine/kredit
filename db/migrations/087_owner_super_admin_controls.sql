-- +goose Up

-- 1. Platform Owner Role & Support
ALTER TABLE app.platform_role_assignments DROP CONSTRAINT IF EXISTS platform_role_assignments_role_check;
ALTER TABLE app.platform_role_assignments ADD CONSTRAINT platform_role_assignments_role_check
  CHECK(role IN ('support_agent','compliance_reviewer','dispute_reviewer','platform_admin','finance_operator','policy_manager','approver','access_administrator','platform_owner'));

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION app.is_platform_owner(actor uuid) RETURNS boolean
LANGUAGE sql STABLE SECURITY DEFINER SET search_path=pg_catalog,app AS $$
  SELECT EXISTS(
    SELECT 1
    FROM app.platform_role_assignments r
    JOIN app.users u ON u.id = r.user_id
    WHERE r.user_id = actor
      AND r.role = 'platform_owner'
      AND r.revoked_at IS NULL
      AND (r.expires_at IS NULL OR r.expires_at > now())
      AND u.status = 'active'
  );
$$;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION app.is_active_policy_admin(actor uuid) RETURNS boolean
LANGUAGE sql STABLE SECURITY DEFINER SET search_path=pg_catalog,app AS $$
  SELECT app.has_admin_role(actor, ARRAY['platform_admin','policy_manager','platform_owner']);
$$;
-- +goose StatementEnd

-- 2. Last Platform Owner Protection
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION app.protect_last_platform_owner() RETURNS trigger
LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,app AS $$
BEGIN
  IF OLD.role = 'platform_owner' AND OLD.revoked_at IS NULL THEN
    IF TG_OP = 'DELETE' OR (TG_OP = 'UPDATE' AND (NEW.revoked_at IS NOT NULL OR NEW.role <> 'platform_owner')) THEN
      IF (SELECT count(*) FROM app.platform_role_assignments r JOIN app.users u ON u.id = r.user_id WHERE r.role = 'platform_owner' AND r.revoked_at IS NULL AND (r.expires_at IS NULL OR r.expires_at > now()) AND u.status = 'active' AND r.id <> OLD.id) = 0 THEN
        RAISE EXCEPTION 'cannot revoke or remove the last active platform owner';
      END IF;
    END IF;
  END IF;
  RETURN COALESCE(NEW, OLD);
END;
$$;
-- +goose StatementEnd

DROP TRIGGER IF EXISTS protect_last_platform_owner_trigger ON app.platform_role_assignments;
CREATE TRIGGER protect_last_platform_owner_trigger
BEFORE UPDATE OR DELETE ON app.platform_role_assignments
FOR EACH ROW EXECUTE FUNCTION app.protect_last_platform_owner();

-- 3. Platform Governance Mode
CREATE TABLE IF NOT EXISTS app.platform_governance (
    id text PRIMARY KEY DEFAULT 'singleton' CHECK (id = 'singleton'),
    mode text NOT NULL CHECK (mode IN ('solo_owner', 'delegated_team')) DEFAULT 'solo_owner',
    updated_at timestamptz NOT NULL DEFAULT now(),
    updated_by uuid REFERENCES app.users(id),
    reason text NOT NULL DEFAULT 'Initial configuration'
);

INSERT INTO app.platform_governance (id, mode, reason) VALUES ('singleton', 'solo_owner', 'Initial system setup')
ON CONFLICT (id) DO NOTHING;

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION app.current_governance_mode() RETURNS text
LANGUAGE sql STABLE SECURITY DEFINER SET search_path=pg_catalog,app AS $$
  SELECT COALESCE((SELECT mode FROM app.platform_governance WHERE id = 'singleton'), 'solo_owner');
$$;
-- +goose StatementEnd

-- 4. Relax Four-Eyes Constraints for Solo Owner Mode
ALTER TABLE app.business_policy_changes DROP CONSTRAINT IF EXISTS business_policy_changes_check;
ALTER TABLE app.business_policy_changes ADD CONSTRAINT business_policy_changes_check
  CHECK(state <> 'approved' OR (decided_by IS NOT NULL AND decided_at IS NOT NULL AND effective_at >= decided_at));

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION app.guard_business_policy_change() RETURNS trigger
LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,app AS $$
BEGIN
  IF TG_OP = 'DELETE' THEN RAISE EXCEPTION 'policy history is immutable'; END IF;
  IF TG_OP = 'UPDATE' THEN
    IF (to_jsonb(NEW)-ARRAY['state','decided_by','decided_at']) IS DISTINCT FROM (to_jsonb(OLD)-ARRAY['state','decided_by','decided_at']) THEN
      RAISE EXCEPTION 'proposed policy is immutable';
    END IF;
    IF NOT ((OLD.state='pending' AND NEW.state IN ('approved','rejected','cancelled')) OR (OLD.state='approved' AND OLD.effective_at>clock_timestamp() AND NEW.state='cancelled')) THEN
      RAISE EXCEPTION 'invalid policy transition';
    END IF;
    IF NEW.state = 'approved' AND NEW.decided_by = NEW.proposed_by THEN
      IF NOT (app.current_governance_mode() = 'solo_owner' AND app.is_platform_owner(NEW.decided_by)) THEN
        RAISE EXCEPTION 'maker cannot approve own policy change under current governance mode';
      END IF;
    END IF;
  END IF;
  RETURN NEW;
END;
$$;
-- +goose StatementEnd

ALTER TABLE app.admin_change_requests DROP CONSTRAINT IF EXISTS admin_change_requests_check1;

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION app.guard_admin_change() RETURNS trigger
LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,app AS $$
BEGIN
  IF TG_OP = 'DELETE' THEN RAISE EXCEPTION 'approval history is immutable'; END IF;
  IF (to_jsonb(NEW)-ARRAY['state','approved_by','buyer_decided_by','decided_at']) IS DISTINCT FROM (to_jsonb(OLD)-ARRAY['state','approved_by','buyer_decided_by','decided_at']) THEN
    RAISE EXCEPTION 'proposed financial change is immutable';
  END IF;
  IF NOT ((OLD.state='pending' AND NEW.state IN ('awaiting_buyer','applied','rejected','cancelled')) OR (OLD.state='awaiting_buyer' AND NEW.state IN ('applied','rejected','cancelled'))) THEN
    RAISE EXCEPTION 'invalid approval transition';
  END IF;
  IF OLD.approved_by IS NOT NULL AND NEW.approved_by IS DISTINCT FROM OLD.approved_by THEN
    RAISE EXCEPTION 'approval evidence is immutable';
  END IF;
  IF NEW.approved_by IS NOT NULL AND NEW.approved_by = NEW.proposed_by THEN
    IF NOT (app.current_governance_mode() = 'solo_owner' AND app.is_platform_owner(NEW.approved_by)) THEN
      RAISE EXCEPTION 'maker cannot approve own financial change under current governance mode';
    END IF;
  END IF;
  RETURN NEW;
END;
$$;
-- +goose StatementEnd

-- 5. Platform Settings Registry & History
CREATE TABLE IF NOT EXISTS app.platform_settings (
    key text PRIMARY KEY,
    category text NOT NULL,
    value jsonb NOT NULL,
    is_secret boolean NOT NULL DEFAULT false,
    secret_fingerprint text,
    description text NOT NULL DEFAULT '',
    version integer NOT NULL DEFAULT 1,
    updated_at timestamptz NOT NULL DEFAULT now(),
    updated_by uuid REFERENCES app.users(id),
    reason text NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS app.platform_settings_history (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    key text NOT NULL,
    old_value jsonb,
    new_value jsonb NOT NULL,
    version integer NOT NULL,
    action text NOT NULL CHECK (action IN ('create', 'update', 'delete', 'secret_rotated')),
    actor_id uuid REFERENCES app.users(id),
    reason text NOT NULL,
    recorded_at timestamptz NOT NULL DEFAULT now()
);

-- Seed baseline typed settings
INSERT INTO app.platform_settings (key, category, value, is_secret, description) VALUES
('launch.mode', 'launch', '"pre_launch"', false, 'Operational launch phase: pre_launch, private_launch, public_launch'),
('launch.banner_enabled', 'launch', 'false', false, 'Enable top announcement banner'),
('launch.banner_text', 'launch', '"Welcome to Kredit. Launching soon in private beta."', false, 'Banner display text'),
('launch.waitlist_enabled', 'launch', 'true', false, 'Enable customer waitlist signup'),

('features.trade_lines', 'features', 'false', false, 'Enable trade line accounts and revolving facilities'),
('features.drawdowns', 'features', 'false', false, 'Enable drawdown requests on active facilities'),
('features.repayment_extensions', 'features', 'true', false, 'Enable buyer requested repayment extensions'),
('features.disputes', 'features', 'true', false, 'Enable buyer dispute workflows'),
('features.early_settlement_discounts', 'features', 'false', false, 'Enable early settlement discounts'),
('features.notifications_whatsapp', 'features', 'false', false, 'Enable WhatsApp message delivery'),
('features.mono_direct_debit', 'features', 'false', false, 'Enable Mono direct debit mandate sweeps'),

('integrations.mono.enabled', 'integrations', 'false', false, 'Enable Mono Open Banking integration'),
('integrations.mono.app_id', 'integrations', '""', false, 'Mono App ID'),
('integrations.mono.secret_key', 'integrations', '""', true, 'Mono Secret Key (encrypted)'),
('integrations.mono.public_key', 'integrations', '""', false, 'Mono Public Key'),
('integrations.mono.status', 'integrations', '"unconfigured"', false, 'Mono verification status'),

('integrations.paystack.enabled', 'integrations', 'false', false, 'Enable Paystack payment integration'),
('integrations.paystack.secret_key', 'integrations', '""', true, 'Paystack Secret Key (encrypted)'),
('integrations.paystack.public_key', 'integrations', '""', false, 'Paystack Public Key'),
('integrations.paystack.status', 'integrations', '"unconfigured"', false, 'Paystack verification status'),

('integrations.termii.enabled', 'integrations', 'false', false, 'Enable Termii SMS integration'),
('integrations.termii.api_key', 'integrations', '""', true, 'Termii API Key (encrypted)'),
('integrations.termii.sender_id', 'integrations', '""', false, 'Termii Sender ID'),
('integrations.termii.status', 'integrations', '"unconfigured"', false, 'Termii verification status'),

('integrations.resend.enabled', 'integrations', 'false', false, 'Enable Resend transactional email integration'),
('integrations.resend.api_key', 'integrations', '""', true, 'Resend API Key (encrypted)'),
('integrations.resend.from_email', 'integrations', '""', false, 'Resend From Address'),
('integrations.resend.status', 'integrations', '"unconfigured"', false, 'Resend verification status'),

('security.mfa_enforced', 'security', 'true', false, 'Enforce MFA for platform administrators and financial operations'),
('security.session_idle_minutes', 'security', '15', false, 'Session idle timeout before re-authentication is required'),
('security.max_login_attempts', 'security', '5', false, 'Maximum failed login attempts before throttle'),
('security.ip_allowlist_enabled', 'security', 'false', false, 'Enable IP allowlist for super admin console'),

('kyc.tier1_max_kobo', 'kyc', '50000000', false, 'Tier 1 single obligation limit in kobo'),
('kyc.tier2_bvn_required', 'kyc', 'true', false, 'Require verified BVN for Tier 2 limits'),
('kyc.tier3_cac_required', 'kyc', 'true', false, 'Require verified CAC corporate registration for Tier 3'),

('fees.supplier_rate_bps', 'fees', '350', false, 'Default platform supplier fee rate in basis points (350 = 3.5%)'),
('fees.late_fee_rate_bps', 'fees', '100', false, 'Default late penalty fee rate in basis points (100 = 1.0%)'),
('fees.grace_period_days', 'fees', '3', false, 'Grace period days before late penalty applies'),

('notifications.channels', 'notifications', '["in_app","email"]', false, 'Active notification delivery channels'),
('notifications.pre_debit_reminder_days', 'notifications', '2', false, 'Days prior to due date to send pre-debit notice')
ON CONFLICT (key) DO NOTHING;

-- 6. Row Level Security & Runtime Grants
ALTER TABLE app.platform_governance ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS platform_governance_runtime ON app.platform_governance;
CREATE POLICY platform_governance_runtime ON app.platform_governance
  USING (current_user IN ('kredit_app','kredit_worker'))
  WITH CHECK (current_user = 'kredit_app');

ALTER TABLE app.platform_settings ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS platform_settings_runtime ON app.platform_settings;
CREATE POLICY platform_settings_runtime ON app.platform_settings
  USING (current_user IN ('kredit_app','kredit_worker'))
  WITH CHECK (current_user = 'kredit_app');

ALTER TABLE app.platform_settings_history ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS platform_settings_history_runtime ON app.platform_settings_history;
CREATE POLICY platform_settings_history_runtime ON app.platform_settings_history
  USING (current_user IN ('kredit_app','kredit_worker'))
  WITH CHECK (current_user = 'kredit_app');

-- +goose StatementBegin
DO $$ BEGIN
  IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'kredit_app') THEN
    GRANT SELECT, INSERT, UPDATE, DELETE ON app.platform_governance, app.platform_settings, app.platform_settings_history TO kredit_app;
    GRANT EXECUTE ON FUNCTION app.is_platform_owner(uuid) TO kredit_app;
    GRANT EXECUTE ON FUNCTION app.current_governance_mode() TO kredit_app;
  END IF;
  IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'kredit_worker') THEN
    GRANT SELECT ON app.platform_governance, app.platform_settings TO kredit_worker;
    GRANT EXECUTE ON FUNCTION app.current_governance_mode() TO kredit_worker;
  END IF;
END $$;
-- +goose StatementEnd

-- +goose Down
DROP TRIGGER IF EXISTS protect_last_platform_owner_trigger ON app.platform_role_assignments;
DROP TABLE IF EXISTS app.platform_settings_history;
DROP TABLE IF EXISTS app.platform_settings;
DROP TABLE IF EXISTS app.platform_governance;
DROP FUNCTION IF EXISTS app.current_governance_mode() CASCADE;
DROP FUNCTION IF EXISTS app.protect_last_platform_owner() CASCADE;
DROP FUNCTION IF EXISTS app.is_platform_owner(uuid) CASCADE;
