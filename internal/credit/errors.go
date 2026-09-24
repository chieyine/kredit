package credit

import (
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
)

// ErrRequestNotFound deliberately does not disclose whether another tenant owns the request.
var ErrRequestNotFound = errors.New("credit request not found")

// SQLSTATE codes raised by the credit-approval triggers (migration 205).
const (
	sqlStateApprovalRequired        = "KR001"
	sqlStateReviewerCeilingExceeded = "KR002"
)

// NeedsIndependentApproval reports whether the database refused an offer
// because it lacks an independent approval for these exact terms, or because
// the approving reviewer's ceiling is below the offer amount.
func NeedsIndependentApproval(err error) bool {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return false
	}
	return pgErr.Code == sqlStateApprovalRequired || pgErr.Code == sqlStateReviewerCeilingExceeded
}

// ReviewerCeilingExceeded reports whether the approving reviewer's ceiling is
// below the offer amount.
func ReviewerCeilingExceeded(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == sqlStateReviewerCeilingExceeded
}
