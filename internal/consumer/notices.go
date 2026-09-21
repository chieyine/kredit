package consumer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"kredit/internal/notifications"
	"time"

	"github.com/jackc/pgx/v5"
)

func purchaseNoticeLabel(action string) (string, error) {
	switch action {
	case "accept":
		return "Purchase accepted", nil
	case "decline":
		return "Customer declined the purchase", nil
	case "claim":
		return "Payment reported; awaiting retailer confirmation", nil
	case "payment":
		return "Payment confirmed by the retailer", nil
	case "reject_claim":
		return "Payment report reviewed", nil
	case "reverse_payment":
		return "Payment record reversed", nil
	case "release":
		return "Delivery recorded; customer confirmation needed", nil
	case "received":
		return "Customer confirmed receipt", nil
	case "cancel":
		return "Purchase cancelled; check any refund due", nil
	case "request_return":
		return "Return or delivery issue opened", nil
	case "approve_return":
		return "Return approved; check the refund due", nil
	case "reject_return":
		return "Return request declined; review the reason or escalate", nil
	case "escalate":
		return "Return escalated to super admin", nil
	case "reduce_price":
		return "Purchase price reduced", nil
	case "refund":
		return "Refund recorded by the retailer", nil
	default:
		return "", errors.New("unsupported consumer notification action")
	}
}

func notice(ctx context.Context, tx pgx.Tx, s Sale, eventID, action string) error {
	label, err := purchaseNoticeLabel(action)
	if err != nil {
		return err
	}
	for _, buyer := range []bool{true, false} {
		if buyer && s.BuyerID == "" {
			continue
		}
		event := notifications.Event{ID: fmt.Sprintf("consumer:%s:%t", eventID, buyer), Type: "ConsumerPurchaseUpdated", Priority: notifications.PriorityCritical, Reference: s.Terms.Item, NextAction: label, Currency: "NGN"}
		if buyer {
			event.RecipientID = s.BuyerID
			event.SecurePath = "/personal/purchases/" + s.ID
		} else {
			event.OrganizationID = s.OrganizationID
			event.SecurePath = "/workspace/sales/consumers/" + s.ID + "?organization=" + s.OrganizationID
		}
		raw, e := json.Marshal(map[string]any{"notification": event})
		if e != nil {
			return e
		}
		if _, e = tx.Exec(ctx, `INSERT INTO app.outbox_events(aggregate_type,aggregate_id,event_type,payload,idempotency_key) VALUES('consumer_sale',$1,'notification.requested',$2::jsonb,$3) ON CONFLICT(idempotency_key) DO NOTHING`, s.ID, raw, event.ID); e != nil {
			return e
		}
	}
	return nil
}

// EnqueueReminders uses the same durable notification queue as other sales.
func (s *Store) EnqueueReminders(ctx context.Context) error {
	if s == nil || s.Pool == nil {
		return errors.New("consumer notification storage is unavailable")
	}
	rows, err := s.Pool.Query(ctx, `SELECT sale_id::text,organization_id::text,buyer_user_id::text FROM app.consumer_reminder_work()`)
	if err != nil {
		return err
	}
	type work struct{ id, org, buyer string }
	worklist := []work{}
	for rows.Next() {
		var w work
		if err = rows.Scan(&w.id, &w.org, &w.buyer); err != nil {
			rows.Close()
			return err
		}
		worklist = append(worklist, w)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	for _, w := range worklist {
		if err = s.enqueueDue(ctx, w.id, w.org, w.buyer); err != nil {
			return err
		}
	}
	return nil
}
func (s *Store) enqueueDue(ctx context.Context, id, org, buyer string) error {
	tx, e := s.begin(ctx, buyer, org)
	if e != nil {
		return e
	}
	// Preserve the primary operation or commit error during best-effort cleanup.
	defer func() { _ = tx.Rollback(ctx) }()
	v, e := scan(tx.QueryRow(ctx, saleSelect+` WHERE id=$1::uuid`, id))
	if e != nil {
		return e
	}
	if e = history(ctx, tx, &v); e != nil {
		return e
	}
	today := time.Now().In(time.FixedZone("Africa/Lagos", 3600)).Format("2006-01-02")
	due := v.DueAmount(today)
	if due <= 0 {
		return nil
	}
	ev := notifications.Event{ID: "consumer-due:" + id + ":" + today, Type: "ConsumerPaymentDue", RecipientID: buyer, Priority: notifications.PriorityRoutine, AmountKobo: due, Currency: "NGN", Reference: v.Terms.Item, NextAction: "Pay the retailer or report a payment already made.", SecurePath: "/personal/purchases/" + id}
	raw, e := json.Marshal(map[string]any{"notification": ev})
	if e != nil {
		return e
	}
	_, e = tx.Exec(ctx, `INSERT INTO app.outbox_events(aggregate_type,aggregate_id,event_type,payload,idempotency_key) VALUES('consumer_sale',$1,'notification.requested',$2::jsonb,$3) ON CONFLICT(idempotency_key) DO NOTHING`, id, raw, ev.ID)
	if e != nil {
		return e
	}
	return tx.Commit(ctx)
}
func (s *Sale) DueAmount(today string) int64 {
	if s.State != "active" || s.CaseState == "requested" || s.CaseState == "escalated" {
		return 0
	}
	// An unresolved transfer claim must be reviewed before another payment prompt.
	for _, ev := range s.Events {
		if ev.Action != "claim" {
			continue
		}
		done := false
		for _, d := range s.Events {
			if d.RelatedID == ev.ID && (d.Action == "payment" || d.Action == "reject_claim") {
				done = true
			}
		}
		if !done {
			return 0
		}
	}
	var due int64
	for _, d := range s.Schedule {
		if d.Date <= today {
			due += d.Amount - d.Paid
		}
	}
	return min(max(due, 0), s.Outstanding)
}
