package collections

import "errors"

const (
	PolicyStandard  = "STANDARD"
	PolicyGrace     = "GRACE"
	PolicyImmediate = "IMMEDIATE"
)

// ValidatePolicy keeps unavailable staged recovery from silently becoming an
// immediate multi-account sweep. Mono Sweep chooses primary/fallback accounts
// within a debit; it does not expose Kredit-controlled primary-only staging.
func ValidatePolicy(policy string, capabilities Capabilities) error {
	switch policy {
	case "", PolicyGrace, PolicyImmediate:
		return nil
	case PolicyStandard:
		if capabilities.MultiAccount {
			return errors.New("provider does not support separately staged primary and sweep actions")
		}
		return nil
	default:
		return errors.New("unknown collection policy")
	}
}
