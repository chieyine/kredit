package web

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"kredit/internal/access"
	"kredit/internal/auth"
	"kredit/internal/config"
	"kredit/internal/onboarding"
)

func TestNewSupplierOwnerReachesPilotReadyAndCanInviteSales(t *testing.T) {
	t.Setenv("APP_ENV", "development")
	cfg, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	server := NewServer(cfg, slog.New(slog.NewTextHandler(io.Discard, nil)))
	client := newTestClient(server.Handler())
	challengeResponse := doJSON(t, client, "/api/v1/auth/otp/challenges", http.MethodPost, map[string]string{"identifier": "new-owner@example.test", "channel": "email", "purpose": "login"}, nil, http.StatusAccepted)
	var challenge struct {
		ChallengeID string `json:"challenge_id"`
		Code        string `json:"development_code"`
	}
	decodeResponse(t, challengeResponse, &challenge)
	verify := doJSON(t, client, "/api/v1/auth/otp/verify", http.MethodPost, map[string]string{"challenge_id": challenge.ChallengeID, "code": challenge.Code}, nil, http.StatusOK)
	_ = verify.Body.Close()
	csrf := ""
	for _, cookie := range client.cookies {
		if cookie.Name == csrfCookieName {
			csrf = cookie.Value
		}
	}
	created := doJSON(t, client, "/api/v1/organizations", http.MethodPost, map[string]any{"legal_name": "Fresh Foods Ltd", "trading_name": "Fresh Foods", "business_type": "limited_company", "registration_info": "RC-100", "business_address": "Lagos", "industry": "food distribution"}, map[string]string{"X-CSRF-Token": csrf}, http.StatusCreated)
	var orgPayload struct {
		Organization struct {
			ID string `json:"id"`
		} `json:"organization"`
	}
	decodeResponse(t, created, &orgPayload)
	org := orgPayload.Organization.ID
	enroll := doJSON(t, client, "/api/v1/mfa/totp/enroll", http.MethodPost, map[string]string{}, map[string]string{"X-CSRF-Token": csrf}, http.StatusOK)
	var mfa struct {
		Secret string `json:"secret"`
	}
	decodeResponse(t, enroll, &mfa)
	elevate := doJSON(t, client, "/api/v1/mfa/totp/verify", http.MethodPost, map[string]string{"code": auth.TOTPCode(mfa.Secret, time.Now().UTC())}, map[string]string{"X-CSRF-Token": csrf}, http.StatusOK)
	_ = elevate.Body.Close()
	for _, cookie := range client.cookies {
		if cookie.Name == csrfCookieName {
			csrf = cookie.Value
		}
	}
	getVersion := func() int64 {
		r := doJSON(t, client, "/api/v1/organizations/"+org+"/onboarding", http.MethodGet, nil, nil, http.StatusOK)
		var v struct {
			Profile struct {
				Version int64 `json:"version"`
			} `json:"profile"`
		}
		decodeResponse(t, r, &v)
		return v.Profile.Version
	}
	headers := func(key string) map[string]string {
		return map[string]string{"X-CSRF-Token": csrf, "Idempotency-Key": key}
	}
	rep := doJSON(t, client, "/api/v1/organizations/"+org+"/onboarding/representative", http.MethodPatch, map[string]any{"expected_version": getVersion(), "name": "Ada Okafor", "title": "Managing Director"}, headers("onboard-representative"), http.StatusOK)
	_ = rep.Body.Close()
	contactChallenge := doJSON(t, client, "/api/v1/organizations/"+org+"/onboarding/contacts/challenges", http.MethodPost, map[string]string{"identifier": "+2348012345678", "channel": "phone"}, headers("onboard-phone-request"), http.StatusAccepted)
	var contact struct {
		ChallengeID string `json:"challenge_id"`
		Code        string `json:"development_code"`
	}
	decodeResponse(t, contactChallenge, &contact)
	contactVerify := doJSON(t, client, "/api/v1/organizations/"+org+"/onboarding/contacts/verify", http.MethodPost, map[string]string{"identifier": "+2348012345678", "channel": "phone", "challenge_id": contact.ChallengeID, "code": contact.Code}, headers("onboard-phone-verify"), http.StatusOK)
	_ = contactVerify.Body.Close()
	for _, step := range []struct {
		path, method, key string
		body              map[string]any
	}{
		{"kyb", http.MethodPost, "onboard-kyb", map[string]any{}},
		{"settlement", http.MethodPut, "onboard-settlement", map[string]any{"provider": "mock-settlement", "provider_reference": "settlement-ref", "bank_name": "Demo Bank", "account_name": "Fresh Foods Ltd", "account_last4": "1234"}},
		{"billing", http.MethodPut, "onboard-billing", map[string]any{"method": "split_settlement", "provider_reference": "billing-ref", "cycle": "per_settlement"}},
		{"credit-policy", http.MethodPut, "onboard-policy", map[string]any{"credit_limit_kobo": 100000000, "payment_days": 30, "grace_hours": 48}},
		{"consents", http.MethodPost, "onboard-consents", map[string]any{"terms_version": "supplier-terms-v1", "privacy_version": "privacy-v1"}},
	} {
		step.body["expected_version"] = getVersion()
		r := doJSON(t, client, "/api/v1/organizations/"+org+"/onboarding/"+step.path, step.method, step.body, headers(step.key), http.StatusOK)
		_ = r.Body.Close()
	}
	readyResponse := doJSON(t, client, "/api/v1/organizations/"+org+"/onboarding", http.MethodGet, nil, nil, http.StatusOK)
	var ready struct {
		Readiness struct {
			Ready   bool  `json:"ready"`
			Missing []any `json:"missing"`
		} `json:"readiness"`
	}
	decodeResponse(t, readyResponse, &ready)
	if !ready.Readiness.Ready || len(ready.Readiness.Missing) != 0 {
		t.Fatalf("expected pilot ready: %#v", ready)
	}
	invite := doJSON(t, client, "/api/v1/organizations/"+org+"/members", http.MethodPost, map[string]string{"target": "sales@fresh-foods.test", "channel": "email", "role": "sales"}, headers("onboard-invite-sales"), http.StatusAccepted)
	_ = invite.Body.Close()
}

