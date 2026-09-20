package web

import (
	"context"
	"kredit/internal/db"
	"strings"
)

// Branch-scoped staff use customer-linked workflows. Company-wide administration,
// consumer records and unallocated money require company-wide authority until
// those records have their own branch attribution.
func branchScopedRoute(pattern string) bool {
	parts := strings.SplitN(pattern, " ", 2)
	if len(parts) != 2 {
		return false
	}
	method, path := parts[0], parts[1]
	const root = "/api/v1/organizations/{organizationID}"
	if !strings.HasPrefix(path, root) {
		return false
	}
	suffix := strings.TrimPrefix(path, root)
	if method == "GET" {
		switch suffix {
		case "", "/network-operations", "/branch-access", "/customers", "/payments", "/collections", "/reports/receivables", "/reports/ageing", "/reports/fees", "/reports/enterprise", "/purchasing-authority":
			return true
		case "/credit-requests", "/credit-requests/{requestID}", "/credit-requests/{requestID}/deliveries", "/credit-requests/{requestID}/payments", "/credit-requests/{requestID}/reconciliation", "/credit-requests/{requestID}/agreement-document", "/credit-requests/{requestID}/payment-claims", "/credit-requests/{requestID}/schedule", "/credit-requests/{requestID}/collection/eligibility", "/credit-requests/{requestID}/collections", "/trade-lines", "/trade-lines/{lineID}", "/trade-lines/{lineID}/agreement-document", "/trade-lines/{lineID}/drawdowns/{drawdownID}/agreement-document":
			return true
		}
	}
	if method == "POST" {
		switch suffix {
		case "/credit-terms/preview", "/credit-requests", "/credit-requests/{requestID}/send", "/credit-requests/{requestID}/cancel", "/credit-requests/{requestID}/release", "/credit-requests/{requestID}/line-items", "/credit-requests/{requestID}/shipments", "/credit-requests/{requestID}/receipts", "/credit-requests/{requestID}/credit-notes", "/credit-notes/{noteID}/approve", "/drawdowns/{drawdownID}/approval", "/drawdowns/{drawdownID}/release", "/drawdowns/{drawdownID}/cancel", "/trade-lines/{lineID}/drawdowns", "/trade-lines/{lineID}/drawdowns/{drawdownID}/release", "/trade-lines/{lineID}/drawdowns/{drawdownID}/cancel":
			return true
		}
	}
	if method == "PATCH" && suffix == "/credit-requests/{requestID}" {
		return true
	}

	return false
}
func (s *Server) hasCompanyWideAccess(ctx context.Context, actor, org string) (bool, error) {
	if s.runtime.Database == nil {
		return true, nil
	}
	tx, err := s.runtime.Database.Raw().Begin(db.WithTenantContext(ctx, actor, org))
	if err != nil {
		return false, err
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, `SELECT set_config('app.current_user_id',$1,true),set_config('app.current_organization_id',$2,true)`, actor, org); err != nil {
		return false, err
	}
	var allowed bool
	err = tx.QueryRow(ctx, `SELECT app.branch_scope_all($1::uuid)`, org).Scan(&allowed)
	return allowed, err
}
