package platformsettings

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/mail"
	"net/url"
	"regexp"
	"strings"
	"time"
)

const (
	CategoryFeatures   = "features"
	CategoryGovernance = "governance"

	GovernanceSoloOwner     = "solo_owner"
	GovernanceDelegatedTeam = "delegated_team"
)

type Setting struct {
	ConnectionFields  []ConnectionField `json:"connection_fields,omitempty"`
	ConnectionValues  map[string]any    `json:"connection_values,omitempty"`
	RequiresRestart   bool              `json:"requires_restart,omitempty"`
	AppliedVersion    int               `json:"applied_version,omitempty"`
	ConnectionState   string            `json:"connection_state,omitempty"`
	Key               string            `json:"key"`
	Category          string            `json:"category"`
	Value             json.RawMessage   `json:"value"`
	IsSecret          bool              `json:"is_secret"`
	SecretFingerprint string            `json:"secret_fingerprint,omitempty"`
	Description       string            `json:"description"`
	Version           int               `json:"version"`
	UpdatedAt         time.Time         `json:"updated_at"`
	UpdatedBy         string            `json:"updated_by,omitempty"`
	Reason            string            `json:"reason,omitempty"`
}

type SettingHistory struct {
	ID         string          `json:"id"`
	Key        string          `json:"key"`
	OldValue   json.RawMessage `json:"old_value,omitempty"`
	NewValue   json.RawMessage `json:"new_value"`
	Version    int             `json:"version"`
	Action     string          `json:"action"`
	ActorID    string          `json:"actor_id,omitempty"`
	Reason     string          `json:"reason"`
	RecordedAt time.Time       `json:"recorded_at"`
}

type Governance struct {
	Mode      string    `json:"mode"`
	UpdatedAt time.Time `json:"updated_at"`
	UpdatedBy string    `json:"updated_by,omitempty"`
	Reason    string    `json:"reason"`
}

type SettingMeta struct {
	Category    string
	IsSecret    bool
	Description string
	Validate    func(raw json.RawMessage) error
}

var KnownSettings = map[string]SettingMeta{
	"features.system_acceptance": {Category: CategoryFeatures, Description: "Recognize eligible delivered sales after the waiting period using separate system evidence. First-time buyers and delivery issues still require a response.", Validate: validateBool},
	"automation.system_acceptance_hours": {Category: CategoryFeatures, Description: "Minimum hours after confirmed delivery of the goods notice (72–720). Applies to the next recognition attempt.", Validate: func(raw json.RawMessage) error {
		var hours int64
		if err := json.Unmarshal(raw, &hours); err != nil || hours < 72 || hours > 720 {
			return errors.New("waiting period must be a whole number between 72 and 720 hours")
		}
		return nil
	}},
	// Every key here has a consumer, and the consumer is named beside it. A
	// switch that writes a row and changes no behaviour is worse than no switch:
	// it tells the owner something is off when it is on. Keys for adapters that
	// do not exist, launch banners, penalty fees and session limits that nothing
	// read were removed rather than left as decoration, and so were the four
	// feature flags whose features were never built.
	//
	// internal/web/credit_handlers.go, listTradeLines
	"features.trade_lines": {
		Category:    CategoryFeatures,
		Description: "Customer limits: let a customer draw against an agreed limit instead of one sale at a time",
		Validate:    validateBool,
	},
	// internal/web/credit_handlers.go, requestDrawdown
	"features.drawdowns": {
		Category:    CategoryFeatures,
		Description: "Let a customer request goods against their limit",
		Validate:    validateBool,
	},
	// internal/web/credit_handlers.go, openDispute and addEvidence
	"features.disputes": {
		Category:    CategoryFeatures,
		Description: "Let a customer report a problem with a sale",
		Validate:    validateBool,
	},
}

func validateBool(raw json.RawMessage) error {
	var b bool
	if err := json.Unmarshal(raw, &b); err != nil {
		return errors.New("value must be a boolean (true or false)")
	}
	return nil
}

func ValidateKeyAndValue(key string, raw json.RawMessage) (SettingMeta, error) {
	meta, ok := KnownSettings[key]
	if !ok {
		return SettingMeta{}, fmt.Errorf("unknown setting key %q", key)
	}
	if strings.TrimSpace(string(raw)) == "null" {
		return meta, errors.New("setting value cannot be null")
	}
	if meta.Validate != nil {
		if err := meta.Validate(raw); err != nil {
			return meta, fmt.Errorf("validation failed for %s: %w", key, err)
		}
	}
	return meta, nil
}

