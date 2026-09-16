package config

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestNativeSavedAccountCredentialsStayPrivate(t *testing.T) {
	c := productionBase()
	c.CollectionProvider = "new-account"
	entry := RetainedCollectionConnection{Adapter: "monnify", Name: "old-monnify", Token: strings.Repeat("s", 40), APIKey: "private-api-key", ContractCode: "contract"}
	raw, _ := json.Marshal([]RetainedCollectionConnection{entry})
	c.RetainedCollectionProviders = string(raw)
	if _, e := c.RetainedCollections(); e != nil {
		t.Fatal(e)
	}
	visible := PublicConnectionValues(c, "integrations.runtime.retained_collections")
	encoded, _ := json.Marshal(visible)
	if strings.Contains(string(encoded), entry.Token) || strings.Contains(string(encoded), entry.APIKey) {
		t.Fatal("saved native credentials leaked to browser")
	}
	if !strings.Contains(string(encoded), entry.ContractCode) {
		t.Fatal("non-secret contract code missing")
	}
	entry.Endpoint = "https://another-host.example"
	raw, _ = json.Marshal([]RetainedCollectionConnection{entry})
	c.RetainedCollectionProviders = string(raw)
	if _, e := c.RetainedCollections(); e == nil {
		t.Fatal("native credentials allowed to be redirected")
	}
}
func TestNativeSettlementUsesTheConfiguredCollector(t *testing.T) {
	for _, adapter := range []string{"paystack", "flutterwave", "monnify"} {
		c := productionBase()
		c.CollectionAdapter = adapter
		c.CollectionProvider = "native-main"
		c.CollectionProviderToken = strings.Repeat("s", 40)
		switch adapter {
		case "paystack":
			c.CollectionProviderToken = "sk_live_" + strings.Repeat("s", 40)
		case "flutterwave":
			c.CollectionProviderToken = "FLWSECK-" + strings.Repeat("s", 40)
			c.CollectionWebhookSecret = strings.Repeat("w", 40)
		case "monnify":
			c.CollectionAPIKey = "synthetic-api-key"
			c.CollectionContractCode = "contract"
		}
		c.SettlementProvider = c.CollectionProvider
		c.SettlementEnabled = true
		if e := c.Validate(); e != nil {
			t.Fatalf("%s: %v", adapter, e)
		}
	}
}
