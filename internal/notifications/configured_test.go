package notifications

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"kredit/internal/platformsettings"
)

type connectorSettings struct {
	platformsettings.Service
	value platformsettings.Setting
	err   error
}

func (s *connectorSettings) Get(context.Context, string, bool) (platformsettings.Setting, error) {
	return s.value, s.err
}

func TestConfiguredDeliveryRotationDisableAndReadFailure(t *testing.T) {
	if os.Getenv("KREDIT_INTEGRATION") != "1" || os.Getenv("DATABASE_URL") == "" {
		t.Skip("isolated database required for durable provider routes")
	}
	pool, err := pgxpool.New(context.Background(), os.Getenv("DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	var tokens []string
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tokens = append(tokens, r.Header.Get("Authorization"))
		_, _ = w.Write([]byte(`{"message_id":"sent"}`))
	}))
	defer server.Close()
	transport := http.DefaultTransport
	http.DefaultTransport = server.Client().Transport
	defer func() { http.DefaultTransport = transport }()
	settings := &connectorSettings{}
	update := func(enabled bool, token string) {
		encoded, _ := json.Marshal(platformsettings.NotificationConnector{Adapter: "connector", Enabled: enabled, Endpoint: server.URL, Token: token})
		value, _ := json.Marshal(string(encoded))
		settings.value = platformsettings.Setting{Key: "integrations.notifications.sms", Value: value}
	}
	fallback := NewMockProvider(ChannelSMS)
	provider := NewConfiguredProvider(ChannelSMS, settings, fallback).WithPersistence(pool, []byte("fixture-submission-key-0123456789abcdef"))
	provider.encryptor = platformsettings.NewEncryptor("fixture-route-encryption-key-0123456789abcdef")
	message := Message{Channel: ChannelSMS, Destination: "test-recipient", EventID: "test-event"}
	for _, token := range []string{"first", "rotated"} {
		update(true, token)
		message.EventID = server.URL + ":" + token
		if _, err := provider.Send(context.Background(), message); err != nil {
			t.Fatal(err)
		}
	}
	message.EventID = server.URL + ":first"
	if _, err := provider.Send(context.Background(), message); err != nil {
		t.Fatal(err)
	}
	if len(tokens) != 3 || tokens[0] != "Bearer first" || tokens[1] != "Bearer rotated" || tokens[2] != "Bearer first" {
		t.Fatalf("rotation was not applied: %v", tokens)
	}
	update(false, "")
	if _, err := provider.Send(context.Background(), message); err == nil {
		t.Fatal("disabled connector sent")
	}
	settings.err = errors.New("database unavailable")
	if _, err := provider.Send(context.Background(), message); err == nil {
		t.Fatal("read failure used fallback")
	}
	if len(fallback.Messages()) != 0 {
		t.Fatal("failure or disable reactivated fallback")
	}
	settings.err = pgx.ErrNoRows
	if _, err := provider.Send(context.Background(), message); err != nil {
		t.Fatal(err)
	}
	if len(fallback.Messages()) != 1 {
		t.Fatal("missing admin override should preserve deployment configuration")
	}
}
