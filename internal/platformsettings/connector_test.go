package platformsettings

import (
	"encoding/json"
	"testing"
)

func TestConnectorValidation(t *testing.T) {
	for _, tc := range []struct {
		name, endpoint, token string
		enabled, valid        bool
	}{
		{"valid", "https://connector.example/send", "secret", true, true},
		{"plaintext", "http://connector.example/send", "secret", true, false},
		{"userinfo", "https://user:password@connector.example/send", "secret", true, false},
		{"query credential", "https://connector.example/send?token=secret", "secret", true, false},
		{"missing token", "https://connector.example/send", "", true, false},
		{"header injection", "https://connector.example/send", "secret\r\nheader", true, false},
		{"disabled", "", "", false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			payload, _ := json.Marshal(NotificationConnector{Enabled: tc.enabled, Endpoint: tc.endpoint, Token: tc.token})
			raw, _ := json.Marshal(string(payload))
			meta, err := ValidateKeyAndValue("integrations.notifications.email", raw)
			if (err == nil) != tc.valid {
				t.Fatalf("unexpected validation result: %v", err)
			}
			if !meta.IsSecret {
				t.Fatal("connector must be encrypted")
			}
		})
	}
}
