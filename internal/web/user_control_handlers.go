package web

import (
	"net/http"
	"strings"

	"kredit/internal/access"
	"kredit/internal/audit"
	"kredit/internal/auth"
	"kredit/internal/notifications"
)

func (s *Server) getNotificationPreferences(w http.ResponseWriter, r *http.Request) {
	_, user, ok := s.requireAuth(w, r)
	if !ok {
		return
	}
	p, err := s.runtime.Notifications.GetPreferences(r.Context(), user.ID)
	if err != nil {
		writeProblem(w, 503, "notification_preferences_unavailable", "preferences could not be loaded")
		return
	}
	writeJSON(w, 200, map[string]any{"preferences": p, "required_groups": []string{"TRANSACTIONAL_REQUIRED", "SECURITY_REQUIRED"}, "optional_groups": []string{"PAYMENT_REMINDERS", "PRODUCT_UPDATES"}, "required_explanation": "Security and transactional receipts protect your account and financial records and cannot be disabled."})
}

type preferenceUpdate struct {
	PreferredChannel        string `json:"preferred_channel"`
	FallbackChannel         string `json:"fallback_channel"`
	PaymentRemindersEnabled bool   `json:"payment_reminders_enabled"`
	ProductUpdatesEnabled   bool   `json:"product_updates_enabled"`
	QuietStart              int    `json:"quiet_start_hour"`
	QuietEnd                int    `json:"quiet_end_hour"`
	Timezone                string `json:"timezone"`
	ExpectedVersion         int64  `json:"expected_version"`
	DisableRequired         bool   `json:"disable_required"`
}

func (s *Server) updateNotificationPreferences(w http.ResponseWriter, r *http.Request) {
	_, user, ok := s.requireAuth(w, r)
	if !ok || !s.requireCSRF(w, r) {
		return
	}
	var in preferenceUpdate
	if !decodeJSONRequest(w, r, &in) {
		return
	}
	if in.DisableRequired {
		writeProblem(w, 409, "required_notification_cannot_disable", "Security and transactional messages cannot be disabled.")
		return
	}
	p, err := s.runtime.Notifications.UpdatePreferences(r.Context(), user.ID, notifications.Preferences{PreferredChannel: in.PreferredChannel, FallbackChannel: in.FallbackChannel, PaymentRemindersEnabled: in.PaymentRemindersEnabled, ProductUpdatesEnabled: in.ProductUpdatesEnabled, QuietStart: in.QuietStart, QuietEnd: in.QuietEnd, Timezone: in.Timezone}, in.ExpectedVersion)
	if err != nil {
		status := 400
		code := "notification_preference_invalid"
		if strings.Contains(err.Error(), "version") {
			status = 409
			code = "notification_preference_conflict"
		}
		writeProblem(w, status, code, err.Error())
		return
	}
	s.runtime.Audit.Append(audit.Event{ActorUserID: user.ID, Action: "notification.preferences_updated", ResourceType: "notification_preferences", ResourceID: user.ID, Outcome: "success", RequestID: requestIDFromContext(r.Context())})
	_, _ = s.runtime.Notifications.Emit(r.Context(), notifications.Event{ID: "preferences:" + user.ID + ":" + strings.TrimSpace(r.Header.Get("Idempotency-Key")), Type: "NotificationPreferencesChanged", RecipientID: user.ID, Email: user.Email, Phone: user.Phone, Priority: notifications.PriorityCritical, Reference: "preferences updated"})
	writeJSON(w, 200, map[string]any{"preferences": p})
}

func (s *Server) regenerateRecoveryCodes(w http.ResponseWriter, r *http.Request) {
	session, user, ok := s.requireAuth(w, r)
	if !ok || !s.requireFreshMFA(w, session) || !s.requireCSRF(w, r) {
		return
	}
	codes, err := s.runtime.UserControl.GenerateRecoveryCodes(r.Context(), user.ID)
	if err != nil {
		writeProblem(w, 503, "recovery_codes_unavailable", "recovery codes could not be regenerated")
		return
	}
	s.runtime.Audit.Append(audit.Event{ActorUserID: user.ID, Action: "account.recovery_codes_regenerated", ResourceType: "account", ResourceID: user.ID, Severity: "notice"})
	writeJSON(w, 200, map[string]any{"recovery_codes": codes, "warning": "Old recovery codes are revoked. Store these one-time codes securely; they will not be shown again."})
}

type recoveryStart struct {
	Identifier string `json:"identifier"`
	Channel    string `json:"channel"`
}

