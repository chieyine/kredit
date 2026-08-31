package web

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"kredit/internal/access"
	"kredit/internal/audit"
	"kredit/internal/auth"
	"kredit/internal/disputes"
	"kredit/internal/ledger"
	"kredit/internal/notifications"
	"kredit/internal/platformops"
	"kredit/internal/support"

	"github.com/google/uuid"
)

func (s *Server) requirePlatformAccess(w http.ResponseWriter, r *http.Request, permission access.Permission) (auth.Session, auth.User, access.PlatformRole, bool) {
	session, user, ok := s.requireAuth(w, r)
	if !ok {
		return auth.Session{}, auth.User{}, "", false
	}
	if s.runtime.Database == nil || s.runtime.PlatformOps == nil {
		writeProblem(w, http.StatusServiceUnavailable, "operations_unavailable", "the operations service is unavailable")
		return auth.Session{}, auth.User{}, "", false
	}
	rows, err := s.runtime.Database.Raw().Query(r.Context(), `SELECT role FROM app.platform_role_assignments WHERE user_id=$1::uuid AND revoked_at IS NULL AND (expires_at IS NULL OR expires_at>now()) ORDER BY granted_at DESC`, user.ID)
	if err != nil {
		writeProblem(w, http.StatusServiceUnavailable, "operations_unavailable", "platform authorization could not be verified")
		return auth.Session{}, auth.User{}, "", false
	}
	defer rows.Close()
	var selected access.PlatformRole
	for rows.Next() {
		var role access.PlatformRole
		if err := rows.Scan(&role); err != nil {
			writeProblem(w, http.StatusServiceUnavailable, "operations_unavailable", "platform authorization could not be verified")
			return auth.Session{}, auth.User{}, "", false
		}
		if role.Valid() && (permission == "" || access.CanPlatform(role, permission)) {
			selected = role
			break
		}
	}
	if err := rows.Err(); err != nil {
		writeProblem(w, http.StatusServiceUnavailable, "operations_unavailable", "platform authorization could not be verified")
		return auth.Session{}, auth.User{}, "", false
	}
	if selected == "" {
		s.runtime.Audit.Append(audit.Event{ActorUserID: user.ID, Action: "authorization.platform_denied", ResourceType: "platform_operations", Outcome: "denied", Severity: "warning", RequestID: requestIDFromContext(r.Context()), Metadata: map[string]string{"permission": string(permission)}})
		writeProblem(w, http.StatusForbidden, "platform_forbidden", "you do not have access to this operations view")
		return auth.Session{}, auth.User{}, "", false
	}
	if session.AuthenticationLevel != auth.AAL2 {
		writeProblem(w, http.StatusForbidden, "step_up_required", "step-up authentication is required for platform operations")
		return auth.Session{}, auth.User{}, "", false
	}
	return session, user, selected, true
}

func (s *Server) operationsOverview(w http.ResponseWriter, r *http.Request) {
	_, user, role, ok := s.requirePlatformAccess(w, r, "")
	if !ok {
		return
	}
	result, err := s.runtime.PlatformOps.Overview(r.Context())
	if err != nil {
		writeProblem(w, http.StatusServiceUnavailable, "operations_unavailable", "operations health could not be loaded")
		return
	}
	s.auditPlatformRead(r, user.ID, "operations.overview.viewed", "operations_overview", "")
	writeJSON(w, http.StatusOK, map[string]any{"overview": result, "role": role})
}

