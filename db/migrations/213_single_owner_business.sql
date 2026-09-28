-- +goose Up
-- Pre-launch schema cleanup: each business has exactly one owner and no staff subsystem.
LOCK TABLE app.memberships IN SHARE ROW EXCLUSIVE MODE;
-- +goose StatementBegin
DO $$ BEGIN
 IF EXISTS(SELECT 1 FROM app.memberships WHERE role<>'owner') THEN
  RAISE EXCEPTION 'Single-owner setup requires owner-only membership data';
 END IF;
 IF EXISTS(SELECT 1 FROM app.memberships GROUP BY organization_id HAVING count(*)>1) THEN
  RAISE EXCEPTION 'Each business must have one owner';
 END IF;
 IF EXISTS(SELECT 1 FROM app.organizations o WHERE o.status NOT IN ('closed','suspended') AND NOT EXISTS(SELECT 1 FROM app.memberships m WHERE m.organization_id=o.id AND m.status='active')) THEN
  RAISE EXCEPTION 'Each active business must have an active owner';
 END IF;
END $$;
-- +goose StatementEnd
ALTER TABLE app.memberships DROP CONSTRAINT memberships_role_check;
ALTER TABLE app.memberships ADD CONSTRAINT business_owner_role CHECK(role='owner');
ALTER TABLE app.memberships DROP CONSTRAINT memberships_status_check;
ALTER TABLE app.memberships ADD CONSTRAINT business_owner_status CHECK(status IN ('active','suspended','removed'));
ALTER TABLE app.memberships ALTER COLUMN status SET DEFAULT 'active';
DROP INDEX app.active_membership_unique;
CREATE UNIQUE INDEX one_business_owner ON app.memberships(organization_id);

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION app.can_purchase(profile_id uuid, action_name text DEFAULT 'read', amount_kobo bigint DEFAULT 0)
RETURNS boolean LANGUAGE sql STABLE SECURITY DEFINER SET search_path=pg_catalog,app SET row_security=off AS $$
 SELECT EXISTS(SELECT 1 FROM app.businesses b
 JOIN app.organizations o ON o.id=b.organization_id
 JOIN app.memberships m ON m.organization_id=o.id AND m.user_id=app.current_user_id()
 JOIN app.users u ON u.id=m.user_id
 WHERE b.id=profile_id AND b.owner_user_id=m.user_id AND m.role='owner'
 AND m.status='active' AND u.status='active' AND o.status NOT IN ('suspended','closed'));
$$;
-- +goose StatementEnd

-- Remove internal staff review. Buyer acceptance, agreement fingerprints,
-- risk holds, ledger controls and platform financial review remain unchanged.
DROP TRIGGER credit_offer_approval ON app.credit_requests;
DROP TRIGGER reviewer_limit_send ON app.credit_requests;
DROP TRIGGER drawdown_approval_guard ON app.drawdowns;

-- Credit notes keep an explicit draft/review/apply step, owned by the business
-- operator. Retain the immutable financial intent and current-owner lock.
ALTER TABLE app.order_credit_notes DROP CONSTRAINT order_credit_notes_check;
ALTER TABLE app.order_credit_notes ADD CONSTRAINT credit_note_confirmation_evidence
 CHECK(status<>'approved' OR (approved_by IS NOT NULL AND approved_at IS NOT NULL));
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION app.guard_order_credit_note() RETURNS trigger
LANGUAGE plpgsql SECURITY INVOKER SET search_path=pg_catalog,app,pg_temp AS $$
DECLARE reviewer uuid;
BEGIN
 IF TG_OP='INSERT' THEN
  IF NEW.status<>'draft' OR NEW.issued_by IS DISTINCT FROM app.current_user_id()
   OR NEW.approved_by IS NOT NULL OR NEW.approved_at IS NOT NULL THEN
   RAISE EXCEPTION 'credit note must start as an unapproved draft' USING ERRCODE='23514';
  END IF;
  RETURN NEW;
 END IF;
 IF (NEW.id,NEW.order_id,NEW.obligation_id,NEW.supplier_organization_id,NEW.amount_kobo,NEW.reason,NEW.issued_by,NEW.created_at)
 IS DISTINCT FROM (OLD.id,OLD.order_id,OLD.obligation_id,OLD.supplier_organization_id,OLD.amount_kobo,OLD.reason,OLD.issued_by,OLD.created_at) THEN
  RAISE EXCEPTION 'credit note intent is immutable' USING ERRCODE='23514';
 END IF;
 IF OLD.status<>'draft' OR NEW.status<>'approved' OR NEW.approved_by IS DISTINCT FROM app.current_user_id()
 OR NEW.approved_at IS NULL THEN
  RAISE EXCEPTION 'current owner confirmation required' USING ERRCODE='42501';
 END IF;
 -- Lock current reviewer membership against concurrent revocation.
 SELECT m.user_id INTO reviewer FROM app.memberships m JOIN app.users u ON u.id=m.user_id
 JOIN app.organizations o ON o.id=m.organization_id
 WHERE m.organization_id=NEW.supplier_organization_id AND m.user_id=app.current_user_id()
 AND m.status='active' AND m.role='owner' AND u.status='active' AND o.status NOT IN ('suspended','closed')
 FOR SHARE OF m,u,o;
 IF NOT FOUND THEN RAISE EXCEPTION 'current credit reviewer required' USING ERRCODE='42501'; END IF;
 RETURN NEW;
