package web

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"kredit/internal/auth"
	"kredit/internal/config"
)

func TestSupplierOnboardingAndTenantBoundaries(t *testing.T) {
	t.Setenv("APP_ENV", "development")
	cfg, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	server := NewServer(cfg, slog.New(slog.NewTextHandler(io.Discard, nil)))
	client := newTestClient(server.Handler())

	challengeResponse := doJSON(t, client, "/api/v1/auth/otp/challenges", http.MethodPost, map[string]string{"identifier": "owner@example.test", "channel": "email", "purpose": "login"}, nil, http.StatusAccepted)
	var challenge struct {
		ChallengeID     string `json:"challenge_id"`
		DevelopmentCode string `json:"development_code"`
	}
	decodeResponse(t, challengeResponse, &challenge)

	verifyResponse := doJSON(t, client, "/api/v1/auth/otp/verify", http.MethodPost, map[string]string{"challenge_id": challenge.ChallengeID, "code": challenge.DevelopmentCode, "device_label": "test"}, nil, http.StatusOK)
	if len(verifyResponse.Cookies()) < 2 {
		t.Fatal("expected session and csrf cookies")
	}
	csrfValue := ""
	for _, cookie := range client.cookies {
		if cookie.Name == csrfCookieName {
			csrfValue = cookie.Value
		}
	}
	if csrfValue == "" {
		t.Fatal("expected csrf cookie")
	}

	organizationResponse := doJSON(t, client, "/api/v1/organizations", http.MethodPost, map[string]string{"legal_name": "ABC Pharmaceuticals Ltd", "business_type": "limited_company", "business_address": "Lagos", "industry": "pharmaceuticals"}, map[string]string{"X-CSRF-Token": csrfValue}, http.StatusCreated)
	var organizationPayload struct {
		Organization struct {
			ID string `json:"id"`
		} `json:"organization"`
	}
	decodeResponse(t, organizationResponse, &organizationPayload)
	if organizationPayload.Organization.ID == "" {
		t.Fatal("expected organization id")
	}
	feedbackResponse := doJSON(t, client, "/api/v1/me/product-feedback", http.MethodPost, map[string]string{"organization_id": organizationPayload.Organization.ID, "area": "seller", "screen": "overview", "answer": "yes"}, map[string]string{"X-CSRF-Token": csrfValue, "Idempotency-Key": "seller-feedback-1"}, http.StatusCreated)
	var feedbackPayload struct {
		Feedback struct {
			Answer string `json:"answer"`
		} `json:"feedback"`
	}
	decodeResponse(t, feedbackResponse, &feedbackPayload)
	if feedbackPayload.Feedback.Answer != "yes" {
		t.Fatalf("unexpected product feedback response: %#v", feedbackPayload)
	}

	organizationsResponse := doJSON(t, client, "/api/v1/organizations", http.MethodGet, nil, nil, http.StatusOK)
	_ = organizationsResponse.Body.Close()
	inviteResponse := doJSON(t, client, "/api/v1/organizations/"+organizationPayload.Organization.ID+"/members", http.MethodPost, map[string]string{"target": "sales@example.test", "channel": "email", "role": "sales"}, map[string]string{"X-CSRF-Token": csrfValue, "Idempotency-Key": "invite-step-up-denied"}, http.StatusForbidden)
	var problem struct {
		Title string `json:"title"`
	}
	decodeResponse(t, inviteResponse, &problem)
	if problem.Title != "step_up_required" {
		t.Fatalf("expected step-up denial, got %q", problem.Title)
	}

	enrollmentResponse := doJSON(t, client, "/api/v1/mfa/totp/enroll", http.MethodPost, map[string]string{}, map[string]string{"X-CSRF-Token": csrfValue}, http.StatusOK)
	var enrollment struct {
		Secret string `json:"secret"`
	}
	decodeResponse(t, enrollmentResponse, &enrollment)
	verifyMFAResponse := doJSON(t, client, "/api/v1/mfa/totp/verify", http.MethodPost, map[string]string{"code": auth.TOTPCode(enrollment.Secret, time.Now().UTC())}, map[string]string{"X-CSRF-Token": csrfValue}, http.StatusOK)
	_ = verifyMFAResponse.Body.Close()
	inviteSuccessResponse := doJSON(t, client, "/api/v1/organizations/"+organizationPayload.Organization.ID+"/members", http.MethodPost, map[string]string{"target": "sales@example.test", "channel": "email", "role": "sales"}, map[string]string{"X-CSRF-Token": csrfValue, "Idempotency-Key": "invite-success"}, http.StatusAccepted)
	_ = inviteSuccessResponse.Body.Close()
	auditResponse := doJSON(t, client, "/api/v1/organizations/"+organizationPayload.Organization.ID+"/audit-events", http.MethodGet, nil, nil, http.StatusOK)
	var auditPayload struct {
		Events []map[string]any `json:"events"`
	}
	decodeResponse(t, auditResponse, &auditPayload)
	if len(auditPayload.Events) == 0 {
		t.Fatal("expected organization audit events")
	}

	buyerInvitationResponse := doJSON(t, client, "/api/v1/organizations/"+organizationPayload.Organization.ID+"/buyer-invitations", http.MethodPost, map[string]string{"target": "buyer@example.test", "target_type": "email", "legal_name": "Royal Pharmacy Ltd", "business_type": "limited_company", "business_address": "Lagos", "industry": "pharmacy"}, map[string]string{"X-CSRF-Token": csrfValue, "Idempotency-Key": "buyer-invitation-1"}, http.StatusAccepted)
	var buyerInvitationPayload struct {
		InvitationURL string `json:"invitation_url"`
	}
	decodeResponse(t, buyerInvitationResponse, &buyerInvitationPayload)
	tokenMarker := "/buyer-invitations/"
	tokenIndex := strings.Index(buyerInvitationPayload.InvitationURL, tokenMarker)
	if tokenIndex < 0 {
		t.Fatalf("invitation URL did not contain token path: %q", buyerInvitationPayload.InvitationURL)
	}
	token := buyerInvitationPayload.InvitationURL[tokenIndex+len(tokenMarker):]
	previewResponse := doJSON(t, client, "/api/v1/buyer-invitations/"+token, http.MethodGet, nil, nil, http.StatusOK)
	_ = previewResponse.Body.Close()
	buyerOTPResponse := doJSON(t, client, "/api/v1/buyer-invitations/"+token+"/otp", http.MethodPost, nil, nil, http.StatusAccepted)
	var buyerOTP struct {
		ChallengeID string `json:"challenge_id"`
		Code        string `json:"development_code"`
	}
	decodeResponse(t, buyerOTPResponse, &buyerOTP)
	acceptBuyerResponse := doJSON(t, client, "/api/v1/buyer-invitations/"+token+"/accept", http.MethodPost, map[string]string{"challenge_id": buyerOTP.ChallengeID, "code": buyerOTP.Code, "full_name": "Royal Pharmacy Representative"}, map[string]string{"Idempotency-Key": "buyer-accept-1"}, http.StatusCreated)
	_ = acceptBuyerResponse.Body.Close()
	buyerPortalResponse := doJSON(t, client, "/api/v1/buyer/me", http.MethodGet, nil, nil, http.StatusOK)
	_ = buyerPortalResponse.Body.Close()
}

