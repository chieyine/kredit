package web

import (
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"kredit/internal/config"
	platformlogging "kredit/internal/platform/logging"
)

func TestResolveSecureLink(t *testing.T) {
	cfg := config.Config{Environment: "development", TokenHashKey: "secure-link-test"}
	runtime := NewRuntime(cfg)
	server := NewServerWithRuntime(cfg, platformlogging.New(), runtime)
	path := "/buyer/credit-requests/request-1"
	expires := time.Now().Add(time.Minute).Truncate(time.Second)
	link, err := url.Parse(runtime.Notifications.SecureLink(path, expires))
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodGet, "/api/v1/secure-link?"+link.RawQuery, nil)
	response := httptest.NewRecorder()
	server.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
}

func TestResolveSecureLinkRejectsExternalRedirect(t *testing.T) {
	cfg := config.Config{Environment: "development", TokenHashKey: "secure-link-test"}
	runtime := NewRuntime(cfg)
	server := NewServerWithRuntime(cfg, platformlogging.New(), runtime)
	path := "//attacker.example"
	expires := time.Now().Add(time.Minute).Truncate(time.Second)
	signatureURL, _ := url.Parse(runtime.Notifications.SecureLink(path, expires))
	request := httptest.NewRequest(http.MethodGet, "/api/v1/secure-link?path="+hex.EncodeToString([]byte(path))+"&exp="+signatureURL.Query().Get("exp")+"&sig="+signatureURL.Query().Get("sig"), nil)
	response := httptest.NewRecorder()
	server.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusGone {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
}