END $$;
-- +goose StatementEnd


-- +goose StatementBegin
CREATE OR REPLACE FUNCTION app.lock_purchase_permission(profile_id uuid,action_name text,amount_kobo bigint) RETURNS void
LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,app SET row_security=off AS $$
BEGIN
 PERFORM 1 FROM app.businesses b JOIN app.organizations o ON o.id=b.organization_id
 JOIN app.memberships m ON m.organization_id=o.id AND m.user_id=b.owner_user_id
 JOIN app.users u ON u.id=m.user_id
 WHERE b.id=profile_id AND m.user_id=app.current_user_id() AND m.role='owner'
 AND m.status='active' AND u.status='active' AND o.status NOT IN ('suspended','closed')
 FOR SHARE OF b,o,m,u;
 IF NOT FOUND THEN RAISE EXCEPTION 'current business owner required' USING ERRCODE='42501'; END IF;
END $$;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION app.credit_snapshot_by_id(p_request_id text) RETURNS jsonb LANGUAGE sql SECURITY DEFINER STABLE SET search_path=pg_catalog,app SET row_security=off AS $$
 SELECT aggregate FROM app.credit_aggregate_snapshots WHERE credit_request_id=p_request_id AND ((current_setting('role',true)='kredit_worker' OR (current_setting('role',true)='none' AND pg_has_role(session_user,'kredit_worker','member') AND NOT pg_has_role(session_user,'kredit_app','member'))) OR app.request_authority_current(credit_request_id::uuid)) AND
 (supplier_organization_id=NULLIF(current_setting('app.current_organization_id',true),'') OR buyer_user_id=NULLIF(current_setting('app.current_user_id',true),'') OR app.purchase_request_read(credit_request_id::uuid));
$$;

CREATE OR REPLACE FUNCTION app.credit_snapshot_by_obligation(p_obligation_id text) RETURNS TABLE(credit_request_id text,aggregate jsonb) LANGUAGE sql SECURITY DEFINER STABLE SET search_path=pg_catalog,app SET row_security=off AS $$
 SELECT credit_request_id,aggregate FROM app.credit_aggregate_snapshots WHERE aggregate->'obligation'->>'id'=p_obligation_id AND ((current_setting('role',true)='kredit_worker' OR (current_setting('role',true)='none' AND pg_has_role(session_user,'kredit_worker','member') AND NOT pg_has_role(session_user,'kredit_app','member'))) OR app.request_authority_current(credit_request_id::uuid)) AND
 (supplier_organization_id=NULLIF(current_setting('app.current_organization_id',true),'') OR buyer_user_id=NULLIF(current_setting('app.current_user_id',true),'') OR app.purchase_request_read(credit_request_id::uuid)) LIMIT 1;
$$;

