BEGIN;

INSERT INTO app_meta (key, value)
VALUES ('seed_dataset', 'milestone-1-auth-org')
ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value, updated_at = NOW();

INSERT INTO app.users (id, normalized_email, normalized_phone, display_name, status)
VALUES
    ('00000000-0000-7000-8000-000000000001', 'owner@abc-pharmaceuticals.test', '+2348000000001', 'ABC Supplier Owner', 'active'),
    ('00000000-0000-7000-8000-000000000002', 'finance@abc-pharmaceuticals.test', '+2348000000002', 'ABC Finance Officer', 'active'),
    ('00000000-0000-7000-8000-000000000003', 'sales@abc-pharmaceuticals.test', '+2348000000003', 'ABC Sales Representative', 'active'),
    ('00000000-0000-7000-8000-000000000004', 'buyer@royal-pharmacy.test', '+2348000000004', 'Royal Pharmacy Representative', 'active')
ON CONFLICT (id) DO NOTHING;

INSERT INTO app.organizations (id, legal_name, trading_name, business_type, registration_info, business_address, industry, status)
VALUES ('00000000-0000-7000-8000-000000000010', 'ABC Pharmaceuticals Ltd', 'ABC Pharmaceuticals', 'limited_company', '{"number":"DEMO-REG-001"}'::jsonb, 'Lagos, Nigeria', 'pharmaceuticals', 'onboarding')
ON CONFLICT (id) DO NOTHING;

INSERT INTO app.mfa_methods (id, user_id, method_type, credential_reference, verified_at)
VALUES
    ('00000000-0000-7000-8000-000000000014', '00000000-0000-7000-8000-000000000001', 'passkey', 'development-owner-mfa', NOW()),
    ('00000000-0000-7000-8000-000000000015', '00000000-0000-7000-8000-000000000002', 'passkey', 'development-finance-mfa', NOW())
ON CONFLICT (id) DO NOTHING;

INSERT INTO app.supplier_onboarding_profiles (
    organization_id, authorized_representative_name, authorized_representative_title,
    owner_email_verified_at, owner_phone_verified_at, kyb_state, kyb_provider_reference,
    kyb_submitted_at, kyb_decided_at, kyb_expires_at, settlement_state, settlement_provider,
    settlement_provider_reference, settlement_bank_name, settlement_account_name,
    settlement_account_last4, settlement_changed_at, billing_state, billing_method,
    billing_provider_reference, billing_cycle, billing_changed_at, default_credit_limit_kobo,
    default_payment_days, default_grace_hours, default_credit_policy_updated_at, terms_version,
    terms_accepted_at, terms_accepted_by, privacy_version, privacy_accepted_at,
    privacy_accepted_by, owner_mfa_verified_at, finance_mfa_complete, readiness_state,
    readiness_changed_at
)
VALUES (
    '00000000-0000-7000-8000-000000000010', 'Ada Okafor', 'Managing Director',
    NOW(), NOW(), 'approved', 'demo-kyb-approved', NOW(), NOW(), NOW() + INTERVAL '1 year',
    'verified', 'mock-settlement', 'demo-settlement-destination', 'Demo Bank',
    'ABC Pharmaceuticals Ltd', '0001', NOW(), 'configured', 'split_settlement',
    'demo-billing-reference', 'per_settlement', NOW(), 500000000, 30, 48, NOW(),
    'supplier-terms-v2-2026-09-07', NOW(), '00000000-0000-7000-8000-000000000001',
    'privacy-v2-2026-09-07', NOW(), '00000000-0000-7000-8000-000000000001', NOW(), TRUE,
    'pilot_ready', NOW()
)
ON CONFLICT (organization_id) DO NOTHING;

INSERT INTO app.supplier_onboarding_revisions
    (organization_id, profile_version, change_type, actor_user_id, actor_reference, snapshot)
