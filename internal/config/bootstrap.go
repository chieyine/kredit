package config

import (
	"errors"
	"fmt"
	"strings"
)

// Load validates an environment-only configuration for callers that do not
// apply stored connections. Its existing validation contract is unchanged.
func Load() (Config, error) {
	return loadEnvironment(Config.Validate)
}

// LoadBootstrap validates only the prerequisites for reading encrypted
// settings. Entrypoints must apply stored connections (which performs the full
// Validate call) before constructing providers or exposing application routes.
func LoadBootstrap() (Config, error) {
	return loadEnvironment(Config.validateBootstrap)
}

func (c Config) validateBootstrap() error {
	if strings.TrimSpace(c.Version) == "" || strings.TrimSpace(c.APIListenAddr) == "" {
		return errors.New("APP_VERSION and API_ADDR are required")
	}
	switch c.Environment {
	case "development", "staging", "production":
	default:
		return fmt.Errorf("APP_ENV must be development, staging, or production, got %q", c.Environment)
	}
	for _, source := range []struct{ name, value string }{
		{"DATABASE_URL", c.DatabaseURL},
		{"RIVER_DATABASE_URL", c.RiverDatabaseURL},
	} {
		if strings.TrimSpace(source.value) == "" {
			return fmt.Errorf("%s is required to load saved configuration", source.name)
		}
		if c.Environment == "production" {
			if err := validateProductionDatabaseURL(source.name, source.value); err != nil {
				return err
			}
		}
	}
	if c.Environment == "production" {
		return validateSecret("SETTINGS_ENCRYPTION_KEY", c.SettingsEncryptionKey)
	}
	return nil
}
