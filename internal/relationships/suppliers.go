package relationships

import (
	"context"
	"errors"

	"github.com/google/uuid"
)

type Supplier struct {
	ID          string `json:"id"`
	LegalName   string `json:"legal_name"`
	TradingName string `json:"trading_name"`
}
type SupplierReader interface {
	Suppliers(context.Context, string) ([]Supplier, error)
}

func (s *PostgresStore) Suppliers(ctx context.Context, buyerID string) ([]Supplier, error) {
	if s == nil || s.pool == nil {
		return nil, errors.New("supplier directory unavailable")
	}
	if _, err := uuid.Parse(buyerID); err != nil {
		return nil, errors.New("valid buyer identity required")
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err = tx.Exec(ctx, `SELECT set_config('app.current_user_id',$1,true),set_config('app.current_organization_id','',true)`, buyerID); err != nil {
		return nil, err
	}
	rows, err := tx.Query(ctx, `SELECT supplier_organization_id::text,legal_name,trading_name FROM app.buyer_suppliers()`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []Supplier{}
	for rows.Next() {
		var item Supplier
		if err = rows.Scan(&item.ID, &item.LegalName, &item.TradingName); err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	rows.Close()
	if err = tx.Commit(ctx); err != nil {
		return nil, err
	}
	return result, nil
}