SELECT p.organization_id, p.version, 'profile.seeded',
       '00000000-0000-7000-8000-000000000001', 'seed:001_demo', to_jsonb(p)
FROM app.supplier_onboarding_profiles p
WHERE p.organization_id = '00000000-0000-7000-8000-000000000010'
ON CONFLICT (organization_id, profile_version) DO NOTHING;

INSERT INTO app.memberships (id, organization_id, user_id, role, status, accepted_at)
VALUES
    ('00000000-0000-7000-8000-000000000011', '00000000-0000-7000-8000-000000000010', '00000000-0000-7000-8000-000000000001', 'owner', 'active', NOW()),
    ('00000000-0000-7000-8000-000000000012', '00000000-0000-7000-8000-000000000010', '00000000-0000-7000-8000-000000000002', 'finance', 'active', NOW()),
    ('00000000-0000-7000-8000-000000000013', '00000000-0000-7000-8000-000000000010', '00000000-0000-7000-8000-000000000003', 'sales', 'active', NOW())
ON CONFLICT (id) DO NOTHING;

-- The acceptance seed is deliberately deterministic and contains no usable
-- production credential. The development OTP flow prints its code locally.
INSERT INTO app.persons (id, user_id, full_name, status)
VALUES ('00000000-0000-7000-8000-000000000020', '00000000-0000-7000-8000-000000000004', 'Royal Pharmacy Representative', 'verified')
ON CONFLICT (id) DO NOTHING;

INSERT INTO app.businesses (id, owner_user_id, legal_name, trading_name, business_type, registration_info, business_address, industry, status)
VALUES ('00000000-0000-7000-8000-000000000021', '00000000-0000-7000-8000-000000000004', 'Royal Pharmacy Ltd', 'Royal Pharmacy', 'limited_company', '{"number":"DEMO-REG-ROYAL"}'::jsonb, 'Lagos, Nigeria', 'pharmacy', 'verified')
ON CONFLICT (id) DO NOTHING;

INSERT INTO app.business_representatives (id, business_id, person_id, role_title, authority_type, authority_verification_status, starts_at, evidence_reference)
VALUES ('00000000-0000-7000-8000-000000000022', '00000000-0000-7000-8000-000000000021', '00000000-0000-7000-8000-000000000020', 'Managing Director', 'director', 'verified', DATE '2026-01-01', 'DEMO-AUTHORITY-EVIDENCE')
ON CONFLICT (id) DO NOTHING;

INSERT INTO app.verification_cases (id, subject_type, subject_id, provider, provider_reference, verification_level, state, safe_result, completed_at, expires_at)
VALUES
    ('00000000-0000-7000-8000-000000000023', 'person', '00000000-0000-7000-8000-000000000020', 'mock-identity', 'demo-person-verification', 2, 'verified', '{"name_match":true}'::jsonb, NOW(), NOW() + INTERVAL '1 year'),
    ('00000000-0000-7000-8000-000000000024', 'business', '00000000-0000-7000-8000-000000000021', 'mock-identity', 'demo-business-verification', 2, 'verified', '{"registration_match":true}'::jsonb, NOW(), NOW() + INTERVAL '1 year'),
    ('00000000-0000-7000-8000-000000000025', 'authority', '00000000-0000-7000-8000-000000000022', 'mock-identity', 'demo-authority-verification', 2, 'verified', '{"authority_match":true}'::jsonb, NOW(), NOW() + INTERVAL '1 year')
ON CONFLICT (provider, provider_reference) DO NOTHING;

INSERT INTO app.identity_consents (id, user_id, consent_type, version, evidence_hash)
VALUES ('00000000-0000-7000-8000-000000000026', '00000000-0000-7000-8000-000000000004', 'identity_verification', 'identity-consent-v1', 'demo-consent-evidence')
ON CONFLICT (id) DO NOTHING;

