package outbox

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"time"
)

// Publisher hands a committed domain event to a durable downstream transport.
// It must use Event.IdempotencyKey when the transport supports deduplication.
type Publisher interface {
	Publish(context.Context, Event) error
}

type PublishFunc func(context.Context, Event) error

func (f PublishFunc) Publish(ctx context.Context, event Event) error {
	return f(ctx, event)
}

// Dispatcher drains transactionally-created outbox rows. A row is marked
// published only after the downstream durable transport accepts it; failures
// receive bounded exponential backoff with jitter and expired claims are
// recovered by Store.Claim.
type Dispatcher struct {
	store     *Store
	publisher Publisher
	now       func() time.Time
}

func NewDispatcher(store *Store, publisher Publisher) *Dispatcher {
	return &Dispatcher{store: store, publisher: publisher, now: func() time.Time { return time.Now().UTC() }}
}

func (d *Dispatcher) DispatchOnce(ctx context.Context, limit int) (int, error) {
	if d == nil || d.store == nil || d.publisher == nil {
		return 0, errors.New("outbox store and publisher are required")
	}
	events, err := d.store.Claim(ctx, limit)
	if err != nil {
		return 0, err
	}
	published := 0
	var dispatchErr error
	for _, event := range events {
		if err := d.publisher.Publish(ctx, event); err != nil {
			if event.Attempts >= 10 {
				markErr := d.store.MarkFailed(ctx, event.ID, fmt.Sprintf("dead_letter: max retries reached: %v", err), d.now().Add(24*time.Hour))
				dispatchErr = errors.Join(dispatchErr, fmt.Errorf("outbox event %s reached max retries: %w", event.ID, err), markErr)
				continue
			}
			delay := retryDelay(event.Attempts + 1)
			markErr := d.store.MarkFailed(ctx, event.ID, err.Error(), d.now().Add(delay))
			dispatchErr = errors.Join(dispatchErr, fmt.Errorf("publish outbox event %s: %w", event.ID, err), markErr)
			continue
		}
		if err := d.store.MarkPublished(ctx, event.ID); err != nil {
			dispatchErr = errors.Join(dispatchErr, fmt.Errorf("mark outbox event %s published: %w", event.ID, err))
			continue
		}
		published++
	}
	return published, dispatchErr
}

func retryDelay(attempt int) time.Duration {
	if attempt < 1 {
		attempt = 1
	}
	if attempt > 8 {
		attempt = 8
	}
	base := time.Duration(1<<(attempt-1)) * 5 * time.Second
	var jitterBytes [2]byte
	_, _ = rand.Read(jitterBytes[:])
	fraction := float64(binary.BigEndian.Uint16(jitterBytes[:])) / 65535.0
	jitter := time.Duration(float64(base) * 0.25 * fraction)
	return base + jitter
}
