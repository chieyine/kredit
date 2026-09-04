package web

import (
	"net/http"

	"kredit/internal/audit"
)

func (s *Server) acknowledgeCollectionNotice(w http.ResponseWriter, r *http.Request) {
	_, user, ok := s.requireAuth(w, r)
	if !ok || !s.requireCSRF(w, r) {
		return
	}
	if s.runtime.Database == nil {
		writeProblem(w, http.StatusServiceUnavailable, "notice_acknowledgement_unavailable", "notice acknowledgement is unavailable")
		return
	}
	itemID, err := pathID(r, "itemID")
	if err != nil {
		writeProblem(w, http.StatusBadRequest, "invalid_path", err.Error())
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
	var notificationID, receiptChannel, receiptEventID string
	if err = tx.QueryRow(r.Context(), `SELECT n.id::text,receipt.channel,receipt.event_id
		FROM app.schedule_items i
		JOIN app.repayment_schedules s ON s.id=i.schedule_id
		JOIN app.obligations o ON o.id=s.obligation_id
		JOIN app.credit_requests c ON c.id=o.credit_request_id
		JOIN app.outbox_events e ON e.idempotency_key=app.collection_notice_key(i)
		JOIN app.notifications n ON n.event_reference='outbox:'||e.id::text AND n.state IN ('delivered','read')
		JOIN app.notification_delivery_receipts receipt ON receipt.notification_id=n.id
		WHERE i.id=$1::uuid AND c.buyer_user_id=$2::uuid AND i.state NOT IN('PAID','CANCELLED')
		ORDER BY receipt.received_at DESC LIMIT 1`, itemID, user.ID).Scan(&notificationID, &receiptChannel, &receiptEventID); err != nil {
		writeProblem(w, http.StatusConflict, "collection_notice_not_delivered", "a confirmed collection notice must be delivered before it can be acknowledged")
		return
	}
	if _, err = tx.Exec(r.Context(), `INSERT INTO app.collection_notice_acknowledgements(schedule_item_id,buyer_user_id,notification_id,receipt_channel,receipt_event_id) VALUES($1::uuid,$2::uuid,$3::uuid,$4,$5) ON CONFLICT(schedule_item_id) DO NOTHING`, itemID, user.ID, notificationID, receiptChannel, receiptEventID); err != nil || tx.Commit(r.Context()) != nil {
		writeProblem(w, http.StatusServiceUnavailable, "notice_acknowledgement_unavailable", "notice acknowledgement could not be saved")
		return
	}
	s.runtime.Audit.Append(audit.Event{ActorUserID: user.ID, Action: "collection.notice.acknowledged", ResourceType: "schedule_item", ResourceID: itemID, Outcome: "success", RequestID: requestIDFromContext(r.Context()), Metadata: map[string]string{"notification_id": notificationID, "receipt_channel": receiptChannel, "receipt_event_id": receiptEventID}})
	w.WriteHeader(http.StatusNoContent)
}
