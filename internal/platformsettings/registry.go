package platformsettings

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

const (
	CategoryLaunch        = "launch"
	CategoryFeatures      = "features"
	CategoryIntegrations  = "integrations"
	CategorySecurity      = "security"
	CategoryKYC           = "kyc"
	CategoryFees          = "fees"
	CategoryNotifications = "notifications"
	CategoryGovernance    = "governance"

	GovernanceSoloOwner     = "solo_owner"
	GovernanceDelegatedTeam = "delegated_team"

	LaunchModePreLaunch     = "pre_launch"
	LaunchModePrivateLaunch = "private_launch"
	LaunchModePublicLaunch  = "public_launch"

	IntegrationStatusUnconfigured = "unconfigured"
	IntegrationStatusConfigured   = "configured"
	IntegrationStatusVerified     = "verified"
	IntegrationStatusDegraded     = "degraded"
	IntegrationStatusDisabled     = "disabled"
)

type Setting struct {
	Key               string          `json:"key"`
	Category          string          `json:"category"`
	Value             json.RawMessage `json:"value"`
	IsSecret          bool            `json:"is_secret"`
	SecretFingerprint string          `json:"secret_fingerprint,omitempty"`
	Description       string          `json:"description"`
	Version           int             `json:"version"`
	UpdatedAt         time.Time       `json:"updated_at"`
	UpdatedBy         string          `json:"updated_by,omitempty"`
	Reason            string          `json:"reason,omitempty"`
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
	// Launch
	"launch.mode": {
		Category:    CategoryLaunch,
		Description: "Operational launch phase: pre_launch, private_launch, public_launch",
		Validate: func(raw json.RawMessage) error {
			var s string
			if err := json.Unmarshal(raw, &s); err != nil {
				return errors.New("must be a valid string")
			}
			switch s {
			case LaunchModePreLaunch, LaunchModePrivateLaunch, LaunchModePublicLaunch:
				return nil
			default:
				return fmt.Errorf("invalid launch mode %q (must be pre_launch, private_launch, or public_launch)", s)
			}
		},
	},
	"launch.banner_enabled": {
		Category:    CategoryLaunch,
		Description: "Enable announcement banner",
		Validate:    validateBool,
	},
	"launch.banner_text": {
		Category:    CategoryLaunch,
		Description: "Banner display text",
		Validate:    validateString,
	},
	"launch.waitlist_enabled": {
		Category:    CategoryLaunch,
		Description: "Enable customer waitlist signup",
		Validate:    validateBool,
	},

	// Features
	"features.trade_lines": {
		Category:    CategoryFeatures,
		Description: "Enable trade line accounts and revolving facilities",
		Validate:    validateBool,
	},
	"features.drawdowns": {
		Category:    CategoryFeatures,
		Description: "Enable drawdown requests on active facilities",
		Validate:    validateBool,
	},
	"features.repayment_extensions": {
		Category:    CategoryFeatures,
		Description: "Enable buyer requested repayment extensions",
		Validate:    validateBool,
	},
	"features.disputes": {
		Category:    CategoryFeatures,
		Description: "Enable buyer dispute workflows",
		Validate:    validateBool,
	},
	"features.early_settlement_discounts": {
		Category:    CategoryFeatures,
		Description: "Enable early settlement discounts",
		Validate:    validateBool,
	},
	"features.notifications_whatsapp": {
		Category:    CategoryFeatures,
		Description: "Enable WhatsApp message delivery",
		Validate:    validateBool,
	},
	"features.mono_direct_debit": {
		Category:    CategoryFeatures,
		Description: "Enable Mono direct debit mandate sweeps",
		Validate:    validateBool,
	},

	// Integrations - Mono
	"integrations.mono.enabled": {
		Category:    CategoryIntegrations,
		Description: "Enable Mono Open Banking integration",
		Validate:    validateBool,
	},
	"integrations.mono.app_id": {
		Category:    CategoryIntegrations,
		Description: "Mono App ID",
		Validate:    validateString,
	},
	"integrations.mono.secret_key": {
		Category:    CategoryIntegrations,
		IsSecret:    true,
		Description: "Mono Secret Key (encrypted)",
		Validate:    validateString,
	},
	"integrations.mono.public_key": {
		Category:    CategoryIntegrations,
		Description: "Mono Public Key",
		Validate:    validateString,
	},
	"integrations.mono.status": {
		Category:    CategoryIntegrations,
		Description: "Mono verification status",
		Validate:    validateIntegrationStatus,
	},

	// Integrations - Paystack
	"integrations.paystack.enabled": {
		Category:    CategoryIntegrations,
		Description: "Enable Paystack payment integration",
		Validate:    validateBool,
	},
	"integrations.paystack.secret_key": {
		Category:    CategoryIntegrations,
		IsSecret:    true,
		Description: "Paystack Secret Key (encrypted)",
		Validate:    validateString,
	},
	"integrations.paystack.public_key": {
		Category:    CategoryIntegrations,
		Description: "Paystack Public Key",
		Validate:    validateString,
	},
	"integrations.paystack.status": {
		Category:    CategoryIntegrations,
		Description: "Paystack verification status",
		Validate:    validateIntegrationStatus,
	},

	// Integrations - Termii
	"integrations.termii.enabled": {
		Category:    CategoryIntegrations,
		Description: "Enable Termii SMS integration",
		Validate:    validateBool,
	},
	"integrations.termii.api_key": {
		Category:    CategoryIntegrations,
		IsSecret:    true,
		Description: "Termii API Key (encrypted)",
		Validate:    validateString,
	},
	"integrations.termii.sender_id": {
		Category:    CategoryIntegrations,
		Description: "Termii Sender ID",
		Validate:    validateString,
	},
	"integrations.termii.status": {
		Category:    CategoryIntegrations,
		Description: "Termii verification status",
		Validate:    validateIntegrationStatus,
	},

	// Integrations - Resend
	"integrations.resend.enabled": {
		Category:    CategoryIntegrations,
		Description: "Enable Resend transactional email integration",
		Validate:    validateBool,
	},
	"integrations.resend.api_key": {
		Category:    CategoryIntegrations,
		IsSecret:    true,
		Description: "Resend API Key (encrypted)",
		Validate:    validateString,
	},
	"integrations.resend.from_email": {
		Category:    CategoryIntegrations,
		Description: "Resend From Address",
		Validate:    validateString,
	},
	"integrations.resend.status": {
		Category:    CategoryIntegrations,
		Description: "Resend verification status",
		Validate:    validateIntegrationStatus,
	},

	// Security
	"security.mfa_enforced": {
		Category:    CategorySecurity,
		Description: "Enforce MFA for platform administrators and financial operations",
		Validate:    validateBool,
	},
	"security.session_idle_minutes": {
		Category:    CategorySecurity,
		Description: "Session idle timeout before re-authentication is required (minutes)",
		Validate:    validateIntRange(5, 1440),
	},
	"security.max_login_attempts": {
		Category:    CategorySecurity,
		Description: "Maximum failed login attempts before throttle",
		Validate:    validateIntRange(1, 20),
	},
	"security.ip_allowlist_enabled": {
		Category:    CategorySecurity,
		Description: "Enable IP allowlist for super admin console",
		Validate:    validateBool,
	},

	// KYC
	"kyc.tier1_max_kobo": {
		Category:    CategoryKYC,
		Description: "Tier 1 single obligation limit in kobo",
		Validate:    validateIntRange(0, 100000000000),
	},
	"kyc.tier2_bvn_required": {
		Category:    CategoryKYC,
		Description: "Require verified BVN for Tier 2 limits",
		Validate:    validateBool,
	},
	"kyc.tier3_cac_required": {
		Category:    CategoryKYC,
		Description: "Require verified CAC corporate registration for Tier 3",
		Validate:    validateBool,
	},

	// Fees
	"fees.supplier_rate_bps": {
		Category:    CategoryFees,
		Description: "Default platform supplier fee rate in basis points (350 = 3.5%)",
		Validate:    validateIntRange(0, 5000),
	},
	"fees.late_fee_rate_bps": {
		Category:    CategoryFees,
		Description: "Default late penalty fee rate in basis points (100 = 1.0%)",
		Validate:    validateIntRange(0, 5000),
	},
	"fees.grace_period_days": {
		Category:    CategoryFees,
		Description: "Grace period days before late penalty applies",
		Validate:    validateIntRange(0, 90),
	},

	// Notifications
	"notifications.channels": {
		Category:    CategoryNotifications,
		Description: "Active notification delivery channels",
		Validate: func(raw json.RawMessage) error {
			var channels []string
			if err := json.Unmarshal(raw, &channels); err != nil {
				return errors.New("must be an array of strings")
			}
			for _, ch := range channels {
				switch ch {
				case "in_app", "email", "sms", "whatsapp":
				default:
					return fmt.Errorf("unknown notification channel %q", ch)
				}
			}
			return nil
		},
	},
	"notifications.pre_debit_reminder_days": {
		Category:    CategoryNotifications,
		Description: "Days prior to due date to send pre-debit notice",
		Validate:    validateIntRange(1, 14),
	},
}

