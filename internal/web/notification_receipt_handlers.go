package web

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"kredit/internal/notifications"
	"net/http"
)

func (s *Server) notificationDeliveryReceipt(w http.ResponseWriter, r *http.Request) {
	channel := r.PathValue("channel")
	token := ""
	switch channel {
	case "email":
		token = s.config.NotificationEmailToken
	case "sms":
		token = s.config.NotificationSMSToken
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
		token = ""
		if connector.Enabled {
			token = connector.Token
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
	if err = s.runtime.Notifications.RecordDeliveryReceipt(r.Context(), channel, receipt); err != nil {
		switch {
		case errors.Is(err, notifications.ErrInvalidDeliveryReceipt):
			writeProblem(w, 400, "invalid_receipt", "The delivery receipt is incomplete or invalid.")
		case errors.Is(err, notifications.ErrDeliveryReceiptConflict):
			writeProblem(w, 409, "receipt_conflict", "The receipt reference was already used for different evidence.")
		case errors.Is(err, notifications.ErrDeliveryReceiptPending):
			writeProblem(w, 409, "receipt_pending", "The sent message is not yet available. Retry this receipt later.")
		default:
			writeProblem(w, 503, "receipt_unavailable", "The delivery receipt could not be saved. Retry this receipt later.")
		}
		return
	}
	writeJSON(w, 200, map[string]any{"status": "recorded"})
}
