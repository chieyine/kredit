package web

import (
	"context"
	"encoding/json"
	"io"
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
	"kredit/internal/platformsettings"

	"github.com/google/uuid"
)

type auditRetainedSettings struct {
	platformsettings.Service
	saved map[string]json.RawMessage
}

func (s auditRetainedSettings) GetAll(ctx context.Context, includeSecrets bool) ([]platformsettings.Setting, error) {
	// Model the real store: the list is masked; individually decrypted values
	// are available only to the subsequent, explicitly private Get call.
	items := []platformsettings.Setting{}
	for key := range s.saved {
		items = append(items, platformsettings.Setting{Key: key, IsSecret: true, Category: "integrations", Version: 7, Value: json.RawMessage(`""`)})
	}
	return items, nil
}
func (s auditRetainedSettings) Get(ctx context.Context, key string, includeSecret bool) (platformsettings.Setting, error) {
	if value, ok := s.saved[key]; ok {
		if !includeSecret {
			value = json.RawMessage(`""`)
		}
		return platformsettings.Setting{Key: key, IsSecret: true, Version: 7, Value: value}, nil
	}
	return s.Service.Get(ctx, key, includeSecret)
}

func TestAuditSavedRetainedSecretsNeverReachSettingsHTTP(t *testing.T) {
	if os.Getenv("KREDIT_INTEGRATION") != "1" || os.Getenv("DATABASE_URL") == "" {
		t.Skip("isolated database required")
	}
	ctx := context.Background()
	database, err := db.Open(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(database.Close)
	cfg := config.Config{Environment: "development", Version: "test", APIListenAddr: ":0", Currency: "NGN", MoneyUnit: "kobo", CollectionProvider: "mock", TokenHashKey: "isolated-settings-projection-test-key", PublicBaseURL: "http://localhost"}
	runtime := NewRuntimeWithDB(cfg, database)
	server := NewServerWithRuntime(cfg, slog.New(slog.NewTextHandler(io.Discard, nil)), runtime)
	challenge, code, err := runtime.Auth.RequestOTP(uuid.NewString()+"@example.test", "email", "login")
	if err != nil {
		t.Fatal(err)
	}
	user, _, token, err := runtime.Auth.VerifyOTP(challenge.ID, code, "test")
	if err != nil {
		t.Fatal(err)
	}
	method, err := runtime.Auth.BeginTOTPEnrollment(user.ID)
	if err != nil {
		t.Fatal(err)
	}
	_, elevated, err := runtime.Auth.StepUpSession(token, auth.TOTPCode(method.Secret, time.Now().UTC()))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := database.Raw().Exec(ctx, `INSERT INTO app.platform_role_assignments(user_id,role,granted_by,reason) VALUES($1::uuid,'platform_owner',$1::uuid,'Synthetic settings projection test')`, user.ID); err != nil {
		t.Fatal(err)
	}
	// Leave the last synthetic owner for isolated-database teardown rather than
	// bypassing the production guard against removing the final owner.
	t.Cleanup(func() {
		_, err := database.Raw().Exec(context.Background(), `UPDATE app.platform_role_assignments SET revoked_at=now() WHERE user_id=$1::uuid AND revoked_at IS NULL AND EXISTS(SELECT 1 FROM app.platform_role_assignments other WHERE other.role='platform_owner' AND other.user_id<>$1::uuid AND other.revoked_at IS NULL)`, user.ID)
		if err != nil {
			t.Error(err)
		}
	})
	marshal := func(v any) json.RawMessage {
		t.Helper()
		b, e := json.Marshal(v)
		if e != nil {
			t.Fatal(e)
		}
		return b
	}
	saved := map[string]json.RawMessage{}
	for _, item := range []struct{ key, field string }{{"integrations.runtime.retained_collections", "RetainedCollectionProviders"}, {"integrations.runtime.retained_identity", "RetainedIdentityProviders"}} {
		account := []map[string]any{{"name": "saved-account", "endpoint": "https://saved.example.test", "token": "synthetic-http-token", "api_key": "synthetic-http-api-key", "webhook_secret": "synthetic-http-webhook"}}
		saved[item.key] = marshal(string(marshal(map[string]any{item.field: string(marshal(account))})))
	}
	runtime.PlatformSettings = auditRetainedSettings{Service: runtime.PlatformSettings, saved: saved}
	request := httptest.NewRequest(http.MethodGet, "/api/v1/ops/platform-settings", nil)
	request.AddCookie(&http.Cookie{Name: sessionCookieName, Value: elevated})
	response := httptest.NewRecorder()
	server.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("settings returned %d: %s", response.Code, response.Body.String())
	}
	body := response.Body.String()
	for _, secret := range []string{"synthetic-http-token", "synthetic-http-api-key", "synthetic-http-webhook"} {
		if strings.Contains(body, secret) {
			t.Fatal("decrypted retained credential reached the HTTP response")
		}
	}
	var result struct {
		Settings []platformsettings.Setting `json:"settings"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	found := 0
	for _, item := range result.Settings {
		if _, ok := saved[item.Key]; ok {
			found++
			if item.Version != 7 || item.ConnectionState != "restart_required" {
				t.Fatal("saved-state metadata changed")
			}
			visible, err := json.Marshal(item.ConnectionValues)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(string(visible), "saved-account") || !strings.Contains(string(visible), "saved.example.test") {
				t.Fatal("saved public values missing before restart")
			}
		}
	}
	if found != 2 {
		t.Fatalf("expected both saved account editors, got %d", found)
	}
}