func validateBool(raw json.RawMessage) error {
	var b bool
	if err := json.Unmarshal(raw, &b); err != nil {
		return errors.New("value must be a boolean (true or false)")
	}
	return nil
}

func validateString(raw json.RawMessage) error {
	var s string
	if err := json.Unmarshal(raw, &s); err != nil {
		return errors.New("value must be a string")
	}
	return nil
}

func validateIntegrationStatus(raw json.RawMessage) error {
	var s string
	if err := json.Unmarshal(raw, &s); err != nil {
		return errors.New("status must be a string")
	}
	switch s {
	case IntegrationStatusUnconfigured, IntegrationStatusConfigured, IntegrationStatusVerified, IntegrationStatusDegraded, IntegrationStatusDisabled:
		return nil
	default:
		return fmt.Errorf("invalid integration status %q", s)
	}
}

func validateIntRange(minVal, maxVal int64) func(raw json.RawMessage) error {
	return func(raw json.RawMessage) error {
		var n int64
		if err := json.Unmarshal(raw, &n); err != nil {
			return errors.New("value must be an integer")
		}
		if n < minVal || n > maxVal {
			return fmt.Errorf("value must be between %d and %d", minVal, maxVal)
		}
		return nil
	}
}

func ValidateKeyAndValue(key string, raw json.RawMessage) (SettingMeta, error) {
	meta, ok := KnownSettings[key]
	if !ok {
		// If key not statically in map, check category prefix
		parts := strings.SplitN(key, ".", 2)
		if len(parts) < 2 {
			return SettingMeta{}, fmt.Errorf("invalid setting key format %q (expected category.name)", key)
		}
		category := parts[0]
		switch category {
		case CategoryLaunch, CategoryFeatures, CategoryIntegrations, CategorySecurity, CategoryKYC, CategoryFees, CategoryNotifications, CategoryGovernance:
		default:
			return SettingMeta{}, fmt.Errorf("unknown setting category %q", category)
		}
		meta = SettingMeta{Category: category, Description: key}
	}
	if meta.Validate != nil {
		if err := meta.Validate(raw); err != nil {
			return meta, fmt.Errorf("validation failed for %s: %w", key, err)
		}
	}
	return meta, nil
}