type testClient struct {
	handler http.Handler
	cookies []*http.Cookie
}

func newTestClient(handler http.Handler) *testClient {
	return &testClient{handler: handler}
}

func doJSON(t *testing.T, client *testClient, url, method string, body any, headers map[string]string, expectedStatus int) *http.Response {
	t.Helper()
	var payload io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		payload = bytes.NewReader(encoded)
	}
	request := httptest.NewRequest(method, url, payload)
	for _, cookie := range client.cookies {
		request.AddCookie(cookie)
	}
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	for key, value := range headers {
		request.Header.Set(key, value)
	}
	responseRecorder := httptest.NewRecorder()
	client.handler.ServeHTTP(responseRecorder, request)
	response := responseRecorder.Result()
	for _, cookie := range response.Cookies() {
		updated := false
		for index, existing := range client.cookies {
			if existing.Name == cookie.Name {
				client.cookies[index] = cookie
				updated = true
				break
			}
		}
		if !updated {
			client.cookies = append(client.cookies, cookie)
		}
	}
	if response.StatusCode != expectedStatus {
		defer func() { _ = response.Body.Close() }()
		bodyBytes, _ := io.ReadAll(response.Body)
		t.Fatalf("%s %s returned %d, want %d: %s", method, url, response.StatusCode, expectedStatus, bodyBytes)
	}
	return response
}

func decodeResponse(t *testing.T, response *http.Response, destination any) {
	t.Helper()
	defer func() { _ = response.Body.Close() }()
	if err := json.NewDecoder(response.Body).Decode(destination); err != nil {
		t.Fatal(err)
	}
}