CREATE OR REPLACE FUNCTION app.trade_line_mandate(p_mandate_id uuid,p_buyer_user_id uuid,p_buyer_business_id uuid,p_supplier_organization_id uuid)
RETURNS TABLE(id text,provider text,provider_mandate_id text,buyer_business_id text,buyer_user_id text,supplier_organization_id text,amount_ceiling_kobo bigint,state text,created_at timestamptz)
LANGUAGE sql STABLE SECURITY DEFINER SET search_path=pg_catalog,app SET row_security=off AS $$
 SELECT m.id::text,m.provider,m.provider_mandate_id,m.buyer_subject_id::text,b.owner_user_id::text,m.supplier_organization_id::text,m.amount_ceiling_kobo,m.state,m.created_at FROM app.payment_mandates m JOIN app.businesses b ON b.id=m.buyer_subject_id WHERE m.id=p_mandate_id AND m.buyer_subject_type='business' AND m.buyer_subject_id=p_buyer_business_id AND b.owner_user_id=p_buyer_user_id AND m.supplier_organization_id=p_supplier_organization_id AND ((current_setting('role',true)='kredit_worker' OR (current_setting('role',true)='none' AND pg_has_role(session_user,'kredit_worker','member') AND NOT pg_has_role(session_user,'kredit_app','member'))) OR app.purchasing_authority_current(p_buyer_business_id,p_supplier_organization_id));
$$;

CREATE OR REPLACE FUNCTION app.supplier_customers(p_organization_id UUID)
RETURNS TABLE (
    buyer_user_id UUID,
    buyer_business_id UUID,
    legal_name TEXT,
    trading_name TEXT,
    industry TEXT,
    status TEXT
)
LANGUAGE sql
SECURITY DEFINER
STABLE
SET search_path = app, pg_catalog
AS $$
    SELECT b.owner_user_id, b.id, b.legal_name, b.trading_name, b.industry, b.status
    FROM app.trade_relationships r
    JOIN app.businesses b ON b.id = r.buyer_business_id
    WHERE r.supplier_organization_id = p_organization_id
      AND r.status IN ('active', 'paused')
      AND p_organization_id = NULLIF(current_setting('app.current_organization_id', true), '')::UUID
    ORDER BY b.legal_name
$$;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION app.ensure_business_workspace(profile_id uuid) RETURNS uuid
LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,app SET row_security=off AS $$
DECLARE profile app.businesses; workspace uuid; actor uuid;
BEGIN
 actor := app.current_user_id();
 IF actor IS NULL THEN RAISE EXCEPTION 'business authority required' USING ERRCODE='42501'; END IF;
 SELECT * INTO profile FROM app.businesses WHERE id=profile_id AND owner_user_id=actor FOR UPDATE;
 IF NOT FOUND THEN RAISE EXCEPTION 'business authority required' USING ERRCODE='42501'; END IF;
 IF profile.status IN ('suspended','closed') THEN RAISE EXCEPTION 'business is restricted' USING ERRCODE='42501'; END IF;
 IF profile.organization_id IS NOT NULL THEN
  IF NOT EXISTS(SELECT 1 FROM app.memberships WHERE organization_id=profile.organization_id AND user_id=actor AND status='active' AND role='owner') THEN
   RAISE EXCEPTION 'business authority required' USING ERRCODE='42501';
  END IF;
  RETURN profile.organization_id;
 END IF;
 workspace := profile.id;
 -- Never attach a colliding identifier to an unrelated existing workspace.
 INSERT INTO app.organizations(id,legal_name,trading_name,business_type,registration_info,business_address,industry,status)
 VALUES(workspace,profile.legal_name,profile.trading_name,profile.business_type,profile.registration_info,profile.business_address,profile.industry,'onboarding');
 INSERT INTO app.memberships(organization_id,user_id,role,status) VALUES(workspace,actor,'owner','active');
 UPDATE app.businesses SET organization_id=workspace WHERE id=profile_id;
 RETURN workspace;
END $$;

CREATE OR REPLACE FUNCTION app.provision_purchasing_workspace(profile_id uuid) RETURNS void
LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,app AS $$
DECLARE b app.businesses; inserted_profile app.supplier_onboarding_profiles;
BEGIN
 SELECT * INTO b FROM app.businesses WHERE id=profile_id FOR UPDATE;
 IF NOT FOUND OR b.organization_id IS NOT NULL THEN RETURN; END IF;
 IF b.owner_user_id IS NULL THEN RAISE EXCEPTION 'purchasing profile requires a verified ownership mapping before consolidation' USING ERRCODE='23514'; END IF;
 IF EXISTS(SELECT 1 FROM app.organizations WHERE id=b.id) THEN RAISE EXCEPTION 'business identity collision requires explicit reconciliation' USING ERRCODE='23514'; END IF;
 INSERT INTO app.organizations(id,legal_name,trading_name,business_type,registration_info,business_address,industry,status)
 VALUES(b.id,b.legal_name,b.trading_name,b.business_type,b.registration_info,b.business_address,b.industry,'onboarding');
 INSERT INTO app.memberships(organization_id,user_id,role,status) VALUES(b.id,b.owner_user_id,'owner','active');
 UPDATE app.businesses SET organization_id=b.id WHERE id=b.id;
 INSERT INTO app.supplier_onboarding_profiles(organization_id) VALUES(b.id) RETURNING * INTO inserted_profile;
 INSERT INTO app.supplier_onboarding_revisions(id,organization_id,profile_version,change_type,actor_user_id,actor_reference,snapshot)
 VALUES(gen_random_uuid(),b.id,inserted_profile.version,'profile.created',b.owner_user_id,b.owner_user_id::text,to_jsonb(inserted_profile));
