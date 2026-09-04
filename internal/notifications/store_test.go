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

func TestDeferredEventQueuesWithoutSendingAndKeepsItsIdentity(t *testing.T) {
	store := NewStore("secret")
	provider := NewMockProvider(ChannelEmail)
	store.RegisterProvider(provider)
	event := Event{ID: "deferred", Type: "PaymentRecorded", RecipientID: "buyer", Priority: PriorityCritical, DeferDelivery: true}
	first, err := store.Emit(context.Background(), event)
	if err != nil {
		t.Fatal(err)
	}
	second, err := store.Emit(context.Background(), event)
	if err != nil || len(first) == 0 || len(first) != len(second) {
		t.Fatalf("queue: %v", err)
	}
	for i, delivery := range first {
		if delivery.State != StateScheduled || delivery.ID != second[i].ID {
			t.Fatal("queue identity or state changed")
		}
	}
	if len(provider.Messages()) != 0 {
		t.Fatal("outbox dispatch sent a message directly")
	}
}

func TestNotificationMoneyUsesNairaNotKobo(t *testing.T) {
	for _, tc := range []struct {
		amount int64
		want   string
	}{{2500, "NGN 25.00"}, {1, "NGN 0.01"}, {-50, "NGN -0.50"}, {9223372036854775807, "NGN 92233720368547758.07"}} {
		if got := formatAmount(tc.amount, "NGN"); got != tc.want {
			t.Fatalf("money=%s want=%s", got, tc.want)
		}
	}
}

func TestSupplierReminderConsentIsCheckedBeforeQueueing(t *testing.T) {
	s := NewStore("secret")
	s.SetPreferences("buyer", Preferences{PreferredChannel: ChannelEmail, FallbackChannel: ChannelEmail, PaymentRemindersEnabled: true, Timezone: "Africa/Lagos"})
	allowed := false
	s.SetReminderConsent(func(_ context.Context, buyer, org string) (bool, error) {
		if buyer != "buyer" || org != "org" {
			t.Fatal("wrong consent scope")
		}
		return allowed, nil
	})
	event := Event{ID: "due-1", Type: "PaymentDueSoon", RecipientID: "buyer", OrganizationID: "org", Email: "buyer@example.test", Priority: PriorityRoutine}
	if deliveries, err := s.Emit(context.Background(), event); err != nil || len(deliveries) != 0 {
		t.Fatalf("withdrawn consent queued reminder: %+v %v", deliveries, err)
	}
	allowed = true
	if deliveries, err := s.Emit(context.Background(), event); err != nil || len(deliveries) != 1 {
		t.Fatalf("granted consent blocked reminder: %+v %v", deliveries, err)
	}
}