INSERT INTO app.trade_relationships (id, supplier_organization_id, buyer_business_id, status, first_transaction_at, last_transaction_at, supplier_customer_code)
VALUES ('00000000-0000-7000-8000-000000000027', '00000000-0000-7000-8000-000000000010', '00000000-0000-7000-8000-000000000021', 'active', TIMESTAMPTZ '2026-08-01 09:00:00+01', NOW(), 'ROYAL-001')
ON CONFLICT (supplier_organization_id, buyer_business_id) DO NOTHING;

INSERT INTO app.payment_mandates (id, buyer_subject_type, buyer_subject_id, provider, provider_mandate_id, mandate_type, amount_ceiling_kobo, state, capability_snapshot, accepted_disclosure_version, provider_updated_at, created_at)
VALUES
    ('00000000-0000-7000-8000-000000000028', 'business', '00000000-0000-7000-8000-000000000021', 'mock-collection', 'demo-active-mandate', 'variable', 500000000, 'active', '{"one_time":true,"recurring":true,"variable":true}'::jsonb, 'mandate-v1', NOW(), NOW()),
    ('00000000-0000-7000-8000-000000000029', 'business', '00000000-0000-7000-8000-000000000021', 'mock-collection', 'demo-cancelled-mandate', 'variable', 180000000, 'cancelled', '{"one_time":true,"recurring":true}'::jsonb, 'mandate-v1', NOW() - INTERVAL '30 days', NOW() - INTERVAL '30 days')
ON CONFLICT (provider, provider_mandate_id) DO NOTHING;

INSERT INTO app.mandate_events (id, mandate_id, provider_event_id, old_state, new_state, reason_code, event_at)
VALUES ('00000000-0000-7000-8000-000000000030', '00000000-0000-7000-8000-000000000029', 'demo-mandate-cancelled-event', 'active', 'cancelled', 'buyer_cancelled', TIMESTAMPTZ '2026-08-15 09:00:00+01')
ON CONFLICT (mandate_id, provider_event_id) DO NOTHING;

-- FIX-README-C-TRADE-LINE / Scenario C — recurring trade line: ₦5m limit,
-- Friday cadence and three fully evidenced activated drawdowns totalling
-- ₦2.75m. The first obligation records ₦1m already repaid, leaving ₦1.75m
-- current exposure across the line.
INSERT INTO app.trade_lines (id, supplier_organization_id, buyer_user_id, buyer_business_id, approved_limit_kobo, current_exposure_kobo, reserved_pending_kobo, available_limit_kobo, cadence, default_grace_hours, start_at, end_at, state, mandate_id, mandate_active, terms_version)
VALUES ('00000000-0000-7000-8000-000000000040', '00000000-0000-7000-8000-000000000010', '00000000-0000-7000-8000-000000000004', '00000000-0000-7000-8000-000000000021', 500000000, 275000000, 0, 225000000, 'friday', 48, TIMESTAMPTZ '2026-08-01 00:00:00+01', TIMESTAMPTZ '2027-08-01 00:00:00+01', 'ACTIVE', '00000000-0000-7000-8000-000000000028', true, 'trade-line-v1')
ON CONFLICT (id) DO NOTHING;

