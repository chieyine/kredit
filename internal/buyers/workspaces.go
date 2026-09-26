package buyers

import (
	"context"
	"errors"
	"sort"
)

// Purchasing profiles are scoped to the person with recorded buyer authority.
// Supplier membership alone never grants access to a customer's other trades.
func (s *PostgresStore) ListBusinessProfiles(ctx context.Context, userID string) ([]Business, error) {
	if s == nil || s.pool == nil {
		return nil, errors.New("business database unavailable")
	}
	tx, err := s.beginTxContext(ctx, userID, "")
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	rows, err := tx.Query(ctx, `SELECT id::text,COALESCE(organization_id::text,''),owner_user_id::text,legal_name,COALESCE(trading_name,''),business_type,business_address,industry,status,created_at FROM app.businesses WHERE (owner_user_id=$1::uuid OR organization_id IN (SELECT organization_id FROM app.memberships WHERE user_id=$1::uuid AND status='active')) AND app.can_purchase(id) ORDER BY created_at,id`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []Business{}
	for rows.Next() {
		var b Business
		if err := rows.Scan(&b.ID, &b.WorkspaceID, &b.OwnerUserID, &b.LegalName, &b.TradingName, &b.BusinessType, &b.BusinessAddress, &b.Industry, &b.Status, &b.CreatedAt); err != nil {
			return nil, err
		}
		result = append(result, b)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	rows.Close()
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return result, nil
}

func (s *PostgresStore) GetProfileByOrganization(ctx context.Context, userID, orgID string) (Business, error) {
	if s == nil || s.pool == nil {
		return Business{}, errors.New("business database unavailable")
	}
	tx, err := s.beginTxContext(ctx, userID, orgID)
	if err != nil {
		return Business{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var b Business
	err = tx.QueryRow(ctx, `SELECT id::text,COALESCE(organization_id::text,''),owner_user_id::text,legal_name,COALESCE(trading_name,''),business_type,business_address,industry,status,created_at FROM app.businesses WHERE organization_id=$1::uuid AND app.can_purchase(id) ORDER BY created_at LIMIT 1`, orgID).Scan(&b.ID, &b.WorkspaceID, &b.OwnerUserID, &b.LegalName, &b.TradingName, &b.BusinessType, &b.BusinessAddress, &b.Industry, &b.Status, &b.CreatedAt)
	if err != nil {
		return Business{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Business{}, err
	}
	return b, nil
}

func (s *Store) ListBusinessProfiles(ctx context.Context, userID string) ([]Business, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := []Business{}
	for _, b := range s.businesses {
		if b.OwnerUserID == userID {
			result = append(result, *b)
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result, nil
}

func (s *Store) GetProfileByOrganization(ctx context.Context, userID, orgID string) (Business, error) {
	if err := ctx.Err(); err != nil {
		return Business{}, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, b := range s.businesses {
		if b.WorkspaceID == orgID && b.OwnerUserID == userID {
			return *b, nil
		}
	}
	return Business{}, errors.New("business profile not found")
}
