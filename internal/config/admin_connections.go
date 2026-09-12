package config

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strings"

	"kredit/internal/platformsettings"

	"github.com/jackc/pgx/v5"
)

// ApplyStoredConnections is shared by API and worker startup. Credentials remain
// encrypted in storage and never become environment variables or config files.
func ApplyStoredConnections(ctx context.Context, base Config, settings platformsettings.Service, candidateKey string, candidate json.RawMessage) (Config, error) {
	result := base
	result.AdminConnectionVersions = map[string]int{}
	keys := make([]string, 0, len(platformsettings.RuntimeConnections))
	for key := range platformsettings.RuntimeConnections {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		var raw json.RawMessage
		if key == candidateKey {
			raw = candidate
		} else {
			item, err := settings.Get(ctx, key, true)
			if errors.Is(err, pgx.ErrNoRows) {
				continue
			}
			if err != nil {
				return base, fmt.Errorf("saved %s configuration could not be read", platformsettings.RuntimeConnections[key].Title)
			}
			raw = item.Value
			result.AdminConnectionVersions[key] = item.Version
		}
		values, err := platformsettings.DecodeRuntimeConnection(key, raw)
		if err != nil {
			return base, err
		}
		target := reflect.ValueOf(&result).Elem()
		for name, value := range values {
			// DecodeRuntimeConnection has already checked the explicit allowlist.
			field := target.FieldByName(name)
			if !field.IsValid() || !field.CanAddr() {
				return base, errors.New("unsupported connection field")
			}
			if err := json.Unmarshal(value, field.Addr().Interface()); err != nil {
				return base, errors.New("invalid connection value")
			}
		}
		if key == "integrations.runtime.scanner" && !result.DocumentScannerEnabled {
			result.DocumentScannerEndpoint = ""
			result.DocumentScannerToken = ""
		}
		if key == "integrations.runtime.mono" {
			if result.MonoSecretKey != "" {
				if result.RealCollections && result.MonoSweepEnabled {
					return base, errors.New("pause Mono collections before enabling another collection connector")
				}
				if !result.RealCollections {
					result.CollectionProvider = result.MonoAccount()
				}
			} else if result.CollectionProvider == result.MonoAccount() {
				result.CollectionProvider = "mock"
			}
		}
	}
	// Validate saved reconciliation credentials without pretending paused
	// collections are enabled. Approval gates apply to new money movement.

	if err := result.Validate(); err != nil {
		return base, err
	}
	return result, nil
}

// PublicConnectionValues supports editing ordinary fields without exposing
// passwords, tokens, signing secrets or any unregistered deployment setting.
func PublicConnectionValues(c Config, key string) map[string]any {
	values := map[string]any{}
	source := reflect.ValueOf(c)
	for _, field := range platformsettings.RuntimeConnections[key].Fields {
		if field.Kind == "password" {
			continue
		}
		if field.Kind == "retained" {
			entries, err := c.RetainedCollections()
			if field.Key == "RetainedIdentityProviders" {
				entries, err = c.RetainedIdentities()
			}
			if err != nil {
				continue
			}
			if entries == nil {
				entries = []RetainedCollectionConnection{}
			}
			for i := range entries {
				entries[i].Token = ""
				entries[i].WebhookSecret = ""
			}
			raw, _ := json.Marshal(entries)
			values[field.Key] = string(raw)
			continue
		}
		if field.Key == "DocumentScannerEnabled" {
			values[field.Key] = c.DocumentScannerEndpoint != ""
			continue
		}
		value := source.FieldByName(field.Key)
		if value.IsValid() {
			values[field.Key] = value.Interface()
		}
	}
	return values
}

