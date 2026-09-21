package web

import (
	"context"
	"encoding/json"
	"errors"
	"mime"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"kredit/internal/platformsettings"
)

type auditMetaVerificationSettings struct {
	platformsettings.Service
	connector platformsettings.NotificationConnector
	err       error
}

func (s auditMetaVerificationSettings) Get(context.Context, string, bool) (platformsettings.Setting, error) {
	if s.err != nil {
		return platformsettings.Setting{}, s.err
	}
	encoded, err := json.Marshal(s.connector)
	if err != nil {
		return platformsettings.Setting{}, err
	}
	value, err := json.Marshal(string(encoded))
	return platformsettings.Setting{Key: "integrations.notifications.whatsapp", Value: value}, err
}

func TestMetaVerificationEchoIsBoundedAuthenticatedPlainText(t *testing.T) {
	const verifyToken = "synthetic-meta-verification-token-for-tests"
	for _, tc := range []struct {
		name, challenge, token, mode, adapter string
		disabled, unavailable                 bool
		want                                  int
	}{
		{name: "ordinary challenge", challenge: "12345", want: 200},
		{name: "HTML-shaped challenge is text", challenge: "<not-html>&\"'", want: 200},
		{name: "maximum length", challenge: strings.Repeat("x", 512), want: 200},
		{name: "empty", want: 400},
		{name: "too long", challenge: strings.Repeat("x", 513), want: 400},
		{name: "wrong token", challenge: "never-echo", token: "wrong-token", want: 403},
		{name: "wrong mode", challenge: "never-echo", mode: "not-subscribe", want: 403},
		{name: "disabled", challenge: "never-echo", disabled: true, want: 403},
		{name: "wrong adapter", challenge: "never-echo", adapter: "connector", want: 403},
		{name: "settings unavailable", challenge: "never-echo", unavailable: true, want: 503},
	} {
		t.Run(tc.name, func(t *testing.T) {
			adapter, mode, token := tc.adapter, tc.mode, tc.token
			if adapter == "" {
				adapter = "meta"
			}
			if mode == "" {
				mode = "subscribe"
			}
			if token == "" {
				token = verifyToken
			}
			settings := auditMetaVerificationSettings{connector: platformsettings.NotificationConnector{
				Enabled: !tc.disabled, Adapter: adapter, Endpoint: "https://graph.facebook.com/v23.0/123/messages",
				Token: "synthetic-access-token", VerifyToken: verifyToken, WebhookSecret: strings.Repeat("s", 32),
				Language: "en", AuthenticationTemplate: "test_auth", UtilityTemplate: "test_utility", MarketingTemplate: "test_marketing",
			}}
			if tc.unavailable {
				settings.err = errors.New("synthetic settings outage")
			}
			server := &Server{runtime: &Runtime{PlatformSettings: settings}}
			query := url.Values{"hub.mode": {mode}, "hub.verify_token": {token}, "hub.challenge": {tc.challenge}}
			request := httptest.NewRequest(http.MethodGet, "/api/v1/webhooks/meta?"+query.Encode(), nil)
			response := httptest.NewRecorder()
			// Call the handler directly: its response must remain safe even when
			// tested independently of the global nosniff middleware.
			server.metaWebhook(response, request)
			if response.Code != tc.want {
				t.Fatalf("status = %d, want %d", response.Code, tc.want)
			}
			if tc.want == http.StatusOK {
				kind, parameters, err := mime.ParseMediaType(response.Header().Get("Content-Type"))
				if err != nil || kind != "text/plain" || parameters["charset"] != "utf-8" || response.Header().Get("X-Content-Type-Options") != "nosniff" {
					t.Fatalf("challenge response is not explicit non-sniffable UTF-8 text: %v", response.Header())
				}
				if response.Body.String() != tc.challenge {
					t.Fatal("provider challenge bytes changed")
				}
			} else if strings.Contains(response.Body.String(), "never-echo") {
				t.Fatal("rejected challenge was reflected")
			}
		})
	}
}