func (s *Server) operationsAnalyticsScorecard(w http.ResponseWriter, r *http.Request) {
	_, user, _, ok := s.requirePlatformAccess(w, r, access.PermissionReviewCompliance)
	if !ok {
		return
	}
	now := time.Now().UTC()
	from := now.AddDate(0, 0, -7)
	to := now
	var err error
	if value := strings.TrimSpace(r.URL.Query().Get("from")); value != "" {
		from, err = time.Parse("2006-01-02", value)
		if err != nil {
			writeProblem(w, http.StatusBadRequest, "invalid_scorecard_window", "from must use YYYY-MM-DD")
			return
		}
	}
	if value := strings.TrimSpace(r.URL.Query().Get("to")); value != "" {
		to, err = time.Parse("2006-01-02", value)
		if err != nil {
			writeProblem(w, http.StatusBadRequest, "invalid_scorecard_window", "to must use YYYY-MM-DD")
			return
		}
		to = to.Add(24 * time.Hour)
	}
	organizationID := strings.TrimSpace(r.URL.Query().Get("organization_id"))
	if organizationID != "" {
		if _, err := uuid.Parse(organizationID); err != nil {
			writeProblem(w, http.StatusBadRequest, "invalid_organization", "organization_id must be a UUID")
			return
		}
	}
	result, err := s.runtime.Reports.PilotScorecard(r.Context(), from, to, organizationID)
	if err != nil {
		if strings.Contains(err.Error(), "window") {
			writeProblem(w, http.StatusBadRequest, "invalid_scorecard_window", err.Error())
			return
		}
		writeProblem(w, http.StatusServiceUnavailable, "analytics_unavailable", "the pilot scorecard could not be calculated")
		return
	}
	s.auditPlatformRead(r, user.ID, "operations.analytics.viewed", "pilot_scorecard", organizationID)
	writeJSON(w, http.StatusOK, map[string]any{"scorecard": result})
}

func (s *Server) operationsJobs(w http.ResponseWriter, r *http.Request) {
	_, user, _, ok := s.requirePlatformAccess(w, r, access.PermissionProviderOperations)
	if !ok {
		return
	}
	items, err := s.runtime.PlatformOps.Jobs(r.Context(), queryLimit(r))
	if err != nil {
		writeProblem(w, http.StatusServiceUnavailable, "jobs_unavailable", "job activity could not be loaded")
		return
	}
	s.auditPlatformRead(r, user.ID, "operations.jobs.viewed", "job", "")
	writeJSON(w, http.StatusOK, map[string]any{"jobs": items})
}

func (s *Server) operationsProviderEvents(w http.ResponseWriter, r *http.Request) {
	_, user, _, ok := s.requirePlatformAccess(w, r, access.PermissionProviderOperations)
	if !ok {
		return
	}
	items, err := s.runtime.PlatformOps.ProviderEvents(r.Context(), queryLimit(r))
	if err != nil {
		writeProblem(w, http.StatusServiceUnavailable, "provider_events_unavailable", "provider event activity could not be loaded")
		return
	}
	s.auditPlatformRead(r, user.ID, "operations.provider_events.viewed", "provider_event", "")
	writeJSON(w, http.StatusOK, map[string]any{"events": items})
}

func (s *Server) previewOperationsCommand(w http.ResponseWriter, r *http.Request) {
	_, user, _, ok := s.requirePlatformAccess(w, r, access.PermissionProviderOperations)
	if !ok {
		return
	}
	var in platformops.CommandInput
	if !decodeJSONRequest(w, r, &in) {
		return
	}
	if in.Type == "place_risk_hold" || in.Type == "lift_risk_hold" {
		if _, _, _, ok = s.requirePlatformAccess(w, r, access.PermissionManageRiskHold); !ok {
			return
		}
	}
	preview, err := s.runtime.PlatformOps.PreviewCommand(r.Context(), in)
	if err != nil {
		writeProblem(w, http.StatusUnprocessableEntity, "command_preview_failed", err.Error())
		return
	}
	s.auditPlatformRead(r, user.ID, "operations.command.previewed", in.TargetType, in.TargetID)
	writeJSON(w, http.StatusOK, map[string]any{"command": preview})
}

