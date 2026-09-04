package notifications

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPostgresNotificationIsDurableAndDeduplicatedBeforeProviderSend(t *testing.T) {
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
	var userID string
	email := fmt.Sprintf("notification-%d@example.test", time.Now().UnixNano())
	if err := pool.QueryRow(ctx, `INSERT INTO app.users(normalized_email) VALUES($1) RETURNING id::text`, email).Scan(&userID); err != nil {
		t.Fatal(err)
	}
	eventID := "notification-event-" + userID
	defer func() {
		_, _ = pool.Exec(ctx, `DELETE FROM app.notifications WHERE event_reference=$1`, eventID)
		_, _ = pool.Exec(ctx, `DELETE FROM app.notification_preferences WHERE recipient_id=$1::uuid`, userID)
		_, _ = pool.Exec(ctx, `DELETE FROM app.users WHERE id=$1::uuid`, userID)
	}()
	provider := NewMockProvider(ChannelEmail)
	store := NewPostgresStore(pool, "notification-secret")
	store.RegisterProvider(provider)
	store.SetPreferences(userID, Preferences{PreferredChannel: ChannelEmail, FallbackChannel: ChannelEmail, QuietStart: 0, QuietEnd: 0, Timezone: "Africa/Lagos"})
	event := Event{ID: eventID, Type: "PaymentRecorded", RecipientID: userID, Email: email, Priority: PriorityRoutine, AmountKobo: 2500, Currency: "NGN", Reference: "payment-1"}
	deliveries, err := store.Emit(ctx, event)
	if err != nil {
		t.Fatal(err)
	}
	if len(deliveries) != 1 || deliveries[0].State != StateSent {
		t.Fatalf("unexpected delivery: %+v", deliveries)
	}
	restarted := NewPostgresStore(pool, "notification-secret")
	restarted.RegisterProvider(provider)
	duplicate, err := restarted.Emit(ctx, event)
	if err != nil || len(duplicate) != 1 || duplicate[0].ID != deliveries[0].ID {
		t.Fatalf("durable dedupe failed: %v %+v", err, duplicate)
	}
	if len(provider.Messages()) != 1 {
		t.Fatalf("provider was called %d times", len(provider.Messages()))
	}
	loaded := restarted.ListDeliveries(userID)
	if len(loaded) != 1 || loaded[0].ProviderMessageID == "" {
		t.Fatalf("restart-safe list failed: %+v", loaded)
	}
	var plaintext bool
	if err := pool.QueryRow(ctx, `SELECT position($2::bytea in destination_ciphertext)>0 FROM app.notifications WHERE event_reference=$1`, eventID, []byte(email)).Scan(&plaintext); err != nil {
		t.Fatal(err)
	}
	if plaintext {
		t.Fatal("notification destination was stored in plaintext")
	}
}

func TestScheduledNotificationIsRecoveredAndDeliveredOnce(t *testing.T) {
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
	var userID string
	email := fmt.Sprintf("scheduled-notification-%d@example.test", time.Now().UnixNano())
	if err := pool.QueryRow(ctx, `INSERT INTO app.users(normalized_email) VALUES($1) RETURNING id::text`, email).Scan(&userID); err != nil {
		t.Fatal(err)
	}
	eventID := "scheduled-notification-" + userID
	defer func() {
		_, _ = pool.Exec(ctx, `DELETE FROM app.notifications WHERE event_reference=$1`, eventID)
		_, _ = pool.Exec(ctx, `DELETE FROM app.notification_preferences WHERE recipient_id=$1::uuid`, userID)
		_, _ = pool.Exec(ctx, `DELETE FROM app.users WHERE id=$1::uuid`, userID)
	}()
	store := NewPostgresStore(pool, "scheduled-secret")
	store.now = func() time.Time { return time.Date(2026, 8, 17, 23, 0, 0, 0, time.FixedZone("Africa/Lagos", 3600)) }
	store.SetPreferences(userID, Preferences{PreferredChannel: ChannelEmail, FallbackChannel: ChannelEmail, QuietStart: 22, QuietEnd: 7, Timezone: "Africa/Lagos"})
	deliveries, err := store.Emit(ctx, Event{ID: eventID, Type: "PaymentDueSoon", RecipientID: userID, Email: email, Priority: PriorityRoutine})
	if err != nil || len(deliveries) != 1 || deliveries[0].State != StateScheduled {
		t.Fatalf("notification was not scheduled: %v %+v", err, deliveries)
	}
	if _, err := pool.Exec(ctx, `UPDATE app.notifications SET scheduled_at=now()-interval '1 minute' WHERE id=$1::uuid`, deliveries[0].ID); err != nil {
		t.Fatal(err)
	}
	provider := NewMockProvider(ChannelEmail)
	restarted := NewPostgresStore(pool, "scheduled-secret")
	restarted.RegisterProvider(provider)
	ids, err := restarted.DueDeliveryIDs(ctx, 10)
	found := false
	for _, id := range ids {
		if id == deliveries[0].ID {
			found = true
			break
		}
	}
	if err != nil || !found {
		t.Fatalf("scheduled work was not recovered: %v %v", err, ids)
	}
	if err := restarted.DeliverScheduled(ctx, deliveries[0].ID); err != nil {
		t.Fatal(err)
	}
	if err := restarted.DeliverScheduled(ctx, deliveries[0].ID); err != nil {
		t.Fatal(err)
	}
	if len(provider.Messages()) != 1 || provider.Messages()[0].Destination != email {
		t.Fatalf("scheduled delivery was not exactly once: %+v", provider.Messages())
	}
}

