package notifications

import (
	"context"
	"encoding/json"
	"errors"

	"kredit/internal/platformsettings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ConfiguredProvider resolves configuration for each delivery, across API and
// worker processes. A read/decryption failure never reactivates old credentials.
type ConfiguredProvider struct {
	channel       string
	settings      platformsettings.Service
	fallback      Provider
	pool          *pgxpool.Pool
	submissionKey []byte
	deployment    *platformsettings.NotificationConnector
	encryptor     *platformsettings.Encryptor
}

func NewConfiguredProvider(channel string, settings platformsettings.Service, fallback Provider) *ConfiguredProvider {
	return &ConfiguredProvider{channel: channel, settings: settings, fallback: fallback}
}
func (p *ConfiguredProvider) WithPersistence(pool *pgxpool.Pool, key []byte) *ConfiguredProvider {
	p.pool = pool
	p.submissionKey = append([]byte(nil), key...)
	return p
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

// WithDeployment retains the deployment connection for new work when no admin
// override exists. Saved message routes keep their original encrypted connection.
func (p *ConfiguredProvider) WithDeployment(config platformsettings.NotificationConnector, encryptionKey string) *ConfiguredProvider {
	p.deployment = &config
	p.encryptor = platformsettings.NewEncryptor(encryptionKey)
	return p
}
func (p *ConfiguredProvider) Send(ctx context.Context, message Message) (string, error) {
	config, err := ResolveConnector(ctx, p.settings, p.channel)
	if err != nil {
		return "", err
	}
	if config == nil {
		config = p.deployment
	}
	if config == nil {
		if p.fallback == nil {
			return "", errors.New("notification connector is not configured")
		}
		return sendWithSubmission(ctx, p.pool, p.submissionKey, p.fallback, message)
	}
	if !config.Enabled {
		return "", errors.New("notification channel is disabled")
	}
	// Validate before saving a route. A malformed adapter must not trap new work.
	if _, err = NewAdapter(p.channel, *config); err != nil {
		return "", err
	}
	config, err = p.messageRoute(ctx, message.EventID, config)
	if err != nil {
		return "", err
	}
	provider, err := NewAdapter(p.channel, *config)
	if err != nil {
		return "", err
	}
	return sendWithSubmission(ctx, p.pool, p.submissionKey, provider, message)
}

// DeliveryStatusForEvent avoids collisions when two providers return the same ID.
func (p *ConfiguredProvider) DeliveryStatusForEvent(ctx context.Context, event, id string) (DeliveryStatus, error) {
	config, err := p.messageRoute(ctx, event, nil)
	if err != nil {
		return DeliveryStatus{}, err
	}
	provider, err := NewAdapter(p.channel, *config)
	if err != nil {
		return DeliveryStatus{}, err
	}
	reader, ok := provider.(DeliveryStatusProvider)
	if !ok {
		return DeliveryStatus{}, errors.New("this adapter does not support authenticated delivery lookup")
	}
	return reader.DeliveryStatus(ctx, id)
}
func (p *ConfiguredProvider) DeliveryStatus(ctx context.Context, id string) (DeliveryStatus, error) {
	return DeliveryStatus{}, errors.New("delivery lookup requires the original notification event")
}
