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
    'supplier-terms-v1', NOW(), '00000000-0000-7000-8000-000000000001',
    'privacy-v1', NOW(), '00000000-0000-7000-8000-000000000001', NOW(), TRUE,
    'pilot_ready', NOW()
)
ON CONFLICT (organization_id) DO UPDATE SET
    authorized_representative_name=EXCLUDED.authorized_representative_name,
    authorized_representative_title=EXCLUDED.authorized_representative_title,
    owner_email_verified_at=EXCLUDED.owner_email_verified_at,
    owner_phone_verified_at=EXCLUDED.owner_phone_verified_at,
    kyb_state=EXCLUDED.kyb_state, kyb_provider_reference=EXCLUDED.kyb_provider_reference,
    kyb_expires_at=EXCLUDED.kyb_expires_at, settlement_state=EXCLUDED.settlement_state,
    settlement_provider=EXCLUDED.settlement_provider,
    settlement_provider_reference=EXCLUDED.settlement_provider_reference,
    settlement_bank_name=EXCLUDED.settlement_bank_name,
    settlement_account_name=EXCLUDED.settlement_account_name,
    settlement_account_last4=EXCLUDED.settlement_account_last4,
    billing_state=EXCLUDED.billing_state, billing_method=EXCLUDED.billing_method,
    billing_provider_reference=EXCLUDED.billing_provider_reference,
    billing_cycle=EXCLUDED.billing_cycle, default_credit_limit_kobo=EXCLUDED.default_credit_limit_kobo,
    default_payment_days=EXCLUDED.default_payment_days, default_grace_hours=EXCLUDED.default_grace_hours,
    default_credit_policy_updated_at=EXCLUDED.default_credit_policy_updated_at,
    terms_version=EXCLUDED.terms_version, terms_accepted_at=EXCLUDED.terms_accepted_at,
    terms_accepted_by=EXCLUDED.terms_accepted_by, privacy_version=EXCLUDED.privacy_version,
    privacy_accepted_at=EXCLUDED.privacy_accepted_at, privacy_accepted_by=EXCLUDED.privacy_accepted_by,
    owner_mfa_verified_at=EXCLUDED.owner_mfa_verified_at, finance_mfa_complete=TRUE,
    readiness_state='pilot_ready', readiness_changed_at=NOW(), updated_at=NOW();

INSERT INTO app.supplier_onboarding_revisions
    (organization_id, profile_version, change_type, actor_user_id, actor_reference, snapshot)
SELECT p.organization_id, p.version, 'profile.seeded',
       '00000000-0000-7000-8000-000000000001', 'seed:001_demo', to_jsonb(p)
FROM app.supplier_onboarding_profiles p
WHERE p.organization_id = '00000000-0000-7000-8000-000000000010'
ON CONFLICT (organization_id, profile_version) DO UPDATE SET
    change_type = EXCLUDED.change_type,
    actor_user_id = EXCLUDED.actor_user_id,
    actor_reference = EXCLUDED.actor_reference,
    snapshot = EXCLUDED.snapshot;

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
VALUES ('00000000-0000-7000-8000-000000000040', '00000000-0000-7000-8000-000000000010', '00000000-0000-7000-8000-000000000004', '00000000-0000-7000-8000-000000000021', 500000000, 175000000, 0, 325000000, 'friday', 48, TIMESTAMPTZ '2026-08-01 00:00:00+01', TIMESTAMPTZ '2027-08-01 00:00:00+01', 'ACTIVE', '00000000-0000-7000-8000-000000000028', true, 'trade-line-v1')
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

