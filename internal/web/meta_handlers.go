package web

import (
	"crypto/hmac"
	"encoding/json"
	"errors"
	"io"
	"kredit/internal/notifications"
	"kredit/internal/whatsapp"
	"net/http"
	"strings"
)

func (s *Server) metaWebhook(w http.ResponseWriter, r *http.Request) {
	config, err := notifications.ResolveConnector(r.Context(), s.runtime.PlatformSettings, notifications.ChannelWhatsApp)
	if err != nil {
		writeProblem(w, 503, "meta_unavailable", "WhatsApp configuration is unavailable.")
		return
	}
	current := config != nil && config.Enabled && config.Adapter == "meta"
	if r.Method == http.MethodGet {
		if !current || r.URL.Query().Get("hub.mode") != "subscribe" || !hmac.Equal([]byte(r.URL.Query().Get("hub.verify_token")), []byte(config.VerifyToken)) || len(config.VerifyToken) < 32 {
			writeProblem(w, 403, "meta_verification_failed", "Webhook verification failed.")
			return
		}
		challenge := r.URL.Query().Get("hub.challenge")
		if challenge == "" || len(challenge) > 512 {
			writeProblem(w, 400, "invalid_challenge", "Invalid challenge.")
			return
		}
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(200)
		_, _ = w.Write([]byte(challenge))
		return
	}
	raw, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 128<<10))
	if err != nil {
		writeProblem(w, 400, "invalid_webhook", "Invalid webhook body.")
		return
	}
	signature := r.Header.Get("X-Hub-Signature-256")
	authenticated := current && notifications.VerifyMetaSignature(config.WebhookSecret, signature, raw)
	var envelope struct {
		Object  string `json:"object"`
		Entries []struct {
			Changes []struct {
				Field string `json:"field"`
				Value struct {
					Product  string `json:"messaging_product"`
					Metadata struct {
						PhoneID string `json:"phone_number_id"`
					} `json:"metadata"`
					Statuses []struct {
						ID        string `json:"id"`
						Recipient string `json:"recipient_id"`
						Status    string `json:"status"`
						Timestamp string `json:"timestamp"`
					} `json:"statuses"`
					Messages []struct {
						ID   string `json:"id"`
						From string `json:"from"`
						Type string `json:"type"`
						Text struct {
							Body string `json:"body"`
						} `json:"text"`
					} `json:"messages"`
				} `json:"value"`
			} `json:"changes"`
		} `json:"entry"`
	}
	if json.Unmarshal(raw, &envelope) != nil || envelope.Object != "whatsapp_business_account" || len(envelope.Entries) > 100 {
		writeProblem(w, 400, "invalid_webhook", "Invalid Meta event.")
		return
	}
	confirmed := false
	for _, entry := range envelope.Entries {
		for _, change := range entry.Changes {
			if change.Field != "messages" || change.Value.Product != "whatsapp" {
				continue
			}
			value := change.Value
			if len(value.Statuses)+len(value.Messages) > 100 {
				writeProblem(w, 400, "invalid_webhook", "Too many events.")
				return
			}
			for _, status := range value.Statuses {
				err = s.runtime.Notifications.RecordMetaStatus(r.Context(), value.Metadata.PhoneID, status.ID, status.Recipient, status.Status, status.Timestamp, signature, raw)
				if errors.Is(err, notifications.ErrDeliveryReceiptPending) && authenticated {
					currentProvider, currentErr := notifications.NewMetaProvider(*config)
					if currentErr != nil || currentProvider.PhoneID() != value.Metadata.PhoneID {
						writeProblem(w, 401, "wrong_phone", "The WhatsApp phone identity does not match.")
						return
					}
					var standalone bool
					if s.runtime.Database != nil {
						err = s.runtime.Database.Raw().QueryRow(r.Context(), `SELECT EXISTS(SELECT 1 FROM app.message_submissions WHERE channel='whatsapp' AND provider_reference=$1 AND notification_id IS NULL AND state='ACCEPTED')`, status.ID).Scan(&standalone)
					}
					if err == nil && standalone {
						confirmed = true
						continue
					}
					writeProblem(w, 503, "delivery_pending", "The original send is not committed yet. Retry this event.")
					return
				}
				if err != nil {
					writeProblem(w, 503, "delivery_unconfirmed", "The original delivery evidence could not be confirmed.")
					return
				}
				confirmed = true
			}
			if len(value.Messages) > 0 {
				if !authenticated {
					writeProblem(w, 401, "invalid_signature", "Meta authentication failed.")
					return
				}
				provider, e := notifications.NewMetaProvider(*config)
				if e != nil || provider.PhoneID() != value.Metadata.PhoneID {
					writeProblem(w, 401, "wrong_phone", "The WhatsApp phone identity does not match.")
					return
				}
				for _, message := range value.Messages {
					if message.Type != "text" {
						continue
					}
					event := whatsapp.Event{ID: message.ID, From: message.From, Text: message.Text.Body}
					event.Signature = s.runtime.WhatsApp.Sign(event)
					_, e = s.runtime.WhatsApp.Handle(r.Context(), event)
					command, parseErr := whatsapp.ParseCommand(message.Text.Body)
					if e != nil && (parseErr == nil || e.Error() != parseErr.Error()) {
						writeProblem(w, 503, "message_pending", "The message could not be recorded.")
						return
					}
					reply := "Open Kredit to review your account: " + strings.TrimRight(s.config.AppBaseURL, "/") + "/app"
					if parseErr != nil {
						reply = "To create a sale, send: create credit Buyer name, amount, due date. Write the date as day month year. You can also open Kredit: " + strings.TrimRight(s.config.AppBaseURL, "/") + "/app"
					} else if command.RequiresConfirmation {
						reply = whatsapp.ConfirmationSummary(command) + " Review and confirm in Kredit: " + strings.TrimRight(s.config.AppBaseURL, "/") + "/app"
					}
					if e = s.runtime.Notifications.SendWhatsAppReply(r.Context(), message.ID, message.From, reply); e != nil && !errors.Is(e, notifications.ErrSubmissionUnknown) {
						writeProblem(w, 503, "reply_pending", "The reply could not be confirmed.")
						return
					}
				}
				confirmed = true
			}
		}
	}
	if !authenticated && !confirmed {
		writeProblem(w, 401, "invalid_signature", "Meta authentication failed.")
		return
	}
	writeJSON(w, 200, map[string]string{"status": "accepted"})
}