// NotificationConnector is stored as one encrypted setting so endpoint and token
// changes become visible atomically to every API and worker process.
type NotificationConnector struct {
	VerifyToken            string `json:"verify_token,omitempty"`
	Language               string `json:"language,omitempty"`
	AuthenticationTemplate string `json:"authentication_template,omitempty"`
	UtilityTemplate        string `json:"utility_template,omitempty"`
	MarketingTemplate      string `json:"marketing_template,omitempty"`
	Adapter                string `json:"adapter,omitempty"`
	From                   string `json:"from,omitempty"`
	WebhookSecret          string `json:"webhook_secret,omitempty"`
	Enabled                bool   `json:"enabled"`
	Endpoint               string `json:"endpoint"`
	Token                  string `json:"token"`
}

func init() {
	for _, channel := range []string{"email", "sms", "whatsapp"} {
		KnownSettings["integrations.notifications."+channel] = SettingMeta{
			Category: "integrations", IsSecret: true,
			Description: channel + " delivery connector (takes effect on the next delivery)",
			Validate: func(raw json.RawMessage) error {
				if err := validateNotificationConnector(raw); err != nil {
					return err
				}

				var encoded string
				var config NotificationConnector
				_ = json.Unmarshal(raw, &encoded)
				_ = json.Unmarshal([]byte(encoded), &config)
				if !config.Enabled {
					return nil
				}
				adapter := config.Adapter
				if adapter == "" {
					adapter = "connector"
					if channel == "email" {
						adapter = "sendly"
					}
					if channel == "sms" {
						adapter = "mesaj"
					}
				}
				if adapter == "meta" && channel == "whatsapp" {
					if !regexp.MustCompile(`^https://graph\.facebook\.com/v[0-9]+\.[0-9]+/[0-9]+/messages$`).MatchString(config.Endpoint) || len(config.WebhookSecret) < 32 || len(config.VerifyToken) < 32 || !regexp.MustCompile(`^[a-z]{2,3}(_[A-Z]{2})?$`).MatchString(config.Language) || !regexp.MustCompile(`^[a-z0-9_]{1,512}$`).MatchString(config.AuthenticationTemplate) || !regexp.MustCompile(`^[a-z0-9_]{1,512}$`).MatchString(config.UtilityTemplate) || !regexp.MustCompile(`^[a-z0-9_]{1,512}$`).MatchString(config.MarketingTemplate) {
						return errors.New("complete the Meta endpoint, app secret, verification token, language and approved template names")
					}
					return nil
				}
				if adapter == "connector" {
					return nil
				}
				if (adapter != "sendly" || channel != "email") && (adapter != "mesaj" || channel != "sms") {
					return errors.New("select an installed adapter for this channel")
				}
				if channel == "sms" {
					if config.Endpoint != "https://api.mesaj.cloud:25274/client/sms/send/bulk" || !regexp.MustCompile(`^[A-Za-z0-9]{1,11}$`).MatchString(config.From) {
						return errors.New("SMS requires the Mesaj send endpoint and an approved sender ID")
					}
					return nil
				}
				sender, err := mail.ParseAddress(config.From)
				if config.Endpoint != "https://api.sendlyai.com/v1/messages" || !strings.HasPrefix(config.Token, "sk_live_") || len(config.WebhookSecret) < 32 || err != nil || !strings.HasSuffix(strings.ToLower(sender.Address), "@kredit.ng") {
					return errors.New("email requires Sendly's endpoint, a live API key, its webhook secret, and a kredit.ng sender")
				}
				return nil
			},
		}
	}
}

func validateNotificationConnector(raw json.RawMessage) error {
	var encoded string
	if json.Unmarshal(raw, &encoded) != nil {
		return errors.New("connector must be an encoded configuration")
	}
	var config NotificationConnector
	if json.Unmarshal([]byte(encoded), &config) != nil {
		return errors.New("invalid connector configuration")
	}
	if !config.Enabled {
		return nil
	}
	endpoint, err := url.Parse(config.Endpoint)
	if err != nil || endpoint.Scheme != "https" || endpoint.Hostname() == "" || endpoint.User != nil || endpoint.Fragment != "" || endpoint.RawQuery != "" || endpoint.ForceQuery {
		return errors.New("connector endpoint must be an HTTPS URL without embedded credentials, query parameters or a fragment")
	}
	if strings.TrimSpace(config.Token) == "" || strings.ContainsAny(config.Token, "\r\n") {
		return errors.New("connector token is required and must be a single line")
	}
	return nil
}

func setConnectionState(item *Setting, plaintext string) {
	if !strings.HasPrefix(item.Key, "integrations.notifications.") {
		return
	}
	item.ConnectionState = "unavailable"
	var config NotificationConnector
	if json.Unmarshal([]byte(plaintext), &config) != nil {
		return
	}
	item.ConnectionState = "disabled"
	if config.Enabled {
		item.ConnectionState = "enabled_unverified"
	}
}