func TestScheduledSupplierReminderRechecksWithdrawnConsent(t *testing.T) {
	if os.Getenv("KREDIT_INTEGRATION") != "1" {
		t.Skip("integration database required")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	var user, org string
	if err = pool.QueryRow(ctx, `INSERT INTO app.users(normalized_email) VALUES($1) RETURNING id::text`, fmt.Sprintf("consent-reminder-%d@example.test", time.Now().UnixNano())).Scan(&user); err != nil {
		t.Fatal(err)
	}
	if err = pool.QueryRow(ctx, `INSERT INTO app.organizations(legal_name,business_type,business_address,industry) VALUES('Consent reminder','limited_company','Lagos','retail') RETURNING id::text`).Scan(&org); err != nil {
		t.Fatal(err)
	}
	defer func() {
		_, _ = pool.Exec(ctx, `DELETE FROM app.notifications WHERE recipient_id=$1::uuid;DELETE FROM app.notification_preferences WHERE recipient_id=$1::uuid;DELETE FROM app.organizations WHERE id=$2::uuid;DELETE FROM app.users WHERE id=$1::uuid`, user, org)
	}()
	allowed := true
	store := NewPostgresStore(pool, "consent-secret")
	store.SetReminderConsent(func(context.Context, string, string) (bool, error) { return allowed, nil })
	store.SetPreferences(user, Preferences{PreferredChannel: ChannelEmail, FallbackChannel: ChannelEmail, PaymentRemindersEnabled: true, QuietStart: 0, QuietEnd: 0, Timezone: "Africa/Lagos"})
	d, err := store.Emit(ctx, Event{ID: "consent-due-" + user, Type: "PaymentDueSoon", RecipientID: user, OrganizationID: org, Email: "buyer@example.test", Priority: PriorityRoutine, DeferDelivery: true})
	if err != nil || len(d) != 1 {
		t.Fatalf("queue: %+v %v", d, err)
	}
	allowed = false
	provider := NewMockProvider(ChannelEmail)
	store.RegisterProvider(provider)
	if err = store.DeliverScheduled(ctx, d[0].ID); err != nil {
		t.Fatal(err)
	}
	var state string
	if err = pool.QueryRow(ctx, `SELECT state FROM app.notifications WHERE id=$1::uuid`, d[0].ID).Scan(&state); err != nil || state != StateSuppressed || len(provider.Messages()) != 0 {
		t.Fatalf("withdrawn reminder state=%s sent=%d err=%v", state, len(provider.Messages()), err)
	}
}

func TestPostgresPreferenceVersionAndFutureSuppression(t *testing.T) {
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
	var userID string
	email := fmt.Sprintf("preferences-%d@example.test", time.Now().UnixNano())
	if err = pool.QueryRow(ctx, `INSERT INTO app.users(normalized_email) VALUES($1) RETURNING id::text`, email).Scan(&userID); err != nil {
		t.Fatal(err)
	}
	defer func() {
		_, _ = pool.Exec(ctx, `DELETE FROM app.notifications WHERE recipient_id=$1::uuid;DELETE FROM app.notification_preferences WHERE recipient_id=$1::uuid;DELETE FROM app.users WHERE id=$1::uuid`, userID)
	}()
	store := NewPostgresStore(pool, "preferences-secret")
	emailProvider := NewMockProvider(ChannelEmail)
	smsProvider := NewMockProvider(ChannelSMS)
	store.RegisterProvider(emailProvider)
	store.RegisterProvider(smsProvider)
	p, err := store.UpdatePreferences(ctx, userID, Preferences{PreferredChannel: ChannelEmail, FallbackChannel: ChannelSMS, PaymentRemindersEnabled: false, QuietStart: 22, QuietEnd: 7, Timezone: "Africa/Lagos"}, 1)
	if err != nil || p.Version != 1 {
		t.Fatalf("initial=%+v err=%v", p, err)
	}
	optional, err := store.Emit(ctx, Event{ID: "optional-" + userID, Type: "PaymentDueSoon", RecipientID: userID, Email: email, Priority: PriorityRoutine})
	if err != nil || len(optional) != 0 {
		t.Fatalf("optional=%+v err=%v", optional, err)
	}
	required, err := store.Emit(ctx, Event{ID: "required-" + userID, Type: "AccountRecoveryRequested", RecipientID: userID, Email: email, Priority: PriorityCritical})
	if err != nil || len(required) != 1 || required[0].Channel != ChannelEmail || len(smsProvider.Messages()) != 0 {
		t.Fatalf("required=%+v err=%v", required, err)
	}
	restarted := NewPostgresStore(pool, "preferences-secret")
	loaded, err := restarted.GetPreferences(ctx, userID)
	if err != nil || loaded.PaymentRemindersEnabled || loaded.Version != 1 {
		t.Fatalf("loaded=%+v err=%v", loaded, err)
	}
	if _, err = restarted.UpdatePreferences(ctx, userID, Preferences{PreferredChannel: ChannelEmail, FallbackChannel: ChannelSMS, Timezone: "Africa/Lagos"}, 99); err == nil {
		t.Fatal("stale preference update accepted")
	}
}
