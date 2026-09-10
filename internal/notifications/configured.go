package notifications

import (
	"context"
	"encoding/json"
	"errors"

	"kredit/internal/platformsettings"

	"github.com/jackc/pgx/v5"
)

// ConfiguredProvider resolves configuration for each delivery, across API and
// worker processes. A read/decryption failure never reactivates old credentials.
type ConfiguredProvider struct {
	channel  string
	settings platformsettings.Service
	fallback Provider
}

func NewConfiguredProvider(channel string, settings platformsettings.Service, fallback Provider) *ConfiguredProvider {
	return &ConfiguredProvider{channel: channel, settings: settings, fallback: fallback}
}
func (p *ConfiguredProvider) Channel() string { return p.channel }

// ResolveConnector returns the live admin override, or nil when deployment
// configuration applies. A failed read never falls back to stale credentials.
func ResolveConnector(ctx context.Context, settings platformsettings.Service, channel string) (*platformsettings.NotificationConnector, error) {
	if settings == nil {
		return nil, nil
	}
	key := "integrations.notifications." + channel
	setting, err := settings.Get(ctx, key, true)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, errors.New("notification connector configuration is unavailable")
	}
	var encoded string
	var config platformsettings.NotificationConnector
	if json.Unmarshal(setting.Value, &encoded) != nil || json.Unmarshal([]byte(encoded), &config) != nil {
		return nil, errors.New("notification connector configuration is invalid")
	}
	if _, err := platformsettings.ValidateKeyAndValue(key, setting.Value); err != nil {
		return nil, errors.New("notification connector configuration is invalid")
	}
	return &config, nil
}

func (p *ConfiguredProvider) Send(ctx context.Context, message Message) (string, error) {
	config, err := ResolveConnector(ctx, p.settings, p.channel)
	if err != nil {
		return "", err
	}
	if config != nil {
		if !config.Enabled {
			return "", errors.New("notification connector is disabled")
		}
		provider, err := NewWebhookProvider(p.channel, config.Endpoint, config.Token)
		if err != nil {
			return "", err
		}
		return provider.Send(ctx, message)
	}
	if p.fallback == nil {
		return "", errors.New("notification connector is not configured")
	}
	return p.fallback.Send(ctx, message)
}
