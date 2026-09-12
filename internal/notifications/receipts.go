package notifications

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

var ErrInvalidDeliveryReceipt = errors.New("invalid delivery receipt")
var ErrDeliveryReceiptConflict = errors.New("delivery receipt identity conflicts with previous evidence")
var ErrDeliveryReceiptPending = errors.New("matching sent notification is not yet available")

type DeliveryReceipt struct {
	EventID             string    `json:"event_id"`
	NotificationEventID string    `json:"notification_event_id"`
	MessageID           string    `json:"message_id"`
	DeliveredAt         time.Time `json:"delivered_at"`
}

// RecordDeliveryReceipt accepts only authenticated connector evidence. Waiting
// periods use received_at from our database clock, never a backdated callback.
func (s *Store) RecordDeliveryReceipt(ctx context.Context, channel string, receipt DeliveryReceipt) error {
	if s.pool == nil {
		return errors.New("delivery receipt persistence is required")
	}
	if receipt.EventID == "" || len(receipt.EventID) > 200 || receipt.NotificationEventID == "" || len(receipt.NotificationEventID) > 512 || receipt.MessageID == "" || len(receipt.MessageID) > 512 || receipt.DeliveredAt.IsZero() || receipt.DeliveredAt.After(time.Now().Add(time.Minute)) {
		return ErrInvalidDeliveryReceipt
	}
	payload, _ := json.Marshal(receipt)
	digest := sha256.Sum256(payload)
	hash := hex.EncodeToString(digest[:])
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err = tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1,0))`, "delivery-receipt:"+channel+":"+receipt.EventID); err != nil {
		return err
	}
	var existing string
	err = tx.QueryRow(ctx, `SELECT payload_hash FROM app.notification_delivery_receipts WHERE channel=$1 AND event_id=$2`, channel, receipt.EventID).Scan(&existing)
	if err == nil {
		if existing != hash {
			return ErrDeliveryReceiptConflict
		}
		return tx.Commit(ctx)
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return err
	}
	var id string
	if err = tx.QueryRow(ctx, `SELECT id::text FROM app.notifications WHERE channel=$1 AND event_reference=$2 AND provider_message_id=$3 AND state IN ('sent','delivered','read','failed') FOR UPDATE`, channel, receipt.NotificationEventID, receipt.MessageID).Scan(&id); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrDeliveryReceiptPending
		}
		return fmt.Errorf("read sent notification: %w", err)
	}
	if _, err = tx.Exec(ctx, `INSERT INTO app.notification_delivery_receipts(channel,event_id,payload_hash,notification_id) VALUES($1,$2,$3,$4::uuid)`, channel, receipt.EventID, hash, id); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `UPDATE app.notifications SET state=CASE WHEN state='read' THEN state ELSE 'delivered' END,delivered_at=COALESCE(delivered_at,$2) WHERE id=$1::uuid`, id, receipt.DeliveredAt); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// ReconcileEmailDelivery is retained for callers that only need email lookup.
func (s *Store) ReconcileEmailDelivery(ctx context.Context, limit int) error {
	return s.ReconcileDelivery(ctx, ChannelEmail, limit)
}

// ReconcileDelivery reads authenticated provider records. A callback alone never
// starts a waiting period, and a missed callback cannot strand a sent notice.
func (s *Store) ReconcileDelivery(ctx context.Context, channel string, limit int) error {
	if channel != ChannelEmail && channel != ChannelSMS && channel != ChannelWhatsApp {
		return errors.New("delivery lookup channel is unsupported")
	}
	if s.pool == nil {
		return nil
	}
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	s.mu.Lock()
	provider, ok := s.providers[channel].(DeliveryStatusProvider)
	s.mu.Unlock()
	if !ok {
		return nil
	}
	rows, err := s.pool.Query(ctx, `SELECT id::text,event_reference,provider_message_id,destination_ciphertext FROM app.notifications WHERE channel=$2 AND state='sent' AND (provider_checked_at IS NULL OR provider_checked_at<now()-interval '1 minute') ORDER BY provider_checked_at NULLS FIRST,created_at LIMIT $1`, limit, channel)
	if err != nil {
		return err
	}
	type pending struct {
		id, event, message string
		destination        []byte
	}
	var pendingMessages []pending
	for rows.Next() {
		var item pending
		if err = rows.Scan(&item.id, &item.event, &item.message, &item.destination); err != nil {
			rows.Close()
			return err
		}
		pendingMessages = append(pendingMessages, item)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	var failures []error
	for _, item := range pendingMessages {
		// Advance the cursor even when one provider record is unavailable.
		if _, err = s.pool.Exec(ctx, `UPDATE app.notifications SET provider_checked_at=now() WHERE id=$1::uuid AND state='sent'`, item.id); err != nil {
			return err
		}
		var status DeliveryStatus
		var lookupErr error
		if routed, ok := provider.(interface {
			DeliveryStatusForEvent(context.Context, string, string) (DeliveryStatus, error)
		}); ok {
			status, lookupErr = routed.DeliveryStatusForEvent(ctx, item.event, item.message)
		} else {
			status, lookupErr = provider.DeliveryStatus(ctx, item.message)
		}
		if errors.Is(lookupErr, ErrDeliveryLookupUnsupported) {
			continue
		}
		if lookupErr != nil {
			failures = append(failures, lookupErr)
			continue
		}
		destination, decodeErr := s.decryptDestination(item.destination)
		if decodeErr != nil || status.ID != item.message || status.Channel != channel || len(status.To) != 1 || !sameDeliveryDestination(channel, status.To[0], string(destination)) {
			failures = append(failures, errors.New("message delivery identity did not match"))
			continue
		}
		switch status.Status {
		case "delivered":
			if status.DeliveredAt == nil {
				failures = append(failures, ErrInvalidDeliveryReceipt)
				continue
			}
			err = s.RecordDeliveryReceipt(ctx, channel, DeliveryReceipt{EventID: deliveryEvidenceID(channel, item.event, item.message), NotificationEventID: item.event, MessageID: item.message, DeliveredAt: *status.DeliveredAt})
		case "bounced", "failed", "complained":
			// The original send was accepted. Never resubmit it on a later bounce.
			_, err = s.pool.Exec(ctx, `UPDATE app.notifications SET state='failed',failed_at=now(),failure_reason=$2,delivery_attempts=GREATEST(delivery_attempts,8),next_attempt_at=NULL,updated_at=now() WHERE id=$1::uuid AND state='sent'`, item.id, channel+" delivery outcome: "+status.Status)
		default:
			err = nil
		}
		if err != nil {
			failures = append(failures, err)
		}
	}
	return errors.Join(failures...)
}

func sameDeliveryDestination(channel, got, expected string) bool {
	if channel == ChannelSMS || channel == ChannelWhatsApp {
		return validMessagingPhone(got) && validMessagingPhone(expected) && strings.TrimPrefix(got, "+") == strings.TrimPrefix(expected, "+")
	}
	return strings.EqualFold(strings.TrimSpace(got), strings.TrimSpace(expected))
}
func deliveryEvidenceID(channel, event, message string) string {
	// Provider message IDs are not globally unique. The pinned notification
	// event separates evidence from different providers and different accounts.
	digest := sha256.Sum256([]byte(event + "\x00" + message))
	return channel + "-delivered-v2:" + hex.EncodeToString(digest[:])
}
