package whatsapp

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPostgresWebhookDeduplicationSurvivesRestart(t *testing.T) {
	url := os.Getenv("DATABASE_URL")
	if url == "" || os.Getenv("KREDIT_INTEGRATION") != "1" {
		t.Skip("integration database required")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	event := Event{ID: fmt.Sprintf("whatsapp-%d", time.Now().UnixNano()), From: "+2348000000000", Text: "how much is due"}
	handler := NewPostgresHandler(pool, "webhook-secret")
	event.Signature = handler.Sign(event)
	command, err := handler.Handle(ctx, event)
	if err != nil || command.Kind != CommandQuery {
		t.Fatalf("first event failed: %v %+v", err, command)
	}
	duplicate, err := NewPostgresHandler(pool, "webhook-secret").Handle(ctx, event)
	if err != nil || duplicate.Kind != "" {
		t.Fatalf("duplicate was reprocessed: %v %+v", err, duplicate)
	}
	defer func() {
		_, _ = pool.Exec(ctx, `DELETE FROM app.messaging_events WHERE provider='whatsapp' AND provider_event_id=$1`, event.ID)
	}()
	var sender string
	if err := pool.QueryRow(ctx, `SELECT sender FROM app.messaging_events WHERE provider='whatsapp' AND provider_event_id=$1`, event.ID).Scan(&sender); err != nil {
		t.Fatal(err)
	}
	if sender == event.From {
		t.Fatal("raw sender was persisted")
	}
}
