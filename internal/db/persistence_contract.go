package db

import (
	"context"
	"errors"
	"fmt"
	"sort"

	"github.com/jackc/pgx/v5"
)

// RequiredPersistenceObjects is the database contract for a deployable API
// and worker. It intentionally names every state-bearing table, rather than
// checking only the ledger and job tables; a migrated-but-partial database
// must never be reported as ready.
var RequiredPersistenceObjects = []string{
	"app.users",
	"app.sessions",
	"app.otp_challenges",
	"app.mfa_methods",
	"app.organizations",
	"app.memberships",
	"app.organization_invitations",
	"app.audit_events",
	"app.persons",
	"app.businesses",
	"app.business_representatives",
	"app.verification_cases",
	"app.identity_consents",
	"app.bank_account_references",
	"app.buyer_invitations",
	"app.trade_relationships",
	"app.credit_requests",
	"app.credit_aggregate_snapshots",
	"app.agreement_versions",
	"app.agreement_acceptances",
	"app.mandates",
	"app.payment_mandates",
	"app.mandate_events",
	"app.goods_releases",
	"app.receipt_confirmations",
	"app.obligations",
	"ledger.accounts",
	"ledger.transactions",
	"ledger.postings",
	"app.payments",
	"app.payment_claims",
	"app.payment_allocations",
	"app.fees",
	"app.settlement_events",
	"app.repayment_schedules",
	"app.schedule_items",
	"app.trade_lines",
	"app.drawdowns",
	"app.drawdown_reservations",
	"app.collection_reservations",
	"app.collection_attempts",
	"app.collection_aggregate_snapshots",
	"app.collection_attempt_index",
	"app.collection_provider_events",
	"app.disputes",
	"app.dispute_evidence",
	"app.dispute_decisions",
	"app.operation_actions",
	"app.notification_templates",
	"app.notification_preferences",
	"app.notifications",
	"app.messaging_events",
	"app.correction_requests",
	"app.correction_decisions",
	"app.analytics_events",
	"app.provider_approvals",
	"app.provider_events",
	"app.provider_reconciliation_events",
	"app.release_evidence",
	"app.pilot_limit_configs",
	"app.relationship_consents",
	"app.documents",
	"app.support_cases",
	"app.support_case_events",
	"app.idempotency_records",
	"app.outbox_events",
	"app.platform_role_assignments",
	"app.provider_webhook_inbox",
	"app.job_dead_letters",
	"jobs.river_job",
}

// RequiredPersistenceFunctions are versioned SQL capabilities used by the
// PostgreSQL authentication adapter. A complete table set without these
// functions would still fail at runtime.
var RequiredPersistenceFunctions = []string{
	"app.find_or_create_user(text,text,timestamptz)",
	"app.session_by_token_hash(bytea)",
	"app.organization_count()",
	"app.buyer_invitation_by_token_hash(bytea)",
	"app.business_count()",
	"app.supplier_customers(uuid)",
	"app.credit_snapshot_by_id(text)",
	"app.credit_snapshot_by_obligation(text)",
	"app.payment_mandate_by_provider(text,text)",
	"app.trade_line_mandate(uuid,uuid,uuid)",
}

var RequiredPersistenceColumns = []string{
	"app.otp_challenges.target_ciphertext",
	"app.buyer_invitations.target_ciphertext",
	"app.payment_mandates.primary_account_token_ciphertext",
}

// MissingPersistenceObjects returns the names from the contract that are not
// present in PostgreSQL. The result is sorted and contains no tenant or row
// data, making it safe to include in startup diagnostics.
func (p *Pool) MissingPersistenceObjects(ctx context.Context) ([]string, error) {
	if p == nil || p.inner == nil {
		return nil, errors.New("postgres pool is not configured")
	}
	required := append([]string(nil), RequiredPersistenceObjects...)
	rows, err := p.inner.Query(ctx, `
		SELECT required.object_name
		FROM unnest($1::text[]) AS required(object_name)
		WHERE to_regclass(required.object_name) IS NULL
		ORDER BY required.object_name`, required)
	if err != nil {
		return nil, fmt.Errorf("check postgres persistence contract: %w", err)
	}
	defer rows.Close()
	var missing []string
	for rows.Next() {
		var objectName string
		if err := rows.Scan(&objectName); err != nil {
			return nil, fmt.Errorf("scan postgres persistence contract: %w", err)
		}
		missing = append(missing, objectName)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read postgres persistence contract: %w", err)
	}
	sort.Strings(missing)
	return missing, nil
}

// CheckPersistenceContract enforces the complete state-table contract. This
// check complements CheckSchema: a healthy connection and a few anchor tables
// are not sufficient for a multi-tenant financial deployment.
func (p *Pool) CheckPersistenceContract(ctx context.Context) error {
	missing, err := p.MissingPersistenceObjects(ctx)
	if err != nil {
		return err
	}
	missingFunctions, err := p.missingPersistenceFunctions(ctx)
	if err != nil {
		return err
	}
	missingColumns, err := p.missingPersistenceColumns(ctx)
	if err != nil {
		return err
	}
	missing = append(missing, missingFunctions...)
	missing = append(missing, missingColumns...)
	sort.Strings(missing)
	if len(missing) > 0 {
		return fmt.Errorf("database persistence contract is incomplete (missing: %v)", missing)
	}
	return nil
}

func (p *Pool) missingPersistenceFunctions(ctx context.Context) ([]string, error) {
	rows, err := p.inner.Query(ctx, `
		SELECT required.function_name
		FROM unnest($1::text[]) AS required(function_name)
		WHERE to_regprocedure(required.function_name) IS NULL
		ORDER BY required.function_name`, RequiredPersistenceFunctions)
	if err != nil {
		return nil, fmt.Errorf("check postgres persistence functions: %w", err)
	}
	defer rows.Close()
	return scanMissingNames(rows, "persistence function")
}

func (p *Pool) missingPersistenceColumns(ctx context.Context) ([]string, error) {
	rows, err := p.inner.Query(ctx, `
		SELECT required.column_name
		FROM unnest($1::text[]) AS required(column_name)
		WHERE NOT EXISTS (
			SELECT 1
			FROM information_schema.columns c
			WHERE c.table_schema = split_part(required.column_name, '.', 1)
			  AND c.table_name = split_part(required.column_name, '.', 2)
			  AND c.column_name = split_part(required.column_name, '.', 3)
		)
		ORDER BY required.column_name`, RequiredPersistenceColumns)
	if err != nil {
		return nil, fmt.Errorf("check postgres persistence columns: %w", err)
	}
	defer rows.Close()
	return scanMissingNames(rows, "persistence column")
}

func scanMissingNames(rows pgx.Rows, label string) ([]string, error) {
	var missing []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("scan %s: %w", label, err)
		}
		missing = append(missing, name)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read %s: %w", label, err)
	}
	sort.Strings(missing)
	return missing, nil
}
