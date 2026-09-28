package consumer

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
)

// Summary contains only what a seller needs for balances and next actions.
// Bank details, agreement text and event evidence stay on the purchase record.
type Summary struct {
	ID             string       `json:"id"`
	OrganizationID string       `json:"organization_id"`
	Target         string       `json:"target"`
	CustomerName   string       `json:"customer_name"`
	State          string       `json:"state"`
	CaseState      string       `json:"case_state"`
	Terms          SummaryTerms `json:"terms"`
	Schedule       []Due        `json:"schedule_progress"`
	Paid           int64        `json:"paid_kobo"`
	Refunded       int64        `json:"refunded_kobo"`
	Outstanding    int64        `json:"outstanding_kobo"`
	RefundDue      int64        `json:"refund_due_kobo"`
	Eligible       bool         `json:"release_eligible"`
	PendingClaims  int64        `json:"pending_claim_count"`
}
type SummaryTerms struct {
	Item  string `json:"item"`
	Total int64  `json:"total_kobo"`
}

// Summaries reads one scoped database snapshot. Amounts are aggregated by SQL;
// schedule allocation and release eligibility use the same rules as detail.
func (s *Store) Summaries(ctx context.Context, actor, org string) ([]Summary, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	tx, err := s.beginOptions(ctx, actor, org, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var authorized bool
	err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM app.memberships m JOIN app.users u ON u.id=m.user_id JOIN app.organizations o ON o.id=m.organization_id WHERE m.organization_id=$1::uuid AND m.user_id=$2::uuid AND m.role='owner' AND m.status='active' AND u.status='active' AND o.status NOT IN ('suspended','closed'))`, org, actor).Scan(&authorized)
	if err != nil {
		return nil, err
	}
	if !authorized {
		return nil, ErrUnavailable
	}
	rows, err := tx.Query(ctx, `WITH totals AS (
 SELECT e.sale_id,
 COALESCE(sum(CASE e.action WHEN 'payment' THEN e.amount_kobo WHEN 'reverse_payment' THEN -e.amount_kobo ELSE 0 END),0)::bigint paid,
 COALESCE(sum(e.amount_kobo) FILTER(WHERE e.action='refund'),0)::bigint refunded,
 COALESCE(sum(e.amount_kobo) FILTER(WHERE e.action='reduce_price'),0)::bigint reduction,
 count(*) FILTER(WHERE e.action='claim' AND NOT EXISTS(SELECT 1 FROM app.consumer_events d WHERE d.sale_id=e.sale_id AND d.related_id=e.id AND d.action IN ('payment','reject_claim'))) pending
 FROM app.consumer_events e JOIN app.consumer_sales s ON s.id=e.sale_id WHERE s.organization_id=$1::uuid GROUP BY e.sale_id
 ) SELECT s.id::text,s.target_value,s.customer_name,s.state,s.case_state,s.terms,s.released_at,
 COALESCE(t.paid,0),COALESCE(t.refunded,0),COALESCE(t.reduction,0),COALESCE(t.pending,0)
 FROM app.consumer_sales s LEFT JOIN totals t ON t.sale_id=s.id WHERE s.organization_id=$1::uuid ORDER BY s.id DESC`, org)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []Summary{}
	for rows.Next() {
		var sale Sale
		var raw []byte
		var claims int64
		if err = rows.Scan(&sale.ID, &sale.Target, &sale.CustomerName, &sale.State, &sale.CaseState, &raw, &sale.ReleasedAt, &sale.Paid, &sale.Refunded, &sale.Reduction, &claims); err != nil {
			return nil, err
		}
		if err = json.Unmarshal(raw, &sale.Terms); err != nil {
			return nil, err
		}
		if sale.Paid < 0 || sale.Refunded < 0 || sale.Reduction < 0 || sale.Reduction > sale.Terms.Total {
			return nil, errors.New("invalid personal sale balance")
		}
		sale.calculateBalances()
		result = append(result, Summary{ID: sale.ID, OrganizationID: org, Target: sale.Target, CustomerName: sale.CustomerName, State: sale.State, CaseState: sale.CaseState, Terms: SummaryTerms{sale.Terms.Item, sale.Terms.Total}, Schedule: sale.Schedule, Paid: sale.Paid, Refunded: sale.Refunded, Outstanding: sale.Outstanding, RefundDue: sale.RefundDue, Eligible: sale.Eligible, PendingClaims: claims})
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

// Fetch history once per page rather than once per purchase. The caller holds
// a repeatable-read transaction so list rows and their events agree.
func histories(ctx context.Context, tx pgx.Tx, sales []Sale) error {
	if len(sales) == 0 {
		return nil
	}
	ids := make([]string, 0, len(sales))
	byID := make(map[string]int, len(sales))
	for i := range sales {
		ids = append(ids, sales[i].ID)
		byID[sales[i].ID] = i
		sales[i].Events = []Event{}
	}
	rows, err := tx.Query(ctx, `SELECT sale_id::text,id::text,action,amount_kobo,reference,COALESCE(related_id::text,''),note,occurred_at,created_at FROM app.consumer_events WHERE sale_id=ANY($1::uuid[]) ORDER BY sale_id,created_at,id`, ids)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		var event Event
		if err = rows.Scan(&id, &event.ID, &event.Action, &event.Amount, &event.Reference, &event.RelatedID, &event.Note, &event.At, &event.RecordedAt); err != nil {
			return err
		}
		i, ok := byID[id]
		if !ok {
			return errors.New("purchase event scope mismatch")
		}
		sales[i].Events = append(sales[i].Events, event)
	}
	if err = rows.Err(); err != nil {
		return err
	}
	for i := range sales {
		sales[i].Calculate()
	}
	return nil
}
