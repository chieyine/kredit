#!/usr/bin/env bash
set -euo pipefail

root_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$root_dir"

failures=0
require_file() { if [[ ! -f "$1" ]]; then printf 'missing required file: %s\n' "$1" >&2; failures=$((failures+1)); fi; }
require_text() { local file="$1" text="$2"; if ! grep -Fq -- "$text" "$file"; then printf 'missing required contract evidence: %s in %s\n' "$text" "$file" >&2; failures=$((failures+1)); fi; }

for file in README.md IMPLEMENTATION_STATUS.md CHANGELOG.md docs/api/openapi.yaml docs/threat-model.md docs/data-map.md docs/product/open-questions.md docs/product/readme-completion-plan.md docs/testing/test-matrix.md docs/release/readiness-checklist.md docs/operations/readme-gap-audit.md; do require_file "$file"; done
require_file docs/product/readme-completion-traceability.md
require_file docs/product/world-class-product-standard.md
require_file web/tests/product-quality.spec.ts
require_file web/src/routes/+error.svelte
require_file web/src/lib/seo.ts
require_file web/src/lib/components/PortalNav.svelte
require_file web/src/lib/components/ProtectedActionDialog.svelte
for asset in web/static/og.png web/static/icon-192.png web/static/icon-512.png web/static/apple-touch-icon.png web/static/manifest.webmanifest web/static/robots.txt web/static/sitemap.xml; do require_file "$asset"; done

for command in dev build test test:unit test:integration test:e2e test:race test:fuzz lint security generate api:lint readme:check plan:check db:migrate db:rollback db:reset db:seed db:check data:inventory:generate data:inventory:check openapi:generate sqlc:generate web:check web:test ci; do require_text Taskfile.yml "  ${command}:"; done

for command in api worker migrate seed reconcile provider-simulator; do require_file "cmd/${command}/main.go"; done

components=(Money MoneyInput Percentage DateTime DueDate StatusPill RiskFact ReferenceCode Timeline AuditTimeline AgreementSummary MandateStatus PaymentBreakdown OutstandingBalance FeeBreakdown TradeLineMeter ScheduleTable CollectionAttemptCard DisputePanel CustomerIdentityCard BusinessVerificationCard DocumentUploader DocumentViewer ConfirmFinancialAction StepUpAuthDialog EmptyState InlineError SystemBanner)
for component in "${components[@]}"; do require_file "web/src/lib/components/${component}.svelte"; done

routes=(
  '+page.svelte' 'how-it-works/+page.svelte' 'pricing/+page.svelte' 'security/+page.svelte' 'for-suppliers/+page.svelte' 'for-buyers/+page.svelte' 'faq/+page.svelte'
  'legal/terms/+page.svelte' 'legal/privacy/+page.svelte' 'legal/complaints/+page.svelte'
  'app/+page.svelte' 'app/overview/+page.svelte' 'app/credit/new/+page.svelte' 'app/credit/[id]/+page.svelte' 'app/customers/+page.svelte' 'app/customers/[id]/+page.svelte' 'app/trade-lines/+page.svelte' 'app/trade-lines/[id]/+page.svelte' 'app/payments/+page.svelte' 'app/collections/+page.svelte' 'app/overdue/+page.svelte' 'app/disputes/+page.svelte' 'app/reports/+page.svelte' 'app/team/+page.svelte' 'app/settings/+page.svelte' 'app/settings/billing/+page.svelte' 'app/settings/settlement/+page.svelte' 'app/settings/security/+page.svelte'
  'app/onboarding/+page.svelte' 'app/settings/credit-policy/+page.svelte'
  'app/settings/notifications/+page.svelte' 'app/settings/privacy/+page.svelte' 'recover/+page.svelte' 'admin/recovery/+page.svelte' 'admin/privacy/+page.svelte'
  'buyer/+page.svelte' 'buyer/requests/+page.svelte' 'buyer/obligations/+page.svelte' 'buyer/obligations/[id]/+page.svelte' 'buyer/trade-lines/+page.svelte' 'buyer/history/+page.svelte' 'buyer/mandates/+page.svelte' 'buyer/disputes/+page.svelte' 'buyer/settings/+page.svelte'
  'c/[token]/+page.svelte' 'pay/[token]/+page.svelte' 'receipt/[public_token]/+page.svelte'
  'admin/+page.svelte' 'admin/search/+page.svelte' 'admin/cases/[id]/+page.svelte' 'admin/disputes/[id]/+page.svelte' 'admin/provider-events/+page.svelte' 'admin/jobs/+page.svelte' 'admin/audit/+page.svelte'
  'admin/analytics/+page.svelte'
)
for route in "${routes[@]}"; do require_file "web/src/routes/${route}"; done

