-- +goose Up
CREATE TABLE app.dsa_program (
 id integer PRIMARY KEY CHECK(id=1),version bigint NOT NULL DEFAULT 1,enabled boolean NOT NULL DEFAULT true,
 onboarding_kobo bigint NOT NULL DEFAULT 100000 CHECK(onboarding_kobo BETWEEN 0 AND 1000000),
 activation_kobo bigint NOT NULL DEFAULT 200000 CHECK(activation_kobo BETWEEN 0 AND 10000000),
 threshold_kobo bigint NOT NULL DEFAULT 500000 CHECK(threshold_kobo BETWEEN 10000 AND 100000000),
 share_bps integer NOT NULL DEFAULT 1000 CHECK(share_bps BETWEEN 0 AND 5000),
 share_months integer NOT NULL DEFAULT 6 CHECK(share_months BETWEEN 1 AND 12),
 initial_limit integer NOT NULL DEFAULT 20 CHECK(initial_limit BETWEEN 1 AND 1000)
);
INSERT INTO app.dsa_program(id) VALUES(1);
CREATE TABLE app.dsa_agents (
 user_id uuid PRIMARY KEY REFERENCES app.users(id),code text NOT NULL UNIQUE,
 name text NOT NULL,phone text NOT NULL,bank_name text NOT NULL,account_name text NOT NULL,account_number text NOT NULL CHECK(account_number ~ '^[0-9]{10}$'),
 status text NOT NULL DEFAULT 'active' CHECK(status IN ('active','suspended')),onboarding_limit integer NOT NULL CHECK(onboarding_limit BETWEEN 0 AND 100000),
 terms_version text NOT NULL,terms_accepted_at timestamptz NOT NULL DEFAULT now(),bank_updated_at timestamptz NOT NULL DEFAULT now(),created_at timestamptz NOT NULL DEFAULT now(),version bigint NOT NULL DEFAULT 1
);
CREATE TABLE app.dsa_referrals (
 organization_id uuid PRIMARY KEY REFERENCES app.organizations(id),agent_id uuid NOT NULL REFERENCES app.dsa_agents(user_id),
 business_name text NOT NULL,confirmed_by uuid NOT NULL REFERENCES app.users(id),created_at timestamptz NOT NULL DEFAULT now(),
 terms jsonb NOT NULL,registration_key text UNIQUE,qualified_at timestamptz,activated_at timestamptz,share_ends_at timestamptz,
 activation_baseline_kobo bigint NOT NULL DEFAULT 0,onboarding_slot boolean NOT NULL DEFAULT false,
 blocked boolean NOT NULL DEFAULT false,reason text NOT NULL DEFAULT '',progress text NOT NULL DEFAULT 'Complete business verification and bank setup',
 fees_kobo bigint NOT NULL DEFAULT 0,checked_at timestamptz,version bigint NOT NULL DEFAULT 1
);
CREATE INDEX dsa_referrals_agent ON app.dsa_referrals(agent_id,organization_id);
CREATE TABLE app.dsa_earnings (
 id uuid PRIMARY KEY DEFAULT uuidv7(),organization_id uuid NOT NULL REFERENCES app.dsa_referrals(organization_id),agent_id uuid NOT NULL REFERENCES app.dsa_agents(user_id),
 kind text NOT NULL CHECK(kind IN ('onboarding','activation','share')),amount_kobo bigint NOT NULL CHECK(amount_kobo<>0),
 reason text NOT NULL,available_at timestamptz NOT NULL,created_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE app.dsa_payouts (
 id uuid PRIMARY KEY DEFAULT uuidv7(),agent_id uuid NOT NULL REFERENCES app.dsa_agents(user_id),amount_kobo bigint NOT NULL CHECK(amount_kobo>0),
 bank_name text NOT NULL,account_name text NOT NULL,account_number text NOT NULL,bank_updated_at timestamptz NOT NULL,
 state text NOT NULL DEFAULT 'pending' CHECK(state IN ('pending','paid','cancelled')),bank_reference text UNIQUE,evidence text NOT NULL DEFAULT '',
 created_by uuid NOT NULL REFERENCES app.users(id),completed_by uuid REFERENCES app.users(id),created_at timestamptz NOT NULL DEFAULT now(),completed_at timestamptz,version bigint NOT NULL DEFAULT 1
);
CREATE UNIQUE INDEX dsa_one_pending_payout ON app.dsa_payouts(agent_id) WHERE state='pending';
ALTER TABLE app.dsa_program ENABLE ROW LEVEL SECURITY;
ALTER TABLE app.dsa_agents ENABLE ROW LEVEL SECURITY;
ALTER TABLE app.dsa_referrals ENABLE ROW LEVEL SECURITY;
ALTER TABLE app.dsa_earnings ENABLE ROW LEVEL SECURITY;
ALTER TABLE app.dsa_payouts ENABLE ROW LEVEL SECURITY;
CREATE POLICY dsa_program_read ON app.dsa_program FOR SELECT USING(true);
CREATE POLICY dsa_program_owner ON app.dsa_program FOR UPDATE USING(app.has_admin_role(app.current_user_id(),ARRAY['platform_owner']));
CREATE POLICY dsa_agent_read ON app.dsa_agents FOR SELECT USING(user_id=app.current_user_id() OR app.has_admin_role(app.current_user_id(),ARRAY['platform_owner']));
CREATE POLICY dsa_agent_create ON app.dsa_agents FOR INSERT WITH CHECK(user_id=app.current_user_id());
CREATE POLICY dsa_agent_update ON app.dsa_agents FOR UPDATE USING(user_id=app.current_user_id() OR app.has_admin_role(app.current_user_id(),ARRAY['platform_owner']));
CREATE POLICY dsa_referral_read ON app.dsa_referrals FOR SELECT USING(agent_id=app.current_user_id() OR confirmed_by=app.current_user_id() OR app.has_admin_role(app.current_user_id(),ARRAY['platform_owner']));
CREATE POLICY dsa_referral_create ON app.dsa_referrals FOR INSERT WITH CHECK(confirmed_by=app.current_user_id() AND EXISTS(SELECT 1 FROM app.memberships WHERE organization_id=dsa_referrals.organization_id AND user_id=app.current_user_id() AND status='active' AND role='owner'));
CREATE POLICY dsa_referral_update ON app.dsa_referrals FOR UPDATE USING(app.has_admin_role(app.current_user_id(),ARRAY['platform_owner']));
CREATE POLICY dsa_earnings_read ON app.dsa_earnings FOR SELECT USING(agent_id=app.current_user_id() OR app.has_admin_role(app.current_user_id(),ARRAY['platform_owner']));
CREATE POLICY dsa_payout_read ON app.dsa_payouts FOR SELECT USING(agent_id=app.current_user_id() OR app.has_admin_role(app.current_user_id(),ARRAY['platform_owner']));
CREATE POLICY dsa_payout_owner ON app.dsa_payouts FOR ALL USING(app.has_admin_role(app.current_user_id(),ARRAY['platform_owner']));
-- Worker writes are constrained to the referral engine, not merchants or agents.
CREATE POLICY dsa_agent_worker ON app.dsa_agents FOR ALL USING(current_user='kredit_worker');
CREATE POLICY dsa_referral_worker ON app.dsa_referrals FOR ALL USING(current_user='kredit_worker');
CREATE POLICY dsa_earnings_worker ON app.dsa_earnings FOR ALL USING(current_user='kredit_worker');
CREATE POLICY dsa_payout_worker ON app.dsa_payouts FOR SELECT USING(current_user='kredit_worker');
CREATE POLICY dsa_earnings_owner ON app.dsa_earnings FOR INSERT WITH CHECK(app.has_admin_role(app.current_user_id(),ARRAY['platform_owner']));
INSERT INTO ledger.accounts(code,name,normal_balance) VALUES
 ('DSA_ACQUISITION_EXPENSE','DSA acquisition rewards','debit'),('DSA_COMMISSION_PAYABLE','DSA commissions payable','credit'),('DSA_PAYOUT_CASH','Completed DSA bank payouts','debit') ON CONFLICT(code) DO NOTHING;
-- Only this small public lookup is exposed before referral confirmation.
-- +goose StatementBegin
CREATE FUNCTION app.dsa_code(code_arg text) RETURNS jsonb LANGUAGE sql STABLE SECURITY DEFINER SET search_path=pg_catalog,app AS $$
 SELECT jsonb_build_object('code',a.code,'name',a.name) FROM app.dsa_agents a,app.dsa_program p WHERE a.code=upper(code_arg) AND a.status='active' AND p.enabled AND EXISTS(SELECT 1 FROM app.users u WHERE u.id=a.user_id AND u.status='active')
$$;
-- +goose StatementEnd
-- +goose StatementBegin
CREATE FUNCTION app.dsa_registration(value text,business_type text) RETURNS text LANGUAGE plpgsql IMMUTABLE SET search_path=pg_catalog AS $$
DECLARE normalized text; prefix text; digits text;
BEGIN
 normalized:=regexp_replace(upper(COALESCE(value,'')),'[[:space:]-]','','g');
 IF normalized ~ '^[0-9]+$' THEN prefix:=CASE WHEN business_type IN ('registered_business','sole_proprietor') THEN 'BN' ELSE 'RC' END; digits:=normalized;
 ELSIF normalized ~ '^(RC|BN|IT|LP|LLP)[0-9]+$' THEN prefix:=substring(normalized FROM '^[A-Z]+');digits:=substring(normalized FROM '[0-9]+$');
 ELSE RETURN ''; END IF;
 digits:=ltrim(digits,'0');IF digits='' THEN RETURN '';END IF;
 RETURN prefix||digits;
END $$;
-- +goose StatementEnd
-- CAC identity and fee records stay private. Only the engine can read these facts.
-- +goose StatementBegin
CREATE FUNCTION app.dsa_facts(org uuid,started timestamptz,cutoff timestamptz) RETURNS jsonb LANGUAGE sql STABLE SECURITY DEFINER SET search_path=pg_catalog,app AS $$
 WITH invoice_caps AS (
 SELECT i.id,GREATEST(0,COALESCE(sum(LEAST(l.amount_kobo,GREATEST(0,f.amount_kobo-f.waived_kobo-LEAST(l.collected_at_issue_kobo,f.collected_kobo)))) FILTER(WHERE f.state IN ('accrued','paid') AND f.fee_type IN ('base_service','collection')),0)) AS cap
 FROM app.fee_invoices i JOIN app.fee_invoice_lines l ON l.invoice_id=i.id JOIN app.fees f ON f.id=l.fee_id
 WHERE i.organization_id=org AND i.issued_at>=started AND i.issued_at<=cutoff GROUP BY i.id
 ), invoice_cash AS (
 SELECT c.id,c.cap,GREATEST(0,LEAST(c.cap,COALESCE(sum(CASE WHEN r.direction='received' AND r.created_at<=cutoff THEN r.amount_kobo WHEN r.direction='refunded' THEN -r.amount_kobo ELSE 0 END),0))) cash
 FROM invoice_caps c LEFT JOIN app.fee_invoice_receipts r ON r.invoice_id=c.id GROUP BY c.id,c.cap
 ), provider_allocations AS (
 SELECT p.provider,SUM(LEAST(x.amount_kobo,GREATEST(0,f.amount_kobo-f.waived_kobo))) amount
 FROM app.split_fee_allocations x JOIN app.payments p ON p.id=x.payment_id JOIN app.fees f ON f.id=x.fee_id
 WHERE x.supplier_organization_id=org AND p.state='recognized' AND p.recognized_at>=started AND p.recognized_at<=cutoff AND f.state IN ('accrued','paid') AND f.fee_type IN ('base_service','collection') GROUP BY p.provider
 UNION ALL
 SELECT a.provider,SUM(LEAST(d.amount_kobo,GREATEST(0,c.cap-c.cash))) FROM app.fee_debits d JOIN app.fee_authorizations a ON a.id=d.authorization_id JOIN invoice_cash c ON c.id=d.invoice_id
 WHERE d.organization_id=org AND d.state='succeeded' AND NOT d.review_required AND d.created_at<=cutoff GROUP BY a.provider
 ), provider_cash AS (
 SELECT p.provider,GREATEST(0,LEAST(SUM(p.amount),(SELECT COALESCE(sum(CASE WHEN b.direction='received' AND b.created_at<=cutoff THEN b.amount_kobo WHEN b.direction='returned' THEN -b.amount_kobo ELSE 0 END),0) FROM app.fee_bank_receipts b WHERE b.organization_id=org AND b.provider=p.provider AND b.created_at>=started))) cash FROM provider_allocations p GROUP BY p.provider
 )
 SELECT jsonb_build_object('registration',app.dsa_registration(o.registration_info #>> '{}',o.business_type),
 'ready',COALESCE((EXISTS(SELECT 1 FROM app.native_identity_sessions n WHERE n.subject_id=org AND n.kind='business' AND n.state='verified' AND n.safe_result->>'cac_status'='verified' AND (n.expires_at IS NULL OR n.expires_at>now())) OR EXISTS(SELECT 1 FROM app.verification_cases v WHERE v.subject_id=org AND v.state='verified' AND v.safe_result->>'cac_status'='verified' AND (v.expires_at IS NULL OR v.expires_at>now()))) AND p.kyb_state='approved' AND (p.kyb_expires_at IS NULL OR p.kyb_expires_at>now()) AND p.settlement_state='verified' AND (p.owner_email_verified_at IS NOT NULL OR p.owner_phone_verified_at IS NOT NULL) AND o.status NOT IN ('suspended','closed'),false),
 'accepted',EXISTS(SELECT 1 FROM app.agreement_acceptances accepted JOIN app.credit_requests c ON c.id=accepted.credit_request_id WHERE c.supplier_organization_id=org AND accepted.accepted_at>=started) OR EXISTS(SELECT 1 FROM app.consumer_sales s WHERE s.organization_id=org AND s.accepted_at>=started AND s.state NOT IN ('cancelled','returned')),
 'self_referral',EXISTS(SELECT 1 FROM app.dsa_referrals r JOIN app.memberships m ON m.organization_id=r.organization_id AND m.user_id=r.agent_id WHERE r.organization_id=org),
 'fees_kobo',(COALESCE((SELECT SUM(cash) FROM invoice_cash),0)+COALESCE((SELECT SUM(cash) FROM provider_cash),0))::bigint)
 FROM app.organizations o LEFT JOIN app.supplier_onboarding_profiles p ON p.organization_id=o.id WHERE o.id=org AND (app.has_admin_role(app.current_user_id(),ARRAY['platform_owner']) OR current_setting('role',true)='kredit_worker' OR session_user='kredit_worker')
$$;
-- +goose StatementEnd
REVOKE ALL ON FUNCTION app.dsa_code(text),app.dsa_facts(uuid,timestamptz,timestamptz) FROM PUBLIC;
-- +goose StatementBegin
DO $$ BEGIN
 IF EXISTS(SELECT 1 FROM pg_roles WHERE rolname='kredit_app') THEN
 GRANT SELECT ON app.dsa_program,app.dsa_agents,app.dsa_referrals,app.dsa_earnings,app.dsa_payouts TO kredit_app;
 GRANT INSERT,UPDATE ON app.dsa_agents,app.dsa_referrals,app.dsa_payouts TO kredit_app;
 GRANT INSERT ON app.dsa_earnings TO kredit_app;
 GRANT UPDATE ON app.dsa_program TO kredit_app;
 GRANT EXECUTE ON FUNCTION app.dsa_code(text),app.dsa_facts(uuid,timestamptz,timestamptz) TO kredit_app;
 END IF;
 IF EXISTS(SELECT 1 FROM pg_roles WHERE rolname='kredit_worker') THEN
 GRANT SELECT ON app.dsa_program,app.dsa_agents,app.dsa_referrals,app.dsa_earnings,app.dsa_payouts TO kredit_worker;
 GRANT INSERT ON app.dsa_earnings TO kredit_worker;
 GRANT UPDATE ON app.dsa_agents,app.dsa_referrals TO kredit_worker;
 GRANT EXECUTE ON FUNCTION app.dsa_facts(uuid,timestamptz,timestamptz) TO kredit_worker;
 END IF;
END $$;
-- +goose StatementEnd
-- +goose StatementBegin
CREATE FUNCTION app.dsa_claim(org uuid,code_arg text) RETURNS void LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,app AS $$
DECLARE a app.dsa_agents; p app.dsa_program; o app.organizations;
BEGIN
 SELECT * INTO p FROM app.dsa_program WHERE id=1 FOR SHARE;
 IF NOT p.enabled THEN RAISE EXCEPTION 'Referral enrolment is paused'; END IF;
 SELECT * INTO a FROM app.dsa_agents WHERE code=upper(code_arg) AND status='active' FOR UPDATE;
 IF NOT FOUND THEN RAISE EXCEPTION 'Referral code is unavailable'; END IF;
 SELECT * INTO o FROM app.organizations WHERE id=org FOR SHARE;
 IF NOT EXISTS(SELECT 1 FROM app.memberships m JOIN app.users u ON u.id=m.user_id WHERE m.organization_id=org AND m.user_id=app.current_user_id() AND m.role='owner' AND m.status='active' AND u.status='active' FOR SHARE OF m,u) THEN RAISE EXCEPTION 'The business owner must confirm the referral'; END IF;
 IF EXISTS(SELECT 1 FROM app.dsa_referrals WHERE organization_id=org AND agent_id=a.user_id AND confirmed_by=app.current_user_id()) THEN RETURN; END IF;
 IF o.created_at<now()-interval '30 days' OR EXISTS(SELECT 1 FROM app.agreement_acceptances accepted JOIN app.credit_requests c ON c.id=accepted.credit_request_id WHERE c.supplier_organization_id=org) OR EXISTS(SELECT 1 FROM app.consumer_sales WHERE organization_id=org AND accepted_at IS NOT NULL) THEN RAISE EXCEPTION 'Only new businesses within 30 days, before their first accepted sale, qualify'; END IF;
 IF EXISTS(SELECT 1 FROM app.memberships WHERE organization_id=org AND user_id=a.user_id) THEN RAISE EXCEPTION 'You cannot refer a business you belong to'; END IF;
 IF NOT EXISTS(SELECT 1 FROM app.users WHERE id=a.user_id AND status='active') THEN RAISE EXCEPTION 'Referral agent is unavailable'; END IF;
 INSERT INTO app.dsa_referrals(organization_id,agent_id,business_name,confirmed_by,terms) VALUES(org,a.user_id,o.legal_name,app.current_user_id(),to_jsonb(p));
 INSERT INTO app.audit_events(actor_user_id,organization_id,action,resource_type,resource_id,outcome,severity,metadata) VALUES(app.current_user_id(),org,'dsa.referral.confirmed','organization',org::text,'success','high',jsonb_build_object('agent_id',a.user_id,'terms',to_jsonb(p)));
END $$;
-- +goose StatementEnd
REVOKE ALL ON FUNCTION app.dsa_claim(uuid,text) FROM PUBLIC;
-- +goose StatementBegin
DO $$ BEGIN IF EXISTS(SELECT 1 FROM pg_roles WHERE rolname='kredit_app') THEN GRANT EXECUTE ON FUNCTION app.dsa_claim(uuid,text) TO kredit_app; END IF; END $$;
-- +goose StatementEnd
-- +goose StatementBegin
CREATE FUNCTION app.dsa_agent_active(agent uuid) RETURNS boolean LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,app AS $$
DECLARE active boolean;
BEGIN
 IF NOT app.has_admin_role(app.current_user_id(),ARRAY['platform_owner']) THEN RAISE EXCEPTION 'Super Admin required'; END IF;
 SELECT status='active' INTO active FROM app.users WHERE id=agent FOR SHARE;
 RETURN COALESCE(active,false);
END $$;
-- +goose StatementEnd
REVOKE ALL ON FUNCTION app.dsa_agent_active(uuid) FROM PUBLIC;
-- +goose StatementBegin
DO $$ BEGIN IF EXISTS(SELECT 1 FROM pg_roles WHERE rolname='kredit_app') THEN GRANT EXECUTE ON FUNCTION app.dsa_agent_active(uuid) TO kredit_app; END IF; END $$;
-- +goose StatementEnd
-- +goose StatementBegin
CREATE FUNCTION app.guard_dsa_referral() RETURNS trigger LANGUAGE plpgsql SET search_path=pg_catalog,app AS $$
BEGIN
 IF (to_jsonb(NEW)-ARRAY['registration_key','qualified_at','activated_at','share_ends_at','activation_baseline_kobo','onboarding_slot','blocked','reason','progress','fees_kobo','checked_at','version']) IS DISTINCT FROM (to_jsonb(OLD)-ARRAY['registration_key','qualified_at','activated_at','share_ends_at','activation_baseline_kobo','onboarding_slot','blocked','reason','progress','fees_kobo','checked_at','version']) THEN RAISE EXCEPTION 'Referral attribution and agreed terms are immutable'; END IF;
 IF OLD.registration_key IS NOT NULL AND (NEW.registration_key IS DISTINCT FROM OLD.registration_key OR NEW.qualified_at IS DISTINCT FROM OLD.qualified_at) THEN RAISE EXCEPTION 'Qualified CAC identity is immutable'; END IF;
 IF OLD.activated_at IS NOT NULL AND (NEW.activated_at IS DISTINCT FROM OLD.activated_at OR NEW.share_ends_at IS DISTINCT FROM OLD.share_ends_at OR NEW.activation_baseline_kobo<>OLD.activation_baseline_kobo) THEN RAISE EXCEPTION 'Activation window and baseline are immutable'; END IF;
 IF OLD.onboarding_slot AND NOT NEW.onboarding_slot THEN RAISE EXCEPTION 'Used onboarding slots cannot be recycled'; END IF;
 RETURN NEW;
END $$;
-- +goose StatementEnd
CREATE TRIGGER dsa_referral_immutable BEFORE UPDATE ON app.dsa_referrals FOR EACH ROW EXECUTE FUNCTION app.guard_dsa_referral();
-- +goose StatementBegin
CREATE FUNCTION app.guard_dsa_agent() RETURNS trigger LANGUAGE plpgsql SET search_path=pg_catalog,app AS $$
BEGIN
 IF NEW.user_id<>OLD.user_id OR NEW.code<>OLD.code OR NEW.created_at<>OLD.created_at OR NEW.terms_version<>OLD.terms_version OR NEW.terms_accepted_at<>OLD.terms_accepted_at THEN RAISE EXCEPTION 'Agent identity and enrolment terms are immutable'; END IF;
 IF (NEW.status<>OLD.status OR NEW.onboarding_limit<>OLD.onboarding_limit) AND NOT app.has_admin_role(app.current_user_id(),ARRAY['platform_owner']) AND current_user IN ('kredit_app','kredit_worker') THEN RAISE EXCEPTION 'Super Admin required for agent status and limits'; END IF;
 RETURN NEW;
END $$;
-- +goose StatementEnd
CREATE TRIGGER dsa_agent_guard BEFORE UPDATE ON app.dsa_agents FOR EACH ROW EXECUTE FUNCTION app.guard_dsa_agent();
-- +goose Down
-- +goose StatementBegin
DO $$ BEGIN RAISE EXCEPTION 'DSA financial history requires forward recovery'; END $$;
-- +goose StatementEnd