INSERT INTO app.credit_requests (id,supplier_organization_id,buyer_user_id,buyer_business_id,principal_kobo,currency,goods_description,invoice_reference,due_date,grace_hours,collection_at,state,created_by,created_at,updated_at,version)
VALUES
    ('00000000-0000-7000-8000-000000000051','00000000-0000-7000-8000-000000000010','00000000-0000-7000-8000-000000000004','00000000-0000-7000-8000-000000000021',120000000,'NGN','Pharmaceutical inventory drawdown 1','DEMO-TL-001',DATE '2026-09-02',48,TIMESTAMPTZ '2026-09-04 17:00:00+01','ACTIVE','00000000-0000-7000-8000-000000000003',TIMESTAMPTZ '2026-08-02 09:00:00+01',TIMESTAMPTZ '2026-08-02 10:05:00+01',1),
    ('00000000-0000-7000-8000-000000000052','00000000-0000-7000-8000-000000000010','00000000-0000-7000-8000-000000000004','00000000-0000-7000-8000-000000000021',90000000,'NGN','Pharmaceutical inventory drawdown 2','DEMO-TL-002',DATE '2026-09-08',48,TIMESTAMPTZ '2026-09-10 17:00:00+01','ACTIVE','00000000-0000-7000-8000-000000000003',TIMESTAMPTZ '2026-08-08 09:00:00+01',TIMESTAMPTZ '2026-08-08 10:05:00+01',1),
    ('00000000-0000-7000-8000-000000000053','00000000-0000-7000-8000-000000000010','00000000-0000-7000-8000-000000000004','00000000-0000-7000-8000-000000000021',65000000,'NGN','Pharmaceutical inventory drawdown 3','DEMO-TL-003',DATE '2026-09-15',48,TIMESTAMPTZ '2026-09-17 17:00:00+01','ACTIVE','00000000-0000-7000-8000-000000000003',TIMESTAMPTZ '2026-08-15 09:00:00+01',TIMESTAMPTZ '2026-08-15 10:05:00+01',1)
ON CONFLICT (id) DO NOTHING;

INSERT INTO app.agreement_versions (id,credit_request_id,version,canonical_json,document_hash,terms_version,privacy_version,created_by,created_at)
VALUES
    ('00000000-0000-7000-8000-000000000061','00000000-0000-7000-8000-000000000051',1,'{"fixture":"trade-line-drawdown-1"}',encode(digest('{"fixture":"trade-line-drawdown-1"}','sha256'),'hex'),'trade-line-v1','privacy-v1','00000000-0000-7000-8000-000000000003',TIMESTAMPTZ '2026-08-02 09:00:00+01'),
    ('00000000-0000-7000-8000-000000000062','00000000-0000-7000-8000-000000000052',1,'{"fixture":"trade-line-drawdown-2"}',encode(digest('{"fixture":"trade-line-drawdown-2"}','sha256'),'hex'),'trade-line-v1','privacy-v1','00000000-0000-7000-8000-000000000003',TIMESTAMPTZ '2026-08-08 09:00:00+01'),
    ('00000000-0000-7000-8000-000000000063','00000000-0000-7000-8000-000000000053',1,'{"fixture":"trade-line-drawdown-3"}',encode(digest('{"fixture":"trade-line-drawdown-3"}','sha256'),'hex'),'trade-line-v1','privacy-v1','00000000-0000-7000-8000-000000000003',TIMESTAMPTZ '2026-08-15 09:00:00+01')
ON CONFLICT (id) DO NOTHING;

INSERT INTO app.agreement_acceptances (id,credit_request_id,agreement_version_id,accepting_user_id,person_id,business_id,acceptance_method,authentication_level,agreement_hash,mandate_provider_id,accepted_at)
VALUES
    ('00000000-0000-7000-8000-000000000065','00000000-0000-7000-8000-000000000051','00000000-0000-7000-8000-000000000061','00000000-0000-7000-8000-000000000004','00000000-0000-7000-8000-000000000020','00000000-0000-7000-8000-000000000021','digital_signature','AAL2',encode(digest('{"fixture":"trade-line-drawdown-1"}','sha256'),'hex'),'mock-collection',TIMESTAMPTZ '2026-08-02 09:30:00+01'),
    ('00000000-0000-7000-8000-000000000066','00000000-0000-7000-8000-000000000052','00000000-0000-7000-8000-000000000062','00000000-0000-7000-8000-000000000004','00000000-0000-7000-8000-000000000020','00000000-0000-7000-8000-000000000021','digital_signature','AAL2',encode(digest('{"fixture":"trade-line-drawdown-2"}','sha256'),'hex'),'mock-collection',TIMESTAMPTZ '2026-08-08 09:30:00+01'),
    ('00000000-0000-7000-8000-000000000067','00000000-0000-7000-8000-000000000053','00000000-0000-7000-8000-000000000063','00000000-0000-7000-8000-000000000004','00000000-0000-7000-8000-000000000020','00000000-0000-7000-8000-000000000021','digital_signature','AAL2',encode(digest('{"fixture":"trade-line-drawdown-3"}','sha256'),'hex'),'mock-collection',TIMESTAMPTZ '2026-08-15 09:30:00+01')