for table in users sessions otp_challenges mfa_methods organizations memberships persons businesses business_representatives verification_cases bank_account_references trade_relationships relationship_consents credit_requests agreement_versions agreement_acceptances obligations repayment_schedules schedule_items trade_lines drawdowns goods_releases receipt_confirmations documents payment_mandates mandate_events payments payment_allocations payment_claims collection_reservations collection_attempts settlement_events fees disputes dispute_evidence dispute_decisions provider_events idempotency_records audit_events notifications support_cases; do
  if ! grep -REq "CREATE TABLE( IF NOT EXISTS)? app\.${table}[ (]" db/migrations; then printf 'missing README data model table: app.%s\n' "$table" >&2; failures=$((failures+1)); fi
done
for table in accounts transactions postings; do require_text db/migrations/004_milestone3_credit_ledger.sql "CREATE TABLE ledger.${table}"; done

for fixture in 'ABC Pharmaceuticals Ltd' 'Royal Pharmacy Ltd' 'Scenario A' 'Scenario B' 'Scenario C' 'Scenario D' 'Scenario E' 'Scenario F'; do require_text db/seeds/001_demo.sql "$fixture"; done
require_text docker-compose.yml 'provider-simulator:'
require_text scripts/dev.sh 'provider-simulator'
require_text web/src/service-worker.ts 'Financial'

for file in infra/environments/versions.tf infra/environments/main.tf infra/environments/variables.tf infra/monitoring/prometheus-rules.yaml infra/monitoring/otel-collector.yaml tests/contract/provider_contract_test.go tests/e2e/README.md tests/performance/acceptance.js tests/performance/portfolio_queries.sql; do require_file "$file"; done
for route in '/buyer/credit-requests/{requestID}/payment-claims:' '/buyer/mandates/{mandateID}/cancel:' '/buyer/mandates/{mandateID}/restore:' '/buyer/obligations/{obligationID}:' '/public/receipts/{token}:' '/public/payment-intents/{token}:'; do require_text api/openapi.yaml "  ${route}"; done
for route in '/buyer/credit-requests/{requestID}/agreement-document:' '/organizations/{organizationID}/credit-requests/{requestID}/agreement-document:'; do require_text api/openapi.yaml "  ${route}"; done
require_file internal/agreementdocs/document.go
require_text internal/credit/store.go 'CustomScheduleItems'
require_text internal/credit/store.go 'validateScheduleTerms'
require_text internal/web/server.go 'PATCH /api/v1/organizations/{organizationID}/trade-lines/{lineID}'
require_file db/migrations/045_trade_line_mandate_integrity.sql
require_file db/migrations/046_trade_line_drawdown_lifecycle.sql
require_file db/migrations/047_supplier_onboarding_readiness.sql
require_file db/migrations/048_user_control_and_privacy.sql
require_file db/migrations/050_product_analytics.sql
require_file db/migrations/051_analytics_contract_completion.sql
require_file internal/reports/analytics.go
require_file docs/product/analytics-event-catalog.md
require_file docs/product/pilot-scorecard.md
require_file docs/runbooks/pilot-scorecard.md
require_text api/openapi.yaml '/ops/analytics/scorecard:'
require_file internal/onboarding/store.go
require_file internal/onboarding/postgres.go
require_text internal/web/server.go '/onboarding/settlement'
require_text internal/web/credit_handlers.go 'requireSupplierReady'
require_text internal/jobs/client.go 'OpReconcileSupplierOnboarding'
require_text api/openapi.yaml '/onboarding/credit-policy:'
require_text api/openapi.yaml '/me/notification-preferences:'
require_text api/openapi.yaml '/account-recovery/requests:'
require_text api/openapi.yaml '/me/privacy-requests:'
require_file internal/usercontrol/store.go
require_file docs/compliance/data-inventory.tsv
require_file scripts/data-inventory-check.sh
require_text internal/tradelines/store.go 'VerifyAgreementHash'
require_text internal/tradelines/postgres.go 'SetTransactionalActivationHandler'
require_text internal/credit/postgres.go 'ActivateTradeLineDrawdownTx'
require_text internal/jobs/client.go 'ExpireDrawdownReservations'
require_text internal/web/server.go 'drawdowns/{drawdownID}/release'
require_text internal/web/server.go 'drawdowns/{drawdownID}/receipt'
require_text api/openapi.yaml '/drawdowns/{drawdownID}/agreement-document:'
if grep -Rq 'ActivateDrawdown' internal api web/src; then printf 'deprecated external drawdown activation remains\n' >&2; failures=$((failures+1)); fi
require_text internal/mandates/provider.go 'ResolveTradeLineMandate'
require_text internal/tradelines/store.go 'MandateVerified'
require_text api/openapi.yaml 'mandate_id:'
require_text internal/config/config.go 'OFF_PLATFORM_PAYMENT_CLAIMS_ENABLED'
require_file docs/product/workstream-evidence.tsv
require_file docs/product/wave0-contracts.md
require_file docs/product/interface-copy.md
require_file scripts/implementation-plan-conformance.sh
bash scripts/implementation-plan-conformance.sh

if [[ "$failures" -ne 0 ]]; then printf 'README conformance failed with %d missing item(s).\n' "$failures" >&2; exit 1; fi
printf 'README structural conformance passed.\n'
