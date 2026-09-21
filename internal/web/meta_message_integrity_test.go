package web

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"kredit/internal/notifications"
	"kredit/internal/platformsettings"
	"kredit/internal/whatsapp"

	"github.com/jackc/pgx/v5/pgxpool"
)

type auditMetaReplyProvider struct {
	calls int
	err   error
}

func (*auditMetaReplyProvider) Channel() string { return notifications.ChannelWhatsApp }
func (p *auditMetaReplyProvider) Send(context.Context, notifications.Message) (string, error) {
	p.calls++
	return "synthetic-reply", p.err
}

func auditMetaMessageServer(provider *auditMetaReplyProvider) *Server {
	store := notifications.NewStore("synthetic-message-storage-key")
	store.RegisterProvider(provider)
	return &Server{runtime: &Runtime{
		Notifications: store,
		WhatsApp:      whatsapp.NewHandler("synthetic-message-event-key"),
		PlatformSettings: auditMetaVerificationSettings{connector: platformsettings.NotificationConnector{
			Enabled: true, Adapter: "meta", Endpoint: "https://graph.facebook.com/v23.0/123/messages",
			Token: "synthetic-access-token", VerifyToken: strings.Repeat("v", 32), WebhookSecret: strings.Repeat("s", 32),
			Language: "en", AuthenticationTemplate: "test_auth", UtilityTemplate: "test_utility", MarketingTemplate: "test_marketing",
		}},
	}}
}

func auditMetaMessageRequest(t *testing.T, server *Server, message map[string]any) *httptest.ResponseRecorder {
	t.Helper()
	raw, err := json.Marshal(map[string]any{
		"object": "whatsapp_business_account",
		"entry": []any{map[string]any{"changes": []any{map[string]any{
			"field": "messages", "value": map[string]any{
				"messaging_product": "whatsapp", "metadata": map[string]string{"phone_number_id": "123"}, "messages": []any{message},
			},
		}}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	mac := hmac.New(sha256.New, []byte(strings.Repeat("s", 32)))
	_, _ = mac.Write(raw)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/meta", bytes.NewReader(raw))
	request.Header.Set("X-Hub-Signature-256", "sha256="+hex.EncodeToString(mac.Sum(nil)))
	response := httptest.NewRecorder()
	server.metaWebhook(response, request)
	return response
}

func TestMetaVoiceFallbackDoesNotAcknowledgeLostReplies(t *testing.T) {
	for _, tc := range []struct {
		name string
		err  error
		want int
	}{
		{name: "confirmed reply", want: 200},
		{name: "ordinary failure needs retry", err: errors.New("synthetic provider outage"), want: 503},
		{name: "uncertain submission remains fenced", err: notifications.ErrSubmissionUnknown, want: 200},
	} {
		t.Run(tc.name, func(t *testing.T) {
			provider := &auditMetaReplyProvider{err: tc.err}
			server := auditMetaMessageServer(provider)
			response := auditMetaMessageRequest(t, server, map[string]any{"id": "voice-1", "from": "2348000000000", "type": "audio", "audio": map[string]string{"id": "321"}})
			if response.Code != tc.want || provider.calls != 1 {
				t.Fatalf("status=%d, reply calls=%d, want %d/1", response.Code, provider.calls, tc.want)
			}
		})
	}
}

func TestMetaIncomingEvidenceFailureStopsReplies(t *testing.T) {
	provider := &auditMetaReplyProvider{}
	server := auditMetaMessageServer(provider)
	// A closed pool fails locally; no database or external network is contacted.
	pool, err := pgxpool.New(context.Background(), "postgres://audit@127.0.0.1:1/unused")
	if err != nil {
		t.Fatal(err)
	}
	pool.Close()
	server.runtime.WhatsApp = whatsapp.NewPostgresHandler(pool, "synthetic-message-event-key")
	response := auditMetaMessageRequest(t, server, map[string]any{"id": "message-1", "from": "2348000000000", "type": "text", "text": map[string]string{"body": "how much is due"}})
	if response.Code != http.StatusServiceUnavailable || provider.calls != 0 {
		t.Fatalf("failed durable receipt continued: status=%d replies=%d", response.Code, provider.calls)
	}
}

func TestMetaFreeformTextIsAcceptedButConflictingReplayIsNot(t *testing.T) {
	provider := &auditMetaReplyProvider{}
	server := auditMetaMessageServer(provider)
	message := map[string]any{"id": "message-1", "from": "2348000000000", "type": "text", "text": map[string]string{"body": "Please help me understand this sale"}}
	response := auditMetaMessageRequest(t, server, message)
	if response.Code != http.StatusOK || provider.calls != 1 {
		t.Fatalf("freeform text incorrectly treated as storage failure: %d", response.Code)
	}
	message["text"] = map[string]string{"body": "A different message using the same ID"}
	response = auditMetaMessageRequest(t, server, message)
	if response.Code != http.StatusConflict || provider.calls != 1 {
		t.Fatalf("conflicting replay continued: status=%d replies=%d", response.Code, provider.calls)
	}
}

func TestMetaRejectsInvalidMessageIdentityBeforeReplying(t *testing.T) {
	for _, id := range []string{"", "bad|identity", strings.Repeat("x", 513)} {
		provider := &auditMetaReplyProvider{}
		response := auditMetaMessageRequest(t, auditMetaMessageServer(provider), map[string]any{"id": id, "from": "2348000000000", "type": "text", "text": map[string]string{"body": "help"}})
		if response.Code != http.StatusBadRequest || provider.calls != 0 {
			t.Fatalf("invalid message caused work: status=%d replies=%d", response.Code, provider.calls)
		}
	}
}
