package config

import (
	"encoding/json"
	"errors"

	"kredit/internal/platformsettings"
)

// PublicStoredConnectionValues projects a complete saved connection for the
// settings editor. Password fields and nested retained credentials never enter
// the result, including before a newly saved connection has been applied.
func PublicStoredConnectionValues(key string, raw json.RawMessage) (map[string]any, error) {
	values, err := platformsettings.DecodeRuntimeConnection(key, raw)
	if err != nil {
		return nil, err
	}
	result := make(map[string]any)
	for _, field := range platformsettings.RuntimeConnections[key].Fields {
		switch field.Kind {
		case "password":
			continue
		case "retained":
			var encoded string
			if err := json.Unmarshal(values[field.Key], &encoded); err != nil {
				return nil, errors.New("saved account configuration is invalid")
			}
			var entries []RetainedCollectionConnection
			if err := json.Unmarshal([]byte(encoded), &entries); err != nil || len(entries) > 8 {
				return nil, errors.New("saved account configuration is invalid")
			}
			visible, err := publicRetainedAccounts(entries)
			if err != nil {
				return nil, err
			}
			result[field.Key] = visible
		default:
			// DecodeRuntimeConnection already validated the exact JSON type. Keeping
			// it as RawMessage avoids an unnecessary conversion of numbers to float64.
			result[field.Key] = append(json.RawMessage(nil), values[field.Key]...)
		}
	}
	return result, nil
}

// The public representation is an allowlist, not a copy with selected secrets
// blanked. Adding a private field to RetainedCollectionConnection cannot expose
// it accidentally through either saved or environment-backed editor values.
func publicRetainedAccounts(entries []RetainedCollectionConnection) (string, error) {
	type publicAccount struct {
		ContractCode string `json:"contract_code,omitempty"`
		Adapter      string `json:"adapter,omitempty"`
		Partial      bool   `json:"partial,omitempty"`
		Name         string `json:"name"`
		Endpoint     string `json:"endpoint"`
	}
	visible := make([]publicAccount, 0, len(entries))
	for _, entry := range entries {
		visible = append(visible, publicAccount{entry.ContractCode, entry.Adapter, entry.Partial, entry.Name, entry.Endpoint})
	}
	raw, err := json.Marshal(visible)
	if err != nil {
		return "", errors.New("public account configuration could not be encoded")
	}
	return string(raw), nil
}
