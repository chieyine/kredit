package platformops

import (
	"context"
	"encoding/json"
)

func (s *Store) ProviderWork(ctx context.Context) (json.RawMessage, error) {
	actor, err := adminReadActor(ctx)
	if err != nil {
		return nil, err
	}
	tx, err := s.beginAdminRead(ctx, actor)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var result []byte
	err = tx.QueryRow(ctx, `SELECT app.provider_work()`).Scan(&result)
	return result, err
}
