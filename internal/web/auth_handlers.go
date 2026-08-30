package web

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"time"

	"kredit/internal/audit"
	"kredit/internal/auth"
)

const (
	sessionCookieName = "kredit_session"
	csrfCookieName    = "kredit_csrf"
)

type otpRequest struct {
	Identifier string `json:"identifier"`
	Channel    string `json:"channel"`
	Purpose    string `json:"purpose"`
}

type otpVerifyRequest struct {
	ChallengeID string `json:"challenge_id"`
	Code        string `json:"code"`
	DeviceLabel string `json:"device_label"`
}

type totpVerifyRequest struct {
	Code string `json:"code"`
}

func (s *Server) requestOTP(w http.ResponseWriter, r *http.Request) {
	var input otpRequest
	if err := decodeJSON(w, r, &input); err != nil {
		writeProblem(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	challenge, code, err := s.runtime.Auth.RequestOTP(input.Identifier, input.Channel, input.Purpose)
	if err != nil {
		writeProblem(w, http.StatusTooManyRequests, "otp_unavailable", err.Error())
		return
	}
	if err := s.runtime.Notifications.SendOTP(r.Context(), input.Identifier, input.Channel, code); err != nil {
		writeProblem(w, http.StatusServiceUnavailable, "otp_delivery_unavailable", "verification code delivery is unavailable")
		return
	}
	s.runtime.Audit.Append(audit.Event{Action: "auth.otp.requested", ResourceType: "otp_challenge", ResourceID: challenge.ID, Outcome: "success", RequestID: requestIDFromContext(r.Context()), Metadata: map[string]string{"channel": challenge.TargetType, "purpose": challenge.Purpose}})
	response := map[string]any{
		"challenge_id": challenge.ID,
		"expires_at":   challenge.ExpiresAt,
		"channel":      challenge.TargetType,
		"message":      "If the account can receive this code, a verification code has been sent.",
	}
	if s.config.Environment == "development" {
		response["development_code"] = code
	}
	writeJSON(w, http.StatusAccepted, response)
}

func (s *Server) verifyOTP(w http.ResponseWriter, r *http.Request) {
	var input otpVerifyRequest
	if err := decodeJSON(w, r, &input); err != nil {
		writeProblem(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	user, session, token, err := s.runtime.Auth.VerifyOTP(input.ChallengeID, input.Code, input.DeviceLabel)
	if err != nil {
		writeProblem(w, http.StatusUnauthorized, "otp_invalid", err.Error())
		return
	}
	activated := s.runtime.Organizations.ActivateInvitations(user.ID)
	s.runtime.UserControl.BindUser(user.ID, user.Email, user.Phone)
	setSessionCookies(w, s.config.Environment != "development", token)
	s.runtime.Audit.Append(audit.Event{ActorUserID: user.ID, Action: "auth.login.succeeded", ResourceType: "session", ResourceID: session.ID, Outcome: "success", RequestID: requestIDFromContext(r.Context()), Metadata: map[string]string{"authentication_level": session.AuthenticationLevel}})
	writeJSON(w, http.StatusOK, map[string]any{"user": user, "session": session, "activated_memberships": activated})
}

func (s *Server) logout(w http.ResponseWriter, r *http.Request) {
	session, user, ok := s.requireAuth(w, r)
	if !ok {
		return
	}
	if !s.requireCSRF(w, r) {
		return
	}
	if err := s.runtime.Auth.RevokeSession(sessionTokenFromRequest(r)); err != nil {
		writeProblem(w, http.StatusUnauthorized, "session_invalid", err.Error())
		return
	}
	clearSessionCookies(w, s.config.Environment != "development")
	s.runtime.Audit.Append(audit.Event{ActorUserID: user.ID, Action: "auth.logout", ResourceType: "session", ResourceID: session.ID, Outcome: "success", RequestID: requestIDFromContext(r.Context())})
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) me(w http.ResponseWriter, r *http.Request) {
	session, user, ok := s.requireAuth(w, r)
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"user": user, "session": session, "mfa_enrolled": s.runtime.Auth.IsMFAEnrolled(user.ID), "organizations": s.runtime.Organizations.ListForUser(user.ID)})
}

func (s *Server) enrollTOTP(w http.ResponseWriter, r *http.Request) {
	_, user, ok := s.requireAuth(w, r)
	if !ok {
		return
	}
	if !s.requireCSRF(w, r) {
		return
	}
	method, err := s.runtime.Auth.BeginTOTPEnrollment(user.ID)
	if err != nil {
		writeProblem(w, http.StatusConflict, "mfa_enrollment_failed", err.Error())
		return
	}
	label := user.Email
	if label == "" {
		label = user.Phone
	}
	issuer := "Kredit"
	uri := "otpauth://totp/" + issuer + ":" + label + "?secret=" + method.Secret + "&issuer=" + issuer + "&algorithm=SHA1&digits=6&period=30"
	s.runtime.Audit.Append(audit.Event{ActorUserID: user.ID, Action: "auth.mfa.enrollment.started", ResourceType: "mfa_method", ResourceID: method.ID, Outcome: "success", RequestID: requestIDFromContext(r.Context()), Metadata: map[string]string{"type": method.Type}})
	writeJSON(w, http.StatusOK, map[string]any{"method_id": method.ID, "type": method.Type, "secret": method.Secret, "otpauth_uri": uri, "warning": "Store this secret securely. It is shown only during enrollment."})
}

func (s *Server) verifyTOTP(w http.ResponseWriter, r *http.Request) {
	session, user, ok := s.requireAuth(w, r)
	if !ok {
		return
	}
	if !s.requireCSRF(w, r) {
		return
	}
	var input totpVerifyRequest
	if err := decodeJSON(w, r, &input); err != nil {
		writeProblem(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	wasEnrolled := s.runtime.Auth.IsMFAEnrolled(user.ID)
	if err := s.runtime.Auth.VerifyTOTP(user.ID, input.Code); err != nil {
		writeProblem(w, http.StatusUnauthorized, "mfa_invalid", err.Error())
		return
	}
	if err := s.runtime.Auth.ElevateSession(sessionTokenFromRequest(r)); err != nil {
		writeProblem(w, http.StatusConflict, "mfa_step_up_failed", err.Error())
		return
	}
	session.AuthenticationLevel = auth.AAL2
	var recoveryCodes []string
	if !wasEnrolled {
		var err error
		recoveryCodes, err = s.runtime.UserControl.GenerateRecoveryCodes(r.Context(), user.ID)
		if err != nil {
			writeProblem(w, http.StatusServiceUnavailable, "recovery_codes_unavailable", "MFA was enabled but recovery codes could not be issued")
			return
		}
	}
	s.runtime.Audit.Append(audit.Event{ActorUserID: user.ID, Action: "auth.mfa.verified", ResourceType: "session", ResourceID: session.ID, Outcome: "success", RequestID: requestIDFromContext(r.Context()), Severity: "notice"})
	writeJSON(w, http.StatusOK, map[string]any{"session": session, "authentication_level": auth.AAL2, "recovery_codes": recoveryCodes, "recovery_codes_warning": "Store these one-time codes securely. They are shown only once."})
}

func (s *Server) requireAuth(w http.ResponseWriter, r *http.Request) (auth.Session, auth.User, bool) {
	token := sessionTokenFromRequest(r)
	if token == "" {
		s.recordSecurityEvent(r, "auth.authentication_required", "authentication", "denied", "warning")
		writeProblem(w, http.StatusUnauthorized, "authentication_required", "authentication is required")
		return auth.Session{}, auth.User{}, false
	}
	session, user, err := s.runtime.Auth.SessionFromToken(token)
	if err != nil {
		s.recordSecurityEvent(r, "auth.session_invalid", "session", "denied", "warning")
		writeProblem(w, http.StatusUnauthorized, "session_invalid", "session is invalid or expired")
		return auth.Session{}, auth.User{}, false
	}
	return session, user, true
}

func (s *Server) requireCSRF(w http.ResponseWriter, r *http.Request) bool {
	cookie, err := r.Cookie(csrfCookieName)
	if err != nil || cookie.Value == "" || cookie.Value != r.Header.Get("X-CSRF-Token") {
		s.recordSecurityEvent(r, "auth.csrf_failed", "csrf", "denied", "warning")
		writeProblem(w, http.StatusForbidden, "csrf_failed", "csrf token is required")
		return false
	}
	return true
}

func (s *Server) recordSecurityEvent(r *http.Request, action, resourceType, outcome, severity string) {
	if s == nil || s.runtime == nil || s.runtime.Audit == nil {
		return
	}
	s.runtime.Audit.Append(audit.Event{Action: action, ResourceType: resourceType, Outcome: outcome, Severity: severity, RequestID: requestIDFromContext(r.Context())})
}

func sessionTokenFromRequest(r *http.Request) string {
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil {
		return ""
	}
	return cookie.Value
}

func setSessionCookies(w http.ResponseWriter, secure bool, token string) {
	csrf := opaqueToken()
	maxAge := 30 * 24 * 60 * 60
	http.SetCookie(w, &http.Cookie{Name: sessionCookieName, Value: token, Path: "/", MaxAge: maxAge, HttpOnly: true, Secure: secure, SameSite: http.SameSiteLaxMode})
	http.SetCookie(w, &http.Cookie{Name: csrfCookieName, Value: csrf, Path: "/", MaxAge: maxAge, HttpOnly: false, Secure: secure, SameSite: http.SameSiteLaxMode})
}

func clearSessionCookies(w http.ResponseWriter, secure bool) {
	for _, name := range []string{sessionCookieName, csrfCookieName} {
		http.SetCookie(w, &http.Cookie{Name: name, Value: "", Path: "/", MaxAge: -1, HttpOnly: name == sessionCookieName, Secure: secure, SameSite: http.SameSiteLaxMode})
	}
}

func opaqueToken() string {
	value := make([]byte, 32)
	if _, err := rand.Read(value); err != nil {
		return hex.EncodeToString([]byte(time.Now().String()))
	}
	return hex.EncodeToString(value)
}
