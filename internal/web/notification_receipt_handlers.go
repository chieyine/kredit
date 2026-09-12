package web

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"kredit/internal/notifications"
	"net/http"
	"strconv"
	"time"
)

func (s *Server) notificationDeliveryReceipt(w http.ResponseWriter, r *http.Request) {
	channel := r.PathValue("channel")
	token := ""
	adapter := notifications.DefaultAdapter(channel)
	sendlySecret := s.config.NotificationEmailWebhookSecret
	switch channel {
	case "email":
		token = s.config.NotificationEmailToken
		if s.config.NotificationEmailAdapter != "" {
			adapter = s.config.NotificationEmailAdapter
		}
	case "sms":
		token = s.config.NotificationSMSToken
		if s.config.NotificationSMSAdapter != "" {
			adapter = s.config.NotificationSMSAdapter
		}
	case "whatsapp":
		token = s.config.NotificationWhatsAppToken
	default:
		writeProblem(w, 404, "notification_channel_invalid", "Notification channel is not available")
		return
	}
	connector, err := notifications.ResolveConnector(r.Context(), s.runtime.PlatformSettings, channel)
	if err != nil {
		writeProblem(w, 503, "connector_unavailable", "Connector configuration is unavailable")
		return
	}
	if connector != nil {
		if connector.Adapter != "" {
			adapter = connector.Adapter
		}
		token = ""
		if connector.Enabled {
			token = connector.Token
			sendlySecret = connector.WebhookSecret
		}
	}
	if token == "" {
		writeProblem(w, 401, "invalid_signature", "Connector authentication required")
		return
	}
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 16<<10))
	if err != nil {
		writeProblem(w, 400, "invalid_request", "Invalid receipt body")
		return
	}
	if channel == notifications.ChannelSMS && adapter == "mesaj" {
		// Mesaj bulk callbacks are unauthenticated signals. They never establish
		// delivery evidence; the published bulk API has no authenticated lookup.
		var envelope struct {
			MessageID string `json:"message_id"`
			Recipient string `json:"recipient"`
			Status    string `json:"status"`
		}
		if json.Unmarshal(body, &envelope) != nil || envelope.MessageID == "" || len(envelope.MessageID) > 512 || envelope.Recipient == "" || len(envelope.Recipient) > 32 || envelope.Status == "" || len(envelope.Status) > 64 {
			writeProblem(w, 400, "invalid_request", "Invalid Mesaj event")
			return
		}

		writeJSON(w, 200, map[string]any{"status": "accepted"})
		return
	}
	if channel == notifications.ChannelEmail && adapter == "sendly" {
		stamp := r.Header.Get("sendly-timestamp")
		seconds, stampErr := strconv.ParseInt(stamp, 10, 64)
		age := time.Now().Unix() - seconds
		signature, sigErr := hex.DecodeString(r.Header.Get("sendly-signature"))
		mac := hmac.New(sha256.New, []byte(sendlySecret))
		mac.Write([]byte(stamp + "."))
		mac.Write(body)
		if sendlySecret == "" || stampErr != nil || sigErr != nil || age < -300 || age > 300 || !hmac.Equal(signature, mac.Sum(nil)) {
			writeProblem(w, 401, "invalid_signature", "Sendly authentication failed")
			return
		}
		var envelope struct {
			ID        string          `json:"id"`
			Type      string          `json:"type"`
			CreatedAt time.Time       `json:"createdAt"`
			Data      json.RawMessage `json:"data"`
		}
		if json.Unmarshal(body, &envelope) != nil || envelope.ID == "" || envelope.Type == "" || envelope.CreatedAt.IsZero() || len(envelope.Data) == 0 {
			writeProblem(w, 400, "invalid_request", "Invalid Sendly event")
			return
		}
		// The worker independently reconciles saved message IDs through Sendly's
		// authenticated lookup. Receipt of this event alone never starts a timer.
		writeJSON(w, 200, map[string]any{"status": "accepted"})
		return
	}
	signature, err := hex.DecodeString(r.Header.Get("X-Notification-Signature"))
	mac := hmac.New(sha256.New, []byte(token))
	mac.Write(body)
	if err != nil || !hmac.Equal(signature, mac.Sum(nil)) {
		writeProblem(w, 401, "invalid_signature", "Connector authentication failed")
		return
	}
	var receipt notifications.DeliveryReceipt
	if err = json.Unmarshal(body, &receipt); err != nil {
		writeProblem(w, 400, "invalid_request", "Invalid delivery receipt")
		return
	}
	// Authenticated connector callbacks are signals only. A replacement
	// connector cannot attest to messages sent through an earlier connection.
	// The worker looks up the original event through its pinned provider route.
	writeJSON(w, 200, map[string]any{"status": "accepted"})
}