END $$;

CREATE OR REPLACE FUNCTION app.purchasing_authority_current(profile_id uuid,supplier_id uuid DEFAULT NULL) RETURNS boolean
LANGUAGE sql STABLE SECURITY DEFINER SET search_path=pg_catalog,app AS $$
 SELECT NOT EXISTS(SELECT 1 FROM app.businesses b WHERE b.id=profile_id AND b.organization_id IS NOT NULL
 AND (b.owner_user_id=app.current_user_id()
 OR EXISTS(SELECT 1 FROM app.business_representatives r JOIN app.persons p ON p.id=r.person_id WHERE r.business_id=b.id AND p.user_id=app.current_user_id()))
 AND NOT app.can_purchase(b.id)
 AND NOT EXISTS(SELECT 1 FROM app.memberships m JOIN app.users u ON u.id=m.user_id JOIN app.organizations o ON o.id=m.organization_id WHERE m.organization_id=supplier_id AND supplier_id=app.current_organization_id() AND m.user_id=app.current_user_id() AND m.status='active' AND u.status='active' AND o.status<>'suspended'));
$$;

CREATE OR REPLACE FUNCTION app.lock_buyer_payment_claim(p_obligation_id uuid)
RETURNS TABLE(outstanding_kobo bigint,supplier_organization_id uuid,currency character(3))
LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,app,pg_temp AS $$
DECLARE actor uuid:=app.current_user_id(); profile uuid; owner_id uuid;
BEGIN
 PERFORM 1 FROM app.users WHERE id=actor AND status='active' FOR SHARE;
 IF NOT FOUND THEN RAISE EXCEPTION 'payment claim obligation is unavailable' USING ERRCODE='42501'; END IF;
 SELECT o.buyer_business_id,c.buyer_user_id INTO profile,owner_id
 FROM app.obligations o JOIN app.credit_requests c ON c.id=o.credit_request_id
 WHERE o.id=p_obligation_id AND c.buyer_business_id=o.buyer_business_id
 AND c.supplier_organization_id=o.supplier_organization_id;
 IF NOT FOUND OR NOT app.obligation_authority_current(p_obligation_id) THEN
  RAISE EXCEPTION 'payment claim obligation is unavailable' USING ERRCODE='42501';
 END IF;
 IF owner_id=actor THEN
  PERFORM app.lock_obligation_authority(p_obligation_id);
 ELSE
  PERFORM app.lock_purchase_permission(profile,'claim',0);
 END IF;
 RETURN QUERY SELECT o.outstanding_kobo,o.supplier_organization_id,o.currency
 FROM app.obligations o JOIN app.credit_requests c ON c.id=o.credit_request_id
 WHERE o.id=p_obligation_id AND o.buyer_business_id=profile AND c.buyer_business_id=profile
 AND c.supplier_organization_id=o.supplier_organization_id
 AND (c.buyer_user_id=actor OR app.can_purchase(profile,'claim',0))
 AND app.obligation_authority_current(o.id)
 FOR UPDATE OF o;
 IF NOT FOUND THEN RAISE EXCEPTION 'payment claim obligation is unavailable' USING ERRCODE='42501'; END IF;
END $$;

CREATE OR REPLACE FUNCTION app.idempotency_authority_version(actor uuid) RETURNS text
LANGUAGE sql STABLE SECURITY DEFINER SET search_path=pg_catalog,app,pg_temp SET row_security=off AS $$
 SELECT encode(sha256(convert_to(jsonb_build_object(
 'memberships',(SELECT jsonb_agg(jsonb_build_array(m.id,m.organization_id,m.user_id,m.role,m.status,o.status) ORDER BY m.id)
 FROM app.memberships m JOIN app.organizations o ON o.id=m.organization_id WHERE m.user_id=actor),
 'platform',(SELECT jsonb_agg(jsonb_build_array(a.id,a.role,a.revoked_at,a.expires_at IS NULL OR a.expires_at>statement_timestamp()) ORDER BY a.id)
 FROM app.platform_role_assignments a WHERE a.user_id=actor)
 )::text,'UTF8')),'hex');
