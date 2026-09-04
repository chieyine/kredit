package web

import (
	"net/http"
	"strings"
	"time"

	"kredit/internal/access"
	"kredit/internal/audit"
	"kredit/internal/auth"
	"kredit/internal/organizations"
)

type organizationRequest struct {
	LegalName        string `json:"legal_name"`
	TradingName      string `json:"trading_name"`
	BusinessType     string `json:"business_type"`
	RegistrationInfo string `json:"registration_info"`
	BusinessAddress  string `json:"business_address"`
	Industry         string `json:"industry"`
	Timezone         string `json:"timezone"`
	Currency         string `json:"currency"`
}

type memberInvitationRequest struct {
	Target  string `json:"target"`
	Channel string `json:"channel"`
	Role    string `json:"role"`
}

type memberRoleRequest struct {
	Role   string `json:"role"`
	Status string `json:"status"`
}

func (s *Server) listOrganizations(w http.ResponseWriter, r *http.Request) {
	_, user, ok := s.requireAuth(w, r)
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"organizations": s.runtime.Organizations.ListForUser(user.ID)})
}

func (s *Server) createOrganization(w http.ResponseWriter, r *http.Request) {
	_, user, ok := s.requireAuth(w, r)
	if !ok {
		return
	}
	if !s.requireCSRF(w, r) {
		return
	}
	var input organizationRequest
	if err := decodeJSON(w, r, &input); err != nil {
		writeProblem(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	organization, membership, err := s.runtime.Organizations.Create(user.ID, organizations.CreateInput{LegalName: input.LegalName, TradingName: input.TradingName, BusinessType: input.BusinessType, RegistrationInfo: input.RegistrationInfo, BusinessAddress: input.BusinessAddress, Industry: input.Industry, Timezone: input.Timezone, Currency: input.Currency})
	if err != nil {
		writeProblem(w, http.StatusUnprocessableEntity, "organization_invalid", err.Error())
		return
	}
	emailVerified, phoneVerified := ownerContactEvidence(user)
	if _, err := s.runtime.Onboarding.Ensure(organization.ID, user.ID, emailVerified, phoneVerified); err != nil {
		writeProblem(w, http.StatusServiceUnavailable, "onboarding_unavailable", "organization was created but its onboarding profile could not be initialized")
		return
	}
	s.runtime.Audit.Append(audit.Event{ActorUserID: user.ID, OrganizationID: organization.ID, Action: "organization.created", ResourceType: "organization", ResourceID: organization.ID, Outcome: "success", RequestID: requestIDFromContext(r.Context())})
	writeJSON(w, http.StatusCreated, map[string]any{"organization": organization, "membership": membership})
}

func (s *Server) getOrganization(w http.ResponseWriter, r *http.Request) {
	organizationID, err := pathID(r, "organizationID")
	if err != nil {
		writeProblem(w, http.StatusBadRequest, "invalid_path", err.Error())
		return
	}
	_, user, membership, ok := s.requireOrganizationAccess(w, r, organizationID, access.PermissionReadOrganization)
	if !ok {
		return
	}
	organization, exists := s.runtime.Organizations.Get(organizationID)
	if !exists {
		writeProblem(w, http.StatusNotFound, "organization_not_found", "organization was not found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"organization": organization, "membership": membership, "user_id": user.ID})
}

func (s *Server) listMembers(w http.ResponseWriter, r *http.Request) {
	organizationID, err := pathID(r, "organizationID")
	if err != nil {
		writeProblem(w, http.StatusBadRequest, "invalid_path", err.Error())
		return
	}
	if _, _, _, ok := s.requireOrganizationAccess(w, r, organizationID, access.PermissionReadOrganization); !ok {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"members": s.runtime.Organizations.ListMembers(organizationID)})
}

func (s *Server) inviteMember(w http.ResponseWriter, r *http.Request) {
	organizationID, err := pathID(r, "organizationID")
	if err != nil {
		writeProblem(w, http.StatusBadRequest, "invalid_path", err.Error())
		return
	}
	session, user, _, ok := s.requireOrganizationAccess(w, r, organizationID, access.PermissionInviteMembers)
	if !ok {
		return
	}
	if session.AuthenticationLevel != auth.AAL2 {
		writeProblem(w, http.StatusForbidden, "step_up_required", "MFA step-up is required for member invitations")
		return
	}
	if !s.requireCSRF(w, r) {
		return
	}
	var input memberInvitationRequest
	if err := decodeJSON(w, r, &input); err != nil {
		writeProblem(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	role, err := access.ParseRole(strings.ToLower(input.Role))
	if err != nil {
		writeProblem(w, http.StatusUnprocessableEntity, "role_invalid", err.Error())
		return
	}
	targetUser, err := s.runtime.Auth.FindOrCreateUser(input.Target, input.Channel)
	if err != nil {
		writeProblem(w, http.StatusUnprocessableEntity, "invite_target_invalid", err.Error())
		return
	}
	invitation, membership, err := s.runtime.Organizations.Invite(user.ID, organizationID, input.Target, input.Channel, targetUser.ID, role)
	if err != nil {
		writeProblem(w, http.StatusConflict, "member_invite_failed", err.Error())
		return
	}
	s.runtime.Audit.Append(audit.Event{ActorUserID: user.ID, OrganizationID: organizationID, Action: "organization.member.invited", ResourceType: "membership", ResourceID: membership.ID, Outcome: "success", RequestID: requestIDFromContext(r.Context()), Metadata: map[string]string{"role": string(role), "target_type": input.Channel}})
	writeJSON(w, http.StatusAccepted, map[string]any{"invitation": invitation, "membership": membership})
}

func (s *Server) changeMemberRole(w http.ResponseWriter, r *http.Request) {
	organizationID, err := pathID(r, "organizationID")
	if err != nil {
		writeProblem(w, http.StatusBadRequest, "invalid_path", err.Error())
		return
	}
	targetUserID, err := pathID(r, "userID")
	if err != nil {
		writeProblem(w, http.StatusBadRequest, "invalid_path", err.Error())
		return
	}
	session, user, _, ok := s.requireOrganizationAccess(w, r, organizationID, access.PermissionManageMembers)
	if !ok {
		return
	}
	if session.AuthenticationLevel != auth.AAL2 {
		writeProblem(w, http.StatusForbidden, "step_up_required", "MFA step-up is required for role changes")
		return
	}
	if !s.requireCSRF(w, r) {
		return
	}
	var input memberRoleRequest
	if err := decodeJSON(w, r, &input); err != nil {
		writeProblem(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	var membership organizations.Membership
	var action string
	var metadata map[string]string
	if input.Status != "" {
		membership, err = s.runtime.Organizations.ChangeStatus(organizationID, user.ID, targetUserID, strings.ToLower(input.Status))
		action = "organization.member." + strings.ToLower(input.Status)
		metadata = map[string]string{"status": strings.ToLower(input.Status)}
	} else {
		var role access.Role
		role, err = access.ParseRole(strings.ToLower(input.Role))
		if err == nil {
			membership, err = s.runtime.Organizations.ChangeRole(organizationID, user.ID, targetUserID, role)
		}
		action = "organization.member.role_changed"
		metadata = map[string]string{"role": string(role)}
	}
	if err != nil {
		writeProblem(w, http.StatusConflict, "member_access_change_failed", err.Error())
		return
	}
	s.runtime.Audit.Append(audit.Event{ActorUserID: user.ID, OrganizationID: organizationID, Action: action, ResourceType: "membership", ResourceID: membership.ID, Outcome: "success", RequestID: requestIDFromContext(r.Context()), Severity: "notice", Metadata: metadata})
	writeJSON(w, http.StatusOK, map[string]any{"membership": membership})
}

func (s *Server) listAuditEvents(w http.ResponseWriter, r *http.Request) {
	organizationID, err := pathID(r, "organizationID")
	if err != nil {
		writeProblem(w, http.StatusBadRequest, "invalid_path", err.Error())
		return
	}
	if _, _, _, ok := s.requireOrganizationAccess(w, r, organizationID, access.PermissionReadAudit); !ok {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"events": s.runtime.Audit.ListForOrganization(organizationID)})
}

func (s *Server) requireOrganizationAccess(w http.ResponseWriter, r *http.Request, organizationID string, permission access.Permission) (auth.Session, auth.User, organizations.Membership, bool) {
	session, user, ok := s.requireAuth(w, r)
	if !ok {
		return auth.Session{}, auth.User{}, organizations.Membership{}, false
	}
	membership, ok := s.runtime.Organizations.Membership(organizationID, user.ID)
	if !ok || membership.Status != "active" || !access.Can(membership.Role, permission) {
		s.runtime.Audit.Append(audit.Event{ActorUserID: user.ID, OrganizationID: organizationID, Action: "authorization.organization_denied", ResourceType: "organization", ResourceID: organizationID, Outcome: "denied", Severity: "warning", RequestID: requestIDFromContext(r.Context()), Metadata: map[string]string{"permission": string(permission)}})
		writeProblem(w, http.StatusForbidden, "organization_forbidden", "you do not have access to this organization")
		return auth.Session{}, auth.User{}, organizations.Membership{}, false
	}
	if organization, found := s.runtime.Organizations.Get(organizationID); found && organization.Status == "suspended" && access.RequiresStepUp(permission) {
		writeProblem(w, http.StatusLocked, "organization_suspended", "sensitive changes are blocked while this organization is suspended")
		return auth.Session{}, auth.User{}, organizations.Membership{}, false
	}
	if access.RequiresStepUp(permission) && s.runtime.PlatformOps != nil {
		scope := "credit"
		if permission == access.PermissionReleaseGoods {
			scope = "release"
		}
		if permission == access.PermissionManageFinancial {
			scope = "settlement"
		}
		blocked, err := s.runtime.PlatformOps.ActiveHold(r.Context(), "supplier", organizationID, scope)
		if err != nil {
			writeProblem(w, http.StatusServiceUnavailable, "risk_hold_unavailable", "risk hold status could not be verified")
			return auth.Session{}, auth.User{}, organizations.Membership{}, false
		}
		if blocked {
			writeProblem(w, http.StatusLocked, "risk_hold_active", "this sensitive action is blocked by an active risk hold")
			return auth.Session{}, auth.User{}, organizations.Membership{}, false
		}
	}
	if access.RequiresStepUp(permission) && s.runtime.UserControl.SensitiveActionsBlocked(r.Context(), user.ID) {
		s.runtime.Audit.Append(audit.Event{ActorUserID: user.ID, OrganizationID: organizationID, Action: "authorization.recovery_cooling_off", ResourceType: "organization", ResourceID: organizationID, Outcome: "denied", Severity: "warning", RequestID: requestIDFromContext(r.Context())})
		writeProblem(w, http.StatusLocked, "recovery_cooling_off", "Sensitive financial and administrative changes are blocked during account-recovery cooling-off.")
		return auth.Session{}, auth.User{}, organizations.Membership{}, false
	}
	if access.RequiresStepUp(permission) && (session.AuthenticationLevel != auth.AAL2 || session.MFAVerifiedAt.IsZero() || time.Since(session.MFAVerifiedAt) > 15*time.Minute) {
		s.runtime.Audit.Append(audit.Event{ActorUserID: user.ID, OrganizationID: organizationID, Action: "authorization.step_up_required", ResourceType: "organization", ResourceID: organizationID, Outcome: "denied", Severity: "warning", RequestID: requestIDFromContext(r.Context()), Metadata: map[string]string{"permission": string(permission)}})
		writeProblem(w, http.StatusForbidden, "step_up_required", "step-up authentication is required")
		return auth.Session{}, auth.User{}, organizations.Membership{}, false
	}
	return session, user, membership, true
}
