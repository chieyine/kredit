package notifications

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
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
	if err = tx.QueryRow(ctx, `SELECT id::text FROM app.notifications WHERE channel=$1 AND event_reference=$2 AND provider_message_id=$3 AND state IN ('sent','delivered','read') FOR UPDATE`, channel, receipt.NotificationEventID, receipt.MessageID).Scan(&id); err != nil {
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