func (s *Server) requestAccountRecovery(w http.ResponseWriter, r *http.Request) {
	var in recoveryStart
	if !decodeJSONRequest(w, r, &in) {
		return
	}
	id, _ := s.runtime.UserControl.RequestRecovery(r.Context(), in.Identifier, in.Channel, r.RemoteAddr)
	if id != "" {
		if req, err := s.runtime.UserControl.Recovery(r.Context(), id); err == nil {
			s.notifyUser(r, req.TargetUserID, "AccountRecoveryRequested", "recovery-requested:"+id, id)
		}
	}
	response := map[string]any{"message": "If the account is eligible, recovery instructions have been sent."}
	if s.config.Environment == "development" && id != "" {
		response["development_request_id"] = id
	}
	writeJSON(w, 202, response)
}

type recoveryEvidenceInput struct {
	FactorType  string `json:"factor_type"`
	Proof       string `json:"proof"`
	ChallengeID string `json:"challenge_id"`
	Code        string `json:"code"`
	Channel     string `json:"channel"`
	Identifier  string `json:"identifier"`
}

func (s *Server) addAccountRecoveryEvidence(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("requestID")
	var in recoveryEvidenceInput
	if !decodeJSONRequest(w, r, &in) {
		return
	}
	factor := strings.ToLower(in.FactorType)
	proof := in.Proof
	if factor != "recovery_code" && factor != "verified_email" && factor != "verified_phone" {
		writeProblem(w, http.StatusBadRequest, "recovery_verification_incomplete", "Only a one-time recovery code or independently verified contact can be submitted here.")
		return
	}
	if factor == "verified_email" || factor == "verified_phone" {
		expectedChannel := "email"
		if factor == "verified_phone" {
			expectedChannel = "phone"
		}
		if in.Channel != expectedChannel {
			writeProblem(w, 400, "recovery_verification_incomplete", "independent contact verification is required")
			return
		}
		user, _, token, err := s.runtime.Auth.VerifyOTPForTarget(in.ChallengeID, in.Code, "recovery", in.Channel, in.Identifier)
		if err != nil {
			writeProblem(w, 400, "recovery_verification_incomplete", "independent contact verification is required")
			return
		}
		_ = s.runtime.Auth.RevokeSession(token)
		req, err := s.runtime.UserControl.Recovery(r.Context(), id)
		if err != nil || req.TargetUserID != user.ID {
			writeProblem(w, 400, "recovery_verification_incomplete", "independent contact verification is required")
			return
		}
		proof = in.ChallengeID
	}
	req, err := s.runtime.UserControl.AddRecoveryEvidence(r.Context(), id, factor, proof)
	if err != nil {
		writeProblem(w, 400, "recovery_verification_incomplete", err.Error())
		return
	}
	writeJSON(w, 200, map[string]any{"request": req, "message": "Evidence recorded. Recovery requires two independent factors and compliance review."})
}

func (s *Server) cancelAccountRecovery(w http.ResponseWriter, r *http.Request) {
	_, user, ok := s.requireAuth(w, r)
	if !ok || !s.requireCSRF(w, r) {
		return
	}
	if err := s.runtime.UserControl.CancelRecovery(r.Context(), r.PathValue("requestID"), user.ID); err != nil {
		writeProblem(w, 404, "recovery_request_invalid", err.Error())
		return
	}
	s.runtime.Audit.Append(audit.Event{ActorUserID: user.ID, Action: "account.recovery_cancelled", ResourceType: "account_recovery", ResourceID: r.PathValue("requestID"), Severity: "warning"})
	s.notifyUser(r, user.ID, "AccountRecoveryCancelled", "recovery-cancelled:"+r.PathValue("requestID"), r.PathValue("requestID"))
	w.WriteHeader(204)
}

type recoveryCompleteInput struct {
	Token string `json:"token"`
}

func (s *Server) completeAccountRecovery(w http.ResponseWriter, r *http.Request) {
	var in recoveryCompleteInput
	if !decodeJSONRequest(w, r, &in) {
		return
	}
	userID, err := s.runtime.UserControl.CompleteRecovery(r.Context(), r.PathValue("requestID"), in.Token)
	if err != nil {
		writeProblem(w, 409, "recovery_cooling_off", err.Error())
		return
	}
	if err = s.runtime.Auth.RevokeAllSessions(userID); err != nil {
		writeProblem(w, 503, "recovery_completion_failed", "sessions could not be revoked")
		return
	}
	clearSessionCookies(w, s.config.Environment != "development")
	s.runtime.Audit.Append(audit.Event{ActorUserID: userID, Action: "account.recovery_completed", ResourceType: "account_recovery", ResourceID: r.PathValue("requestID"), Severity: "warning"})
	s.notifyUser(r, userID, "AccountRecoveryCompleted", "recovery-completed:"+r.PathValue("requestID"), r.PathValue("requestID"))
	writeJSON(w, 200, map[string]any{"status": "COMPLETED", "message": "Account recovery completed. Existing sessions and old recovery codes were revoked."})
}

