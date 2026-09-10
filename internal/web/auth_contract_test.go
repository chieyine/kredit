package web

import (
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"testing"

	"kredit/internal/config"
)

func TestSignInAndAuthenticatorEnrollmentContracts(t *testing.T) {
	for _, tc := range []struct{ channel, identifier, label string }{
		{"phone", "0801 234 5678", "+2348012345678"},
		{"email", "Owner#finance@example.test", "owner#finance@example.test"},
	} {
		t.Run(tc.channel, func(t *testing.T) {
			t.Setenv("APP_ENV", "development")
			cfg, err := config.Load()
			if err != nil {
				t.Fatal(err)
			}
			server := NewServer(cfg, slog.New(slog.NewTextHandler(io.Discard, nil)))
			client := newTestClient(server.Handler())
			response := doJSON(t, client, "/api/v1/auth/otp/challenges", http.MethodPost, map[string]string{"identifier": tc.identifier, "channel": tc.channel, "purpose": "login"}, nil, http.StatusAccepted)
			var challenge struct {
				ID   string `json:"challenge_id"`
				Code string `json:"development_code"`
			}
			decodeResponse(t, response, &challenge)
			login := doJSON(t, client, "/api/v1/auth/otp/verify", http.MethodPost, map[string]string{"challenge_id": challenge.ID, "code": challenge.Code}, nil, http.StatusOK)
			_ = login.Body.Close()
			var csrf string
			for _, cookie := range client.cookies {
				if cookie.Name == csrfCookieName {
					csrf = cookie.Value
				}
			}
			enrollment := doJSON(t, client, "/api/v1/mfa/totp/enroll", http.MethodPost, map[string]string{}, map[string]string{"X-CSRF-Token": csrf}, http.StatusOK)
			var body struct {
				Secret string `json:"secret"`
				URI    string `json:"otpauth_uri"`
			}
			decodeResponse(t, enrollment, &body)
			uri, err := url.Parse(body.URI)
			if err != nil {
				t.Fatal(err)
			}
			if uri.Scheme != "otpauth" || uri.Host != "totp" || uri.Path != "/Kredit:"+tc.label || uri.Fragment != "" || uri.Query().Get("secret") != body.Secret || body.Secret == "" {
				t.Fatal("authenticator URI did not preserve account label and setup key")
			}
		})
	}
}
