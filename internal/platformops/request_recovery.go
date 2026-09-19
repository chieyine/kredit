package platformops

import (
	"context"
	"errors"
	"regexp"
	"time"

	"kredit/internal/access"
	"kredit/internal/db"
)

var recoveryReferencePattern = regexp.MustCompile(`^kredit-[0-9a-f]{32}$`)

// requestRecovery exposes only receipt metadata for an exact random reference.
// A recorded HTTP response is not evidence that a bank payment settled.
func (s *Store) requestRecovery(ctx context.Context, reference string) ([]SearchResult, error) {
	identity, ok := db.TenantFromContext(ctx)
	if !ok || identity.UserID == "" {
		return nil, errors.New("operator identity is required")
	}
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err = access.LockPlatformAuthority(ctx, tx, identity.UserID, access.PermissionSupportSearch); err != nil {
		return nil, err
	}
	rows, err := tx.Query(ctx, `SELECT id::text,idempotency_key,
 CASE WHEN completed_at IS NULL THEN 'RESULT_UNCONFIRMED'
      ELSE 'HTTP_RESPONSE_RECORDED_' || COALESCE(response_status::text,'UNKNOWN') END
 FROM app.idempotency_records WHERE idempotency_key=$1 ORDER BY created_at DESC,id LIMIT 25`, reference)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	results := []SearchResult{}
	for rows.Next() {
		item := SearchResult{Type: "request_receipt"}
		if err := rows.Scan(&item.ID, &item.Reference, &item.State); err != nil {
			return nil, err
		}
		results = append(results, item)
	}
	return results, rows.Err()
}
