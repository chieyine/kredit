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

					if message.Type == "text" {
						rawText = message.Text.Body
					} else if message.Type == "audio" || message.Type == "voice" {
						isAudio = true
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

					// Use Gemini AI parser if enabled
					if s.runtime.WhatsAppAI != nil && s.runtime.WhatsAppAI.Enabled() {
						var aiResult whatsapp.AIResult
						var aiErr error
						if isAudio {
							aiResult, aiErr = s.runtime.WhatsAppAI.ParseAudio(r.Context(), audioBytes, audioMime)
						} else {
							aiResult, aiErr = s.runtime.WhatsAppAI.ParseText(r.Context(), rawText)
						}

						if aiErr != nil {
							reply = "Could not parse your message. To record a sale, send: 'create credit Buyer, amount, due date' or send a voice note. Dashboard: " + appURL + "/app"
						} else {
							switch aiResult.Intent {
							case whatsapp.IntentCreateCredit:
								amountStr := fmt.Sprintf("₦%s", formatKoboAmount(aiResult.AmountKobo))
								itemsLine := ""
								if aiResult.Items != "" {
									itemsLine = fmt.Sprintf("• *Items:* %s\n", aiResult.Items)
								}
								dueLine := ""
								if aiResult.DueDate != "" {
									dueLine = fmt.Sprintf("• *Due Date:* %s\n", aiResult.DueDate)
								}
								reply = fmt.Sprintf("📋 *Credit Sale Draft:*\n• *Buyer:* %s\n• *Amount:* %s\n%s%s\nReply *YES* to confirm and send payment link, or reply with edits.", aiResult.BuyerName, amountStr, itemsLine, dueLine)

							case whatsapp.IntentConfirm:
								reply = "✅ *Sale Confirmed!*\nInvoice recorded. A notification and payment link have been dispatched to the buyer.\n\nOpen Kredit: " + appURL + "/app"

							case whatsapp.IntentRecordPayment:
								reply = fmt.Sprintf("💰 *Payment Recorded:*\n%s\n\nView updated ledger: %s/app", aiResult.Summary, appURL)

							case whatsapp.IntentQueryBalance:
								reply = fmt.Sprintf("📊 *Your Kredit Account:*\nView your current debtors, receivables, and invoices anytime at: %s/app", appURL)

							case whatsapp.IntentHelp:
								reply = "👋 *Welcome to Kredit on WhatsApp!*\nYou can send text or voice notes anytime:\n• *\"I gave Emeka 50 cartons for 150k to pay on Friday\"*\n• *\"Who owes me?\"*\n• *\"Emeka paid 50,000\"*\n\nDashboard: " + appURL + "/app"

							default:
								if aiResult.Summary != "" {
									reply = aiResult.Summary + "\n\nOpen Kredit: " + appURL + "/app"
								} else {
									reply = "👋 Send a voice note or message with your sale details (e.g. 'I gave Alhassan goods for 200k to pay next week').\nDashboard: " + appURL + "/app"
								}
							}
						}
					} else {
						command, parseErr := whatsapp.ParseCommand(rawText)
						if parseErr != nil {
							reply = "To create a sale, send: create credit Buyer name, amount, due date. Dashboard: " + appURL + "/app"
						} else if command.RequiresConfirmation {
							reply = whatsapp.ConfirmationSummary(command) + " Review and confirm in Kredit: " + appURL + "/app"
						} else {
							reply = "Open Kredit to review your account: " + appURL + "/app"
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
