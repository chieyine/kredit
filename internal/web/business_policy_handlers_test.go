package web

import (
	"encoding/json"
	"kredit/internal/access"
	"kredit/internal/config"
	"log/slog"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPublicPricingOnlyExposesRates(t *testing.T) {
	s := NewServer(config.Config{Environment: "development", MonoSecretKey: "must-stay-secret", CollectionProvider: "mock"}, slog.Default())
	r := httptest.NewRequest("GET", "/api/v1/pricing", nil)
	w := httptest.NewRecorder()
	s.Handler().ServeHTTP(w, r)
	if w.Code != 200 || strings.Contains(w.Body.String(), "must-stay-secret") || w.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("unsafe pricing response: %d %s", w.Code, w.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil || len(body) != 3 || body["base_bps"] != float64(50) {
		t.Fatal("unexpected public pricing contract", err, body)
	}
	for _, role := range []access.PlatformRole{access.PlatformSupportAgent, access.PlatformComplianceReviewer, access.PlatformDisputeReviewer} {
		if access.CanPlatform(role, access.PermissionManagePolicies) {
			t.Fatal("non-administrator can alter business policy")
		}
	}
	if !requiresIdempotencyKey(httptest.NewRequest("POST", "/api/v1/ops/business-policies", nil)) {
		t.Fatal("policy mutation lacks idempotency envelope")
	}
}
