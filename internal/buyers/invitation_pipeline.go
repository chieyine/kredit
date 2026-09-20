package buyers

import (
	"context"
	"errors"
	"sort"
	"time"
)

type InvitationProgress struct {
	ID                     string    `json:"id"`
	Name                   string    `json:"name"`
	SourceReference        string    `json:"source_reference"`
	State                  string    `json:"state"`
	ExpiresAt              time.Time `json:"expires_at"`
	CreatedAt              time.Time `json:"created_at"`
	BusinessID             string    `json:"business_id"`
	Trades                 int64     `json:"trades"`
	ZeroBalanceObligations int64     `json:"zero_balance_obligations"`
}

func (s *PostgresStore) InvitationPipeline(ctx context.Context, userID, organizationID string) ([]InvitationProgress, error) {
	if s == nil || s.pool == nil {
		return nil, errors.New("invitation database unavailable")
	}
	tx, err := s.beginTxContext(ctx, userID, organizationID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	rows, err := tx.Query(ctx, `SELECT i.id::text,i.proposed_legal_name,COALESCE(i.source_reference,''),CASE WHEN i.status='pending' AND i.expires_at<=now() THEN 'expired' ELSE i.status END,i.expires_at,i.created_at,COALESCE(i.accepted_business_id::text,''),
 (SELECT count(*) FROM app.credit_requests c WHERE c.supplier_organization_id=i.organization_id AND c.buyer_business_id=i.accepted_business_id),
 (SELECT count(*) FROM app.obligations o WHERE o.supplier_organization_id=i.organization_id AND o.buyer_business_id=i.accepted_business_id AND o.outstanding_kobo=0 AND o.principal_kobo>0)
 FROM app.buyer_invitations i WHERE i.organization_id=$1::uuid ORDER BY i.created_at DESC,i.id DESC LIMIT 500`, organizationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []InvitationProgress{}
	for rows.Next() {
		var item InvitationProgress
		if err := rows.Scan(&item.ID, &item.Name, &item.SourceReference, &item.State, &item.ExpiresAt, &item.CreatedAt, &item.BusinessID, &item.Trades, &item.ZeroBalanceObligations); err != nil {
			return nil, err
		}
		result = append(result, item)
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

func (s *Store) InvitationPipeline(ctx context.Context, userID, organizationID string) ([]InvitationProgress, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := []InvitationProgress{}
	for _, record := range s.invitations {
		i := record.Invitation
		if i.OrganizationID != organizationID {
			continue
		}
		state := i.Status
		if state == "pending" && !i.ExpiresAt.After(s.now()) {
			state = "expired"
		}
		result = append(result, InvitationProgress{ID: i.ID, Name: i.ProposedLegalName, State: state, CreatedAt: i.CreatedAt, ExpiresAt: i.ExpiresAt})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].CreatedAt.After(result[j].CreatedAt) })
	if len(result) > 500 {
		result = result[:500]
	}
	return result, nil
}