$$;
-- +goose StatementEnd

-- Remove branch-only restrictions and triggers; tenant, owner, purchaser and ledger policies remain.
-- +goose StatementBegin
DO $$ DECLARE item record; BEGIN
 FOR item IN SELECT schemaname,tablename,policyname FROM pg_policies WHERE schemaname='app' AND policyname='branch_boundary' LOOP
  EXECUTE format('DROP POLICY %I ON %I.%I',item.policyname,item.schemaname,item.tablename);
 END LOOP;
 FOR item IN SELECT n.nspname,c.relname,t.tgname FROM pg_trigger t JOIN pg_class c ON c.oid=t.tgrelid JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname='app' AND t.tgname='branch_financial_write' LOOP
  EXECUTE format('DROP TRIGGER %I ON %I.%I',item.tgname,item.nspname,item.relname);
 END LOOP;
END $$;
-- +goose StatementEnd
DROP TRIGGER purchasing_membership_authority ON app.memberships;
DROP TRIGGER membership_invitation_lifecycle ON app.memberships;
ALTER TABLE app.drawdowns DROP COLUMN branch_id;
DROP TABLE app.partner_terms_import_rows, app.partner_terms_import_batches;
DROP TABLE app.purchasing_delegations, app.purchasing_delegation_history;
DROP TABLE app.member_branch_scopes, app.member_branch_scope_history;
DROP TABLE app.partner_assignments, app.business_branches, app.network_operation_history;
DROP TABLE app.credit_offer_approvals, app.tradeline_drawdown_approvals;
DROP TABLE app.business_credit_controls, app.business_credit_control_history;
DROP TABLE app.credit_reviewer_limits, app.credit_reviewer_limit_history;
DROP TABLE app.organization_invitations;
DROP TABLE app.runtime_process_status;
ALTER TABLE app.supplier_onboarding_profiles DROP COLUMN finance_mfa_complete;
ALTER TABLE app.memberships DROP COLUMN purchasing_authority_version, DROP COLUMN invited_by, DROP COLUMN invited_at, DROP COLUMN accepted_at;
DROP FUNCTION app.close_membership_invitation(), app.guard_terms_import_review(), app.guard_terms_import_row(), app.guard_drawdown_approval_evidence(),
 app.guard_branch_financial_write(), app.lock_branch_customer(uuid,uuid),
 app.branch_drawdown_access(uuid), app.branch_line_access(uuid), app.branch_obligation_access(uuid),
 app.branch_credit_access(uuid), app.branch_customer_access(uuid,uuid), app.branch_scope_all(uuid),
 app.record_branch_scope(), app.append_branch_scope_history(), app.record_network_operation(),
 app.record_purchasing_delegation(), app.bind_purchasing_delegation_membership(), app.advance_purchasing_membership_authority(),
 app.guard_credit_offer_approval(), app.guard_credit_approval_evidence(), app.record_credit_control_version(),
 app.record_credit_reviewer_limit(), app.enforce_credit_reviewer_limit(), app.enforce_offer_reviewer_limit(),
 app.guard_drawdown_approval(), app.drawdown_approval_fingerprint(app.drawdowns), app.drawdown_approval_proposal(app.drawdowns),
 app.credit_offer_fingerprint(app.credit_requests), app.credit_offer_proposal(app.credit_requests);

-- Owner purchase policies use the existing scoped checks, with no staff permission table.
-- +goose StatementBegin
DO $$ DECLARE item record; BEGIN
 FOR item IN SELECT schemaname,tablename,policyname FROM pg_policies WHERE schemaname='app' AND policyname LIKE 'delegated\_%' ESCAPE '\' LOOP
  EXECUTE format('ALTER POLICY %I ON %I.%I RENAME TO %I',item.policyname,item.schemaname,item.tablename,replace(item.policyname,'delegated_','owner_'));
 END LOOP;
END $$;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DO $$ BEGIN RAISE EXCEPTION 'Single-owner schema requires forward recovery'; END $$;
-- +goose StatementEnd
