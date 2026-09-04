package web

import (
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
	"kredit/internal/config"
	"kredit/internal/db"
)

func TestPlatformOperationsRequiresRoleAndStepUp(t *testing.T) {
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
	cfg := config.Config{Environment: "development", Version: "test", APIListenAddr: ":0", Currency: "NGN", MoneyUnit: "kobo", CollectionProvider: "mock", TokenHashKey: "platform-ops-test-secret", PublicBaseURL: "http://localhost"}
	runtime := NewRuntimeWithDB(cfg, database)
	identifier := fmt.Sprintf("platform-ops-%d@example.test", time.Now().UnixNano())
	challenge, code, err := runtime.Auth.RequestOTP(identifier, "email", "login")
	if err != nil {
		t.Fatal(err)
	}
	user, _, token, err := runtime.Auth.VerifyOTP(challenge.ID, code, "test")
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
	server := NewServerWithRuntime(cfg, slog.Default(), runtime)
	request := httptest.NewRequest(http.MethodGet, "/api/v1/ops/overview", nil)
	request.AddCookie(&http.Cookie{Name: sessionCookieName, Value: token})
	response := httptest.NewRecorder()
	server.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusForbidden {
		t.Fatalf("unassigned user status=%d body=%s", response.Code, response.Body.String())
	}
	if _, err := database.Raw().Exec(ctx, `INSERT INTO app.platform_role_assignments(user_id,role,granted_by,reason) VALUES($1::uuid,'platform_admin',$1::uuid,'Integration test assignment')`, user.ID); err != nil {
		t.Fatal(err)
	}
	response = httptest.NewRecorder()
	server.Handler().ServeHTTP(response, request.Clone(ctx))
	if response.Code != http.StatusForbidden || !strings.Contains(response.Body.String(), "step_up_required") {
		t.Fatalf("AAL1 status=%d body=%s", response.Code, response.Body.String())
	}
	method, err := runtime.Auth.BeginTOTPEnrollment(user.ID)
	if err != nil {
		t.Fatal(err)
	}
	_, rotatedToken, err := runtime.Auth.StepUpSession(token, auth.TOTPCode(method.Secret, time.Now().UTC()))
	if err != nil {
		t.Fatal(err)
	}
	response = httptest.NewRecorder()
	server.Handler().ServeHTTP(response, request.Clone(ctx))
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("pre-step-up session remained usable: %d", response.Code)
	}
	token = rotatedToken
	request = httptest.NewRequest(http.MethodGet, "/api/v1/ops/overview", nil)
	request.AddCookie(&http.Cookie{Name: sessionCookieName, Value: token})
	response = httptest.NewRecorder()
	server.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), "queued_jobs") {
		t.Fatalf("authorized status=%d body=%s", response.Code, response.Body.String())
	}
	for _, path := range []string{
		"/api/v1/ops/metrics",
		"/api/v1/ops/metrics/prometheus",
		"/api/v1/ops/users",
		"/api/v1/ops/organizations",
		"/api/v1/ops/money",
		"/api/v1/ops/cases",
		"/api/v1/ops/disputes",
		"/api/v1/ops/business-policies",
		"/api/v1/ops/capabilities", "/api/v1/ops/approval-inbox", "/api/v1/ops/admin-changes", "/api/v1/ops/change-history", "/api/v1/ops/change-history?format=csv", "/api/v1/ops/attention", "/api/v1/ops/attention/details", "/api/v1/buyer/amendments",
		"/api/v1/ops/team",
		"/api/v1/ops/jobs",
		"/api/v1/ops/provider-events",
		"/api/v1/ops/audit",
		"/api/v1/ops/analytics/scorecard?from=2026-01-01&to=2026-12-31",
		"/api/v1/ops/search?q=reference-that-does-not-exist",
	} {
		request = httptest.NewRequest(http.MethodGet, path, nil)
		request.AddCookie(&http.Cookie{Name: sessionCookieName, Value: token})
		response = httptest.NewRecorder()
		server.Handler().ServeHTTP(response, request)
		if response.Code != http.StatusOK {
			t.Fatalf("%s status=%d body=%s", path, response.Code, response.Body.String())
		}
	}
	currentPolicy, err := runtime.BusinessPolicies.Read(ctx)
	if err != nil {
		t.Fatal(err)
	}
	currentPolicy.Values.UpcomingNoticeDays = 4
	previewBody, _ := json.Marshal(map[string]any{"base_revision": currentPolicy.Revision, "values": currentPolicy.Values})
	previewRequest := httptest.NewRequest(http.MethodPost, "/api/v1/ops/business-policies/preview", strings.NewReader(string(previewBody)))
	previewRequest.AddCookie(&http.Cookie{Name: sessionCookieName, Value: token})
	previewResponse := httptest.NewRecorder()
	server.Handler().ServeHTTP(previewResponse, previewRequest)
	if previewResponse.Code != 200 || !strings.Contains(previewResponse.Body.String(), "due_instalments") {
		t.Fatalf("policy impact preview: %d %s", previewResponse.Code, previewResponse.Body.String())
	}
	// New specialist roles can open their work but do not inherit unrelated powers.
	for _, c := range []struct{ role, allow, deny string }{
		{"finance_operator", "/api/v1/ops/admin-changes", "/api/v1/ops/team"},
		{"policy_manager", "/api/v1/ops/business-policies", "/api/v1/ops/admin-changes"},
		{"approver", "/api/v1/ops/business-policies", "/api/v1/ops/team"},
		{"access_administrator", "/api/v1/ops/team", "/api/v1/ops/admin-changes"},
		{"support_agent", "/api/v1/ops/approval-inbox", "/api/v1/ops/change-context?q=anything"},
	} {
		if _, err = database.Raw().Exec(ctx, `UPDATE app.platform_role_assignments SET role=$2 WHERE user_id=$1::uuid`, user.ID, c.role); err != nil {
			t.Fatal(err)
		}
		for path, want := range map[string]int{c.allow: 200, c.deny: 403} {
			req := httptest.NewRequest(http.MethodGet, path, nil)
			req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: token})
			rec := httptest.NewRecorder()
			server.Handler().ServeHTTP(rec, req)
			if rec.Code != want {
				t.Fatalf("%s %s got %d: %s", c.role, path, rec.Code, rec.Body.String())
			}
		}
	}

}
