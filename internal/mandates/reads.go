package mandates

import (
	"context"
	"encoding/json"
	"errors"
	"sort"
	"strings"
)

// BuyerReader lists persisted permissions even before a sale or limit uses them.
// Reading the account must not contact a provider or change its authorization.
type BuyerReader interface {
	ReadForBuyer(context.Context, string) ([]Mandate, error)
}

func (p *PostgresProvider) ReadForBuyer(ctx context.Context, userID string) ([]Mandate, error) {
	if p == nil || p.pool == nil {
		return nil, errors.New("mandate database is not configured")
	}
	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err = tx.Exec(ctx, `SELECT set_config('app.current_user_id',$1,true)`, userID); err != nil {
		return nil, err
	}
	rows, err := tx.Query(ctx, `SELECT m.id::text,m.provider,m.provider_mandate_id,m.buyer_subject_id::text,COALESCE(b.owner_user_id,p.user_id)::text,COALESCE(m.supplier_organization_id::text,''),m.amount_ceiling_kobo,m.state,m.created_at,COALESCE(m.starts_at,'0001-01-01'::timestamptz),COALESCE(m.ends_at,'0001-01-01'::timestamptz),m.metadata FROM app.payment_mandates m LEFT JOIN app.businesses b ON m.buyer_subject_type='business' AND b.id=m.buyer_subject_id LEFT JOIN app.persons p ON m.buyer_subject_type='person' AND p.id=m.buyer_subject_id WHERE COALESCE(b.owner_user_id,p.user_id)=$1::uuid ORDER BY m.created_at,m.id`, userID)
	if err != nil {
		return nil, err
	}
	result := []Mandate{}
	for rows.Next() {
		var m, stored Mandate
		var metadata []byte
		if err = rows.Scan(&m.ID, &m.Provider, &m.ProviderID, &m.BusinessID, &m.UserID, &m.SupplierOrganizationID, &m.AmountCeiling, &m.Status, &m.CreatedAt, &m.StartsAt, &m.EndsAt, &metadata); err != nil {
			rows.Close()
			return nil, err
		}
		if err = json.Unmarshal(metadata, &stored); err != nil {
			rows.Close()
			return nil, err
		}
		m.ProviderAdapter = stored.ProviderAdapter
		m.AuthorizationURL, m.Variable, m.MultiAccount, m.PartialRecovery, m.ActivatedAt = stored.AuthorizationURL, stored.Variable, stored.MultiAccount, stored.PartialRecovery, stored.ActivatedAt
		m.Status = Status(strings.ToUpper(string(m.Status)))
		result = append(result, m)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, err
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, err
	}
	return result, nil
}
func (p *MockProvider) ReadForBuyer(ctx context.Context, userID string) ([]Mandate, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	p.mu.RLock()
	defer p.mu.RUnlock()
	result := []Mandate{}
	for _, m := range p.items {
		if m.UserID == userID {
			result = append(result, m)
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result, nil
}
