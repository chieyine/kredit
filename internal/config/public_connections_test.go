package config

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestAuditSavedConnectionProjectionOmitsNestedCredentials(t *testing.T) {
	for _, item := range []struct{ key, field string }{
		{"integrations.runtime.retained_collections", "RetainedCollectionProviders"},
		{"integrations.runtime.retained_identity", "RetainedIdentityProviders"},
	} {
		t.Run(item.field, func(t *testing.T) {
			nested := `[{"name":"saved-account","endpoint":"https://connector.example.test","adapter":"connector","partial":true,"contract_code":"visible-contract","api_key":"synthetic-api-secret","token":"synthetic-token-secret","webhook_secret":"synthetic-webhook-secret","future_private_field":"synthetic-future-secret"}]`
			raw := encodedConnection(t, item.key, map[string]any{item.field: nested})
			visible, err := PublicStoredConnectionValues(item.key, raw)
			if err != nil {
				t.Fatal(err)
			}
			encoded, err := json.Marshal(visible)
			if err != nil {
				t.Fatal(err)
			}
			for _, secret := range []string{"synthetic-api-secret", "synthetic-token-secret", "synthetic-webhook-secret", "synthetic-future-secret"} {
				if strings.Contains(string(encoded), secret) {
					t.Fatalf("secret entered the public projection: %s", secret)
				}
			}
			var accounts []map[string]json.RawMessage
			if err := json.Unmarshal([]byte(visible[item.field].(string)), &accounts); err != nil {
				t.Fatal(err)
			}
			if len(accounts) != 1 || string(accounts[0]["name"]) != `"saved-account"` || string(accounts[0]["contract_code"]) != `"visible-contract"` {
				t.Fatal("non-secret editor fields were lost")
			}
			for _, key := range []string{"api_key", "token", "webhook_secret", "future_private_field"} {
				if _, ok := accounts[0][key]; ok {
					t.Fatalf("private field %s is present", key)
				}
			}
			for _, bad := range []string{`{}`, `[{"name":1}]`, `not JSON`, `[null,null,null,null,null,null,null,null,null]`} {
				if _, err := PublicStoredConnectionValues(item.key, encodedConnection(t, item.key, map[string]any{item.field: bad})); err == nil {
					t.Fatalf("accepted malformed saved configuration %s", bad)
				}
			}
		})
	}
}

func TestAuditSavedConnectionProjectionPreservesOrdinaryTypes(t *testing.T) {
	key := "integrations.runtime.settlement"
	raw := encodedConnection(t, key, map[string]any{"SettlementEnabled": true, "SettlementProvider": "provider", "SettlementEndpoint": "https://connector.example.test", "SettlementToken": "synthetic-secret"})
	visible, err := PublicStoredConnectionValues(key, raw)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := visible["SettlementToken"]; ok {
		t.Fatal("password field exposed")
	}
	encoded, err := json.Marshal(visible)
	if err != nil {
		t.Fatal(err)
	}
	var typed struct {
		Enabled  bool   `json:"SettlementEnabled"`
		Provider string `json:"SettlementProvider"`
	}
	if err := json.Unmarshal(encoded, &typed); err != nil {
		t.Fatal(err)
	}
	if !typed.Enabled || typed.Provider != "provider" {
		t.Fatal("ordinary values changed type")
	}
	if _, err := PublicStoredConnectionValues("unregistered", raw); err == nil {
		t.Fatal("unregistered connection projected")
	}
}