func (s *Server) executeOperationsCommand(w http.ResponseWriter, r *http.Request) {
	session, user, _, ok := s.requirePlatformAccess(w, r, access.PermissionProviderOperations)
	if !ok || !s.requireFreshMFA(w, session) || !s.requireCSRF(w, r) {
		return
	}
	var in platformops.CommandInput
	if !decodeJSONRequest(w, r, &in) {
		return
	}
	if in.Type == "place_risk_hold" || in.Type == "lift_risk_hold" {
		if _, _, _, ok = s.requirePlatformAccess(w, r, access.PermissionManageRiskHold); !ok {
			return
		}
	}
	in.IdempotencyKey = strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	in.CorrelationID = requestIDFromContext(r.Context())
	var external any
	var err error
	switch in.Type {
	case "resolve_unknown_submission":
		external, err = s.runtime.Collections.Reconcile(r.Context(), in.TargetID)
	case "retry_collection":
		external, err = s.runtime.Collections.Retry(r.Context(), in.TargetID, time.Now().UTC())
	case "cancel_collection":
		external, err = s.runtime.Collections.Cancel(r.Context(), in.TargetID)
	}
	if err != nil {
		s.runtime.Audit.Append(audit.Event{ActorUserID: user.ID, OrganizationID: in.OrganizationID, Action: "operations.command.failed", ResourceType: in.TargetType, ResourceID: in.TargetID, Outcome: "failure", Severity: "high", RequestID: in.CorrelationID, Metadata: map[string]string{"command_type": in.Type, "reason": in.Reason}})
		writeProblem(w, http.StatusConflict, "operations_command_failed", err.Error())
		return
	}
	if external != nil {
		encoded, _ := json.Marshal(external)
		_ = json.Unmarshal(encoded, &in.ExternalResult)
	}
	command, err := s.runtime.PlatformOps.ExecuteCommand(r.Context(), user.ID, in)
	if err != nil {
		status := http.StatusUnprocessableEntity
		if strings.Contains(err.Error(), "version conflict") {
			status = http.StatusConflict
		}
		writeProblem(w, status, "operations_command_failed", err.Error())
		return
	}
	s.runtime.Audit.Append(audit.Event{ActorUserID: user.ID, OrganizationID: in.OrganizationID, Action: "operations.command.applied", ResourceType: in.TargetType, ResourceID: in.TargetID, Outcome: "success", Severity: "high", RequestID: in.CorrelationID, Metadata: map[string]string{"command_id": command.ID, "command_type": in.Type, "reason": in.Reason}})
	s.notifyOperationsTarget(r, in, command)
	writeJSON(w, http.StatusOK, map[string]any{"command": command})
}

func (s *Server) operationsDiagnostics(w http.ResponseWriter, r *http.Request) {
	_, user, _, ok := s.requirePlatformAccess(w, r, access.PermissionProviderOperations)
	if !ok {
		return
	}
	window, _ := strconv.Atoi(r.URL.Query().Get("window_minutes"))
	diagnostics, err := s.runtime.PlatformOps.Diagnostics(r.Context(), window, requestIDFromContext(r.Context()))
	if err != nil {
		writeProblem(w, http.StatusServiceUnavailable, "diagnostics_unavailable", "operational diagnostics could not be loaded")
		return
	}
	s.auditPlatformRead(r, user.ID, "operations.diagnostics.viewed", "diagnostics", "")
	writeJSON(w, http.StatusOK, map[string]any{"diagnostics": diagnostics})
}

