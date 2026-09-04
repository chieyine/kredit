package web

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"net/http"
	"net/url"
	"strings"

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
	if !setSessionCookies(w, s.config.Environment != "development", token) {
		writeProblem(w, http.StatusServiceUnavailable, "session_unavailable", "a secure session could not be established")
		return
	}
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
	rotatedSession, rotatedToken, err := s.runtime.Auth.StepUpSession(sessionTokenFromRequest(r), input.Code)
	if err != nil {
		if errors.Is(err, auth.ErrMFALocked) {
			s.recordSecurityEvent(r, "auth.mfa.locked", "mfa_method", "denied", "warning")
			w.Header().Set("Retry-After", "900")
			writeProblem(w, http.StatusTooManyRequests, "mfa_locked", "too many incorrect verification codes; try again shortly or use account recovery")
			return
		}
		writeProblem(w, http.StatusUnauthorized, "mfa_invalid", err.Error())
		return
	}
	if !setSessionCookies(w, s.config.Environment != "development", rotatedToken) {
		writeProblem(w, http.StatusServiceUnavailable, "session_unavailable", "a secure session could not be established")
		return
	}
	session = rotatedSession
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

// requireCSRF enforces the two controls README section 21.6 requires for
// cookie-authenticated writes: a double-submit token compared in constant time,
// and a same-origin check on Sec-Fetch-Site/Origin. Either control alone is
// weaker than the pair.
func (s *Server) requireCSRF(w http.ResponseWriter, r *http.Request) bool {
	if !s.sameOriginRequest(r) {
		s.recordSecurityEvent(r, "auth.csrf_cross_origin", "csrf", "denied", "warning")
		writeProblem(w, http.StatusForbidden, "csrf_failed", "cross-origin state change is not permitted")
		return false
	}
	cookie, err := r.Cookie(csrfCookieName)
	submitted := r.Header.Get("X-CSRF-Token")
	if err != nil || cookie.Value == "" || submitted == "" || subtle.ConstantTimeCompare([]byte(cookie.Value), []byte(submitted)) != 1 {
		s.recordSecurityEvent(r, "auth.csrf_failed", "csrf", "denied", "warning")
		writeProblem(w, http.StatusForbidden, "csrf_failed", "csrf token is required")
		return false
	}
	return true
}

// sameOriginRequest rejects a browser write that a different site initiated.
// Sec-Fetch-Site is authoritative where the browser sends it; otherwise the
// Origin header is matched against the configured public origins. A request
// with neither header did not come from a modern browser form or fetch, so it
// still has to satisfy the double-submit token above.
func (s *Server) sameOriginRequest(r *http.Request) bool {
	switch strings.ToLower(strings.TrimSpace(r.Header.Get("Sec-Fetch-Site"))) {
	case "same-origin", "none":
		return true
	case "same-site", "cross-site":
		return false
	}
	origin := strings.TrimSpace(r.Header.Get("Origin"))
	if origin == "" {
		// Neither header is present. Some proxies strip Sec-Fetch-* and Origin on
		// the internal hop, so this is not evidence of a cross-site request; the
		// double-submit token below remains the control.
		return true
	}
	if origin == "null" {
		return false
	}
	requested, err := url.Parse(origin)
	if err != nil || requested.Host == "" {
		return false
	}
	for _, allowed := range []string{s.config.PublicBaseURL, s.config.AppBaseURL} {
		if allowed == "" {
			continue
		}
		permitted, err := url.Parse(allowed)
		if err != nil || permitted.Host == "" {
			continue
		}
		if strings.EqualFold(permitted.Scheme, requested.Scheme) && strings.EqualFold(permitted.Host, requested.Host) {
			return true
		}
	}
	return false
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

func setSessionCookies(w http.ResponseWriter, secure bool, token string) bool {
	csrf := opaqueToken()
	if csrf == "" {
		return false
	}
	maxAge := 30 * 24 * 60 * 60
	http.SetCookie(w, &http.Cookie{Name: sessionCookieName, Value: token, Path: "/", MaxAge: maxAge, HttpOnly: true, Secure: secure, SameSite: http.SameSiteLaxMode})
	http.SetCookie(w, &http.Cookie{Name: csrfCookieName, Value: csrf, Path: "/", MaxAge: maxAge, HttpOnly: false, Secure: secure, SameSite: http.SameSiteLaxMode})
	return true
}

func clearSessionCookies(w http.ResponseWriter, secure bool) {
	for _, name := range []string{sessionCookieName, csrfCookieName} {
		http.SetCookie(w, &http.Cookie{Name: name, Value: "", Path: "/", MaxAge: -1, HttpOnly: name == sessionCookieName, Secure: secure, SameSite: http.SameSiteLaxMode})
	}
}

// opaqueToken returns 256 bits of cryptographic randomness. A clock-derived
// fallback would be guessable, so a randomness failure returns an empty token
// and the caller fails closed rather than issuing a predictable CSRF secret.
func opaqueToken() string {
	value := make([]byte, 32)
	if _, err := rand.Read(value); err != nil {
		return ""
	}
	return hex.EncodeToString(value)
}
