package notifications

import (
	"errors"
	"kredit/internal/platformsettings"
)

// NewAdapter is the boundary between provider protocols and notification work.
// Additional vendors implement Provider and, where available, DeliveryStatusProvider.
func NewAdapter(channel string, config platformsettings.NotificationConnector) (Provider, error) {
	adapter := config.Adapter
	if adapter == "" {
		adapter = DefaultAdapter(channel)
	}
	switch adapter {
	case "sendly":
		if channel != ChannelEmail {
			return nil, errors.New("sendly adapter requires email")
		}
		return NewSendlyProvider(config.Endpoint, config.Token, config.From)
	case "mesaj":
		if channel != ChannelSMS {
			return nil, errors.New("mesaj adapter requires SMS")
		}
		return NewMesajProvider(config.Endpoint, config.Token, config.From)
	case "meta":
		if channel != ChannelWhatsApp {
			return nil, errors.New("meta adapter requires WhatsApp")
		}
		return NewMetaProvider(config)
	case "connector":
		return NewWebhookProvider(channel, config.Endpoint, config.Token)
	default:
		return nil, errors.New("notification adapter is not installed")
	}
}
func DefaultAdapter(channel string) string {
	switch channel {
	case ChannelEmail:
		return "sendly"
	case ChannelSMS:
		return "mesaj"
	default:
		return "connector"
	}
}
