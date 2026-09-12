package web

import (
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"

	"kredit/internal/access"
)

func (s *Server) listMessageSubmissions(w http.ResponseWriter, r *http.Request) {
	_, user, _, ok := s.requirePlatformAccess(w, r, access.PermissionProviderOperations)
	if !ok {
		return
	}
	if s.runtime.Database == nil {
		writeProblem(w, 503, "recovery_unavailable", "Message recovery requires the database.")
		return
	}
	rows, err := s.runtime.Database.Raw().Query(r.Context(), `SELECT event_key,channel,state,COALESCE(notification_id::text,''),COALESCE(provider_reference,''),created_at FROM app.message_submissions WHERE state='STARTED' AND created_at<now()-interval '1 minute' ORDER BY created_at LIMIT 100`)
	if err != nil {
		writeProblem(w, 503, "recovery_unavailable", "Message submissions could not be read.")
		return
	}
	items := []map[string]any{}
	for rows.Next() {
		var key, channel, state, id, reference string
		var at time.Time
		if err = rows.Scan(&key, &channel, &state, &id, &reference, &at); err != nil {
			rows.Close()
			writeProblem(w, 503, "recovery_unavailable", "Message submissions could not be read.")
			return
		}
		items = append(items, map[string]any{"id": key, "channel": channel, "state": state, "notification_id": id, "provider_reference": reference, "created_at": at})
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		writeProblem(w, 503, "recovery_unavailable", "Message submissions could not be read.")
		return
	}
	s.auditPlatformRead(r, user.ID, "messages.recovery.viewed", "message_submissions", "")
	writeJSON(w, 200, map[string]any{"submissions": items})
}
func (s *Server) resolveMessageSubmission(w http.ResponseWriter, r *http.Request) {
	session, user, _, ok := s.requirePlatformAccess(w, r, access.PermissionProviderOperations)
	if !ok || !s.requireFreshMFA(w, session) || !s.requireCSRF(w, r) {
		return
	}
	key := r.PathValue("submissionID")
	decoded, err := hex.DecodeString(key)
	if err != nil || len(decoded) != 32 {
		writeProblem(w, 400, "invalid_submission", "Invalid submission reference.")
		return
	}
	var in struct {
		Action    string `json:"action"`
		Reference string `json:"provider_reference"`
		Reason    string `json:"reason"`
	}
	if !decodeJSONRequest(w, r, &in) {
		return
	}
	in.Reference = strings.TrimSpace(in.Reference)
	in.Reason = strings.TrimSpace(in.Reason)
	if len(in.Reason) < 8 || len(in.Reason) > 2000 || (in.Action != "record_reference" && in.Action != "close_without_resend") || (in.Action == "record_reference" && (in.Reference == "" || len(in.Reference) > 512)) {
		writeProblem(w, 400, "invalid_resolution", "Choose a resolution and explain the provider evidence reviewed.")
		return
	}
	if s.runtime.Database == nil {
		writeProblem(w, 503, "recovery_unavailable", "Message recovery requires the database.")
		return
	}
	tx, err := s.runtime.Database.Raw().Begin(r.Context())
	if err != nil {
		writeProblem(w, 503, "recovery_unavailable", "Recovery could not begin.")
		return
	}
	defer func() { _ = tx.Rollback(r.Context()) }()
	if _, err = tx.Exec(r.Context(), `SELECT set_config('app.current_user_id',$1,true)`, user.ID); err != nil {
		writeProblem(w, 503, "recovery_unavailable", "Recovery could not begin.")
		return
	}
	var state, id string
	var created time.Time
	err = tx.QueryRow(r.Context(), `SELECT state,COALESCE(notification_id::text,''),created_at FROM app.message_submissions WHERE event_key=$1 FOR UPDATE`, key).Scan(&state, &id, &created)
	if err != nil || state != "STARTED" || time.Since(created) < time.Minute {
		writeProblem(w, 409, "recovery_conflict", "This send is still active or has already been resolved. Refresh its status.")
		return
	}
	next := "REJECTED"
	reference := ""
	if in.Action == "record_reference" {
		next = "ACCEPTED"
		reference = in.Reference
	}
	_, err = tx.Exec(r.Context(), `UPDATE app.message_submissions SET state=$2,provider_reference=NULLIF($3,''),updated_at=now() WHERE event_key=$1 AND state='STARTED'`, key, next, reference)
	if err == nil && id != "" {
		var affected pgconn.CommandTag
		if in.Action == "record_reference" {
			affected, err = tx.Exec(r.Context(), `UPDATE app.notifications SET state='sent',provider_message_id=$2,sent_at=COALESCE(sent_at,send_started_at,now()),failure_reason='Provider acceptance reference recorded by an operator; delivery awaits confirmation',lease_expires_at=NULL,next_attempt_at=NULL,updated_at=now() WHERE id=$1::uuid AND state IN ('sending','failed') AND (provider_message_id IS NULL OR provider_message_id='')`, id, reference)
		} else {
			affected, err = tx.Exec(r.Context(), `UPDATE app.notifications SET state='failed',failure_reason='Closed by an operator without resending; original delivery remains unconfirmed',failed_at=now(),delivery_attempts=GREATEST(delivery_attempts,8),next_attempt_at=NULL,lease_expires_at=NULL,updated_at=now() WHERE id=$1::uuid AND state IN ('sending','failed')`, id)
		}
		if err == nil && affected.RowsAffected() != 1 {
			writeProblem(w, 409, "recovery_conflict", "The message changed during review. Refresh its status before continuing.")
			return
		}
	}
	metadata, _ := json.Marshal(map[string]string{"action": in.Action, "reason": in.Reason, "provider_reference": reference, "notification_id": id})
	if err == nil {
		_, err = tx.Exec(r.Context(), `INSERT INTO app.audit_events(actor_user_id,action,resource_type,resource_id,outcome,severity,metadata) VALUES($1::uuid,'message_submission.resolved','message_submission',$2,'success','high',$3::jsonb)`, user.ID, key, metadata)
	}
	if err == nil {
		err = tx.Commit(r.Context())
	}
	if err != nil {
		writeProblem(w, 503, "recovery_unavailable", "Recovery was not saved. Refresh before trying again.")
		return
	}
	writeJSON(w, 200, map[string]string{"state": next, "delivery": "not_confirmed"})
}
