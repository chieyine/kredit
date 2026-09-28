package reports

import (
	"context"
	"errors"
	"time"

	"kredit/internal/db"

	"github.com/jackc/pgx/v5"
)

type CustomerBalance struct {
	ID              string `json:"id"`
	BuyerUserID     string `json:"buyer_user_id"`
	BuyerBusinessID string `json:"buyer_business_id"`
	LegalName       string `json:"legal_name"`
	TradingName     string `json:"trading_name"`
	Industry        string `json:"industry"`
	State           string `json:"state"`
	RequestCount    int64  `json:"request_count"`
	Outstanding     string `json:"outstanding_kobo"`
}

// CustomerBalances reads identities and balances in one scoped snapshot. Full
// agreement snapshots and event histories never leave this projection; there
// are no per-customer queries.
func (s *Store) CustomerBalances(ctx context.Context, org string) ([]CustomerBalance, error) {
	tenant, ok := db.TenantFromContext(ctx)
	if !ok || tenant.OrganizationID != org || tenant.UserID == "" || s.pool == nil {
		return nil, errors.New("authorized customer scope required")
	}
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err = tx.Exec(ctx, `SELECT set_config('app.current_user_id',$1,true),set_config('app.current_organization_id',$2,true)`, tenant.UserID, org); err != nil {
		return nil, err
	}
	rows, err := tx.Query(ctx, `WITH names AS (
 SELECT buyer_user_id,buyer_business_id,legal_name,COALESCE(trading_name,'') trading_name,industry,status state,0 priority FROM app.supplier_customers($1::uuid)
 UNION ALL
 SELECT c.buyer_user_id,c.buyer_business_id,COALESCE(s.aggregate->'request'->>'buyer_legal_name','Business customer'),COALESCE(s.aggregate->'request'->>'buyer_trading_name',''),'','ACTIVE',1 FROM app.credit_requests c LEFT JOIN app.credit_aggregate_snapshots s ON s.credit_request_id=c.id::text WHERE c.supplier_organization_id=$1::uuid
 ), identities AS (
 SELECT DISTINCT ON (buyer_user_id,buyer_business_id) * FROM names ORDER BY buyer_user_id,buyer_business_id,priority,legal_name
 ), balances AS (
 SELECT c.buyer_user_id,c.buyer_business_id,count(*) request_count,COALESCE(sum(o.outstanding_kobo),0)::text outstanding
 FROM app.credit_requests c LEFT JOIN app.obligations o ON o.credit_request_id=c.id AND o.supplier_organization_id=c.supplier_organization_id
 WHERE c.supplier_organization_id=$1::uuid GROUP BY c.buyer_user_id,c.buyer_business_id
 ) SELECT n.buyer_user_id::text,n.buyer_business_id::text,n.legal_name,n.trading_name,n.industry,n.state,COALESCE(b.request_count,0),COALESCE(b.outstanding,'0')
 FROM identities n LEFT JOIN balances b ON b.buyer_user_id=n.buyer_user_id AND b.buyer_business_id=n.buyer_business_id
 ORDER BY n.legal_name,n.buyer_business_id,n.buyer_user_id`, org)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []CustomerBalance{}
	for rows.Next() {
		var row CustomerBalance
		if err = rows.Scan(&row.BuyerUserID, &row.BuyerBusinessID, &row.LegalName, &row.TradingName, &row.Industry, &row.State, &row.RequestCount, &row.Outstanding); err != nil {
			return nil, err
		}
		row.ID = row.BuyerUserID
		result = append(result, row)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}