func TestIncompleteSupplierGateExplainsRecovery(t *testing.T) {
	t.Setenv("APP_ENV", "development")
	cfg, _ := config.Load()
	runtime := NewRuntime(cfg)
	_, _ = runtime.Onboarding.Ensure("org-incomplete", "owner", true, false)
	server := NewServerWithRuntime(cfg, slog.Default(), runtime)
	rec := httptest.NewRecorder()
	if server.requireSupplierReady(rec, "org-incomplete", "owner", "releasing goods") {
		t.Fatal("incomplete supplier must be blocked")
	}
	if rec.Code != http.StatusConflict {
		t.Fatalf("expected conflict, got %d", rec.Code)
	}
	body := rec.Body.String()
	for _, want := range []string{"supplier_not_ready", "phone_verified", "kyb_approved", "settlement_verified"} {
		if !contains(body, want) {
			t.Fatalf("missing recovery guidance %q in %s", want, body)
		}
	}
}

func TestOnboardingRoleViewsAndMFAFreshness(t *testing.T) {
	profile := onboarding.Profile{KYBProviderReference: "kyb-secret-ref", SettlementProviderReference: "settlement-secret-ref", BillingProviderReference: "billing-secret-ref", SettlementAccountName: "Sensitive Account Name"}
	sales := visibleOnboardingProfile(profile, access.RoleSales)
	if sales.KYBProviderReference != "" || sales.SettlementProviderReference != "" || sales.BillingProviderReference != "" || sales.SettlementAccountName != "" {
		t.Fatal("sales view exposed restricted onboarding references")
	}
	financePermissions := onboardingPermissions(access.RoleFinance)
	if !financePermissions["settlement"] || !financePermissions["billing"] || financePermissions["credit_policy"] || financePermissions["consents"] {
		t.Fatalf("unexpected finance permissions: %#v", financePermissions)
	}
	t.Setenv("APP_ENV", "development")
	cfg, _ := config.Load()
	server := NewServer(cfg, slog.Default())
	stale := httptest.NewRecorder()
	if server.requireFreshMFA(stale, auth.Session{AuthenticationLevel: auth.AAL2, MFAVerifiedAt: time.Now().Add(-16 * time.Minute)}) {
		t.Fatal("stale MFA must not authorize sensitive onboarding")
	}
	if stale.Code != http.StatusForbidden {
		t.Fatalf("expected stale MFA denial, got %d", stale.Code)
	}
	fresh := httptest.NewRecorder()
	if !server.requireFreshMFA(fresh, auth.Session{AuthenticationLevel: auth.AAL2, MFAVerifiedAt: time.Now()}) {
		t.Fatal("fresh MFA should authorize sensitive onboarding")
	}
}

func contains(value, part string) bool {
	return len(part) == 0 || len(value) >= len(part) && func() bool {
		for i := 0; i+len(part) <= len(value); i++ {
			if value[i:i+len(part)] == part {
				return true
			}
		}
		return false
	}()
}
