package whatsapp

import (
	"context"
	"errors"
	"testing"
)

func TestCommandSyntaxIsDistinctFromEventIntegrity(t *testing.T) {
	handler := NewHandler("synthetic-signing-key")
	event := Event{ID: "event-1", From: "2348000000000", Text: "Unstructured request"}
	event.Signature = handler.Sign(event)
	_, err := handler.Handle(context.Background(), event)
	var syntax *CommandSyntaxError
	if !errors.As(err, &syntax) {
		t.Fatalf("unknown syntax not distinguished: %v", err)
	}
	event.Text = "different request"
	event.Signature = handler.Sign(event)
	_, err = handler.Handle(context.Background(), event)
	if !errors.Is(err, ErrEventConflict) || errors.As(err, &syntax) {
		t.Fatalf("conflicting content mistaken for syntax: %v", err)
	}
	event.Signature = "invalid"
	if _, err := handler.Handle(context.Background(), event); !errors.Is(err, ErrInvalidEvent) {
		t.Fatalf("invalid event classification: %v", err)
	}
}

func TestCancelledMessageDoesNotConsumeItsIdentity(t *testing.T) {
	handler := NewHandler("synthetic-signing-key")
	event := Event{ID: "event-1", From: "2348000000000", Text: "how much is due"}
	event.Signature = handler.Sign(event)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := handler.Handle(ctx, event); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled work was accepted: %v", err)
	}
	command, err := handler.Handle(context.Background(), event)
	if err != nil || command.Kind != CommandQuery {
		t.Fatalf("cancelled work poisoned replay: %+v %v", command, err)
	}
}
