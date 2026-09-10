package documents

import (
	"context"
	"errors"
	"time"
)

// ObjectCandidate exposes no document contents or credentials.
type ObjectCandidate struct {
	Key        string
	ModifiedAt time.Time
}
type ObjectJanitor interface {
	ListObjects(context.Context, string) ([]ObjectCandidate, string, error)
	DeleteObject(context.Context, string) error
}

// CleanupOrphans processes one bounded storage page. Completed documents,
// including quarantined/rejected evidence, never qualify for deletion.
func (s *Store) CleanupOrphans(ctx context.Context) error {
	if s == nil || s.pool == nil {
		return nil
	}
	janitor, ok := s.objects.(ObjectJanitor)
	if !ok {
		return nil
	}
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	s.cleanupMu.Lock()
	defer s.cleanupMu.Unlock()
	candidates, next, err := janitor.ListObjects(ctx, s.cleanupCursor)
	if err != nil {
		return err
	}
	for _, candidate := range candidates {
		if candidate.ModifiedAt.IsZero() || candidate.ModifiedAt.After(time.Now().Add(-7*24*time.Hour)) {
			continue
		}
		var eligible bool
		if err = s.pool.QueryRow(ctx, `SELECT app.document_object_is_orphan($1)`, candidate.Key).Scan(&eligible); err != nil {
			return err
		}
		if !eligible {
			continue
		}
		if err = janitor.DeleteObject(ctx, candidate.Key); err != nil {
			return err
		}
		if _, err = s.pool.Exec(ctx, `INSERT INTO app.audit_events(action,resource_type,resource_id,outcome,metadata) VALUES('document.orphan_removed','object_storage',$1,'success','{"reason":"expired_incomplete_or_unreferenced","minimum_age_days":7}')`, candidate.Key); err != nil {
			return errors.New("orphan removal audit could not be saved")
		}
	}
	s.cleanupCursor = next
	return nil
}
