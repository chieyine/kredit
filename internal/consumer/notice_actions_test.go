package consumer

import (
	"context"
	"strings"
	"testing"
)

func TestEveryConsumerActionHasAnExplicitNotice(t *testing.T) {
	for _, action := range []string{
		"accept", "decline", "claim", "payment", "reject_claim", "reverse_payment",
		"release", "received", "cancel", "request_return", "approve_return",
		"reject_return", "escalate", "reduce_price", "refund",
	} {
		label, err := purchaseNoticeLabel(action)
		if err != nil || strings.TrimSpace(label) == "" {
			t.Errorf("%s has no usable notice: %q, %v", action, label, err)
		}
	}
	label, err := purchaseNoticeLabel("decline")
	if err != nil || label != "Customer declined the purchase" {
		t.Fatalf("decline notice must not imply acceptance or payment: %q, %v", label, err)
	}
}

func TestUnknownConsumerNoticeDoesNotSilentlyProduceBlankContent(t *testing.T) {
	for _, action := range []string{"", "unknown", "pyaument", "declined"} {
		if label, err := purchaseNoticeLabel(action); err == nil || label != "" {
			t.Errorf("accepted unknown action %q: %q, %v", action, label, err)
		}
		// Reject before touching a database transaction or emitting either notice.
		if err := notice(context.Background(), nil, Sale{}, "event", action); err == nil {
			t.Errorf("unknown action %q reached the notification queue", action)
		}
	}
}

func TestConsumerReminderWorkerRejectsMissingPersistence(t *testing.T) {
	for _, store := range []*Store{nil, {}} {
		if err := store.EnqueueReminders(context.Background()); err == nil {
			t.Fatal("missing database was reported as a completed reminder scan")
		}
	}
}
