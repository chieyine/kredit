package platformsettings

import "testing"

func TestMissingEncryptionRootFailsClosed(t *testing.T) {
	t.Setenv("SETTINGS_ENCRYPTION_KEY", "")
	t.Setenv("TOKEN_SECRET", "legacy-token-secret-must-not-encrypt-settings")
	enc := NewEncryptor("")
	if _, err := enc.Encrypt("integrations.mono.secret_key", "provider-secret"); err == nil {
		t.Fatal("missing root allowed secret encryption")
	}
	if _, err := enc.Decrypt("integrations.mono.secret_key", "YWJj"); err == nil {
		t.Fatal("missing root allowed secret decryption")
	}
}

func TestShortEncryptionRootFailsClosed(t *testing.T) {
	if _, err := NewEncryptor("short").Encrypt("integrations.mono.secret_key", "secret"); err == nil {
		t.Fatal("short root accepted")
	}
}

func TestRegistryRejectsUnknownNullAndManualVerification(t *testing.T) {
	for _, tc := range []struct{ key, value string }{
		{"features.typo", "true"}, {"features.trade_lines", "null"},
		{"integrations.mono.status", `"verified"`},
	} {
		if _, err := ValidateKeyAndValue(tc.key, []byte(tc.value)); err == nil {
			t.Fatalf("accepted %s=%s", tc.key, tc.value)
		}
	}
}