ON CONFLICT (id) DO NOTHING;

INSERT INTO app.obligations (id,credit_request_id,agreement_version_id,supplier_organization_id,buyer_business_id,principal_kobo,currency,lifecycle_status,payment_status,outstanding_kobo,base_fee_kobo,ledger_transaction_id,activated_at)
VALUES
    ('00000000-0000-7000-8000-000000000071','00000000-0000-7000-8000-000000000051','00000000-0000-7000-8000-000000000061','00000000-0000-7000-8000-000000000010','00000000-0000-7000-8000-000000000021',120000000,'NGN','ACTIVE','UNPAID',120000000,600000,'00000000-0000-7000-8000-000000000081',TIMESTAMPTZ '2026-08-02 10:05:00+01'),
    ('00000000-0000-7000-8000-000000000072','00000000-0000-7000-8000-000000000052','00000000-0000-7000-8000-000000000062','00000000-0000-7000-8000-000000000010','00000000-0000-7000-8000-000000000021',90000000,'NGN','ACTIVE','UNPAID',90000000,450000,'00000000-0000-7000-8000-000000000082',TIMESTAMPTZ '2026-08-08 10:05:00+01'),
    ('00000000-0000-7000-8000-000000000073','00000000-0000-7000-8000-000000000053','00000000-0000-7000-8000-000000000063','00000000-0000-7000-8000-000000000010','00000000-0000-7000-8000-000000000021',65000000,'NGN','ACTIVE','UNPAID',65000000,325000,'00000000-0000-7000-8000-000000000083',TIMESTAMPTZ '2026-08-15 10:05:00+01')
ON CONFLICT (id) DO NOTHING;

UPDATE app.credit_requests r SET agreement_version_id=v.agreement_id, acceptance_id=v.acceptance_id, obligation_id=v.obligation_id
FROM (VALUES
    ('00000000-0000-7000-8000-000000000051'::uuid,'00000000-0000-7000-8000-000000000061'::uuid,'00000000-0000-7000-8000-000000000065'::uuid,'00000000-0000-7000-8000-000000000071'::uuid),
    ('00000000-0000-7000-8000-000000000052'::uuid,'00000000-0000-7000-8000-000000000062'::uuid,'00000000-0000-7000-8000-000000000066'::uuid,'00000000-0000-7000-8000-000000000072'::uuid),
    ('00000000-0000-7000-8000-000000000053'::uuid,'00000000-0000-7000-8000-000000000063'::uuid,'00000000-0000-7000-8000-000000000067'::uuid,'00000000-0000-7000-8000-000000000073'::uuid)
) AS v(request_id,agreement_id,acceptance_id,obligation_id) WHERE r.id=v.request_id;

