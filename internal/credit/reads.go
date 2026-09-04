package credit

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

func (s *PostgresStore) ReadForSupplier(ctx context.Context, id string) ([]View, error) {
	return s.readViews(ctx, "supplier_organization_id", id)
}
func (s *PostgresStore) ReadForBuyer(ctx context.Context, id string) ([]View, error) {
	return s.readViews(ctx, "buyer_user_id", id)
}
func (s *PostgresStore) readViews(ctx context.Context, field, id string) ([]View, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	setting := "app.current_organization_id"
	if field == "buyer_user_id" {
		setting = "app.current_user_id"
	}
	if _, err = tx.Exec(ctx, `SELECT set_config($1,$2,true)`, setting, id); err != nil {
		return nil, err
	}
	rows, err := tx.Query(ctx, fmt.Sprintf(`SELECT CASE WHEN o.id IS NOT NULL THEN jsonb_set(s.aggregate,'{obligation}',to_jsonb(o)) ELSE s.aggregate END FROM app.credit_aggregate_snapshots s LEFT JOIN app.obligations o ON o.credit_request_id::text=s.credit_request_id WHERE s.%s=$1 ORDER BY s.updated_at DESC,s.credit_request_id`, field), id)
	if err != nil {
		return nil, err
	}
	views := []View{}
	for rows.Next() {
		var encoded []byte
		var v View
		if err = rows.Scan(&encoded); err != nil {
			rows.Close()
			return nil, err
		}
		if err = json.Unmarshal(encoded, &v); err != nil {
			rows.Close()
			return nil, err
		}
		views = append(views, v)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, err
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, err
	}
	return views, nil
}
