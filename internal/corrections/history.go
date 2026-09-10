package corrections

import (
	"context"
	"encoding/json"
	"sort"
	"time"
)

// ReviewedRequest joins a request to its durable review evidence. Approval is
// an annotation; balances continue to come from payments and approved changes.
type ReviewedRequest struct {
	Request
	Decisions []Decision `json:"decisions"`
}

func (s *Store) ReadForBuyer(ctx context.Context, userID string) ([]ReviewedRequest, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := []ReviewedRequest{}
	for _, r := range s.requests {
		if r.RequestedBy == userID {
			out = append(out, ReviewedRequest{Request: clone(*r), Decisions: append([]Decision{}, s.decisions[r.ID]...)})
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].CreatedAt.Equal(out[j].CreatedAt) {
			return out[i].ID > out[j].ID
		}
		return out[i].CreatedAt.After(out[j].CreatedAt)
	})
	return out, nil
}
func (s *PostgresStore) ReadForBuyer(ctx context.Context, userID string) ([]ReviewedRequest, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err = tx.Exec(ctx, `SELECT set_config('app.current_user_id',$1,true)`, userID); err != nil {
		return nil, err
	}
	rows, err := tx.Query(ctx, `SELECT to_jsonb(r)||jsonb_build_object('decisions',COALESCE((SELECT jsonb_agg(to_jsonb(d) ORDER BY d.decided_at,d.id) FROM app.correction_decisions d WHERE d.request_id=r.id),'[]'::jsonb)) FROM app.correction_requests r WHERE r.requested_by=$1::uuid ORDER BY r.created_at DESC,r.id DESC`, userID)
	if err != nil {
		return nil, err
	}
	out := []ReviewedRequest{}
	for rows.Next() {
		var raw []byte
		var r ReviewedRequest
		if err = rows.Scan(&raw); err != nil {
			rows.Close()
			return nil, err
		}
		if err = json.Unmarshal(raw, &r); err != nil {
			rows.Close()
			return nil, err
		}
		out = append(out, r)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, err
	}
	return out, tx.Commit(ctx)
}