INSERT INTO app.obligations (id,credit_request_id,agreement_version_id,supplier_organization_id,buyer_business_id,principal_kobo,currency,lifecycle_status,payment_status,outstanding_kobo,base_fee_kobo,ledger_transaction_id,activated_at)
VALUES
    ('00000000-0000-7000-8000-000000000071','00000000-0000-7000-8000-000000000051','00000000-0000-7000-8000-000000000061','00000000-0000-7000-8000-000000000010','00000000-0000-7000-8000-000000000021',120000000,'NGN','ACTIVE','PARTIALLY_PAID',20000000,600000,'00000000-0000-7000-8000-000000000081',TIMESTAMPTZ '2026-08-02 10:05:00+01'),
    ('00000000-0000-7000-8000-000000000072','00000000-0000-7000-8000-000000000052','00000000-0000-7000-8000-000000000062','00000000-0000-7000-8000-000000000010','00000000-0000-7000-8000-000000000021',90000000,'NGN','ACTIVE','UNPAID',90000000,450000,'00000000-0000-7000-8000-000000000082',TIMESTAMPTZ '2026-08-08 10:05:00+01'),
    ('00000000-0000-7000-8000-000000000073','00000000-0000-7000-8000-000000000053','00000000-0000-7000-8000-000000000063','00000000-0000-7000-8000-000000000010','00000000-0000-7000-8000-000000000021',65000000,'NGN','ACTIVE','UNPAID',65000000,325000,'00000000-0000-7000-8000-000000000083',TIMESTAMPTZ '2026-08-15 10:05:00+01')
ON CONFLICT (id) DO NOTHING;

UPDATE app.credit_requests r SET agreement_version_id=v.agreement_id, obligation_id=v.obligation_id
FROM (VALUES
    ('00000000-0000-7000-8000-000000000051'::uuid,'00000000-0000-7000-8000-000000000061'::uuid,'00000000-0000-7000-8000-000000000071'::uuid),
    ('00000000-0000-7000-8000-000000000052'::uuid,'00000000-0000-7000-8000-000000000062'::uuid,'00000000-0000-7000-8000-000000000072'::uuid),
    ('00000000-0000-7000-8000-000000000053'::uuid,'00000000-0000-7000-8000-000000000063'::uuid,'00000000-0000-7000-8000-000000000073'::uuid)
) AS v(request_id,agreement_id,obligation_id) WHERE r.id=v.request_id;

INSERT INTO app.drawdowns (id,trade_line_id,principal_kobo,goods_description,invoice_reference,due_date,collection_at,grace_hours,terms_version,agreement_hash,state,obligation_id,buyer_confirmed_at,release_actor_id,delivery_method,release_evidence_reference,released_at,receipt_state,receipt_actor_id,receipt_at,activated_at)
VALUES
    ('00000000-0000-7000-8000-000000000041','00000000-0000-7000-8000-000000000040',120000000,'Pharmaceutical inventory drawdown 1','DEMO-TL-001',DATE '2026-09-02',TIMESTAMPTZ '2026-09-04 17:00:00+01',48,'trade-line-v1','8a631674c3f425aa86b91ddef58a9b2b9c37bb7a187ee30ffe733a0e91689f86','ACTIVATED','00000000-0000-7000-8000-000000000071',TIMESTAMPTZ '2026-08-02 10:00:00+01','00000000-0000-7000-8000-000000000003','delivery','DEMO-RELEASE-1',TIMESTAMPTZ '2026-08-02 10:02:00+01','no_issue','00000000-0000-7000-8000-000000000004',TIMESTAMPTZ '2026-08-02 10:05:00+01',TIMESTAMPTZ '2026-08-02 10:05:00+01'),
    ('00000000-0000-7000-8000-000000000042','00000000-0000-7000-8000-000000000040',90000000,'Pharmaceutical inventory drawdown 2','DEMO-TL-002',DATE '2026-09-08',TIMESTAMPTZ '2026-09-10 17:00:00+01',48,'trade-line-v1','64bf62359598479de600038ef64ae5529abcdb78353f1dcce4f58862a34e03b5','ACTIVATED','00000000-0000-7000-8000-000000000072',TIMESTAMPTZ '2026-08-08 10:00:00+01','00000000-0000-7000-8000-000000000003','delivery','DEMO-RELEASE-2',TIMESTAMPTZ '2026-08-08 10:02:00+01','no_issue','00000000-0000-7000-8000-000000000004',TIMESTAMPTZ '2026-08-08 10:05:00+01',TIMESTAMPTZ '2026-08-08 10:05:00+01'),
    ('00000000-0000-7000-8000-000000000043','00000000-0000-7000-8000-000000000040',65000000,'Pharmaceutical inventory drawdown 3','DEMO-TL-003',DATE '2026-09-15',TIMESTAMPTZ '2026-09-17 17:00:00+01',48,'trade-line-v1','21c61c3462747e630f0edf70872fb928fd088dddb4419e4c73dcc80749f409b6','ACTIVATED','00000000-0000-7000-8000-000000000073',TIMESTAMPTZ '2026-08-15 10:00:00+01','00000000-0000-7000-8000-000000000003','delivery','DEMO-RELEASE-3',TIMESTAMPTZ '2026-08-15 10:02:00+01','no_issue','00000000-0000-7000-8000-000000000004',TIMESTAMPTZ '2026-08-15 10:05:00+01',TIMESTAMPTZ '2026-08-15 10:05:00+01')
