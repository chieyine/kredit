package notifications

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strconv"
	"time"
)

var ErrDeliveryLookupUnsupported = errors.New("delivery is confirmed through authenticated provider callbacks")

func (p *MetaProvider) DeliveryStatus(context.Context, string) (DeliveryStatus, error) {
	return DeliveryStatus{}, ErrDeliveryLookupUnsupported
}

// RecordMetaStatus checks the app secret and phone-number identity saved with
// the original send. A replacement connection cannot confirm an older message.
func (s *Store) RecordMetaStatus(ctx context.Context, phoneID, id, recipient, status, stamp, signature string, body []byte) error {
	if s.pool == nil {
		return errors.New("delivery persistence is unavailable")
	}
	if status != "delivered" && status != "read" && status != "failed" && status != "sent" && status != "deleted" {
		return ErrInvalidDeliveryReceipt
	}
	seconds, err := strconv.ParseInt(stamp, 10, 64)
	if err != nil || seconds <= 0 {
		return ErrInvalidDeliveryReceipt
	}
	at := time.Unix(seconds, 0)
	if at.After(time.Now().Add(time.Minute)) {
		return ErrInvalidDeliveryReceipt
	}
	s.mu.Lock()
	configured, ok := s.providers[ChannelWhatsApp].(*ConfiguredProvider)
	s.mu.Unlock()
	if !ok {
		return errors.New("original WhatsApp connection is unavailable")
	}
	rows, err := s.pool.Query(ctx, `SELECT id::text,event_reference,destination_ciphertext FROM app.notifications WHERE channel='whatsapp' AND provider_message_id=$1 AND state IN ('sent','delivered','read','failed') LIMIT 10`, id)
	if err != nil {
		return err
	}
	type candidate struct {
		id, event   string
		destination []byte
	}
	items := []candidate{}
	for rows.Next() {
		var item candidate
		if err = rows.Scan(&item.id, &item.event, &item.destination); err != nil {
			rows.Close()
			return err
		}
		items = append(items, item)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	matches := []candidate{}
	for _, item := range items {
		config, e := configured.messageRoute(ctx, item.event, nil)
		if e != nil {
			return e
		}
		if config.Adapter != "meta" {
			continue
		}
		provider, e := NewMetaProvider(*config)
		if e != nil {
			return e
		}
		destination, e := s.decryptDestination(item.destination)
		if e != nil {
			return e
		}
		if provider.PhoneID() == phoneID && VerifyMetaSignature(config.WebhookSecret, signature, body) && sameDeliveryDestination(ChannelWhatsApp, recipient, string(destination)) {
			matches = append(matches, item)
		}
	}
	if len(matches) == 0 {
		return ErrDeliveryReceiptPending
	}
	if len(matches) != 1 {
		return ErrDeliveryReceiptConflict
	}
	item := matches[0]
	if status == "sent" || status == "deleted" {
		return nil
	}
	if status == "failed" {
		_, err = s.pool.Exec(ctx, `UPDATE app.notifications SET state='failed',failed_at=now(),failure_reason='Meta reported delivery failure',delivery_attempts=GREATEST(delivery_attempts,8),next_attempt_at=NULL,updated_at=now() WHERE id=$1::uuid AND state='sent'`, item.id)
		return err
	}
	digest := sha256.Sum256([]byte(item.event + ":" + id + ":" + status + ":" + stamp))
	return s.RecordDeliveryReceipt(ctx, ChannelWhatsApp, DeliveryReceipt{EventID: "meta:" + hex.EncodeToString(digest[:]), NotificationEventID: item.event, MessageID: id, DeliveredAt: at})
}

func (s *Store) SendWhatsAppReply(ctx context.Context, event, to, body string) error {
	s.mu.Lock()
	provider := s.providers[ChannelWhatsApp]
	s.mu.Unlock()
	if provider == nil {
		return errors.New("WhatsApp connection is unavailable")
	}
	_, err := provider.Send(ctx, Message{EventID: "whatsapp-reply:" + event, RecipientID: to, Channel: ChannelWhatsApp, Template: "WhatsAppCommandResponse", TemplateVersion: "v1", Body: body, Destination: to})
	return err
}