func (s *Server) notifyOperationsTarget(r *http.Request, in platformops.CommandInput, command platformops.Command) {
	if s.runtime.Database == nil || s.runtime.Notifications == nil {
		return
	}
	recipients := []struct{ ID, Email, Phone string }{}
	if affected, ok := command.Result["affected_target_id"].(string); ok && affected != "" {
		in.TargetID = affected
	}
	if in.TargetType == "user" || in.TargetType == "buyer" {
		var recipient struct{ ID, Email, Phone string }
		recipient.ID = in.TargetID
		if s.runtime.Database.Raw().QueryRow(r.Context(), `SELECT COALESCE(normalized_email,''),COALESCE(normalized_phone,'') FROM app.users WHERE id=$1::uuid`, in.TargetID).Scan(&recipient.Email, &recipient.Phone) == nil {
			recipients = append(recipients, recipient)
		}
	} else if in.TargetType == "organization" || in.TargetType == "supplier" {
		rows, err := s.runtime.Database.Raw().Query(r.Context(), `SELECT u.id::text,COALESCE(u.normalized_email,''),COALESCE(u.normalized_phone,'') FROM app.memberships m JOIN app.users u ON u.id=m.user_id WHERE m.organization_id=$1::uuid AND m.role IN('owner','administrator') AND m.status='active'`, in.TargetID)
		if err == nil {
			for rows.Next() {
				var item struct{ ID, Email, Phone string }
				if rows.Scan(&item.ID, &item.Email, &item.Phone) == nil {
					recipients = append(recipients, item)
				}
			}
			if rows.Err() != nil {
				s.logger.Error("operations target notification recipients could not be fully loaded", "target_type", in.TargetType, "target_id", in.TargetID, "error", rows.Err())
			}
			rows.Close()
		}
	} else if in.TargetType == "collection" {
		rows, err := s.runtime.Database.Raw().Query(r.Context(), `SELECT u.id::text,COALESCE(u.normalized_email,''),COALESCE(u.normalized_phone,'') FROM app.collection_attempts ca JOIN app.obligations o ON o.id=ca.obligation_id JOIN app.credit_requests cr ON cr.id=o.credit_request_id JOIN app.users u ON u.id=cr.buyer_user_id WHERE ca.id=$1::uuid UNION SELECT u.id::text,COALESCE(u.normalized_email,''),COALESCE(u.normalized_phone,'') FROM app.collection_attempts ca JOIN app.obligations o ON o.id=ca.obligation_id JOIN app.memberships m ON m.organization_id=o.supplier_organization_id AND m.role IN('owner','administrator') AND m.status='active' JOIN app.users u ON u.id=m.user_id WHERE ca.id=$1::uuid`, in.TargetID)
		if err == nil {
			for rows.Next() {
				var item struct{ ID, Email, Phone string }
				if rows.Scan(&item.ID, &item.Email, &item.Phone) == nil {
					recipients = append(recipients, item)
				}
			}
			if rows.Err() != nil {
				s.logger.Error("operations target notification recipients could not be fully loaded", "target_type", in.TargetType, "target_id", in.TargetID, "error", rows.Err())
			}
			rows.Close()
		}
	}
	for _, recipient := range recipients {
		_, _ = s.runtime.Notifications.Emit(r.Context(), notifications.Event{ID: "operations:" + command.ID + ":" + recipient.ID, Type: "OperationsControlApplied", RecipientID: recipient.ID, Email: recipient.Email, Phone: recipient.Phone, OrganizationID: in.OrganizationID, Priority: notifications.PriorityCritical, Reference: fmt.Sprintf("%s (%s)", in.Type, command.ID), NextAction: "Review this protected account or organization change and contact support if unexpected.", SecurePath: "/app/settings"})
	}
}

func (s *Server) operationsSearch(w http.ResponseWriter, r *http.Request) {
	_, user, _, ok := s.requirePlatformAccess(w, r, access.PermissionSupportSearch)
	if !ok {
		return
	}
	query := strings.TrimSpace(r.URL.Query().Get("q"))
	items, err := s.runtime.PlatformOps.Search(r.Context(), query)
	if err != nil {
		writeProblem(w, http.StatusBadRequest, "invalid_search", err.Error())
		return
	}
	s.auditPlatformRead(r, user.ID, "operations.reference.searched", "reference", query)
	writeJSON(w, http.StatusOK, map[string]any{"results": items})
}

func (s *Server) operationsUsers(w http.ResponseWriter, r *http.Request) {
	_, user, _, ok := s.requirePlatformAccess(w, r, access.PermissionSupportSearch)
	if !ok {
		return
	}
	items, err := s.runtime.PlatformOps.Users(r.Context(), r.URL.Query().Get("q"), queryLimit(r))
	if err != nil {
		writeProblem(w, http.StatusServiceUnavailable, "users_unavailable", "users could not be loaded")
		return
	}
	s.auditPlatformRead(r, user.ID, "operations.users.viewed", "user_directory", "")
	writeJSON(w, http.StatusOK, map[string]any{"users": items})
}

func (s *Server) operationsOrganizations(w http.ResponseWriter, r *http.Request) {
	_, user, _, ok := s.requirePlatformAccess(w, r, access.PermissionSupportSearch)
	if !ok {
		return
	}
	items, err := s.runtime.PlatformOps.Organizations(r.Context(), r.URL.Query().Get("q"), queryLimit(r))
	if err != nil {
		writeProblem(w, http.StatusServiceUnavailable, "organizations_unavailable", "businesses could not be loaded")
		return
	}
	s.auditPlatformRead(r, user.ID, "operations.organizations.viewed", "organization_directory", "")
	writeJSON(w, http.StatusOK, map[string]any{"organizations": items})
}