ON CONFLICT (id) DO NOTHING;

-- FIX-README-A-ONE-TIME, FIX-README-B-INSTALMENTS,
-- FIX-README-D-MANDATE-CANCEL, and FIX-README-E-PARTIAL-DISPUTE are durable credit aggregates used by the supplier
-- and buyer portals. Their exact expected outcomes are kept beside the data so
-- acceptance checks can compare integer-kobo values without hidden arithmetic.
INSERT INTO app.credit_aggregate_snapshots (credit_request_id, supplier_organization_id, buyer_user_id, aggregate, version, updated_at)
VALUES
    ('00000000-0000-7000-8000-000000000101', '00000000-0000-7000-8000-000000000010', '00000000-0000-7000-8000-000000000004', jsonb_build_object('request', jsonb_build_object('id','00000000-0000-7000-8000-000000000101','supplier_organization_id','00000000-0000-7000-8000-000000000010','supplier_legal_name','ABC Pharmaceuticals Ltd','buyer_user_id','00000000-0000-7000-8000-000000000004','buyer_business_id','00000000-0000-7000-8000-000000000021','buyer_legal_name','Royal Pharmacy Ltd','principal_kobo',120000000,'currency','NGN','goods_description','Scenario A — one-time credit pharmaceutical supplies','due_date','2026-09-30','grace_hours',48,'collection_at','2026-10-02T16:00:00Z','state','ACTIVE','created_by','00000000-0000-7000-8000-000000000003','created_at','2026-08-16T09:00:00Z','updated_at','2026-10-02T17:00:00Z','version',8), 'agreement', jsonb_build_object('id','00000000-0000-7000-8000-000000000201','credit_request_id','00000000-0000-7000-8000-000000000101','version',1,'canonical_json','{}'::jsonb,'document_hash','demo-scenario-a-hash','principal_kobo',120000000,'due_date','2026-09-30','grace_hours',48,'collection_at','2026-10-02T16:00:00Z','terms_version','terms-v1','privacy_version','privacy-v1','created_by','00000000-0000-7000-8000-000000000003','created_at','2026-08-16T09:00:00Z'), 'receipts','[]'::jsonb, 'obligation', jsonb_build_object('id','00000000-0000-7000-8000-000000000301','credit_request_id','00000000-0000-7000-8000-000000000101','agreement_version_id','00000000-0000-7000-8000-000000000201','supplier_organization_id','00000000-0000-7000-8000-000000000010','buyer_business_id','00000000-0000-7000-8000-000000000021','principal_kobo',120000000,'currency','NGN','lifecycle_status','ACTIVE','payment_status','PAID','outstanding_kobo',0,'base_fee_kobo',600000,'ledger_transaction_id','00000000-0000-7000-8000-000000000401','activated_at','2026-08-16T10:00:00Z')), 8, NOW()),
    ('00000000-0000-7000-8000-000000000102', '00000000-0000-7000-8000-000000000010', '00000000-0000-7000-8000-000000000004', jsonb_build_object('request', jsonb_build_object('id','00000000-0000-7000-8000-000000000102','supplier_organization_id','00000000-0000-7000-8000-000000000010','supplier_legal_name','ABC Pharmaceuticals Ltd','buyer_user_id','00000000-0000-7000-8000-000000000004','buyer_business_id','00000000-0000-7000-8000-000000000021','buyer_legal_name','Royal Pharmacy Ltd','principal_kobo',300000000,'currency','NGN','goods_description','Scenario B — six monthly instalments','due_date','2027-02-28','grace_hours',48,'collection_at','2027-03-02T16:00:00Z','state','ACTIVE','created_by','00000000-0000-7000-8000-000000000003','created_at','2026-08-16T09:00:00Z','updated_at','2026-08-16T10:00:00Z','version',6), 'agreement', jsonb_build_object('id','00000000-0000-7000-8000-000000000202','credit_request_id','00000000-0000-7000-8000-000000000102','version',1,'canonical_json','{}'::jsonb,'document_hash','demo-scenario-b-hash','principal_kobo',300000000,'due_date','2027-02-28','grace_hours',48,'collection_at','2027-03-02T16:00:00Z','terms_version','terms-v1','privacy_version','privacy-v1','created_by','00000000-0000-7000-8000-000000000003','created_at','2026-08-16T09:00:00Z'), 'receipts','[]'::jsonb, 'obligation', jsonb_build_object('id','00000000-0000-7000-8000-000000000302','credit_request_id','00000000-0000-7000-8000-000000000102','agreement_version_id','00000000-0000-7000-8000-000000000202','supplier_organization_id','00000000-0000-7000-8000-000000000010','buyer_business_id','00000000-0000-7000-8000-000000000021','principal_kobo',300000000,'currency','NGN','lifecycle_status','ACTIVE','payment_status','PARTIALLY_PAID','outstanding_kobo',237500000,'base_fee_kobo',1500000,'ledger_transaction_id','00000000-0000-7000-8000-000000000402','activated_at','2026-08-16T10:00:00Z')), 6, NOW()),
    ('00000000-0000-7000-8000-000000000104', '00000000-0000-7000-8000-000000000010', '00000000-0000-7000-8000-000000000004', jsonb_build_object('request', jsonb_build_object('id','00000000-0000-7000-8000-000000000104','supplier_organization_id','00000000-0000-7000-8000-000000000010','supplier_legal_name','ABC Pharmaceuticals Ltd','buyer_user_id','00000000-0000-7000-8000-000000000004','buyer_business_id','00000000-0000-7000-8000-000000000021','buyer_legal_name','Royal Pharmacy Ltd','principal_kobo',180000000,'currency','NGN','goods_description','Scenario D — overdue after mandate cancellation','due_date','2026-08-10','grace_hours',48,'collection_at','2026-08-12T16:00:00Z','state','ACTIVE','mandate_id','00000000-0000-7000-8000-000000000029','created_by','00000000-0000-7000-8000-000000000003','created_at','2026-08-01T09:00:00Z','updated_at','2026-08-15T09:00:00Z','version',7), 'agreement', jsonb_build_object('id','00000000-0000-7000-8000-000000000204','credit_request_id','00000000-0000-7000-8000-000000000104','version',1,'canonical_json','{}'::jsonb,'document_hash','demo-scenario-d-hash','principal_kobo',180000000,'due_date','2026-08-10','grace_hours',48,'collection_at','2026-08-12T16:00:00Z','terms_version','terms-v1','privacy_version','privacy-v1','created_by','00000000-0000-7000-8000-000000000003','created_at','2026-08-01T09:00:00Z'), 'mandate', jsonb_build_object('id','00000000-0000-7000-8000-000000000029','provider','mock-collection','provider_id','demo-cancelled-mandate','user_id','00000000-0000-7000-8000-000000000004','business_id','00000000-0000-7000-8000-000000000021','status','CANCELLED','amount_ceiling_kobo',180000000), 'receipts','[]'::jsonb, 'obligation', jsonb_build_object('id','00000000-0000-7000-8000-000000000304','credit_request_id','00000000-0000-7000-8000-000000000104','agreement_version_id','00000000-0000-7000-8000-000000000204','supplier_organization_id','00000000-0000-7000-8000-000000000010','buyer_business_id','00000000-0000-7000-8000-000000000021','principal_kobo',180000000,'currency','NGN','lifecycle_status','ACTIVE','payment_status','UNPAID','outstanding_kobo',180000000,'base_fee_kobo',900000,'ledger_transaction_id','00000000-0000-7000-8000-000000000404','activated_at','2026-08-01T10:00:00Z')), 7, NOW()),
    ('00000000-0000-7000-8000-000000000105', '00000000-0000-7000-8000-000000000010', '00000000-0000-7000-8000-000000000004', jsonb_build_object('request', jsonb_build_object('id','00000000-0000-7000-8000-000000000105','supplier_organization_id','00000000-0000-7000-8000-000000000010','supplier_legal_name','ABC Pharmaceuticals Ltd','buyer_user_id','00000000-0000-7000-8000-000000000004','buyer_business_id','00000000-0000-7000-8000-000000000021','buyer_legal_name','Royal Pharmacy Ltd','principal_kobo',100000000,'currency','NGN','goods_description','Scenario E — partial dispute','due_date','2026-08-20','grace_hours',48,'collection_at','2026-08-22T16:00:00Z','state','ACTIVE','created_by','00000000-0000-7000-8000-000000000003','created_at','2026-08-01T09:00:00Z','updated_at','2026-08-16T09:00:00Z','version',7), 'agreement', jsonb_build_object('id','00000000-0000-7000-8000-000000000205','credit_request_id','00000000-0000-7000-8000-000000000105','version',1,'canonical_json','{}'::jsonb,'document_hash','demo-scenario-e-hash','principal_kobo',100000000,'due_date','2026-08-20','grace_hours',48,'collection_at','2026-08-22T16:00:00Z','terms_version','terms-v1','privacy_version','privacy-v1','created_by','00000000-0000-7000-8000-000000000003','created_at','2026-08-01T09:00:00Z'), 'receipts','[]'::jsonb, 'obligation', jsonb_build_object('id','00000000-0000-7000-8000-000000000305','credit_request_id','00000000-0000-7000-8000-000000000105','agreement_version_id','00000000-0000-7000-8000-000000000205','supplier_organization_id','00000000-0000-7000-8000-000000000010','buyer_business_id','00000000-0000-7000-8000-000000000021','principal_kobo',100000000,'currency','NGN','lifecycle_status','ACTIVE','payment_status','PARTIALLY_PAID','outstanding_kobo',90000000,'base_fee_kobo',500000,'ledger_transaction_id','00000000-0000-7000-8000-000000000405','activated_at','2026-08-01T10:00:00Z')), 7, NOW())