INSERT INTO app.drawdowns (id,trade_line_id,principal_kobo,goods_description,invoice_reference,due_date,collection_at,grace_hours,terms_version,agreement_hash,state,obligation_id,buyer_confirmed_at,release_actor_id,delivery_method,release_evidence_reference,released_at,receipt_state,receipt_actor_id,receipt_at,activated_at)
VALUES
    ('00000000-0000-7000-8000-000000000041','00000000-0000-7000-8000-000000000040',120000000,'Pharmaceutical inventory drawdown 1','DEMO-TL-001',DATE '2026-09-02',TIMESTAMPTZ '2026-09-04 17:00:00+01',48,'trade-line-v1','8a631674c3f425aa86b91ddef58a9b2b9c37bb7a187ee30ffe733a0e91689f86','ACTIVATED','00000000-0000-7000-8000-000000000071',TIMESTAMPTZ '2026-08-02 10:00:00+01','00000000-0000-7000-8000-000000000003','delivery','DEMO-RELEASE-1',TIMESTAMPTZ '2026-08-02 10:02:00+01','no_issue','00000000-0000-7000-8000-000000000004',TIMESTAMPTZ '2026-08-02 10:05:00+01',TIMESTAMPTZ '2026-08-02 10:05:00+01'),
    ('00000000-0000-7000-8000-000000000042','00000000-0000-7000-8000-000000000040',90000000,'Pharmaceutical inventory drawdown 2','DEMO-TL-002',DATE '2026-09-08',TIMESTAMPTZ '2026-09-10 17:00:00+01',48,'trade-line-v1','64bf62359598479de600038ef64ae5529abcdb78353f1dcce4f58862a34e03b5','ACTIVATED','00000000-0000-7000-8000-000000000072',TIMESTAMPTZ '2026-08-08 10:00:00+01','00000000-0000-7000-8000-000000000003','delivery','DEMO-RELEASE-2',TIMESTAMPTZ '2026-08-08 10:02:00+01','no_issue','00000000-0000-7000-8000-000000000004',TIMESTAMPTZ '2026-08-08 10:05:00+01',TIMESTAMPTZ '2026-08-08 10:05:00+01'),
    ('00000000-0000-7000-8000-000000000043','00000000-0000-7000-8000-000000000040',65000000,'Pharmaceutical inventory drawdown 3','DEMO-TL-003',DATE '2026-09-15',TIMESTAMPTZ '2026-09-17 17:00:00+01',48,'trade-line-v1','21c61c3462747e630f0edf70872fb928fd088dddb4419e4c73dcc80749f409b6','ACTIVATED','00000000-0000-7000-8000-000000000073',TIMESTAMPTZ '2026-08-15 10:00:00+01','00000000-0000-7000-8000-000000000003','delivery','DEMO-RELEASE-3',TIMESTAMPTZ '2026-08-15 10:02:00+01','no_issue','00000000-0000-7000-8000-000000000004',TIMESTAMPTZ '2026-08-15 10:05:00+01',TIMESTAMPTZ '2026-08-15 10:05:00+01')
ON CONFLICT (id) DO NOTHING;

-- The portals must read the same debts as the normalized financial tables.
-- Earlier demo-only snapshots invented four obligations that did not exist,
-- so bank collection and payment detail reads failed for those displayed sales.
-- Remove only those known orphan fixtures on a repeated development seed.
DELETE FROM app.credit_aggregate_snapshots s
WHERE s.credit_request_id IN ('00000000-0000-7000-8000-000000000101','00000000-0000-7000-8000-000000000102','00000000-0000-7000-8000-000000000104','00000000-0000-7000-8000-000000000105')
AND NOT EXISTS (SELECT 1 FROM app.credit_requests r WHERE r.id::text=s.credit_request_id);

INSERT INTO app.repayment_schedules(obligation_id,schedule_type,timezone,allocation_policy,cadence,grace_hours,status)
SELECT o.id,'equal','Africa/Lagos','oldest_due_first','custom',r.grace_hours,'ACTIVE'
FROM app.obligations o JOIN app.credit_requests r ON r.id=o.credit_request_id
WHERE r.id IN ('00000000-0000-7000-8000-000000000051','00000000-0000-7000-8000-000000000052','00000000-0000-7000-8000-000000000053')
ON CONFLICT(obligation_id) DO NOTHING;