func (s *Server) listRecoveryReviews(w http.ResponseWriter, r *http.Request) {
	session, _, _, ok := s.requirePlatformAccess(w, r, access.PermissionRecoverAccounts)
	if !ok || !s.requireFreshMFA(w, session) {
		return
	}
	items, err := s.runtime.UserControl.ListRecoveries(r.Context(), r.URL.Query().Get("state"))
	if err != nil {
		writeProblem(w, 503, "recovery_review_unavailable", "recovery queue could not be loaded")
		return
	}
	writeJSON(w, 200, map[string]any{"requests": items})
}

type reviewRecoveryInput struct {
	Decision        string `json:"decision"`
	Reason          string `json:"reason"`
	ExpectedVersion int64  `json:"expected_version"`
}

func (s *Server) reviewAccountRecovery(w http.ResponseWriter, r *http.Request) {
	session, user, _, ok := s.requirePlatformAccess(w, r, access.PermissionRecoverAccounts)
	if !ok || !s.requireFreshMFA(w, session) || !s.requireCSRF(w, r) {
		return
	}
	var in reviewRecoveryInput
	if !decodeJSONRequest(w, r, &in) {
		return
	}
	req, token, err := s.runtime.UserControl.ReviewRecovery(r.Context(), r.PathValue("requestID"), user.ID, in.Decision, in.Reason, in.ExpectedVersion)
	if err != nil {
		writeProblem(w, 409, "recovery_conflict", err.Error())
		return
	}
	response := map[string]any{"request": req}
	if s.config.Environment == "development" && token != "" {
		response["development_completion_token"] = token
	}
	s.runtime.Audit.Append(audit.Event{ActorUserID: user.ID, Action: "account.recovery_reviewed", ResourceType: "account_recovery", ResourceID: req.ID, Severity: "warning", Metadata: map[string]string{"decision": in.Decision, "reason": in.Reason}})
	if req.State == "COOLING_OFF" {
		s.notifyUser(r, req.TargetUserID, "AccountRecoveryCoolingOff", "recovery-cooling:"+req.ID, req.ID)
	}
	writeJSON(w, 200, response)
}

type privacyCreateInput struct {
	RequestType    string `json:"request_type"`
	OrganizationID string `json:"organization_id"`
	Details        string `json:"details"`
}

func (s *Server) createPrivacyRequest(w http.ResponseWriter, r *http.Request) {
	session, user, ok := s.requireAuth(w, r)
	if !ok || !s.requireCSRF(w, r) {
		return
	}
	if session.AuthenticationLevel != auth.AAL2 {
		writeProblem(w, 403, "privacy_identity_check_required", "Step-up authentication is required to submit a privacy request.")
		return
	}
	var in privacyCreateInput
	if !decodeJSONRequest(w, r, &in) {
		return
	}
	item, err := s.runtime.UserControl.CreatePrivacyRequest(r.Context(), user.ID, in.OrganizationID, in.RequestType, in.Details)
	if err != nil {
		writeProblem(w, 400, "privacy_request_invalid", err.Error())
		return
	}
	s.runtime.Audit.Append(audit.Event{ActorUserID: user.ID, Action: "privacy.request_received", ResourceType: "privacy_request", ResourceID: item.ID})
	_, _ = s.runtime.Notifications.Emit(r.Context(), notifications.Event{ID: "privacy-received:" + item.ID, Type: "PrivacyRequestReceived", RecipientID: user.ID, Email: user.Email, Phone: user.Phone, Priority: notifications.PriorityCritical, Reference: item.ID})
	writeJSON(w, 201, map[string]any{"request": item})
}
func (s *Server) listMyPrivacyRequests(w http.ResponseWriter, r *http.Request) {
	_, user, ok := s.requireAuth(w, r)
	if !ok {
		return
	}
	items, err := s.runtime.UserControl.ListPrivacyForUser(r.Context(), user.ID)
	if err != nil {
		writeProblem(w, 503, "privacy_requests_unavailable", "privacy requests could not be loaded")
		return
	}
	writeJSON(w, 200, map[string]any{"requests": items})
}