ON CONFLICT (credit_request_id) DO UPDATE SET aggregate=EXCLUDED.aggregate, version=EXCLUDED.version, updated_at=EXCLUDED.updated_at;

-- FIX-README-F-DUPLICATE-WEBHOOK / Scenario F — duplicate provider webhook. The inbox uniqueness constraint is
-- the executable proof: three deliveries share one provider/event key and
-- therefore resolve to one stored event and one downstream financial effect.
INSERT INTO app.provider_webhook_inbox (id, provider, event_id, event_type, payload, signature_valid, state, attempts, processed_at)
VALUES ('00000000-0000-7000-8000-000000000060', 'mock-collection', 'scenario-f-success-event', 'collection.succeeded', '{"deliveries":3,"financial_effects":1,"receipts":1}'::jsonb, true, 'processed', 3, NOW())
ON CONFLICT (provider, event_id) DO UPDATE SET attempts=3, state='processed', processed_at=NOW();

INSERT INTO app_meta (key, value)
VALUES ('acceptance_dataset', 'Scenario A one-time; Scenario B instalments; Scenario C trade line; Scenario D mandate cancellation and overdue; Scenario E partial dispute; Scenario F duplicate webhook')
ON CONFLICT (key) DO UPDATE SET value=EXCLUDED.value, updated_at=NOW();
