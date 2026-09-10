package config

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"sort"

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
				if result.RealCollections {
					return base, errors.New("disable the other collection connector before configuring Mono")
				}
				result.CollectionProvider = "mono-sweep"
			} else if result.CollectionProvider == "mono-sweep" {
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
	source := reflect.ValueOf(base)
	// A different recipient must not receive an existing credential implicitly.
	tokenField, enabledField, endpointField := "", "", ""
	endpointChanged := false
	switch key {
	case "integrations.runtime.identity":
		tokenField, enabledField, endpointField = "IdentityProviderToken", "RealIdentity", "IdentityProviderEndpoint"
	case "integrations.runtime.collections":
		tokenField, enabledField, endpointField = "CollectionProviderToken", "RealCollections", "CollectionProviderEndpoint"
	case "integrations.runtime.scanner":
		tokenField, enabledField, endpointField = "DocumentScannerToken", "DocumentScannerEnabled", "DocumentScannerEndpoint"
	}
	if tokenField != "" {
		var enabled bool
		var endpoint, token, oldEndpoint string
		_ = json.Unmarshal(values[enabledField], &enabled)
		_ = json.Unmarshal(values[endpointField], &endpoint)
		_ = json.Unmarshal(values[tokenField], &token)
		if prior, ok := previous[endpointField]; ok {
			_ = json.Unmarshal(prior, &oldEndpoint)
		} else {
			oldEndpoint = source.FieldByName(endpointField).String()
		}
		endpointChanged = endpoint != oldEndpoint
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
