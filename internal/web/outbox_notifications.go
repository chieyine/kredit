package web

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"kredit/internal/notifications"
	"kredit/internal/outbox"
)

// QueueOutboxNotification acknowledges a request only after the notification
// store has durably deduplicated it. Actual delivery belongs to its own worker.
func (r *Runtime) QueueOutboxNotification(ctx context.Context, event outbox.Event) error {
	if r.Database == nil || r.Notifications == nil {
		return errors.New("notification persistence unavailable")
	}
	var payload struct {
		Notification *notifications.Event `json:"notification"`
		Event        string               `json:"event"`
		Amount       int64                `json:"amount_kobo"`
		ScheduleItem string               `json:"schedule_item_id"`
		EndsAt       time.Time            `json:"ends_at"`
		RetryAt      time.Time            `json:"next_retry_at"`
	}
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return err
	}
	if payload.Notification != nil {
		notice := *payload.Notification
		notice.DeferDelivery = true
		_, err := r.EmitNotification(ctx, notice)
		return err
	}
	templates := map[string]string{"SCHEDULE_AMENDMENT": "ScheduleAmendment", "PRE_DEBIT_NOTICE": "PriorDebitNotice", "MANDATE_REVOKED": "MandateRevoked", "RETRY_SCHEDULED": "CollectionRetryScheduled", "OBLIGATION_ACCEPTED": "ObligationAccepted", "GOODS_RELEASED": "GoodsReleased", "OBLIGATION_FULLY_REPAID": "ObligationRepaid", "UPCOMING_DUE": "PaymentDueSoon", "PAYMENT_DUE": "PaymentDueSoon", "MANDATE_EXPIRING": "MandateExpiring", "COLLECTION_INITIATED": "CollectionSubmitted", "COLLECTION_UNCERTAIN": "CollectionUncertain", "COLLECTION_SUCCESSFUL": "PaymentRecorded", "PARTIAL_COLLECTION": "PaymentRecorded", "COLLECTION_FAILED": "CollectionFailed", "COLLECTION_CANCELLED": "CollectionCancelled"}
	template := templates[payload.Event]
	if template == "" {
		return errors.New("unsupported notification request")
	}
	var recipient string
	date := payload.EndsAt
	if !payload.RetryAt.IsZero() {
		date = payload.RetryAt
	}
	switch event.AggregateType {
	case "payment_mandate":
		if err := r.Database.Raw().QueryRow(ctx, `SELECT metadata->>'user_id' FROM app.payment_mandates WHERE id=$1::uuid`, event.AggregateID).Scan(&recipient); err != nil {
			return err
		}
	case "credit_request":
		if err := r.Database.Raw().QueryRow(ctx, `SELECT buyer_user_id FROM app.credit_aggregate_snapshots WHERE credit_request_id=$1`, event.AggregateID).Scan(&recipient); err != nil {
			return err
		}
	case "payment":
		if err := r.Database.Raw().QueryRow(ctx, `SELECT buyer_user_id::text FROM app.payments WHERE id=$1::uuid`, event.AggregateID).Scan(&recipient); err != nil {
			return err
		}
	default:
		if err := r.Database.Raw().QueryRow(ctx, `SELECT s.buyer_user_id FROM app.credit_aggregate_snapshots s JOIN app.obligations o ON o.credit_request_id::text=s.credit_request_id WHERE o.id=$1::uuid`, event.AggregateID).Scan(&recipient); err != nil {
			return err
		}
		if payload.ScheduleItem != "" {
			if err := r.Database.Raw().QueryRow(ctx, `SELECT CASE WHEN $3='PRE_DEBIT_NOTICE' THEN i.collection_at ELSE i.due_at END FROM app.schedule_items i JOIN app.repayment_schedules s ON s.id=i.schedule_id WHERE i.id=$1::uuid AND s.obligation_id=$2::uuid`, payload.ScheduleItem, event.AggregateID, payload.Event).Scan(&date); err != nil {
				return err
			}
		}
	}
	tx, err := r.Database.Raw().Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, `SELECT set_config('app.current_user_id',$1,true)`, recipient); err != nil {
		return err
	}
	var email, phone string
	if err := tx.QueryRow(ctx, `SELECT COALESCE(normalized_email,''),COALESCE(normalized_phone,'') FROM app.users WHERE id=$1::uuid`, recipient).Scan(&email, &phone); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}
	priority := notifications.PriorityCritical
	if payload.Event == "UPCOMING_DUE" {
		priority = notifications.PriorityRoutine
	}
	securePath := ""
	if payload.Event == "SCHEDULE_AMENDMENT" {
		securePath = "/buyer/amendments"
	}
	nextAction := "Review your payment details and proposed date changes in Kredit"
	if payload.Event == "GOODS_RELEASED" {
		// This notice is the evidence that makes deemed acceptance permissible,
		// so it must say what silence will be taken to mean and point at the
		// screen where the buyer can object.
		securePath = "/buyer/requests"
		nextAction = "Confirm receipt or report a problem before the date shown"
	}
	_, err = r.Notifications.Emit(ctx, notifications.Event{ID: "outbox:" + event.ID, Type: template, RecipientID: recipient, Phone: phone, Email: email, Priority: priority, AmountKobo: payload.Amount, Currency: "NGN", Date: date, Reference: event.AggregateID, NextAction: nextAction, SecurePath: securePath, DeferDelivery: true})
	return err
}