func (s *Server) operationsMoney(w http.ResponseWriter, r *http.Request) {
	_, user, _, ok := s.requirePlatformAccess(w, r, access.PermissionReviewCompliance)
	if !ok {
		return
	}
	summary, activity, err := s.runtime.PlatformOps.Money(r.Context(), queryLimit(r))
	if err != nil {
		writeProblem(w, http.StatusServiceUnavailable, "money_unavailable", "platform money activity could not be loaded")
		return
	}
	s.auditPlatformRead(r, user.ID, "operations.money.viewed", "platform_money", "")
	writeJSON(w, http.StatusOK, map[string]any{"summary": summary, "activity": activity})
}

func (s *Server) operationsCases(w http.ResponseWriter, r *http.Request) {
	_, user, _, ok := s.requirePlatformAccess(w, r, access.PermissionManageCases)
	if !ok {
		return
	}
	items, err := s.runtime.PlatformOps.Cases(r.Context(), r.URL.Query().Get("state"), queryLimit(r))
	if err != nil {
		writeProblem(w, http.StatusServiceUnavailable, "cases_unavailable", "support cases could not be loaded")
		return
	}
	s.auditPlatformRead(r, user.ID, "operations.cases.viewed", "support_case", "")
	writeJSON(w, http.StatusOK, map[string]any{"cases": items})
}

func (s *Server) operationsDisputes(w http.ResponseWriter, r *http.Request) {
	_, user, _, ok := s.requirePlatformAccess(w, r, access.PermissionReviewDisputes)
	if !ok {
		return
	}
	items, err := s.runtime.PlatformOps.Disputes(r.Context(), r.URL.Query().Get("state"), queryLimit(r))
	if err != nil {
		writeProblem(w, http.StatusServiceUnavailable, "disputes_unavailable", "disputes could not be loaded")
		return
	}
	s.auditPlatformRead(r, user.ID, "operations.disputes.viewed", "dispute", "")
	writeJSON(w, http.StatusOK, map[string]any{"disputes": items})
}

func (s *Server) transitionOperationsCase(w http.ResponseWriter, r *http.Request) {
	session, user, _, ok := s.requirePlatformAccess(w, r, access.PermissionManageCases)
	if !ok || !s.requireFreshMFA(w, session) || !s.requireCSRF(w, r) {
		return
	}
	if _, err := uuid.Parse(r.PathValue("caseID")); err != nil {
		writeProblem(w, http.StatusBadRequest, "invalid_case", "case ID must be valid")
		return
	}
	var input struct {
		State string `json:"state"`
		Note  string `json:"note"`
	}
	if !decodeJSONRequest(w, r, &input) {
		return
	}
	item, event, err := s.runtime.Support.Transition(r.PathValue("caseID"), user.ID, support.State(strings.ToUpper(strings.TrimSpace(input.State))), strings.TrimSpace(input.Note))
	if err != nil {
		writeProblem(w, http.StatusConflict, "case_transition_failed", err.Error())
		return
	}
	s.runtime.Audit.Append(audit.Event{ActorUserID: user.ID, OrganizationID: item.OrganizationID, Action: "operations.case.transitioned", ResourceType: "support_case", ResourceID: item.ID, Outcome: "success", Severity: "notice", RequestID: requestIDFromContext(r.Context()), Metadata: map[string]string{"state": string(item.State)}})
	writeJSON(w, http.StatusOK, map[string]any{"case": item, "event": event})
}