// PrepareConnectionUpdate keeps existing passwords when the owner leaves their
// write-only fields blank. All ordinary fields still form a full replacement.
func PrepareConnectionUpdate(ctx context.Context, base Config, settings platformsettings.Service, key string, raw json.RawMessage, clearCredentials bool) (json.RawMessage, error) {
	values, err := platformsettings.DecodeRuntimeConnection(key, raw)
	if err != nil {
		return nil, err
	}
	previous := map[string]json.RawMessage{}
	existing, readErr := settings.Get(ctx, key, true)
	if readErr == nil {
		previous, err = platformsettings.DecodeRuntimeConnection(key, existing.Value)
		if err != nil {
			return nil, err
		}
	} else if !errors.Is(readErr, pgx.ErrNoRows) {
		return nil, errors.New("existing connection could not be read")
	}
	if key == "integrations.runtime.retained_collections" || key == "integrations.runtime.retained_identity" {
		retainedField := "RetainedCollectionProviders"
		if key == "integrations.runtime.retained_identity" {
			retainedField = "RetainedIdentityProviders"
		}
		var encoded, oldEncoded string
		_ = json.Unmarshal(values[retainedField], &encoded)
		if raw, ok := previous[retainedField]; ok {
			_ = json.Unmarshal(raw, &oldEncoded)
		} else {
			oldEncoded = reflect.ValueOf(base).FieldByName(retainedField).String()
		}
		var entries, old []RetainedCollectionConnection
		if json.Unmarshal([]byte(encoded), &entries) != nil || len(entries) > 8 {
			return nil, errors.New("provide at most eight saved collection accounts")
		}
		if strings.TrimSpace(oldEncoded) != "" && json.Unmarshal([]byte(oldEncoded), &old) != nil {
			return nil, errors.New("saved collection accounts could not be read")
		}
		for i := range entries {
			for _, prior := range old {
				if entries[i].Name != prior.Name {
					continue
				}
				if (entries[i].Endpoint != prior.Endpoint || entries[i].Adapter != prior.Adapter) && (entries[i].Token == "" || (entries[i].Adapter != "mono" && entries[i].WebhookSecret == "")) {
					return nil, errors.New("provide new credentials when changing a saved account address")
				}
				if entries[i].Endpoint == prior.Endpoint && entries[i].Adapter == prior.Adapter && !clearCredentials {
					if entries[i].Token == "" {
						entries[i].Token = prior.Token
					}
					if entries[i].WebhookSecret == "" {
						entries[i].WebhookSecret = prior.WebhookSecret
					}
				}
			}
		}
		merged, _ := json.Marshal(entries)
		values[retainedField], _ = json.Marshal(string(merged))
	}
	source := reflect.ValueOf(base)
	// A different recipient must not receive an existing credential implicitly.
	tokenField, enabledField, endpointField := "", "", ""
	endpointChanged := false
	switch key {
	case "integrations.runtime.settlement":
		tokenField, enabledField, endpointField = "SettlementToken", "SettlementEnabled", "SettlementEndpoint"
	case "integrations.runtime.identity":
		tokenField, enabledField, endpointField = "IdentityProviderToken", "RealIdentity", "IdentityProviderEndpoint"
	case "integrations.runtime.collections":
		tokenField, enabledField, endpointField = "CollectionProviderToken", "RealCollections", "CollectionProviderEndpoint"
	case "integrations.runtime.storage":
		tokenField, enabledField, endpointField = "ObjectStorageSecretKey", "", "ObjectStorageEndpoint"
	case "integrations.runtime.scanner":
		tokenField, enabledField, endpointField = "DocumentScannerToken", "DocumentScannerEnabled", "DocumentScannerEndpoint"
	}
	if tokenField != "" {
		var enabled bool
		var endpoint, token, oldEndpoint string
		if enabledField == "" {
			enabled = true
		} else {
			_ = json.Unmarshal(values[enabledField], &enabled)
		}
		_ = json.Unmarshal(values[endpointField], &endpoint)
		_ = json.Unmarshal(values[tokenField], &token)
		if prior, ok := previous[endpointField]; ok {
			_ = json.Unmarshal(prior, &oldEndpoint)
		} else {
			oldEndpoint = source.FieldByName(endpointField).String()
		}
		endpointChanged = endpoint != oldEndpoint
		if key == "integrations.runtime.storage" && endpointChanged {
			var accessKey string
			_ = json.Unmarshal(values["ObjectStorageAccessKey"], &accessKey)
			if accessKey == "" {
				return nil, errors.New("enter the access key when changing the storage address")
			}
		}
		if enabled && endpointChanged && token == "" {
			return nil, errors.New("enter the access token when changing an enabled connector address")
		}
	}
	for _, field := range platformsettings.RuntimeConnections[key].Fields {
		if field.Kind != "password" {
			continue
		}
		var value string
		_ = json.Unmarshal(values[field.Key], &value)
		if value != "" || clearCredentials {
			continue
		}
		// A disabled retarget must not carry the old recipient's access token
		// into storage, where a later enable could otherwise reuse it silently.
		if field.Key == tokenField && endpointChanged {
			continue
		}
		if saved, ok := previous[field.Key]; ok {
			values[field.Key] = saved
		} else {
			values[field.Key], _ = json.Marshal(source.FieldByName(field.Key).String())
		}
	}
	encoded, err := json.Marshal(values)
	if err != nil {
		return nil, err
	}
	return json.Marshal(string(encoded))
}
