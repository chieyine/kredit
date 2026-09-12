package config

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"kredit/internal/platformsettings"

	"github.com/jackc/pgx/v5"
)

type connectionStore struct {
	platformsettings.Service
	items map[string]platformsettings.Setting
	err   error
}

func (s connectionStore) Get(_ context.Context, key string, _ bool) (platformsettings.Setting, error) {
	if s.err != nil {
		return platformsettings.Setting{}, s.err
	}
	item, ok := s.items[key]
	if !ok {
		return item, pgx.ErrNoRows
	}
	return item, nil
}
func encodedConnection(t *testing.T, key string, values map[string]any) json.RawMessage {
	t.Helper()
	for _, field := range platformsettings.RuntimeConnections[key].Fields {
		if _, ok := values[field.Key]; ok {
			continue
		}
		switch field.Kind {
		case "boolean":
			values[field.Key] = false
		case "number":
			values[field.Key] = 0
		default:
			values[field.Key] = ""
		}
	}
	payload, err := json.Marshal(values)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(string(payload))
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func TestAdminIdentityIsConsumedAtStartupWithoutChangingInfrastructure(t *testing.T) {
	key := "integrations.runtime.identity"
	raw := encodedConnection(t, key, map[string]any{"RealIdentity": true, "IdentityProvider": "chosen-connector", "IdentityProviderEndpoint": "https://identity.kredit.test", "IdentityProviderToken": "identity-token-fixture-0123456789abcdef", "IdentityWebhookSecret": "identity-webhook-fixture-0123456789abcdef", "IdentityApprovalReference": "approved-identity-001"})
	store := connectionStore{items: map[string]platformsettings.Setting{key: {Key: key, Value: raw, Version: 7}}}
	base := productionBase()
	result, err := ApplyStoredConnections(context.Background(), base, store, "", nil)
	if err != nil {
		t.Fatal(err)
	}
	if !result.RealIdentity || result.IdentityProvider != "chosen-connector" || result.AdminConnectionVersions[key] != 7 {
		t.Fatal("saved identity not applied")
	}
	if result.DatabaseURL != base.DatabaseURL || result.SessionSigningKey != base.SessionSigningKey || base.RealIdentity {
		t.Fatal("infrastructure or input config changed")
	}
	visible := PublicConnectionValues(result, key)
	if _, ok := visible["IdentityProviderToken"]; ok {
		t.Fatal("secret exposed in editor values")
	}
}

func TestAdminMonoPauseRetainsReconciliationAndSecret(t *testing.T) {
	key := "integrations.runtime.mono"
	base := productionBase()
	base.ProviderCertificationReference = "certification-001"
	base.CollectionNoticeMinHours = 24
	base.DeemedAcceptanceMinHours = 48
	saved := encodedConnection(t, key, map[string]any{"MonoSweepEnabled": true, "MonoSecretKey": "live-mono-fixture-0123456789abcdef012345", "MonoWebhookSecret": "mono-webhook-fixture-0123456789abcdef", "MonoRedirectURL": "https://app.kredit.test/return"})
	store := connectionStore{items: map[string]platformsettings.Setting{key: {Key: key, Value: saved, Version: 3}}}
	draft := encodedConnection(t, key, map[string]any{"MonoSweepEnabled": false, "MonoRedirectURL": "https://app.kredit.test/return"})
	prepared, err := PrepareConnectionUpdate(context.Background(), base, store, key, draft, false)
	if err != nil {
		t.Fatal(err)
	}
	result, err := ApplyStoredConnections(context.Background(), base, store, key, prepared)
	if err != nil {
		t.Fatal(err)
	}
	if result.MonoSweepEnabled || result.MonoSecretKey == "" || result.CollectionProvider != "mono-sweep" {
		t.Fatal("pause removed reconciliation credentials")
	}
}

func TestAdminConfigurationRejectsUnknownKeysInvalidSecretsAndOutages(t *testing.T) {
	key := "integrations.runtime.identity"
	raw := encodedConnection(t, key, map[string]any{"RealIdentity": true, "IdentityProvider": "connector", "IdentityProviderEndpoint": "https://identity.kredit.test", "IdentityProviderToken": "weak", "IdentityWebhookSecret": "weak", "IdentityApprovalReference": "approval"})
	if _, err := ApplyStoredConnections(context.Background(), productionBase(), connectionStore{}, key, raw); err == nil {
		t.Fatal("weak credentials accepted")
	}
	if _, err := ApplyStoredConnections(context.Background(), productionBase(), connectionStore{err: errors.New("database unavailable")}, "", nil); err == nil {
		t.Fatal("configuration outage ignored")
	}
	var encoded string
	_ = json.Unmarshal(raw, &encoded)
	var values map[string]any
	_ = json.Unmarshal([]byte(encoded), &values)
	delete(values, "IdentityProvider")
	values["DatabaseURL"] = "postgres://attacker"
	forged := encodedConnection(t, key, values)
	if _, err := platformsettings.DecodeRuntimeConnection(key, forged); err == nil {
		t.Fatal("unregistered infrastructure field accepted")
	}
}

func TestAdminScannerDisableKeepsStoredSecretButDisablesRuntime(t *testing.T) {
	key := "integrations.runtime.scanner"
	base := productionBase()
	raw := encodedConnection(t, key, map[string]any{"DocumentScannerEnabled": false, "DocumentScannerEndpoint": "https://scanner.kredit.test", "DocumentScannerToken": "scanner-fixture-token-0123456789abcdef"})
	result, err := ApplyStoredConnections(context.Background(), base, connectionStore{}, key, raw)
	if err != nil {
		t.Fatal(err)
	}
	if result.DocumentScannerEndpoint != "" || result.DocumentScannerToken != "" {
		t.Fatal("disabled scanner remains connected")
	}
}

func TestConnectorAddressChangeRequiresExplicitToken(t *testing.T) {
	key := "integrations.runtime.identity"
	existing := encodedConnection(t, key, map[string]any{"RealIdentity": true, "IdentityProvider": "connector", "IdentityProviderEndpoint": "https://old.kredit.test", "IdentityProviderToken": "saved-secret-token-0123456789abcdef", "IdentityWebhookSecret": "saved-webhook-secret-0123456789abcdef", "IdentityApprovalReference": "approval"})
	draft := encodedConnection(t, key, map[string]any{"RealIdentity": true, "IdentityProvider": "connector", "IdentityProviderEndpoint": "https://new.kredit.test", "IdentityApprovalReference": "approval"})
	store := connectionStore{items: map[string]platformsettings.Setting{key: {Key: key, Value: existing, Version: 1}}}
	if _, err := PrepareConnectionUpdate(context.Background(), productionBase(), store, key, draft, false); err == nil {
		t.Fatal("old access token would have been sent to a different endpoint")
	}
}

func TestDisabledRetargetCannotCarryCredentialsIntoLaterEnable(t *testing.T) {
	for _, test := range []struct{ key, enabled, endpoint, token string }{
		{"integrations.runtime.identity", "RealIdentity", "IdentityProviderEndpoint", "IdentityProviderToken"},
		{"integrations.runtime.collections", "RealCollections", "CollectionProviderEndpoint", "CollectionProviderToken"},
		{"integrations.runtime.scanner", "DocumentScannerEnabled", "DocumentScannerEndpoint", "DocumentScannerToken"},
	} {
		t.Run(test.key, func(t *testing.T) {
			existing := encodedConnection(t, test.key, map[string]any{test.enabled: true, test.endpoint: "https://old.kredit.test", test.token: "old-provider-credential"})
			store := connectionStore{items: map[string]platformsettings.Setting{test.key: {Key: test.key, Value: existing, Version: 1}}}
			draft := encodedConnection(t, test.key, map[string]any{test.enabled: false, test.endpoint: "https://new.kredit.test"})
			prepared, err := PrepareConnectionUpdate(t.Context(), productionBase(), store, test.key, draft, false)
			if err != nil {
				t.Fatal(err)
			}
			store.items[test.key] = platformsettings.Setting{Key: test.key, Value: prepared, Version: 2}
			enable := encodedConnection(t, test.key, map[string]any{test.enabled: true, test.endpoint: "https://new.kredit.test"})
			prepared, err = PrepareConnectionUpdate(t.Context(), productionBase(), store, test.key, enable, false)
			if err != nil {
				t.Fatal(err)
			}
			values, err := platformsettings.DecodeRuntimeConnection(test.key, prepared)
			if err != nil {
				t.Fatal(err)
			}
			var token string
			if err := json.Unmarshal(values[test.token], &token); err != nil || token != "" {
				t.Fatalf("credential followed disabled retarget: token present=%v err=%v", token != "", err)
			}
		})
	}
}

func TestRetainedAccountEditorProtectsCredentialsAndRetargets(t *testing.T) {
	key := "integrations.runtime.retained_collections"
	entry := RetainedCollectionConnection{Name: "original-account", Endpoint: "https://connector.example.test", Token: "access-012345678901234567890123456789", WebhookSecret: "signing-012345678901234567890123456789"}
	raw, _ := json.Marshal([]RetainedCollectionConnection{entry})
	base := productionBase()
	base.RetainedCollectionProviders = string(raw)
	visible := PublicConnectionValues(base, key)
	public, _ := json.Marshal(visible)
	if strings.Contains(string(public), entry.Token) || strings.Contains(string(public), entry.WebhookSecret) {
		t.Fatal("saved credentials exposed")
	}
	saved := encodedConnection(t, key, map[string]any{"RetainedCollectionProviders": string(raw)})
	store := connectionStore{items: map[string]platformsettings.Setting{key: {Key: key, Value: saved, Version: 1}}}
	draft := encodedConnection(t, key, map[string]any{"RetainedCollectionProviders": visible["RetainedCollectionProviders"]})
	prepared, err := PrepareConnectionUpdate(t.Context(), base, store, key, draft, false)
	if err != nil {
		t.Fatal(err)
	}
	applied, err := ApplyStoredConnections(t.Context(), base, store, key, prepared)
	if err != nil {
		t.Fatal(err)
	}
	entries, err := applied.RetainedCollections()
	if err != nil || len(entries) != 1 || entries[0] != entry {
		t.Fatal("blank editor fields lost original account", err)
	}
	entry.Endpoint = "https://other.example.test"
	entry.Token = ""
	entry.WebhookSecret = ""
	changed, _ := json.Marshal([]RetainedCollectionConnection{entry})
	draft = encodedConnection(t, key, map[string]any{"RetainedCollectionProviders": string(changed)})
	if _, err = PrepareConnectionUpdate(t.Context(), base, store, key, draft, false); err == nil {
		t.Fatal("old credentials retargeted to another address")
	}
	original := entries[0]
	base.CollectionProvider = original.Name
	base.CollectionProviderEndpoint = original.Endpoint
	base.CollectionProviderToken = original.Token
	base.CollectionWebhookSecret = original.WebhookSecret
	if _, err = base.RetainedCollections(); err != nil {
		t.Fatal("cannot retain current account before switching", err)
	}
	base.CollectionProviderToken += "changed"
	if _, err = base.RetainedCollections(); err == nil {
		t.Fatal("same identity points to two different accounts")
	}
}

func TestRetainedMonoAccountCanSurviveAccountSwitch(t *testing.T) {
	c := Config{Environment: "production", MonoAccountName: "mono-sweep", MonoSecretKey: "live_sk_0123456789abcdef0123456789abcdef", MonoWebhookSecret: "webhook-0123456789abcdef0123456789abcdef", CollectionProvider: "mono-sweep"}
	accounts := []RetainedCollectionConnection{{Adapter: "mono", Name: c.MonoAccount(), Endpoint: "https://api.withmono.com", Token: c.MonoSecretKey, WebhookSecret: c.MonoWebhookSecret, Partial: true}}
	raw, _ := json.Marshal(accounts)
	c.RetainedCollectionProviders = string(raw)
	if _, err := c.RetainedCollections(); err != nil {
		t.Fatal(err)
	}
	c.MonoAccountName = "mono-new-account"
	c.CollectionProvider = c.MonoAccount()
	c.MonoSecretKey = "live_sk_another0123456789abcdef0123456789"
	saved, err := c.RetainedCollections()
	if err != nil || len(saved) != 1 || saved[0].Name != "mono-sweep" {
		t.Fatal("original Mono route lost", err)
	}
	visible := PublicConnectionValues(c, "integrations.runtime.retained_collections")
	encoded, _ := json.Marshal(visible)
	if strings.Contains(string(encoded), accounts[0].Token) || strings.Contains(string(encoded), accounts[0].WebhookSecret) {
		t.Fatal("saved native account credentials exposed")
	}
}
