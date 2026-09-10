package web

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
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
	t.Setenv("SETTINGS_ENCRYPTION_KEY", "isolated-test-settings-encryption-root-32-bytes")
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
	if _, ok := caps["launch_mode"]; ok {
		t.Fatalf("private launch_mode leaked in caps: %v", caps)
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

	// The console includes consumed connectors before their first save, but
	// never exposes plaintext credentials.
	if len(settingsResp.Settings) != (len(platformsettings.KnownSettings) - len(platformsettings.WebsitePages)) {
		t.Fatalf("expected %d registered settings, got %d", (len(platformsettings.KnownSettings) - len(platformsettings.WebsitePages)), len(settingsResp.Settings))
	}
	for _, item := range settingsResp.Settings {
		meta, known := platformsettings.KnownSettings[item.Key]
		if !known || item.IsSecret != meta.IsSecret {
			t.Fatalf("unexpected setting metadata: %s", item.Key)
		}
		if strings.HasPrefix(item.Key, "integrations.notifications.") && !item.IsSecret {
			t.Fatal("connector not marked secret")
		}
		if item.Description == "" {
			t.Fatalf("%s has no description", item.Key)
		}
	}

	originalSettings := runtime.PlatformSettings
	runtime.PlatformSettings = unavailableGovernance{Service: originalSettings}
	unavailable := httptest.NewRecorder()
	server.Handler().ServeHTTP(unavailable, req.Clone(ctx))
	if unavailable.Code != http.StatusServiceUnavailable {
		t.Fatalf("missing governance must fail closed: %d %s", unavailable.Code, unavailable.Body.String())
	}
	runtime.PlatformSettings = originalSettings

	t.Run("website publication keeps drafts private", func(t *testing.T) { verifyWebsitePublication(t, server, aal2Token, token) })

	// 5. A retired key cannot be written back in through the update endpoint.
	retiredBody, _ := json.Marshal(map[string]any{
		"key":              "integrations.paystack.secret_key",
		"value":            json.RawMessage(`"sk_test_abc1234567890xyz"`),
		"reason":           "Attempting to restore a retired provider key",
		"expected_version": 0,
	})
	retiredReq := httptest.NewRequest(http.MethodPost, "/api/v1/ops/platform-settings", bytes.NewReader(retiredBody))
	retiredReq.AddCookie(&http.Cookie{Name: sessionCookieName, Value: aal2Token})
	retiredReq.Header.Set("Origin", "http://localhost")
	retiredReq.Header.Set("Sec-Fetch-Site", "same-origin")
	csrfToken := "test-csrf-token-1234"
	retiredReq.AddCookie(&http.Cookie{Name: csrfCookieName, Value: csrfToken})
	retiredReq.Header.Set("X-CSRF-Token", csrfToken)
	retiredRec := httptest.NewRecorder()
	server.Handler().ServeHTTP(retiredRec, retiredReq)
	if retiredRec.Code == http.StatusOK {
		t.Fatalf("a retired setting key was accepted: %s", retiredRec.Body.String())
	}

	// 6. Use the version actually read, as an operator must. Other integration
	// tests legitimately advance this global setting's immutable history.
	currentSetting, err := runtime.PlatformSettings.Get(ctx, "features.trade_lines", false)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		latest, readErr := runtime.PlatformSettings.Get(ctx, currentSetting.Key, false)
		if readErr != nil {
			t.Error(readErr)
			return
		}
		if _, restoreErr := runtime.PlatformSettings.Update(ctx, user.ID, currentSetting.Key, currentSetting.Value, "Restore setting after endpoint verification", latest.Version); restoreErr != nil {
			t.Error(restoreErr)
		}
	}()
	updateBody, _ := json.Marshal(map[string]any{
		"key":              "features.trade_lines",
		"value":            json.RawMessage(`true`),
		"reason":           "Turning customer limits on for this test",
		"expected_version": currentSetting.Version,
	})
	updateReq := httptest.NewRequest(http.MethodPost, "/api/v1/ops/platform-settings", bytes.NewReader(updateBody))
	updateReq.AddCookie(&http.Cookie{Name: sessionCookieName, Value: aal2Token})
	updateReq.Header.Set("Origin", "http://localhost")
	updateReq.Header.Set("Sec-Fetch-Site", "same-origin")
	updateReq.AddCookie(&http.Cookie{Name: csrfCookieName, Value: csrfToken})
	updateReq.Header.Set("X-CSRF-Token", csrfToken)
	updateRec := httptest.NewRecorder()
	server.Handler().ServeHTTP(updateRec, updateReq)
	if updateRec.Code != http.StatusOK {
		t.Fatalf("updating a registered setting failed: status=%d body=%s", updateRec.Code, updateRec.Body.String())
	}

	histReq := httptest.NewRequest(http.MethodGet, "/api/v1/ops/platform-settings/history?key=features.trade_lines", nil)
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
		t.Fatalf("a settings change left no history entry")
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

type unavailableGovernance struct{ platformsettings.Service }

func (unavailableGovernance) GetGovernance(context.Context) (platformsettings.Governance, error) {
	return platformsettings.Governance{}, errors.New("governance read unavailable")
}

// A public capability describes the same feature switch the write handlers use.
func TestCapabilitiesReflectDisputeSwitch(t *testing.T) {
	runtime := NewRuntime(config.Config{Environment: "development"})
	runtime.PlatformSettings = disputeDisabledSettings{}
	server := NewServerWithRuntime(config.Config{Environment: "development"}, slog.Default(), runtime)
	response := httptest.NewRecorder()
	server.platformCapabilities(response, httptest.NewRequest(http.MethodGet, "/api/v1/platform/capabilities", nil))
	var payload struct {
		Features map[string]bool `json:"features"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload.Features["disputes"] {
		t.Fatal("disabled disputes advertised as enabled")
	}
}

type disputeDisabledSettings struct{ platformsettings.Service }

func (disputeDisabledSettings) GetBool(_ context.Context, _ string, _ bool) bool { return false }