func (s *Server) decideOperationsDispute(w http.ResponseWriter, r *http.Request) {
	session, user, _, ok := s.requirePlatformAccess(w, r, access.PermissionReviewDisputes)
	if !ok || !s.requireFreshMFA(w, session) || !s.requireCSRF(w, r) {
		return
	}
	if _, err := uuid.Parse(r.PathValue("disputeID")); err != nil {
		writeProblem(w, http.StatusBadRequest, "invalid_dispute", "dispute ID must be valid")
		return
	}
	var input struct {
		Outcome               string `json:"outcome"`
		ValidPrincipalKobo    int64  `json:"valid_principal_kobo"`
		AdjustmentKobo        int64  `json:"adjustment_kobo"`
		RemainingDisputedKobo int64  `json:"remaining_disputed_kobo"`
		Reason                string `json:"reason"`
	}
	if !decodeJSONRequest(w, r, &input) {
		return
	}
	item, decision, err := s.runtime.Disputes.Decide(disputes.DecideInput{DisputeID: r.PathValue("disputeID"), ReviewerID: user.ID, Outcome: strings.TrimSpace(input.Outcome), ValidPrincipalKobo: ledger.Money(input.ValidPrincipalKobo), AdjustmentKobo: ledger.Money(input.AdjustmentKobo), RemainingDisputedKobo: ledger.Money(input.RemainingDisputedKobo), Reason: strings.TrimSpace(input.Reason)})
	if err != nil {
		writeProblem(w, http.StatusConflict, "dispute_decision_failed", err.Error())
		return
	}
	s.runtime.Audit.Append(audit.Event{ActorUserID: user.ID, OrganizationID: item.SupplierOrganizationID, Action: "operations.dispute.decided", ResourceType: "dispute", ResourceID: item.ID, Outcome: "success", Severity: "high", RequestID: requestIDFromContext(r.Context()), Metadata: map[string]string{"outcome": decision.Outcome}})
	writeJSON(w, http.StatusOK, map[string]any{"dispute": item, "decision": decision})
}

func (s *Server) operationsTeam(w http.ResponseWriter, r *http.Request) {
	_, user, _, ok := s.requirePlatformAccess(w, r, access.PermissionBreakGlass)
	if !ok {
		return
	}
	items, err := s.runtime.PlatformOps.Team(r.Context())
	if err != nil {
		writeProblem(w, http.StatusServiceUnavailable, "team_unavailable", "admin access could not be loaded")
		return
	}
	s.auditPlatformRead(r, user.ID, "operations.team.viewed", "platform_role_assignment", "")
	writeJSON(w, http.StatusOK, map[string]any{"members": items})
}

func (s *Server) grantOperationsRole(w http.ResponseWriter, r *http.Request) {
	session, user, _, ok := s.requirePlatformAccess(w, r, access.PermissionBreakGlass)
	if !ok || !s.requireFreshMFA(w, session) || !s.requireCSRF(w, r) {
		return
	}
	if _, err := uuid.Parse(r.PathValue("userID")); err != nil {
		writeProblem(w, http.StatusBadRequest, "invalid_user", "user ID must be valid")
		return
	}
	var input struct {
		Role      string `json:"role"`
		Reason    string `json:"reason"`
		ExpiresAt string `json:"expires_at"`
	}
	if !decodeJSONRequest(w, r, &input) {
		return
	}
	role := access.PlatformRole(strings.TrimSpace(input.Role))
	if !role.Valid() {
		writeProblem(w, http.StatusBadRequest, "invalid_platform_role", "choose a valid admin role")
		return
	}
	var expires *time.Time
	if strings.TrimSpace(input.ExpiresAt) != "" {
		parsed, err := time.Parse(time.RFC3339, input.ExpiresAt)
		if err != nil || !parsed.After(time.Now()) {
			writeProblem(w, http.StatusBadRequest, "invalid_role_expiry", "expiry must be a future date and time")
			return
		}
		expires = &parsed
	}
	item, err := s.runtime.PlatformOps.GrantRole(r.Context(), user.ID, r.PathValue("userID"), string(role), input.Reason, expires)
	if err != nil {
		writeProblem(w, http.StatusConflict, "role_grant_failed", err.Error())
		return
	}
	s.runtime.Audit.Append(audit.Event{ActorUserID: user.ID, Action: "operations.role.granted", ResourceType: "platform_role_assignment", ResourceID: item.AssignmentID, Outcome: "success", Severity: "high", RequestID: requestIDFromContext(r.Context()), Metadata: map[string]string{"target_user_id": item.UserID, "role": item.Role, "reason": strings.TrimSpace(input.Reason)}})
	writeJSON(w, http.StatusOK, map[string]any{"member": item})
}

