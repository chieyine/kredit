package web

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"kredit/internal/auth"
	"kredit/internal/businesspolicy"
	"kredit/internal/config"
	"kredit/internal/db"
	"kredit/internal/platformsettings"

	"github.com/google/uuid"
)

func TestPlatformSettingsEndpoints(t *testing.T) {
	url := os.Getenv("DATABASE_URL")
	if url == "" || os.Getenv("KREDIT_INTEGRATION") != "1" {
		t.Skip("integration database required")
	}
	ctx := context.Background()
	database, err := db.Open(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()

	cfg := config.Config{
		Environment:        "development",
		Version:            "test",
		APIListenAddr:      ":0",
		Currency:           "NGN",
		MoneyUnit:          "kobo",
		CollectionProvider: "mock",
		TokenHashKey:       "platform-settings-test-secret",
		PublicBaseURL:      "http://localhost",
	}
	runtime := NewRuntimeWithDB(cfg, database)
	server := NewServerWithRuntime(cfg, slog.Default(), runtime)

	// Create user
	identifier := fmt.Sprintf("owner-test-%d@example.test", time.Now().UnixNano())
	challenge, code, err := runtime.Auth.RequestOTP(identifier, "email", "login")
	if err != nil {
		t.Fatal(err)
	}
	user, _, token, err := runtime.Auth.VerifyOTP(challenge.ID, code, "test")
	if err != nil {
		t.Fatal(err)
	}

	// Step-up to AAL2
	method, err := runtime.Auth.BeginTOTPEnrollment(user.ID)
	if err != nil {
		t.Fatal(err)
	}
	_, aal2Token, err := runtime.Auth.StepUpSession(token, auth.TOTPCode(method.Secret, time.Now().UTC()))
	if err != nil {
		t.Fatal(err)
	}

	defer func() {
		_, _ = database.Raw().Exec(ctx, `DELETE FROM app.audit_events WHERE actor_user_id=$1::uuid`, user.ID)
		_, _ = database.Raw().Exec(ctx, `DELETE FROM app.platform_role_assignments WHERE user_id=$1::uuid`, user.ID)
		_, _ = database.Raw().Exec(ctx, `DELETE FROM app.mfa_methods WHERE user_id=$1::uuid`, user.ID)
		_, _ = database.Raw().Exec(ctx, `DELETE FROM app.sessions WHERE user_id=$1::uuid`, user.ID)
		_, _ = database.Raw().Exec(ctx, `DELETE FROM app.otp_challenges WHERE target_hash IS NOT NULL AND id=$1::uuid`, challenge.ID)
		_, _ = database.Raw().Exec(ctx, `DELETE FROM app.users WHERE id=$1::uuid`, user.ID)
	}()

	// 1. Test public capabilities endpoint
	req := httptest.NewRequest(http.MethodGet, "/api/v1/platform/capabilities", nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("capabilities status=%d body=%s", rec.Code, rec.Body.String())
	}
	var caps map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &caps); err != nil {
		t.Fatal(err)
	}
	if _, ok := caps["launch_mode"]; !ok {
		t.Fatalf("missing launch_mode in caps: %v", caps)
	}
	if _, ok := caps["features"]; !ok {
		t.Fatalf("missing features in caps: %v", caps)
	}

	// 2. Non-owner (no role assigned) cannot access platform settings
	req = httptest.NewRequest(http.MethodGet, "/api/v1/ops/platform-settings", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: aal2Token})
	rec = httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for unassigned user, got %d: %s", rec.Code, rec.Body.String())
	}

	// 3. Regular admin role (e.g. platform_admin) cannot access platform settings
	if _, err := database.Raw().Exec(ctx, `INSERT INTO app.platform_role_assignments(user_id, role, granted_by, reason) VALUES($1::uuid, 'platform_admin', $1::uuid, 'Admin assignment')`, user.ID); err != nil {
		t.Fatal(err)
	}
	req = httptest.NewRequest(http.MethodGet, "/api/v1/ops/platform-settings", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: aal2Token})
	rec = httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for platform_admin, got %d: %s", rec.Code, rec.Body.String())
	}

	// 4. Platform Owner can access platform settings
	if _, err := database.Raw().Exec(ctx, `UPDATE app.platform_role_assignments SET role='platform_owner' WHERE user_id=$1::uuid`, user.ID); err != nil {
		t.Fatal(err)
	}
	req = httptest.NewRequest(http.MethodGet, "/api/v1/ops/platform-settings", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: aal2Token})
	rec = httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for platform_owner, got %d: %s", rec.Code, rec.Body.String())
	}

	var settingsResp struct {
		Settings   []platformsettings.Setting  `json:"settings"`
		Governance platformsettings.Governance `json:"governance"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &settingsResp); err != nil {
		t.Fatal(err)
	}
	if len(settingsResp.Settings) == 0 {
		t.Fatalf("expected seeded settings, got 0")
	}

	// Verify secrets are masked
	foundSecret := false
	for _, s := range settingsResp.Settings {
		if s.IsSecret {
			foundSecret = true
			var valStr string
			_ = json.Unmarshal(s.Value, &valStr)
			if valStr != "" && !strings.Contains(valStr, "•••") {
				t.Fatalf("secret setting %s should have masked value, got %s", s.Key, valStr)
			}
		}
	}
	if !foundSecret {
		t.Fatalf("expected at least one secret setting in seeded settings")
	}

	// 5. Rotate secret test
	rotateBody, _ := json.Marshal(map[string]any{
		"key":    "integrations.paystack.secret_key",
		"secret": "sk_test_abc1234567890xyz",
		"reason": "Rotating test paystack key",
	})
	rotateReq := httptest.NewRequest(http.MethodPost, "/api/v1/ops/platform-settings/secret", bytes.NewReader(rotateBody))
	rotateReq.AddCookie(&http.Cookie{Name: sessionCookieName, Value: aal2Token})
	rotateReq.Header.Set("Origin", "http://localhost")
	rotateReq.Header.Set("Sec-Fetch-Site", "same-origin")
	csrfToken := "test-csrf-token-1234"
	rotateReq.AddCookie(&http.Cookie{Name: csrfCookieName, Value: csrfToken})
	rotateReq.Header.Set("X-CSRF-Token", csrfToken)

	rotateRec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rotateRec, rotateReq)
	if rotateRec.Code != http.StatusOK {
		t.Fatalf("secret rotation failed: status=%d body=%s", rotateRec.Code, rotateRec.Body.String())
	}

	var rotateResp struct {
		Setting platformsettings.Setting `json:"setting"`
	}
	if err := json.Unmarshal(rotateRec.Body.Bytes(), &rotateResp); err != nil {
		t.Fatal(err)
	}
	var rotVal string
	_ = json.Unmarshal(rotateResp.Setting.Value, &rotVal)
	if !strings.HasSuffix(rotVal, "0xyz") || !strings.Contains(rotVal, "•••") {
		t.Fatalf("expected secret mask ending in 0xyz, got %s", rotVal)
	}
	if rotateResp.Setting.SecretFingerprint == "" {
		t.Fatalf("expected secret fingerprint to be populated")
	}

	// 6. Test History
	histReq := httptest.NewRequest(http.MethodGet, "/api/v1/ops/platform-settings/history?key=integrations.paystack.secret_key", nil)
	histReq.AddCookie(&http.Cookie{Name: sessionCookieName, Value: aal2Token})
	histRec := httptest.NewRecorder()
	server.Handler().ServeHTTP(histRec, histReq)
	if histRec.Code != http.StatusOK {
		t.Fatalf("history fetch failed: status=%d body=%s", histRec.Code, histRec.Body.String())
	}
	var histResp struct {
		History []platformsettings.SettingHistory `json:"history"`
	}
	if err := json.Unmarshal(histRec.Body.Bytes(), &histResp); err != nil {
		t.Fatal(err)
	}
	if len(histResp.History) == 0 {
		t.Fatalf("expected history entries after secret rotation")
	}

	// 7. Test Governance API
	govReq := httptest.NewRequest(http.MethodGet, "/api/v1/ops/governance", nil)
	govReq.AddCookie(&http.Cookie{Name: sessionCookieName, Value: aal2Token})
	govRec := httptest.NewRecorder()
	server.Handler().ServeHTTP(govRec, govReq)
	if govRec.Code != http.StatusOK {
		t.Fatalf("governance GET failed: status=%d body=%s", govRec.Code, govRec.Body.String())
	}

	// 8. Test Solo-Owner Self Approval
	policy, err := runtime.BusinessPolicies.Read(ctx)
	if err != nil {
		t.Fatal(err)
	}
	newValues := policy.Values
	newValues.UpcomingNoticeDays = 4
	proposalUUID := uuid.NewString()
	_, _ = database.Raw().Exec(ctx, `UPDATE app.business_policy_changes SET state='cancelled' WHERE state='pending' OR (state='approved' AND effective_at>now())`)
	defer func() {
		_, _ = database.Raw().Exec(ctx, `UPDATE app.business_policy_changes SET state='cancelled' WHERE id=$1::uuid`, proposalUUID)
	}()
	policyChangeID, err := runtime.BusinessPolicies.Propose(ctx, user.ID, businesspolicy.Proposal{
		ID:           proposalUUID,
		BaseRevision: policy.Revision,
		Values:       newValues,
		Reason:       "Test policy change for solo approval",
		EffectiveAt:  time.Now().UTC().Add(24 * time.Hour),
	})
	if err != nil {
		t.Fatal(err)
	}

	approveBody, _ := json.Marshal(map[string]any{
		"target_type": "policy",
		"target_id":   policyChangeID,
		"reason":      "Solo-owner self-approval automated test",
		"confirm":     true,
	})
	approveReq := httptest.NewRequest(http.MethodPost, "/api/v1/ops/solo-owner/approve", bytes.NewReader(approveBody))
	approveReq.AddCookie(&http.Cookie{Name: sessionCookieName, Value: aal2Token})
	approveReq.Header.Set("Origin", "http://localhost")
	approveReq.Header.Set("Sec-Fetch-Site", "same-origin")
	approveReq.AddCookie(&http.Cookie{Name: csrfCookieName, Value: csrfToken})
	approveReq.Header.Set("X-CSRF-Token", csrfToken)

	approveRec := httptest.NewRecorder()
	server.Handler().ServeHTTP(approveRec, approveReq)
	if approveRec.Code != http.StatusOK {
		t.Fatalf("solo-owner approval failed: status=%d body=%s", approveRec.Code, approveRec.Body.String())
	}
}
