package notifications

import (
	"context"
	"testing"
	"time"
)

func TestCriticalNotificationFallsBackAndDeduplicates(t *testing.T) {
	store := NewStore("secret")
	whatsapp := NewMockProvider(ChannelWhatsApp)
	whatsapp.SetFail(true)
	email := NewMockProvider(ChannelEmail)
	store.RegisterProvider(whatsapp)
	store.RegisterProvider(email)
	deliveries, err := store.Emit(context.Background(), Event{ID: "event-1", Type: "CollectionSubmitted", RecipientID: "buyer", Priority: PriorityCritical, AmountKobo: 700000, Currency: "NGN", Reference: "TCC-1", NextAction: "review", Date: time.Now(), SecurePath: "/buyer/credit"})
	if err != nil {
		t.Fatal(err)
	}
	if len(deliveries) != 3 || deliveries[0].State != StateFailed || deliveries[1].State != StateSent {
		t.Fatalf("deliveries=%+v", deliveries)
	}
	duplicate, err := store.Emit(context.Background(), Event{ID: "event-1", Type: "CollectionSubmitted", RecipientID: "buyer", Priority: PriorityCritical, AmountKobo: 700000, Currency: "NGN", Reference: "TCC-1", NextAction: "review", Date: time.Now(), SecurePath: "/buyer/credit"})
	if err != nil || len(duplicate) != 3 {
		t.Fatalf("duplicate err=%v len=%d", err, len(duplicate))
	}
	if len(email.Messages()) != 1 {
		t.Fatal("duplicate email sent")
	}
}

func TestRoutineNotificationRespectsQuietHoursAndSecureLink(t *testing.T) {
	store := NewStore("secret")
	email := NewMockProvider(ChannelEmail)
	store.RegisterProvider(email)
	store.SetPreferences("buyer", Preferences{PreferredChannel: ChannelEmail, FallbackChannel: ChannelSMS, QuietStart: 0, QuietEnd: 23})
	deliveries, err := store.Emit(context.Background(), Event{ID: "event-quiet", Type: "PaymentDueSoon", RecipientID: "buyer", Priority: PriorityRoutine, AmountKobo: 300000, Currency: "NGN", Date: time.Now(), Reference: "obl-1", NextAction: "pay", SecurePath: "/pay/obl-1"})
	if err != nil {
		t.Fatal(err)
	}
	if len(deliveries) != 2 || deliveries[0].State != StateScheduled {
		t.Fatalf("deliveries=%+v", deliveries)
	}
	link := store.SecureLink("/pay/obl-1", time.Now().Add(time.Minute))
	if link == "" {
		t.Fatal("secure link missing")
	}
}

func TestPreferenceUpdateChangesFutureOptionalDeliveryButNotRequiredHistory(t *testing.T) {
	store := NewStore("secret")
	email := NewMockProvider(ChannelEmail)
	sms := NewMockProvider(ChannelSMS)
	store.RegisterProvider(email)
	store.RegisterProvider(sms)
	updated, err := store.UpdatePreferences(context.Background(), "buyer", Preferences{PreferredChannel: ChannelEmail, FallbackChannel: ChannelSMS, PaymentRemindersEnabled: false, ProductUpdatesEnabled: false, QuietStart: 22, QuietEnd: 7, Timezone: "Africa/Lagos"}, 1)
	if err != nil || updated.Version != 2 {
		t.Fatalf("updated=%+v err=%v", updated, err)
	}
	optional, err := store.Emit(context.Background(), Event{ID: "optional-1", Type: "PaymentDueSoon", RecipientID: "buyer", Priority: PriorityRoutine})
	if err != nil || len(optional) != 0 {
		t.Fatalf("optional=%+v err=%v", optional, err)
	}
	required, err := store.Emit(context.Background(), Event{ID: "security-1", Type: "AccountRecoveryRequested", RecipientID: "buyer", Priority: PriorityCritical})
	if err != nil || len(required) < 2 {
		t.Fatalf("required=%+v err=%v", required, err)
	}
	if len(store.ListDeliveries("buyer")) != len(required) {
		t.Fatal("suppressed future event mutated delivery history")
	}
}
