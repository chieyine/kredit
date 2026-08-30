#!/usr/bin/env bash
set -euo pipefail

root_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$root_dir"
: "${DATABASE_URL:?DATABASE_URL is required}"

output="docs/compliance/data-inventory.tsv"
temporary="${output}.tmp"
{
  printf 'schema\ttable\tfield\tclassification\tsubject\tsource\tpurpose\tlawful_basis\treaders\twriters\tprotection\tretention\tdeletion_hold_behavior\tprocessor\tlocation_transfer\towner\n'
  psql "$DATABASE_URL" -A -F $'\t' -t -c "
    SELECT c.table_schema,c.table_name,c.column_name,
      CASE
        WHEN c.table_name='analytics_events' THEN 'pseudonymous_product_analytics'
        WHEN c.column_name ~* '(token|secret|password|otp|session|mfa|hash|cipher)' THEN 'restricted_authentication'
        WHEN c.column_name ~* '(amount|kobo|account|payment|mandate|settlement|principal|balance|fee|currency)' OR c.table_schema='ledger' THEN 'restricted_financial'
        WHEN c.column_name ~* '(email|phone|name|address|identity|user_id|person|representative|birth|bvn|nin)' THEN 'restricted_identity'
        ELSE 'confidential_commercial' END,
      CASE WHEN c.table_schema='river' THEN 'platform_operator' WHEN c.column_name ~* '(user|person|email|phone)' THEN 'account_user' ELSE 'business_or_transaction' END,
      'application_or_authorized_provider',
      CASE WHEN c.table_schema='ledger' THEN 'financial_accounting' WHEN c.table_name='analytics_events' THEN 'pilot_measurement_and_product_reliability' WHEN c.table_name ~* '(recovery|session|otp|mfa)' THEN 'account_security' WHEN c.table_name ~* '(privacy|consent)' THEN 'privacy_rights' ELSE 'operate_trade_credit_service' END,
      'pending_legal_approval',
      CASE WHEN c.table_name='analytics_events' THEN 'aal2_compliance_operations' WHEN c.table_name ~* '(recovery|privacy|verification|audit)' THEN 'authorized_user;case_bound_compliance' ELSE 'tenant_scoped_application;authorized_operations' END,
      'validated_application_command;authorized_worker',
      CASE WHEN c.column_name ~* '(hash|hmac)' THEN 'one_way_hmac_or_hash' WHEN c.column_name ~* '(cipher|token|secret)' THEN 'encrypted_or_opaque_reference' ELSE 'database_encryption;rls;access_audit' END,
      'environment_retention_register_pending_legal_approval',
      CASE WHEN c.table_schema='ledger' OR c.table_name ~* '(payment|obligation|agreement|audit|legal_hold)' THEN 'retain_under_financial_or_legal_hold;restrict_or_pseudonymize' ELSE 'delete_or_pseudonymize_after_approved_request_and_retention_expiry' END,
      CASE WHEN c.table_name ~* '(provider|mandate|settlement|notification)' THEN 'approved_provider_if_configured' ELSE 'kredit' END,
      'configured_region;transfer_assessment_required_for_external_processor',
      CASE WHEN c.table_schema='ledger' THEN 'Financial Systems Lead' WHEN c.table_name='analytics_events' THEN 'Data Platform Lead' WHEN c.table_name ~* '(privacy|legal_hold|restriction)' THEN 'Data Protection Lead' WHEN c.table_name ~* '(recovery|session|otp|mfa)' THEN 'Security Engineering Lead' ELSE 'Domain Engineering Lead' END
    FROM information_schema.columns c
    JOIN information_schema.tables t USING(table_schema,table_name)
    WHERE t.table_type='BASE TABLE' AND c.table_schema IN ('app','ledger','river')
    ORDER BY c.table_schema,c.table_name,c.ordinal_position;"
} > "$temporary"
mv "$temporary" "$output"
printf 'Generated %s with %s field rows.\n' "$output" "$(( $(wc -l < "$output") - 1 ))"
