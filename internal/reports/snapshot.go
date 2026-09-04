package reports

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"kredit/internal/credit"
	"kredit/internal/disputes"
	"kredit/internal/ledger"
	"kredit/internal/payments"
	"kredit/internal/schedules"

	"github.com/jackc/pgx/v5"
)

// financialSnapshot reads all inputs under one repeatable-read transaction. A
// partially available projection must never be presented as an empty balance.
func (s *Store) financialSnapshot(ctx context.Context, org, buyer string) (*Store, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err = tx.Exec(ctx, `SELECT set_config('app.current_organization_id',$1,true),set_config('app.current_user_id',$2,true)`, org, buyer); err != nil {
		return nil, err
	}
	rows, err := tx.Query(ctx, `SELECT s.aggregate,to_jsonb(o),
 COALESCE((SELECT jsonb_agg(to_jsonb(p)||jsonb_build_object('recorded_by',p.recorded_by_reference) ORDER BY p.recognized_at,p.id) FROM app.payments p WHERE p.obligation_id=o.id),'[]'::jsonb),
 (SELECT to_jsonb(r) FROM app.repayment_schedules r WHERE r.obligation_id=o.id),
 COALESCE((SELECT jsonb_agg(to_jsonb(i) ORDER BY i.sequence) FROM app.schedule_items i JOIN app.repayment_schedules r ON r.id=i.schedule_id WHERE r.obligation_id=o.id),'[]'::jsonb),
 COALESCE((SELECT jsonb_agg(to_jsonb(d)) FROM app.disputes d WHERE d.obligation_id=o.id),'[]'::jsonb),
 COALESCE((SELECT SUM((a.metadata->>'amount_kobo')::bigint) FROM app.operation_actions a WHERE a.resource_id=o.id AND a.organization_id=o.supplier_organization_id AND a.action='fee_waiver'),0)
 FROM app.credit_aggregate_snapshots s JOIN app.obligations o ON o.credit_request_id::text=s.credit_request_id
 WHERE ($1='' OR s.supplier_organization_id=$1) AND ($2='' OR s.buyer_user_id=$2) ORDER BY o.activated_at,o.id`, org, buyer)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	views := []credit.View{}
	paymentMap := map[string][]payments.Payment{}
	scheduleMap := map[string]schedules.Schedule{}
	itemMap := map[string][]schedules.Item{}
	disputeMap := map[string][]disputes.Dispute{}
	waiverMap := map[string]ledger.Money{}
	for rows.Next() {
		var aggregate, obligation, paymentJSON, scheduleJSON, itemJSON, disputeJSON []byte
		var waived ledger.Money
		if err = rows.Scan(&aggregate, &obligation, &paymentJSON, &scheduleJSON, &itemJSON, &disputeJSON, &waived); err != nil {
			return nil, err
		}
		var view credit.View
		var o credit.Obligation
		var pp []payments.Payment
		var schedule schedules.Schedule
		var items []schedules.Item
		var dd []disputes.Dispute
		for _, part := range []struct {
			data   []byte
			target any
		}{{aggregate, &view}, {obligation, &o}, {paymentJSON, &pp}, {itemJSON, &items}, {disputeJSON, &dd}} {
			if err = json.Unmarshal(part.data, part.target); err != nil {
				return nil, err
			}
		}
		if len(scheduleJSON) == 0 || string(scheduleJSON) == "null" {
			return nil, errors.New("obligation schedule is missing")
		}
		if err = json.Unmarshal(scheduleJSON, &schedule); err != nil {
			return nil, err
		}
		if o.Currency != "NGN" || o.OutstandingKobo < 0 || o.OutstandingKobo > o.PrincipalKobo || len(items) == 0 {
			return nil, errors.New("invalid financial report input")
		}
		view.Obligation = &o
		views = append(views, view)
		paymentMap[o.ID] = pp
		scheduleMap[o.ID] = schedule
		itemMap[o.ID] = items
		disputeMap[o.ID] = dd
		waiverMap[o.ID] = waived
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	rows.Close()
	if err = tx.Commit(ctx); err != nil {
		return nil, err
	}
	now := s.source.Now()
	return NewStore(Source{Now: func() time.Time { return now }, SupplierViews: func(string) []credit.View { return views }, BuyerViews: func(string) []credit.View { return views }, Payments: func(id string) ([]payments.Payment, error) { return paymentMap[id], nil }, Schedule: func(id string) (schedules.Schedule, []schedules.Item, error) {
		return scheduleMap[id], itemMap[id], nil
	}, Disputes: func(id string) []disputes.Dispute { return disputeMap[id] }, FeeWaivers: func(string) map[string]ledger.Money { return waiverMap }}), nil
}
