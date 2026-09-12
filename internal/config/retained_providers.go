package config

import (
	"encoding/json"
	"errors"
	"regexp"
	"strings"
)

type RetainedCollectionConnection struct {
	Adapter       string `json:"adapter,omitempty"`
	Partial       bool   `json:"partial,omitempty"`
	Name          string `json:"name"`
	Endpoint      string `json:"endpoint"`
	Token         string `json:"token"`
	WebhookSecret string `json:"webhook_secret"`
}

// Retained connections are for existing collections only. No new debit chooses
// one of these connections as an automatic fallback.
func (c Config) RetainedCollections() ([]RetainedCollectionConnection, error) {
	if strings.TrimSpace(c.RetainedCollectionProviders) == "" {
		return nil, nil
	}
	var connections []RetainedCollectionConnection
	if json.Unmarshal([]byte(c.RetainedCollectionProviders), &connections) != nil || len(connections) > 8 {
		return nil, errors.New("COLLECTION_RETAINED_PROVIDERS must be a JSON array of at most eight connections")
	}
	seen := map[string]bool{}
	for _, connection := range connections {
		if (connection.Adapter != "" && connection.Adapter != "connector" && connection.Adapter != "mono") || !regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]{1,63}$`).MatchString(connection.Name) || (connection.Name == "mono-sweep" && connection.Adapter != "mono") || strings.Contains(strings.ToLower(connection.Name), "mock") || seen[connection.Name] {
			return nil, errors.New("saved collection account names must be unique and use the correct adapter")
		}
		if connection.Adapter == "mono" {
			if c.Environment == "production" && !strings.HasPrefix(connection.Token, "live_sk_") {
				return nil, errors.New("saved production Mono accounts require live keys")
			}
			if c.Environment != "production" && !strings.HasPrefix(connection.Token, "test_sk_") {
				return nil, errors.New("saved test Mono accounts require sandbox keys")
			}
			if connection.Name == c.MonoAccount() && c.MonoSecretKey != "" && (connection.Endpoint != "https://api.withmono.com" || connection.Token != c.MonoSecretKey || connection.WebhookSecret != c.MonoWebhookSecret) {
				return nil, errors.New("save the current Mono account with its exact connection before switching")
			}
		}
		// Save the current connector before switching. The redundant route must be
		// identical, so the same identity can never point at two accounts at once.
		if connection.Adapter != "mono" && connection.Name == c.CollectionProvider && (connection.Endpoint != c.CollectionProviderEndpoint || connection.Token != c.CollectionProviderToken || connection.WebhookSecret != c.CollectionWebhookSecret) {
			return nil, errors.New("the saved active account must match its current connection exactly")
		}
		if err := validateProductionURL("retained collection endpoint", connection.Endpoint); err != nil {
			return nil, err
		}
		if len(connection.Token) < 32 || len(connection.WebhookSecret) < 32 || strings.ContainsAny(connection.Token+connection.WebhookSecret, "\r\n") {
			return nil, errors.New("retained collection credentials require at least 32 characters and no line breaks")
		}
		seen[connection.Name] = true
	}
	return connections, nil
}

// RetainedIdentities preserves access to checks opened with a previous connector.
func (c Config) RetainedIdentities() ([]RetainedCollectionConnection, error) {
	if strings.TrimSpace(c.RetainedIdentityProviders) == "" {
		return nil, nil
	}
	var entries []RetainedCollectionConnection
	if json.Unmarshal([]byte(c.RetainedIdentityProviders), &entries) != nil || len(entries) > 8 {
		return nil, errors.New("provide at most eight saved identity accounts")
	}
	seen := map[string]bool{}
	for _, a := range entries {
		if !regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]{1,63}$`).MatchString(a.Name) || seen[a.Name] || strings.Contains(strings.ToLower(a.Name), "mock") {
			return nil, errors.New("identity account names must be unique")
		}
		if c.Environment == "production" && a.Adapter == "mono" && !strings.HasPrefix(a.Token, "live_sk_") {
			return nil, errors.New("saved production Mono lookup accounts require live secret keys")
		}
		if a.Adapter != "" && a.Adapter != "connector" && a.Adapter != "mono" {
			return nil, errors.New("identity adapter must be mono or connector")
		}
		if err := validateProductionURL("saved identity endpoint", a.Endpoint); err != nil {
			return nil, err
		}
		if len(a.Token) < 32 || strings.ContainsAny(a.Token+a.WebhookSecret, "\r\n") || (a.Adapter != "mono" && len(a.WebhookSecret) < 32) {
			return nil, errors.New("saved identity account credentials are incomplete")
		}
		adapter := a.Adapter
		if adapter == "" {
			adapter = "connector"
		}
		active := c.IdentityAdapter
		if active == "" {
			active = "connector"
		}
		if a.Name == c.IdentityProvider && (a.Endpoint != c.IdentityProviderEndpoint || a.Token != c.IdentityProviderToken || a.WebhookSecret != c.IdentityWebhookSecret || adapter != active) {
			return nil, errors.New("save the active identity account with its exact connection")
		}
		seen[a.Name] = true
	}
	return entries, nil
}

func (c Config) validateMonoAccountName() error {
	if !regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]{1,63}$`).MatchString(c.MonoAccount()) || strings.Contains(strings.ToLower(c.MonoAccount()), "mock") {
		return errors.New("mono account name must be a unique bounded identifier")
	}
	return nil
}
