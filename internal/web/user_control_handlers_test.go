package web

import (
	"io"
	"log/slog"
	"net/http"
	"testing"
	"time"

	"kredit/internal/auth"
	"kredit/internal/config"
)

func TestNotificationPrivacyAndRecoverySelfService(t *testing.T) {
	t.Setenv("APP_ENV", "development")
	cfg, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	server := NewServer(cfg, slog.New(slog.NewTextHandler(io.Discard, nil)))
	client := newTestClient(server.Handler())
	challengeResponse := doJSON(t, client, "/api/v1/auth/otp/challenges", http.MethodPost, map[string]string{"identifier": "wave3-owner@example.test", "channel": "email", "purpose": "login"}, nil, http.StatusAccepted)
	var login struct {
		ChallengeID string `json:"challenge_id"`
		Code        string `json:"development_code"`
	}
	decodeResponse(t, challengeResponse, &login)
	verify := doJSON(t, client, "/api/v1/auth/otp/verify", http.MethodPost, map[string]string{"challenge_id": login.ChallengeID, "code": login.Code}, nil, http.StatusOK)
	_ = verify.Body.Close()
	csrf := ""
	for _, c := range client.cookies {
		if c.Name == csrfCookieName {
			csrf = c.Value
		}
	}
	enroll := doJSON(t, client, "/api/v1/mfa/totp/enroll", http.MethodPost, map[string]string{}, map[string]string{"X-CSRF-Token": csrf}, http.StatusOK)
	var mfa struct {
		Secret string `json:"secret"`
	}
	decodeResponse(t, enroll, &mfa)
	mfaResponse := doJSON(t, client, "/api/v1/mfa/totp/verify", http.MethodPost, map[string]string{"code": auth.TOTPCode(mfa.Secret, time.Now().UTC())}, map[string]string{"X-CSRF-Token": csrf}, http.StatusOK)
	var mfaResult struct {
		RecoveryCodes []string `json:"recovery_codes"`
	}
	decodeResponse(t, mfaResponse, &mfaResult)
	if len(mfaResult.RecoveryCodes) != 10 {
		t.Fatalf("recovery codes=%d", len(mfaResult.RecoveryCodes))
	}
	prefs := doJSON(t, client, "/api/v1/me/notification-preferences", http.MethodGet, nil, nil, http.StatusOK)
	var current struct {
		Preferences struct {
			Version int64 `json:"version"`
		} `json:"preferences"`
	}
	decodeResponse(t, prefs, &current)
	updated := doJSON(t, client, "/api/v1/me/notification-preferences", http.MethodPut, map[string]any{"preferred_channel": "email", "fallback_channel": "sms", "payment_reminders_enabled": false, "product_updates_enabled": false, "quiet_start_hour": 21, "quiet_end_hour": 7, "timezone": "Africa/Lagos", "expected_version": current.Preferences.Version}, map[string]string{"X-CSRF-Token": csrf, "Idempotency-Key": "wave3-prefs"}, http.StatusOK)
	_ = updated.Body.Close()
	requiredDenied := doJSON(t, client, "/api/v1/me/notification-preferences", http.MethodPut, map[string]any{"disable_required": true}, map[string]string{"X-CSRF-Token": csrf, "Idempotency-Key": "wave3-required"}, http.StatusConflict)
	_ = requiredDenied.Body.Close()
	privacy := doJSON(t, client, "/api/v1/me/privacy-requests", http.MethodPost, map[string]string{"request_type": "PORTABILITY", "details": "Provide my portable account data"}, map[string]string{"X-CSRF-Token": csrf, "Idempotency-Key": "wave3-privacy"}, http.StatusCreated)
	_ = privacy.Body.Close()
	listed := doJSON(t, client, "/api/v1/me/privacy-requests", http.MethodGet, nil, nil, http.StatusOK)
	var privacyList struct {
		Requests []any `json:"requests"`
	}
	decodeResponse(t, listed, &privacyList)
	if len(privacyList.Requests) != 1 {
		t.Fatalf("privacy requests=%d", len(privacyList.Requests))
	}
	start := doJSON(t, client, "/api/v1/account-recovery/requests", http.MethodPost, map[string]string{"identifier": "wave3-owner@example.test", "channel": "email"}, map[string]string{"Idempotency-Key": "wave3-recovery"}, http.StatusAccepted)
	var recovery struct {
		RequestID string `json:"development_request_id"`
	}
	decodeResponse(t, start, &recovery)
	if recovery.RequestID == "" {
		t.Fatal("development recovery id missing")
	}
	codeEvidence := doJSON(t, client, "/api/v1/account-recovery/requests/"+recovery.RequestID+"/evidence", http.MethodPost, map[string]string{"factor_type": "recovery_code", "proof": mfaResult.RecoveryCodes[0]}, map[string]string{"Idempotency-Key": "wave3-code"}, http.StatusOK)
	_ = codeEvidence.Body.Close()
	contactStart := doJSON(t, client, "/api/v1/auth/otp/challenges", http.MethodPost, map[string]string{"identifier": "wave3-owner@example.test", "channel": "email", "purpose": "recovery"}, nil, http.StatusAccepted)
	var contact struct {
		ChallengeID string `json:"challenge_id"`
		Code        string `json:"development_code"`
	}
	decodeResponse(t, contactStart, &contact)
	contactEvidence := doJSON(t, client, "/api/v1/account-recovery/requests/"+recovery.RequestID+"/evidence", http.MethodPost, map[string]string{"factor_type": "verified_email", "challenge_id": contact.ChallengeID, "code": contact.Code, "channel": "email", "identifier": "wave3-owner@example.test"}, map[string]string{"Idempotency-Key": "wave3-contact"}, http.StatusOK)
	var evidence struct {
		Request struct {
			State string `json:"state"`
		} `json:"request"`
	}
	decodeResponse(t, contactEvidence, &evidence)
	if evidence.Request.State != "PENDING_REVIEW" {
		t.Fatalf("state=%s", evidence.Request.State)
	}
	unknown := doJSON(t, client, "/api/v1/account-recovery/requests", http.MethodPost, map[string]string{"identifier": "absent@example.test", "channel": "email"}, map[string]string{"Idempotency-Key": "wave3-unknown"}, http.StatusAccepted)
	var unknownBody map[string]any
	decodeResponse(t, unknown, &unknownBody)
	if _, ok := unknownBody["development_request_id"]; ok {
		t.Fatal("unknown account was disclosed")
	}
}
