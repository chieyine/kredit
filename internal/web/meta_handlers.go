package web

import (
	"crypto/hmac"
	"encoding/json"
	"errors"
	"fmt"
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
						Audio struct {
							ID       string `json:"id"`
							MimeType string `json:"mime_type"`
						} `json:"audio"`
						Voice struct {
							ID       string `json:"id"`
							MimeType string `json:"mime_type"`
						} `json:"voice"`
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
					var rawText string
					var isAudio bool
					var audioBytes []byte
					var audioMime string

					// Decide before downloading anything. A voice note costs a media
					// download plus a model call, so an assistant that is switched
					// off or over its per-sender budget must not pay either.
					assistant := s.runtime.WhatsAppAI != nil && s.runtime.WhatsAppAI.Enabled() && s.runtime.WhatsAppAI.Allow(message.From)

					if message.Type == "text" {
						rawText = message.Text.Body
					} else if message.Type == "audio" || message.Type == "voice" {
						isAudio = true
						if !assistant {
							// Without the assistant there is nothing that can read a
							// voice note, so say so rather than downloading it.
							_ = s.runtime.Notifications.SendWhatsAppReply(r.Context(), message.ID, message.From, "We cannot read voice notes right now. Send the goods, the amount and the payment day as a message, or record the sale in Kredit: "+strings.TrimRight(s.config.AppBaseURL, "/")+"/workspace/sales/new")
							continue
						}
						mediaID := message.Audio.ID
						audioMime = message.Audio.MimeType
						if mediaID == "" {
							mediaID = message.Voice.ID
							audioMime = message.Voice.MimeType
						}
						var dlErr error
						audioBytes, audioMime, dlErr = provider.DownloadMedia(r.Context(), mediaID)
						if dlErr != nil {
							_ = s.runtime.Notifications.SendWhatsAppReply(r.Context(), message.ID, message.From, "Sorry, we could not retrieve that voice note. Please try sending it again.")
							continue
						}
					} else {
						continue
					}

					event := whatsapp.Event{ID: message.ID, From: message.From, Text: rawText}
					if isAudio {
						event.Text = "[voice note]"
					}
					event.Signature = s.runtime.WhatsApp.Sign(event)
					_, _ = s.runtime.WhatsApp.Handle(r.Context(), event)

					var reply string
					appURL := strings.TrimRight(s.config.AppBaseURL, "/")

					// The assistant is optional. When it is switched off, over its
					// per-sender budget, or unable to read the message, the
					// deterministic command parser below still answers.
					if assistant {
						var aiResult whatsapp.AIResult
						var aiErr error
						if isAudio {
							aiResult, aiErr = s.runtime.WhatsAppAI.ParseAudio(r.Context(), audioBytes, audioMime)
						} else {
							aiResult, aiErr = s.runtime.WhatsAppAI.ParseText(r.Context(), rawText)
						}

						reply = assistantReply(aiResult, aiErr, appURL)
					} else {
						command, parseErr := whatsapp.ParseCommand(rawText)
						if parseErr != nil {
							reply = "To start a sale, send: create credit Customer name, amount, payment day. Nothing is recorded until you confirm it in Kredit: " + appURL + "/signin"
						} else if command.RequiresConfirmation {
							reply = whatsapp.ConfirmationSummary(command) + " Review and confirm in Kredit: " + appURL + "/signin"
						} else {
							reply = "Open Kredit to review your account: " + appURL + "/signin"
						}
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

func formatKoboAmount(kobo int64) string {
	ngn := kobo / 100
	rem := kobo % 100
	str := fmt.Sprintf("%d", ngn)
	n := len(str)
	var out []byte
	for i := 0; i < n; i++ {
		if i > 0 && (n-i)%3 == 0 {
			out = append(out, ',')
		}
		out = append(out, str[i])
	}
	if rem > 0 {
		return fmt.Sprintf("%s.%02d", string(out), rem)
	}
	return string(out)
}

// assistantReply turns a read message into what the seller must still do. It is
// separated from the webhook so the property that matters can be tested
// directly: no reply may ever state that a sale or a payment has been recorded.
//
// WhatsApp carries no authenticated session, so a chat message cannot create an
// agreement, a mandate or a payment. README section 29.2 requires confirmation
// on an authenticated surface and section 29.6 forbids completing a financial
// action through unauthenticated chat. An earlier version of this code replied
// "Sale Confirmed! Invoice recorded" when nothing had been written, which is the
// one thing a trade-credit product must never say.
func assistantReply(result whatsapp.AIResult, readErr error, appURL string) string {
	if readErr != nil {
		return "Sorry, we could not read that message. Open Kredit to record the sale yourself: " + appURL + "/workspace/sales/new"
	}
	switch result.Intent {
	case whatsapp.IntentCreateCredit:
		itemsLine := ""
		if result.Items != "" {
			itemsLine = fmt.Sprintf("• *Goods:* %s\n", result.Items)
		}
		dueLine := ""
		if result.DueDate != "" {
			dueLine = fmt.Sprintf("• *Payment day:* %s\n", result.DueDate)
		}
		return fmt.Sprintf("📋 *This is what we understood. Nothing is saved yet.*\n• *Customer:* %s\n• *Amount:* ₦%s\n%s%s\nOpen Kredit to check these details and send the sale to your customer: %s/app/credit/new",
			result.BuyerName, formatKoboAmount(result.AmountKobo), itemsLine, dueLine, appURL)
	case whatsapp.IntentConfirm:
		return "A sale cannot be confirmed over WhatsApp. Open Kredit to check the goods, amount and payment day, then send it to your customer: " + appURL + "/workspace/sales/new"
	case whatsapp.IntentRecordPayment:
		return "Nothing is saved yet. Check and record payments in Kredit against your bank account: " + appURL + "/workspace/money/received"
	case whatsapp.IntentQueryBalance:
		return "Open Kredit to see what each customer still owes you: " + appURL + "/workspace/today"
	case whatsapp.IntentHelp:
		return "👋 *Kredit on WhatsApp.*\nSend a message or voice note and we will read the details back to you. Recording a sale, confirming it and recording a payment all happen in Kredit, where your account is protected.\n\nOpen Kredit: " + appURL + "/signin"
	default:
		return "Send the goods, the amount and the payment day and we will read them back to you. Sales are recorded in Kredit: " + appURL + "/workspace/sales/new"
	}
}
