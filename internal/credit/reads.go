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
	if s == nil || s.pool == nil {
		return nil, fmt.Errorf("credit database is not configured")
	}
	if field != "supplier_organization_id" && field != "buyer_user_id" {
		return nil, fmt.Errorf("invalid credit scope")
	}
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
	rows, err := tx.Query(ctx, fmt.Sprintf(`SELECT c.id::text, CASE WHEN o.id IS NOT NULL THEN jsonb_set(s.aggregate,'{obligation}',to_jsonb(o)) ELSE s.aggregate END FROM app.credit_requests c LEFT JOIN app.credit_aggregate_snapshots s ON s.credit_request_id=c.id::text LEFT JOIN app.obligations o ON o.credit_request_id=c.id WHERE c.%s=$1::uuid ORDER BY c.updated_at DESC,c.id`, field), id)
	if err != nil {
		return nil, err
	}
	views := []View{}
	for rows.Next() {
		var encoded []byte
		var requestID string
		var v View
		if err = rows.Scan(&requestID, &encoded); err != nil {
			rows.Close()
			return nil, err
		}
		if err = json.Unmarshal(encoded, &v); err != nil {
			rows.Close()
			return nil, err
		}
		if v.Request.ID != requestID || (field == "supplier_organization_id" && v.Request.SupplierOrganizationID != id) || (field == "buyer_user_id" && v.Request.BuyerUserID != id) {
			rows.Close()
			return nil, fmt.Errorf("credit projection identity does not match its saved record")
		}
		if field == "buyer_user_id" && v.Request.State == Draft {
			continue
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
