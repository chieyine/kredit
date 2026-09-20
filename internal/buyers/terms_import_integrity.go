package buyers

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"github.com/jackc/pgx/v5"
	"kredit/internal/access"
	"kredit/internal/db"
)

var ErrTermsAuthority = errors.New("current terms import authority required")

// TermsSourceHash hashes the canonical typed payload, independently of the HTTP
// idempotency key. Row order is meaningful and is preserved in the fingerprint.
func TermsSourceHash(rows []TermsImportRowInput) (string, error) {
	if len(rows) == 0 || len(rows) > 500 {
		return "", ErrTermsInvalidRow
	}
	for _, row := range rows {
		if row.ProposedCreditLimitKobo < 0 || row.ProposedCreditLimitKobo > 9007199254740991 ||
			row.OpeningBalanceKobo < 0 || row.OpeningBalanceKobo > 9007199254740991 ||
			row.ProposedGraceHours < 0 || row.ProposedGraceHours > 720 ||
			len(row.CustomerName) > 1000 || len(row.ContactEmail) > 320 || len(row.ContactPhone) > 64 || len(row.OpeningBalanceReference) > 500 {
			return "", ErrTermsInvalidRow
		}
	}
	payload, err := json.Marshal(rows)
	if err != nil {
		return "", err
	}
	hash := sha256.Sum256(append([]byte("kredit:terms-import:v1\x00"), payload...))
	return hex.EncodeToString(hash[:]), nil
}

const termsBatchColumns = `id::text,organization_id::text,source_hash,total_rows,valid_rows,invalid_rows,state,
    uploaded_by::text,COALESCE(approved_by::text,''),COALESCE(cancelled_by::text,''),created_at,approved_at,cancelled_at`

type termsRowScanner interface{ Scan(...any) error }

func scanTermsBatch(row termsRowScanner) (TermsImportBatch, error) {
	var batch TermsImportBatch
	err := row.Scan(&batch.ID, &batch.OrganizationID, &batch.SourceHash, &batch.TotalRows, &batch.ValidRows,
		&batch.InvalidRows, &batch.State, &batch.UploadedBy, &batch.ApprovedBy, &batch.CancelledBy,
		&batch.CreatedAt, &batch.ApprovedAt, &batch.CancelledAt)
	batch.FinancialApplication = "not_applied"
	return batch, err
}

func cloneTermsBatch(batch TermsImportBatch) TermsImportBatch {
	if batch.ApprovedAt != nil {
		at := *batch.ApprovedAt
		batch.ApprovedAt = &at
	}
	if batch.CancelledAt != nil {
		at := *batch.CancelledAt
		batch.CancelledAt = &at
	}
	return batch
}

func (p *PostgresTermsImportStore) beginTermsTx(ctx context.Context, actor, org, operation string) (pgx.Tx, error) {
	if p.pool == nil {
		return nil, errors.New("database pool unavailable")
	}
	if actor == "" || org == "" {
		return nil, ErrTermsAuthority
	}
	if identity, ok := db.TenantFromContext(ctx); ok && ((identity.UserID != "" && identity.UserID != actor) || (identity.OrganizationID != "" && identity.OrganizationID != org)) {
		return nil, ErrTermsAuthority
	}
	tx, err := (&db.ScopedDatabase{Pool: p.pool}).Begin(db.WithTenantContext(ctx, actor, org))
	if err != nil {
		return nil, err
	}
	fail := func(err error) (pgx.Tx, error) { _ = tx.Rollback(ctx); return nil, err }
	var role access.Role
	err = tx.QueryRow(ctx, `SELECT m.role FROM app.memberships m
        JOIN app.users u ON u.id=m.user_id JOIN app.organizations o ON o.id=m.organization_id
        WHERE m.organization_id=$1::uuid AND m.user_id=$2::uuid AND m.status='active'
        AND u.status='active' AND o.status<>'suspended' FOR SHARE OF m,u,o`, org, actor).Scan(&role)
	if errors.Is(err, pgx.ErrNoRows) {
		return fail(ErrTermsAuthority)
	}
	if err != nil {
		return fail(err)
	}
	allowed := false
	switch operation {
	case "stage":
		allowed = access.Can(role, access.PermissionInviteBuyers)
	case "review":
		allowed = access.Can(role, access.PermissionApproveBusinessCredit)
	case "read":
		allowed = access.Can(role, access.PermissionInviteBuyers) || access.Can(role, access.PermissionApproveBusinessCredit)
	}
	var companyWide bool
	if err = tx.QueryRow(ctx, `SELECT app.branch_scope_all($1::uuid)`, org).Scan(&companyWide); err != nil {
		return fail(err)
	}
	if !allowed || !companyWide {
		return fail(ErrTermsAuthority)
	}
	return tx, nil
}
