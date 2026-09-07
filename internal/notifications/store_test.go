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
	// Exercise the scheduling branch independently of the CI runner's wall clock.
	now := time.Date(2026, time.September, 6, 22, 30, 0, 0, time.UTC)
	store.now = func() time.Time { return now }
	email := NewMockProvider(ChannelEmail)
	sms := NewMockProvider(ChannelSMS)
	store.RegisterProvider(email)
	store.RegisterProvider(sms)
	store.SetPreferences("buyer", Preferences{PreferredChannel: ChannelEmail, FallbackChannel: ChannelSMS, QuietStart: 0, QuietEnd: 23, Timezone: "UTC"})
	deliveries, err := store.Emit(context.Background(), Event{ID: "event-quiet", Type: "PaymentDueSoon", RecipientID: "buyer", Priority: PriorityRoutine, AmountKobo: 300000, Currency: "NGN", Date: now, Reference: "obl-1", NextAction: "pay", SecurePath: "/pay/obl-1"})
	if err != nil {
		t.Fatal(err)
	}
	if len(deliveries) != 2 {
		t.Fatalf("deliveries=%+v", deliveries)
	}
	want := time.Date(2026, time.September, 6, 23, 0, 0, 0, time.UTC)
	for _, delivery := range deliveries {
		if delivery.State != StateScheduled || !delivery.ScheduledAt.Equal(want) {
			t.Fatalf("delivery=%+v; want scheduled at %v", delivery, want)
		}
	}
	if len(email.Messages()) != 0 || len(sms.Messages()) != 0 {
		t.Fatal("a provider received a routine message during quiet hours")
	}
	if store.SecureLink("/pay/obl-1", now.Add(time.Minute)) == "" {
		t.Fatal("secure link missing")
	}
}

func TestQuietHoursBoundariesAndCalendarRollover(t *testing.T) {
	for _, tc := range []struct {
		name, zone, at, end string
		startHour, endHour  int
		quiet               bool
	}{
		{"before start", "Africa/Lagos", "2026-09-06T20:59:59Z", "", 22, 7, false},
		{"at start", "Africa/Lagos", "2026-09-06T21:00:00Z", "2026-09-07T06:00:00Z", 22, 7, true},
		{"after local midnight", "Africa/Lagos", "2026-09-06T23:30:00Z", "2026-09-07T06:00:00Z", 22, 7, true},
		{"before end", "Africa/Lagos", "2026-09-07T05:59:59Z", "2026-09-07T06:00:00Z", 22, 7, true},
		{"at end", "Africa/Lagos", "2026-09-07T06:00:00Z", "", 22, 7, false},
		{"year rollover", "Africa/Lagos", "2026-12-31T21:30:00Z", "2027-01-01T06:00:00Z", 22, 7, true},
		{"daytime window", "UTC", "2026-09-06T22:59:59Z", "2026-09-06T23:00:00Z", 0, 23, true},
		{"daytime end", "UTC", "2026-09-06T23:00:00Z", "", 0, 23, false},
		{"disabled window", "Africa/Lagos", "2026-09-06T23:30:00Z", "", 0, 0, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			now, err := time.Parse(time.RFC3339, tc.at)
			if err != nil {
				t.Fatal(err)
			}
			prefs := Preferences{QuietStart: tc.startHour, QuietEnd: tc.endHour, Timezone: tc.zone}
			if got := inQuietHours(now, prefs); got != tc.quiet {
				t.Fatalf("quiet=%v, want %v at %v", got, tc.quiet, now)
			}
			if tc.quiet && nextQuietEnd(now, prefs).Format(time.RFC3339) != tc.end {
				t.Fatalf("scheduled end=%v; want %s", nextQuietEnd(now, prefs), tc.end)
			}
		})
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
	}{{2500, "NGN 25"}, {1, "NGN 0.01"}, {-50, "NGN -0.50"}, {120000000, "NGN 1,200,000"}, {120000050, "NGN 1,200,000.50"}, {9223372036854775807, "NGN 92,233,720,368,547,758.07"}} {
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
