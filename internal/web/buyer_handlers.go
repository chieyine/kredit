package web

import (
	"context"
	"net/http"
	"strings"

	"kredit/internal/access"
	"kredit/internal/audit"
	"kredit/internal/buyers"
)

type buyerInvitationRequest struct {
	Target          string `json:"target"`
	TargetType      string `json:"target_type"`
	LegalName       string `json:"legal_name"`
	TradingName     string `json:"trading_name"`
	BusinessType    string `json:"business_type"`
	BusinessAddress string `json:"business_address"`
	Industry        string `json:"industry"`
}

type buyerInvitationAcceptRequest struct {
	ChallengeID     string `json:"challenge_id"`
	Code            string `json:"code"`
	DeviceLabel     string `json:"device_label"`
	FullName        string `json:"full_name"`
	LegalName       string `json:"legal_name"`
	TradingName     string `json:"trading_name"`
	BusinessType    string `json:"business_type"`
	BusinessAddress string `json:"business_address"`
	Industry        string `json:"industry"`
}

func (s *Server) createBuyerInvitation(w http.ResponseWriter, r *http.Request) {
	organizationID, err := pathID(r, "organizationID")
	if err != nil {
		writeProblem(w, http.StatusBadRequest, "invalid_path", err.Error())
		return
	}
	_, user, _, ok := s.requireOrganizationAccess(w, r, organizationID, access.PermissionInviteBuyers)
	if !ok {
		return
	}
	if !s.requireCSRF(w, r) {
		return
	}
	var input buyerInvitationRequest
	if err := decodeJSON(w, r, &input); err != nil {
		writeProblem(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	result, err := s.runtime.Buyers.CreateInvitation(user.ID, organizationID, buyers.CreateInvitationInput{Target: input.Target, TargetType: strings.ToLower(input.TargetType), LegalName: input.LegalName, TradingName: input.TradingName, BusinessType: input.BusinessType, BusinessAddress: input.BusinessAddress, Industry: input.Industry})
	if err != nil {
		writeProblem(w, http.StatusUnprocessableEntity, "buyer_invitation_invalid", err.Error())
		return
	}
	s.runtime.Audit.Append(audit.Event{ActorUserID: user.ID, OrganizationID: organizationID, Action: "buyer.invitation.created", ResourceType: "buyer_invitation", ResourceID: result.Invitation.ID, Outcome: "success", RequestID: requestIDFromContext(r.Context()), Metadata: map[string]string{"target_type": result.Invitation.TargetType}})
	invitationURL := strings.TrimRight(s.config.PublicBaseURL, "/") + "/buyer-invitations/" + result.RawToken
	deliveryState := "sent"
	if err := s.runtime.Notifications.SendInvitation(r.Context(), input.Target, strings.ToLower(input.TargetType), invitationURL); err != nil {
		deliveryState = "manual_handoff_required"
	}
	writeJSON(w, http.StatusAccepted, map[string]any{"invitation": result.Invitation, "invitation_url": invitationURL, "delivery_state": deliveryState})
}

func (s *Server) previewBuyerInvitation(w http.ResponseWriter, r *http.Request) {
	token := r.PathValue("token")
	preview, err := s.runtime.Buyers.Preview(token)
	if err != nil {
		writeProblem(w, http.StatusNotFound, "buyer_invitation_invalid", "buyer invitation is invalid or expired")
		return
	}
	organization, ok := s.runtime.Organizations.Get(preview.OrganizationID)
	if !ok {
		writeProblem(w, http.StatusNotFound, "organization_not_found", "supplier organization was not found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"invitation": preview, "supplier": map[string]string{"legal_name": organization.LegalName, "trading_name": organization.TradingName, "industry": organization.Industry}})
}

func (s *Server) requestBuyerInvitationOTP(w http.ResponseWriter, r *http.Request) {
	token := r.PathValue("token")
	targetType, target, err := s.runtime.Buyers.InvitationTarget(token)
	if err != nil {
		writeProblem(w, http.StatusNotFound, "buyer_invitation_invalid", "buyer invitation is invalid or expired")
		return
	}
	challenge, code, err := s.runtime.Auth.RequestOTP(target, targetType, "buyer_invitation")
	if err != nil {
		writeProblem(w, http.StatusTooManyRequests, "otp_unavailable", err.Error())
		return
	}
	if err := s.runtime.Notifications.SendOTP(r.Context(), target, targetType, code); err != nil {
		writeProblem(w, http.StatusServiceUnavailable, "otp_delivery_unavailable", "verification code delivery is unavailable")
		return
	}
	s.runtime.Audit.Append(audit.Event{Action: "buyer.invitation.otp_requested", ResourceType: "otp_challenge", ResourceID: challenge.ID, Outcome: "success", RequestID: requestIDFromContext(r.Context()), Metadata: map[string]string{"target_type": targetType, "purpose": "buyer_invitation"}})
	response := map[string]any{"challenge_id": challenge.ID, "expires_at": challenge.ExpiresAt, "target_type": targetType, "message": "If the invitation contact can receive this code, a verification code has been sent."}
	if s.config.Environment == "development" {
		response["development_code"] = code
	}
	writeJSON(w, http.StatusAccepted, response)
}

func (s *Server) acceptBuyerInvitation(w http.ResponseWriter, r *http.Request) {
	token := r.PathValue("token")
	targetType, target, err := s.runtime.Buyers.InvitationTarget(token)
	if err != nil {
		writeProblem(w, http.StatusNotFound, "buyer_invitation_invalid", "buyer invitation is invalid or expired")
		return
	}
	var input buyerInvitationAcceptRequest
	if err := decodeJSON(w, r, &input); err != nil {
		writeProblem(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	user, session, rawSessionToken, err := s.runtime.Auth.VerifyOTPForTarget(input.ChallengeID, input.Code, input.DeviceLabel, targetType, target)
	if err != nil {
		writeProblem(w, http.StatusUnauthorized, "otp_invalid", err.Error())
		return
	}
	portal, err := s.runtime.Buyers.Accept(context.Background(), token, user.ID, buyers.AcceptInput{FullName: input.FullName, LegalName: input.LegalName, TradingName: input.TradingName, BusinessType: input.BusinessType, BusinessAddress: input.BusinessAddress, Industry: input.Industry})
	if err != nil {
		writeProblem(w, http.StatusUnprocessableEntity, "buyer_onboarding_failed", err.Error())
		return
	}
	setSessionCookies(w, s.config.Environment != "development", rawSessionToken)
	s.runtime.Audit.Append(audit.Event{ActorUserID: user.ID, Action: "buyer.invitation.accepted", ResourceType: "buyer_invitation", ResourceID: portal.Business.ID, Outcome: "success", RequestID: requestIDFromContext(r.Context()), Metadata: map[string]string{"authentication_level": session.AuthenticationLevel}})
	writeJSON(w, http.StatusCreated, map[string]any{"user": user, "session": session, "portal": portal})
}

func (s *Server) buyerPortal(w http.ResponseWriter, r *http.Request) {
	_, user, ok := s.requireAuth(w, r)
	if !ok {
		return
	}
	portal, err := s.runtime.Buyers.Portal(user.ID)
	if err != nil {
		writeProblem(w, http.StatusNotFound, "buyer_profile_not_found", "buyer portal profile was not found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"portal": portal})
}