func (s *Server) downloadPrivacyExport(w http.ResponseWriter, r *http.Request) {
	_, user, ok := s.requireAuth(w, r)
	if !ok {
		return
	}
	payload, err := s.runtime.UserControl.PrivacyExport(r.Context(), r.PathValue("requestID"), user.ID)
	if err != nil {
		writeProblem(w, 404, "privacy_export_unavailable", err.Error())
		return
	}
	s.runtime.Audit.Append(audit.Event{ActorUserID: user.ID, Action: "privacy.export_downloaded", ResourceType: "privacy_request", ResourceID: r.PathValue("requestID"), Severity: "notice"})
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", "attachment; filename=kredit-privacy-export.json")
	_, _ = w.Write(payload)
}
func (s *Server) listPrivacyReviews(w http.ResponseWriter, r *http.Request) {
	session, _, _, ok := s.requirePlatformAccess(w, r, access.PermissionReviewPrivacy)
	if !ok || !s.requireFreshMFA(w, session) {
		return
	}
	items, err := s.runtime.UserControl.ListPrivacyReview(r.Context())
	if err != nil {
		writeProblem(w, 503, "privacy_requests_unavailable", "privacy review queue could not be loaded")
		return
	}
	writeJSON(w, 200, map[string]any{"requests": items})
}

type privacyDecisionInput struct {
	Decision        string `json:"decision"`
	Reason          string `json:"reason"`
	ExpectedVersion int64  `json:"expected_version"`
}

func (s *Server) decidePrivacyRequest(w http.ResponseWriter, r *http.Request) {
	session, user, _, ok := s.requirePlatformAccess(w, r, access.PermissionReviewPrivacy)
	if !ok || !s.requireFreshMFA(w, session) || !s.requireCSRF(w, r) {
		return
	}
	var in privacyDecisionInput
	if !decodeJSONRequest(w, r, &in) {
		return
	}
	item, err := s.runtime.UserControl.DecidePrivacy(r.Context(), r.PathValue("requestID"), user.ID, in.Decision, in.Reason, in.ExpectedVersion)
	if err != nil {
		writeProblem(w, 409, "privacy_decision_invalid", err.Error())
		return
	}
	s.runtime.Audit.Append(audit.Event{ActorUserID: user.ID, Action: "privacy.request_decided", ResourceType: "privacy_request", ResourceID: item.ID, Metadata: map[string]string{"decision": item.State, "reason": in.Reason}})
	s.notifyUser(r, item.RequesterUserID, "PrivacyRequestDecided", "privacy-decided:"+item.ID, item.ID)
	writeJSON(w, 200, map[string]any{"request": item})
}

type privacyCompleteInput struct {
	DecisionReviewerID string `json:"decision_reviewer_id"`
	ExpectedVersion    int64  `json:"expected_version"`
}

func (s *Server) completePrivacyRequest(w http.ResponseWriter, r *http.Request) {
	session, user, _, ok := s.requirePlatformAccess(w, r, access.PermissionReviewPrivacy)
	if !ok || !s.requireFreshMFA(w, session) || !s.requireCSRF(w, r) {
		return
	}
	var in privacyCompleteInput
	if !decodeJSONRequest(w, r, &in) {
		return
	}
	item, err := s.runtime.UserControl.CompletePrivacy(r.Context(), r.PathValue("requestID"), in.DecisionReviewerID, user.ID, in.ExpectedVersion)
	if err != nil {
		writeProblem(w, 409, "privacy_request_conflict", err.Error())
		return
	}
	s.runtime.Audit.Append(audit.Event{ActorUserID: user.ID, Action: "privacy.request_completed", ResourceType: "privacy_request", ResourceID: item.ID, Severity: "notice"})
	eventType := "PrivacyRequestCompleted"
	if item.ExportReference != "" {
		eventType = "PrivacyExportReady"
	}
	s.notifyUser(r, item.RequesterUserID, eventType, "privacy-complete:"+item.ID, item.ID)
	writeJSON(w, 200, map[string]any{"request": item})
}

func (s *Server) notifyUser(r *http.Request, userID, eventType, eventID, reference string) {
	user, err := s.runtime.Auth.UserByID(userID)
	if err != nil {
		return
	}
	_, _ = s.runtime.Notifications.Emit(r.Context(), notifications.Event{ID: eventID, Type: eventType, RecipientID: userID, Email: user.Email, Phone: user.Phone, Priority: notifications.PriorityCritical, Reference: reference})
}
