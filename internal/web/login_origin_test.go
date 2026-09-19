package web

import (
	"fmt"
	"io"
	"kredit/internal/auth"
	"kredit/internal/config"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestLoginRejectsForeignBrowserOriginBeforeConsumingOTP(t *testing.T) {
	t.Setenv("APP_ENV", "development")
	cfg, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	cfg.PublicBaseURL = "https://www.kredit.ng"
	cfg.AppBaseURL = cfg.PublicBaseURL
	server := NewServer(cfg, slog.New(slog.NewTextHandler(io.Discard, nil)))
	for _, site := range []string{"cross-site", "same-site", ""} {
		t.Run(site, func(t *testing.T) {
			challenge, code, err := server.runtime.Auth.RequestOTP("synthetic-"+site+"@example.test", "email", auth.PurposeLogin)
			if err != nil {
				t.Fatal(err)
			}
			// A text/plain HTML form may put its '=' separator inside device_label.
			body := fmt.Sprintf(`{"challenge_id":%q,"code":%q,"device_label":"="}`, challenge.ID, code)
			request := httptest.NewRequest(http.MethodPost, "/api/v1/auth/otp/verify", strings.NewReader(body))
			request.Header.Set("Content-Type", "text/plain")
			request.Header.Set("Origin", "https://unrelated.example")
			if site != "" {
				request.Header.Set("Sec-Fetch-Site", site)
			}
			response := httptest.NewRecorder()
			server.Handler().ServeHTTP(response, request)
			if response.Code != http.StatusForbidden || len(response.Result().Cookies()) != 0 {
				t.Fatalf("foreign login accepted: status=%d cookies=%d", response.Code, len(response.Result().Cookies()))
			}
			request = httptest.NewRequest(http.MethodPost, "/api/v1/auth/otp/verify", strings.NewReader(body))
			request.Header.Set("Content-Type", "application/json")
			request.Header.Set("Origin", cfg.PublicBaseURL)
			request.Header.Set("Sec-Fetch-Site", "same-origin")
			response = httptest.NewRecorder()
			server.Handler().ServeHTTP(response, request)
			if response.Code != http.StatusOK || len(response.Result().Cookies()) != 2 {
				t.Fatalf("legitimate login failed or rejected attempt consumed OTP: status=%d", response.Code)
			}
		})
	}
}