INSERT INTO app.schedule_items(schedule_id,sequence,principal_due_kobo,due_at,grace_hours,collection_at,allocated_kobo,state)
SELECT s.id,1,o.principal_kobo,r.collection_at-make_interval(hours=>r.grace_hours),r.grace_hours,r.collection_at,
       o.principal_kobo-o.outstanding_kobo,CASE WHEN o.outstanding_kobo=0 THEN 'PAID' WHEN o.outstanding_kobo<o.principal_kobo THEN 'PARTIALLY_PAID' ELSE 'OPEN' END
FROM app.repayment_schedules s JOIN app.obligations o ON o.id=s.obligation_id JOIN app.credit_requests r ON r.id=o.credit_request_id
WHERE r.id IN ('00000000-0000-7000-8000-000000000051','00000000-0000-7000-8000-000000000052','00000000-0000-7000-8000-000000000053')
ON CONFLICT(schedule_id,sequence) DO NOTHING;


-- Real synthetic accounting evidence backs every displayed demo balance.
INSERT INTO ledger.transactions(id,event_type,reference_type,reference_id,idempotency_key,effective_at)
SELECT o.ledger_transaction_id,'principal_activated','obligation',o.id::text,'seed:activation:'||o.id::text,o.activated_at
FROM app.obligations o WHERE o.id IN('00000000-0000-7000-8000-000000000071','00000000-0000-7000-8000-000000000072','00000000-0000-7000-8000-000000000073')
ON CONFLICT(id) DO NOTHING;

INSERT INTO ledger.postings(id,transaction_id,account_id,debit_kobo,credit_kobo)
SELECT md5(o.ledger_transaction_id::text||v.code)::uuid,o.ledger_transaction_id,a.id,v.debit,v.credit
FROM app.obligations o CROSS JOIN LATERAL (VALUES
 ('TRADE_RECEIVABLE_CONTROL',o.principal_kobo,0::bigint),
 ('PRINCIPAL_ORIGINATED_CONTROL',0::bigint,o.principal_kobo),
 ('SUPPLIER_FEE_RECEIVABLE',o.base_fee_kobo,0::bigint),
 ('PLATFORM_SERVICE_REVENUE',0::bigint,o.base_fee_kobo)) v(code,debit,credit)
JOIN ledger.accounts a ON a.code=v.code
WHERE o.id IN('00000000-0000-7000-8000-000000000071','00000000-0000-7000-8000-000000000072','00000000-0000-7000-8000-000000000073')
ON CONFLICT(id) DO NOTHING;

WITH recorded AS (
 INSERT INTO app.payments(id,obligation_id,buyer_user_id,supplier_organization_id,source_type,amount_kobo,currency,state,paid_at,recorded_by,recorded_by_reference,idempotency_key)
 VALUES('00000000-0000-7000-8000-000000000091','00000000-0000-7000-8000-000000000071','00000000-0000-7000-8000-000000000004','00000000-0000-7000-8000-000000000010','supplier_recorded_transfer',100000000,'NGN','recognized',TIMESTAMPTZ '2026-08-25 12:00:00+01','00000000-0000-7000-8000-000000000002','00000000-0000-7000-8000-000000000002','seed:payment:royal-001')
 ON CONFLICT(id) DO NOTHING RETURNING obligation_id,amount_kobo
)
UPDATE app.obligations o SET outstanding_kobo=o.outstanding_kobo-recorded.amount_kobo,payment_status='PARTIALLY_PAID'
FROM recorded WHERE o.id=recorded.obligation_id;

INSERT INTO ledger.transactions(id,event_type,reference_type,reference_id,idempotency_key,effective_at)
VALUES('00000000-0000-7000-8000-000000000092','payment_recognized','payment','00000000-0000-7000-8000-000000000091','seed:payment-journal:royal-001',TIMESTAMPTZ '2026-08-25 12:00:00+01')
ON CONFLICT(id) DO NOTHING;
INSERT INTO ledger.postings(id,transaction_id,account_id,debit_kobo,credit_kobo)
SELECT md5('seed:royal-payment:'||v.code)::uuid,'00000000-0000-7000-8000-000000000092',a.id,v.debit,v.credit
FROM (VALUES ('VOLUNTARY_SETTLEMENT_CONTROL',100000000::bigint,0::bigint),('TRADE_RECEIVABLE_CONTROL',0::bigint,100000000::bigint)) v(code,debit,credit)
JOIN ledger.accounts a ON a.code=v.code ON CONFLICT(id) DO NOTHING;

