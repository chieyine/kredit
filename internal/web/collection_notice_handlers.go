package web

import (
	"context"
	"errors"
	"net/http"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (s *Server) acknowledgeCollectionNotice(w http.ResponseWriter, r *http.Request) {
	_, user, ok := s.requireAuth(w, r)
	if !ok || !s.requireCSRF(w, r) {
		return
	}
	if s.runtime.Database == nil || s.runtime.Database.Raw() == nil {
		writeProblem(w, http.StatusServiceUnavailable, "notice_acknowledgement_unavailable", "notice acknowledgement is unavailable")
		return
	}
	itemID, err := pathID(r, "itemID")
	if err != nil {
		writeProblem(w, http.StatusBadRequest, "invalid_path", err.Error())
		return
	}
	var input struct {
		NotificationID string `json:"notification_id"`
	}
	if err := decodeJSON(w, r, &input); err != nil {
		writeProblem(w, http.StatusBadRequest, "invalid_request", "Choose the notice you have read.")
		return
	}
	if _, err := uuid.Parse(input.NotificationID); err != nil {
		writeProblem(w, http.StatusBadRequest, "invalid_request", "Choose the notice you have read.")
		return
	}

	tx, err := s.runtime.Database.Raw().Begin(r.Context())
	if err != nil {
		writeProblem(w, http.StatusServiceUnavailable, "notice_acknowledgement_unavailable", "notice acknowledgement could not be saved")
		return
	}
	defer func() { _ = tx.Rollback(r.Context()) }()
	if _, err = tx.Exec(r.Context(), `SELECT set_config('app.current_user_id',$1,true)`, user.ID); err != nil {
		writeProblem(w, http.StatusServiceUnavailable, "notice_acknowledgement_unavailable", "notice acknowledgement could not be saved")
		return
	}
	// Establish the supplier context only after proving this buyer owns the item.
	// Lock the obligation before the item, matching payment and amendment writes.
	var obligationID, organizationID string
	err = tx.QueryRow(r.Context(), `SELECT o.id::text,c.supplier_organization_id::text
 FROM app.schedule_items i JOIN app.repayment_schedules s ON s.id=i.schedule_id
 JOIN app.obligations o ON o.id=s.obligation_id JOIN app.credit_requests c ON c.id=o.credit_request_id
 WHERE i.id=$1::uuid AND c.buyer_user_id=$2::uuid`, itemID, user.ID).Scan(&obligationID, &organizationID)
	if errors.Is(err, pgx.ErrNoRows) {
		writeProblem(w, http.StatusNotFound, "schedule_item_not_found", "We could not find that payment day.")
		return
	}
	if err == nil {
		_, err = tx.Exec(r.Context(), `SELECT set_config('app.current_organization_id',$1,true)`, organizationID)
	}
	if err == nil {
		err = tx.QueryRow(r.Context(), `SELECT id::text FROM app.obligations WHERE id=$1::uuid FOR UPDATE`, obligationID).Scan(&obligationID)
	}
	if err != nil {
		writeProblem(w, http.StatusServiceUnavailable, "notice_acknowledgement_unavailable", "notice acknowledgement could not be saved")
		return
	}

	var notificationID, receiptChannel, receiptEventID string
	if err = tx.QueryRow(r.Context(), `SELECT n.id::text,receipt.channel,receipt.event_id
		FROM app.schedule_items i
		JOIN app.repayment_schedules s ON s.id=i.schedule_id
		JOIN app.obligations o ON o.id=s.obligation_id
		JOIN app.credit_requests c ON c.id=o.credit_request_id
		JOIN app.outbox_events e ON e.idempotency_key=app.collection_notice_key(i)
		JOIN app.notifications n ON n.event_reference='outbox:'||e.id::text AND n.state IN ('delivered','read')
		JOIN app.notification_delivery_receipts receipt ON receipt.notification_id=n.id
		WHERE i.id=$1::uuid AND c.buyer_user_id=$2::uuid AND i.state NOT IN('PAID','CANCELLED') AND i.principal_due_kobo>i.allocated_kobo
 AND s.status='ACTIVE' AND n.recipient_id=$2::uuid AND n.id=$3::uuid
		ORDER BY receipt.received_at DESC LIMIT 1`, itemID, user.ID, input.NotificationID).Scan(&notificationID, &receiptChannel, &receiptEventID); err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			writeProblem(w, http.StatusServiceUnavailable, "notice_acknowledgement_unavailable", "notice acknowledgement could not be saved")
			return
		}
		writeProblem(w, http.StatusConflict, "collection_notice_not_delivered", "a confirmed collection notice must be delivered before it can be acknowledged")
		return
	}
	result, err := tx.Exec(r.Context(), `INSERT INTO app.collection_notice_acknowledgements(schedule_item_id,buyer_user_id,notification_id,receipt_channel,receipt_event_id) VALUES($1::uuid,$2::uuid,$3::uuid,$4,$5) ON CONFLICT(schedule_item_id,notification_id) DO NOTHING`, itemID, user.ID, notificationID, receiptChannel, receiptEventID)
	if err == nil && result.RowsAffected() > 0 {
		_, err = tx.Exec(r.Context(), `INSERT INTO app.audit_events(actor_user_id,organization_id,action,resource_type,resource_id,outcome,request_id,metadata)
   VALUES($1::uuid,$2::uuid,'collection.notice.acknowledged','schedule_item',$3,'success',$4,jsonb_build_object('notification_id',$5::text,'receipt_channel',$6::text,'receipt_event_id',$7::text))`, user.ID, organizationID, itemID, requestIDFromContext(r.Context()), notificationID, receiptChannel, receiptEventID)
	}
	if err != nil || tx.Commit(r.Context()) != nil {
		writeProblem(w, http.StatusServiceUnavailable, "notice_acknowledgement_unavailable", "notice acknowledgement could not be saved")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// collectionNotices exposes only delivered notices for the buyer's current
// amount/date. The acknowledgement command must name this exact notification.
type collectionNotice struct {
	ScheduleItemID string `json:"schedule_item_id"`
	NotificationID string `json:"notification_id"`
	Acknowledged   bool   `json:"acknowledged"`
}

func (s *Server) collectionNotices(ctx context.Context, buyerID, obligationID string) ([]collectionNotice, error) {
	notices := []collectionNotice{}
	if s.runtime.Database == nil {
		return notices, nil
	}
	err := s.runtime.Database.WithTenantTx(ctx, buyerID, "", func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `SELECT DISTINCT ON(i.id) i.id::text,n.id::text,
   EXISTS(SELECT 1 FROM app.collection_notice_acknowledgements ack WHERE ack.schedule_item_id=i.id AND ack.notification_id=n.id AND ack.buyer_user_id=$2::uuid)
  FROM app.schedule_items i JOIN app.repayment_schedules s ON s.id=i.schedule_id
  JOIN app.obligations o ON o.id=s.obligation_id JOIN app.credit_requests c ON c.id=o.credit_request_id
  JOIN app.outbox_events e ON e.idempotency_key=app.collection_notice_key(i)
  JOIN app.notifications n ON n.event_reference='outbox:'||e.id::text AND n.state IN('delivered','read') AND n.recipient_id=$2::uuid
  JOIN app.notification_delivery_receipts receipt ON receipt.notification_id=n.id
  WHERE o.id=$1::uuid AND c.buyer_user_id=$2::uuid AND s.status='ACTIVE'
   AND i.state NOT IN('PAID','CANCELLED') AND i.principal_due_kobo>i.allocated_kobo
  ORDER BY i.id,receipt.received_at DESC,n.id`, obligationID, buyerID)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var notice collectionNotice
			if err := rows.Scan(&notice.ScheduleItemID, &notice.NotificationID, &notice.Acknowledged); err != nil {
				return err
			}
			notices = append(notices, notice)
		}
		return rows.Err()
	})
	return notices, err
}
