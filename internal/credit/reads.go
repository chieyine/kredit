package credit

import (
	"context"
	"encoding/json"
	"fmt"
	"kredit/internal/db"
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
	identity, _ := db.TenantFromContext(ctx)
	if field == "supplier_organization_id" {
		if identity.OrganizationID != "" && identity.OrganizationID != id {
			return nil, fmt.Errorf("supplier scope does not match authorized business")
		}
		if _, err = tx.Exec(ctx, `SELECT set_config('app.current_user_id',$1,true)`, identity.UserID); err != nil {
			return nil, err
		}
	}
	setting := "app.current_organization_id"
	if field == "buyer_user_id" {
		setting = "app.current_user_id"
	}
	if _, err = tx.Exec(ctx, `SELECT set_config($1,$2,true)`, setting, id); err != nil {
		return nil, err
	}
	predicate := "c." + field + "=$1::uuid"
	if field == "buyer_user_id" {
		predicate = "(c.buyer_user_id=$1::uuid OR c.buyer_business_id IN (SELECT b.id FROM app.businesses b JOIN app.memberships m ON m.organization_id=b.organization_id WHERE m.user_id=$1::uuid AND m.status='active' AND app.can_purchase(b.id)))"
	}
	rows, err := tx.Query(ctx, fmt.Sprintf(`SELECT c.id::text,c.buyer_user_id::text,c.buyer_business_id::text, CASE WHEN o.id IS NOT NULL THEN jsonb_set(s.aggregate,'{obligation}',to_jsonb(o)) ELSE s.aggregate END FROM app.credit_requests c LEFT JOIN app.credit_aggregate_snapshots s ON s.credit_request_id=c.id::text LEFT JOIN app.obligations o ON o.credit_request_id=c.id WHERE %s ORDER BY c.updated_at DESC,c.id`, predicate), id)
	if err != nil {
		return nil, err
	}
	views := []View{}
	for rows.Next() {
		var encoded []byte
		var requestID, ownerID, profileID string
		var v View
		if err = rows.Scan(&requestID, &ownerID, &profileID, &encoded); err != nil {
			rows.Close()
			return nil, err
		}
		if err = json.Unmarshal(encoded, &v); err != nil {
			rows.Close()
			return nil, err
		}
		if v.Request.ID != requestID || v.Request.BuyerUserID != ownerID || v.Request.BuyerBusinessID != profileID || (field == "supplier_organization_id" && v.Request.SupplierOrganizationID != id) {
			rows.Close()
			return nil, fmt.Errorf("credit projection identity does not match its saved record")
		}
		if field == "buyer_user_id" && v.Request.State == Draft {
			continue
		}
		if v.Mandate != nil && (field == "supplier_organization_id" || v.Request.BuyerUserID != id) {
			v.Mandate.AuthorizationURL = ""
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
