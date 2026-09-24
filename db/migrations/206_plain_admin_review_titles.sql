-- +goose Up
-- The review inbox prints each row's title as written here. The old titles
-- joined raw column values ("business_account support case", "ACCESS
-- request"); these say the same thing in words an operator reads at a glance.
CREATE OR REPLACE VIEW app.admin_review_queue AS
 SELECT 'policy'::text kind,id,state,reason title,proposed_by author_id,created_at,effective_at due_at,'/admin/settings'::text href FROM app.business_policy_changes WHERE state='pending' OR(state='approved' AND effective_at>now())
 UNION ALL SELECT 'financial_change',id,state,initcap(replace(kind,'_',' '))||': '||reason,proposed_by,created_at,expires_at,'/admin/approvals?change='||id FROM app.admin_change_requests WHERE state IN ('pending','awaiting_buyer')
 UNION ALL SELECT 'dispute',id,state,reason,opened_by,opened_at,opened_at+interval '2 days','/admin/disputes/'||id FROM app.disputes WHERE state IN ('OPEN','UNDER_REVIEW','PARTIALLY_RESOLVED')
 UNION ALL SELECT 'financial_review',id,state,'Records disagree: '||lower(replace(kind,'_',' ')),NULL,first_seen_at,first_seen_at+interval '1 day','/admin/reconciliation' FROM app.financial_review_cases WHERE state='OPEN'
 UNION ALL SELECT 'recovery',id,state,'Account recovery',target_user_id,created_at,expires_at,'/admin/recovery' FROM app.account_recovery_requests WHERE state IN ('PENDING_REVIEW','COOLING_OFF','APPROVED')
 UNION ALL SELECT 'privacy',id,state,CASE request_type WHEN 'ACCESS' THEN 'Wants a copy of their information' WHEN 'CORRECTION' THEN 'Wants something corrected' WHEN 'DELETION' THEN 'Wants their information deleted' WHEN 'RESTRICTION' THEN 'Wants some use restricted' WHEN 'OBJECTION' THEN 'Objects to how it is used' WHEN 'CONSENT_WITHDRAWAL' THEN 'Withdrew a permission' WHEN 'PORTABILITY' THEN 'Wants a portable copy' ELSE 'Privacy request' END,requester_user_id,created_at,due_at,'/admin/privacy' FROM app.privacy_requests WHERE state NOT IN ('COMPLETED','CANCELLED','REJECTED')
 UNION ALL SELECT 'support',id,state,CASE subject_type WHEN 'business_account' THEN 'Help request from a business' WHEN 'organization' THEN 'Help request from a business' WHEN 'obligation' THEN 'Help request about an amount owed' ELSE 'Help request' END,opened_by,created_at,created_at+interval '2 days','/admin/cases/'||id FROM app.support_cases WHERE state IN ('OPEN','IN_PROGRESS');

-- +goose Down
CREATE OR REPLACE VIEW app.admin_review_queue AS
 SELECT 'policy'::text kind,id,state,reason title,proposed_by author_id,created_at,effective_at due_at,'/admin/settings'::text href FROM app.business_policy_changes WHERE state='pending' OR(state='approved' AND effective_at>now())
 UNION ALL SELECT 'financial_change',id,state,replace(kind,'_',' ')||': '||reason,proposed_by,created_at,expires_at,'/admin/approvals?change='||id FROM app.admin_change_requests WHERE state IN ('pending','awaiting_buyer')
 UNION ALL SELECT 'dispute',id,state,reason,opened_by,opened_at,opened_at+interval '2 days','/admin/disputes/'||id FROM app.disputes WHERE state IN ('OPEN','UNDER_REVIEW','PARTIALLY_RESOLVED')
 UNION ALL SELECT 'financial_review',id,state,kind||' discrepancy',NULL,first_seen_at,first_seen_at+interval '1 day','/admin/reconciliation' FROM app.financial_review_cases WHERE state='OPEN'
 UNION ALL SELECT 'recovery',id,state,'Account recovery',target_user_id,created_at,expires_at,'/admin/recovery' FROM app.account_recovery_requests WHERE state IN ('PENDING_REVIEW','COOLING_OFF','APPROVED')
 UNION ALL SELECT 'privacy',id,state,request_type||' request',requester_user_id,created_at,due_at,'/admin/privacy' FROM app.privacy_requests WHERE state NOT IN ('COMPLETED','CANCELLED','REJECTED')
 UNION ALL SELECT 'support',id,state,subject_type||' support case',opened_by,created_at,created_at+interval '2 days','/admin/cases/'||id FROM app.support_cases WHERE state IN ('OPEN','IN_PROGRESS');