WITH allocated AS (
 INSERT INTO app.payment_allocations(id,payment_id,obligation_id,schedule_item_id,amount_kobo,allocation_order)
 SELECT '00000000-0000-7000-8000-000000000093','00000000-0000-7000-8000-000000000091',s.obligation_id,i.id,100000000,1
 FROM app.repayment_schedules s JOIN app.schedule_items i ON i.schedule_id=s.id
 WHERE s.obligation_id='00000000-0000-7000-8000-000000000071' AND i.sequence=1
 ON CONFLICT(id) DO NOTHING RETURNING schedule_item_id,amount_kobo
)
UPDATE app.schedule_items i SET allocated_kobo=i.allocated_kobo+allocated.amount_kobo,state='PARTIALLY_PAID'
FROM allocated WHERE i.id=allocated.schedule_item_id;


INSERT INTO app.credit_aggregate_snapshots(credit_request_id,supplier_organization_id,buyer_user_id,aggregate,version,updated_at)
SELECT r.id::text,r.supplier_organization_id::text,r.buyer_user_id::text,
 jsonb_build_object(
  'request',to_jsonb(r)||jsonb_build_object('supplier_legal_name',org.legal_name,'supplier_trading_name',org.trading_name,'buyer_legal_name',b.legal_name,'buyer_trading_name',b.trading_name,'schedule_type','equal','schedule_count',1,'schedule_cadence','custom'),
  'agreement',to_jsonb(a)||jsonb_build_object('principal_kobo',r.principal_kobo,'due_date',r.due_date,'grace_hours',r.grace_hours,'collection_at',r.collection_at),
  'acceptance',to_jsonb(ac),'receipts','[]'::jsonb,'obligation',to_jsonb(o)),r.version,r.updated_at
FROM app.credit_requests r
JOIN app.organizations org ON org.id=r.supplier_organization_id
JOIN app.businesses b ON b.id=r.buyer_business_id
JOIN app.agreement_versions a ON a.id=r.agreement_version_id
JOIN app.agreement_acceptances ac ON ac.id=r.acceptance_id
JOIN app.obligations o ON o.id=r.obligation_id
WHERE r.id IN ('00000000-0000-7000-8000-000000000051','00000000-0000-7000-8000-000000000052','00000000-0000-7000-8000-000000000053')
ON CONFLICT(credit_request_id) DO NOTHING;

-- FIX-README-F-DUPLICATE-WEBHOOK / Scenario F — duplicate provider webhook. The inbox uniqueness constraint is
-- the event row represents a previously processed provider event. Actual
-- replay and financial-effect assertions live in the collection tests.
INSERT INTO app.provider_webhook_inbox (id, provider, event_id, event_type, payload, signature_valid, state, attempts, processed_at)
VALUES ('00000000-0000-7000-8000-000000000060', 'mock-collection', 'scenario-f-success-event', 'collection.succeeded', '{"deliveries":3,"financial_effects":1,"receipts":1}'::jsonb, true, 'processed', 3, NOW())
ON CONFLICT (provider, event_id) DO NOTHING;

INSERT INTO app_meta(key,value)
VALUES ('acceptance_dataset','Normalized Scenario C trade-line sales and schedules; Scenario F processed webhook fixture. Other lifecycle scenarios are verified by domain tests.')
ON CONFLICT(key) DO UPDATE SET value=EXCLUDED.value,updated_at=NOW();

COMMIT;