func (s *Server) revokeOperationsRole(w http.ResponseWriter, r *http.Request) {
	session, user, _, ok := s.requirePlatformAccess(w, r, access.PermissionBreakGlass)
	if !ok || !s.requireFreshMFA(w, session) || !s.requireCSRF(w, r) {
		return
	}
	if _, err := uuid.Parse(r.PathValue("assignmentID")); err != nil {
		writeProblem(w, http.StatusBadRequest, "invalid_role_assignment", "role assignment ID must be valid")
		return
	}
	var input struct {
		Reason string `json:"reason"`
	}
	if !decodeJSONRequest(w, r, &input) {
		return
	}
	if length := len(strings.TrimSpace(input.Reason)); length < 8 || length > 1000 {
		writeProblem(w, http.StatusBadRequest, "role_reason_required", "reason must be between 8 and 1000 characters")
		return
	}
	if err := s.runtime.PlatformOps.RevokeRole(r.Context(), user.ID, r.PathValue("assignmentID")); err != nil {
		writeProblem(w, http.StatusConflict, "role_revoke_failed", err.Error())
		return
	}
	s.runtime.Audit.Append(audit.Event{ActorUserID: user.ID, Action: "operations.role.revoked", ResourceType: "platform_role_assignment", ResourceID: r.PathValue("assignmentID"), Outcome: "success", Severity: "high", RequestID: requestIDFromContext(r.Context()), Metadata: map[string]string{"reason": strings.TrimSpace(input.Reason)}})
	writeJSON(w, http.StatusOK, map[string]any{"revoked": true})
}

func (s *Server) operationsAudit(w http.ResponseWriter, r *http.Request) {
	_, user, _, ok := s.requirePlatformAccess(w, r, access.PermissionReviewCompliance)
	if !ok {
		return
	}
	organizationID := strings.TrimSpace(r.URL.Query().Get("organization_id"))
	items, err := s.runtime.PlatformOps.Audit(r.Context(), organizationID, queryLimit(r))
	if err != nil {
		writeProblem(w, http.StatusServiceUnavailable, "audit_unavailable", "audit history could not be loaded")
		return
	}
	s.auditPlatformRead(r, user.ID, "operations.audit.viewed", "audit_event", organizationID)
	writeJSON(w, http.StatusOK, map[string]any{"events": items})
}

func (s *Server) operationsCase(w http.ResponseWriter, r *http.Request) {
	_, user, _, ok := s.requirePlatformAccess(w, r, access.PermissionManageCases)
	if !ok {
		return
	}
	id := r.PathValue("caseID")
	item, found := s.runtime.Support.Get(id)
	if !found {
		writeProblem(w, http.StatusNotFound, "case_not_found", "support case was not found")
		return
	}
	s.auditPlatformRead(r, user.ID, "operations.case.viewed", "support_case", id)
	writeJSON(w, http.StatusOK, map[string]any{"case": item, "timeline": s.runtime.Support.Timeline(id)})
}

func (s *Server) operationsDispute(w http.ResponseWriter, r *http.Request) {
	_, user, _, ok := s.requirePlatformAccess(w, r, access.PermissionReviewDisputes)
	if !ok {
		return
	}
	id := r.PathValue("disputeID")
	item, evidence, decisions, err := s.runtime.Disputes.Get(id)
	if err != nil {
		writeProblem(w, http.StatusNotFound, "dispute_not_found", "dispute was not found")
		return
	}
	s.auditPlatformRead(r, user.ID, "operations.dispute.viewed", "dispute", id)
	writeJSON(w, http.StatusOK, map[string]any{"dispute": item, "evidence": evidence, "decisions": decisions})
}

func (s *Server) auditPlatformRead(r *http.Request, userID, action, resourceType, resourceID string) {
	s.runtime.Audit.Append(audit.Event{ActorUserID: userID, Action: action, ResourceType: resourceType, ResourceID: resourceID, Outcome: "success", Severity: "notice", RequestID: requestIDFromContext(r.Context())})
}

func queryLimit(r *http.Request) int {
	value, err := strconv.Atoi(r.URL.Query().Get("limit"))
	if err != nil {
		return 100
	}
	return value
}
