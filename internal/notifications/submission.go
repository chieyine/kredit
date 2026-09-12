package notifications

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrSubmissionUnknown = errors.New("message submission is unconfirmed; review the original provider request before retrying")

type submissionProvider interface {
	Provider
	SubmissionIdentity() string
}

// Meta and Mesaj do not document a send idempotency key. An uncertain POST is
// never repeated automatically. This also survives a worker losing its lease.
func sendWithSubmission(ctx context.Context, pool *pgxpool.Pool, key []byte, provider Provider, m Message) (string, error) {
	native, ok := provider.(submissionProvider)
	if !ok {
		return provider.Send(ctx, m)
	}
	if pool == nil || len(key) < 32 {
		return "", errors.New("durable message tracking is required")
	}
	hash := func(value []byte) string {
		mac := hmac.New(sha256.New, key)
		mac.Write(value)
		return hex.EncodeToString(mac.Sum(nil))
	}
	event := hash([]byte(m.Channel + ":" + m.EventID))
	payload, _ := json.Marshal(struct {
		Identity string
		Message  Message
	}{native.SubmissionIdentity(), m})
	fingerprint := hash(payload)
	result, err := pool.Exec(ctx, `INSERT INTO app.message_submissions(event_key,channel,payload_hash,state,notification_id) VALUES($1,$2,$3,'STARTED',(SELECT id FROM app.notifications WHERE channel=$2 AND event_reference=$4)) ON CONFLICT(event_key) DO NOTHING`, event, m.Channel, fingerprint, m.EventID)
	if err != nil {
		return "", err
	}
	if result.RowsAffected() == 0 {
		var state, original, reference string
		if err = pool.QueryRow(ctx, `SELECT state,payload_hash,COALESCE(provider_reference,'') FROM app.message_submissions WHERE event_key=$1`, event).Scan(&state, &original, &reference); err != nil {
			return "", err
		}
		if original != fingerprint {
			return "", errors.New("saved message differs from this request; provider review is required")
		}
		if state == "ACCEPTED" && reference != "" {
			return reference, nil
		}
		if state == "REJECTED" {
			return "", permanentDeliveryError{422}
		}
		return "", ErrSubmissionUnknown
	}
	reference, sendErr := provider.Send(ctx, m)
	if sendErr != nil {
		if permanentDeliveryFailure(sendErr) {
			_, err = pool.Exec(ctx, `UPDATE app.message_submissions SET state='REJECTED',updated_at=now() WHERE event_key=$1 AND state='STARTED'`, event)
			if err != nil {
				return "", err
			}
			return "", sendErr
		}
		return "", ErrSubmissionUnknown
	}
	if result, err = pool.Exec(ctx, `UPDATE app.message_submissions SET state='ACCEPTED',provider_reference=$2,updated_at=now() WHERE event_key=$1 AND state='STARTED'`, event, reference); err != nil {
		return "", ErrSubmissionUnknown
	}
	if result.RowsAffected() != 1 {
		return "", ErrSubmissionUnknown
	}
	return reference, nil
}

// Unknown submission is held for review, not classified as a provider rejection.
func stopAutomaticDeliveryRetry(err error) bool {
	return permanentDeliveryFailure(err) || errors.Is(err, ErrSubmissionUnknown)
}
