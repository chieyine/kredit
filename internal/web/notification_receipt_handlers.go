package web

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
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
		writeProblem(w, 409, "receipt_not_recorded", err.Error())
		return
	}
	writeJSON(w, 200, map[string]any{"status": "recorded"})
}
