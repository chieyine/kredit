package paymentclaims

import (
	"errors"
	"strings"
)

// A replay acknowledges the original decision; it must not imply that a
// different reviewer or different evidence has been recorded. The caller
// still owns authentication, current authority checks and transaction locks.
var ErrReviewConflict = errors.New("payment claim already has different review details")

func sameReview(claim Claim, actor, reason string) bool {
	return claim.ReviewedBy == actor && claim.ReviewReason == strings.TrimSpace(reason)
}
