package notifications

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) { return fn(request) }

func TestWebhookProviderDeliversOTPWithoutLoggingCredential(t *testing.T) {
	provider, err := NewWebhookProvider(ChannelEmail, "https://connector.example.test/send", "connector-secret")
	if err != nil {
		t.Fatal(err)
	}
	provider.client = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.Header.Get("Authorization") != "Bearer connector-secret" {
			t.Fatal("missing connector authorization")
		}
		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatal(err)
		}
		if payload["destination"] != "owner@example.test" || payload["body"] == "" {
			t.Fatalf("unexpected connector payload: %+v", payload)
		}
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"message_id":"message-1"}`)), Header: make(http.Header)}, nil
	})}
	store := NewStore("test-secret")
	store.RegisterProvider(provider)
	if err := store.SendOTP(context.Background(), "owner@example.test", ChannelEmail, "123456"); err != nil {
		t.Fatal(err)
	}
}
