package web

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"kredit/internal/db"
	"kredit/internal/identifier"
	"kredit/internal/jobs"
	"kredit/internal/notifications"
	"kredit/internal/outbox"
)

func TestOutboxNotificationQueuesOnceToActualBuyer(t *testing.T) {
	if os.Getenv("KREDIT_INTEGRATION") != "1" {
		t.Skip("integration database required")
	}
	ctx := context.Background()
	database, err := db.Open(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	pool := database.Raw()
	user, request, eventID := identifier.New(), identifier.New(), identifier.New()
	var org string
	if err = pool.QueryRow(ctx, `INSERT INTO app.organizations(legal_name,business_type,business_address,industry) VALUES('Notice Supplier','limited_company','Lagos','retail') RETURNING id::text`).Scan(&org); err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, `INSERT INTO app.users(id,normalized_email) VALUES($1::uuid,$2)`, user, "notice-"+user+"@example.test"); err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, `INSERT INTO app.credit_aggregate_snapshots(credit_request_id,supplier_organization_id,buyer_user_id,aggregate,version) VALUES($1,$2,$3,'{}',1)`, request, org, user); err != nil {
		t.Fatal(err)
	}
	defer func() {
		_, _ = pool.Exec(ctx, `DELETE FROM app.notifications WHERE event_reference=$1`, "outbox:"+eventID)
	}()
	defer func() {
		_, _ = pool.Exec(ctx, `DELETE FROM app.credit_aggregate_snapshots WHERE credit_request_id=$1`, request)
	}()
	store := notifications.NewPostgresStore(pool, "notification-test-secret")
	provider := notifications.NewMockProvider(notifications.ChannelEmail)
	store.RegisterProvider(provider)
	runtime := &Runtime{Database: database, Notifications: store}
	payload, _ := json.Marshal(map[string]any{"event": "OBLIGATION_ACCEPTED"})
	event := outbox.Event{ID: eventID, AggregateType: "credit_request", AggregateID: request, EventType: "notification.requested", Payload: payload}
	for range 2 {
		if err = runtime.QueueOutboxNotification(ctx, event); err != nil {
			t.Fatal(err)
		}
	}
	list := store.ListDeliveries(user)
	if len(list) != 1 || list[0].Channel != notifications.ChannelEmail || list[0].State != notifications.StateScheduled {
		t.Fatalf("deliveries=%+v", list)
	}
	if len(provider.Messages()) != 0 {
		t.Fatal("outbox worker sent inline instead of queueing")
	}
	if err = store.DeliverScheduled(ctx, list[0].ID); err != nil {
		t.Fatal(err)
	}
	if got := provider.Messages(); len(got) != 1 || !strings.Contains(got[0].Destination, user) {
		t.Fatalf("delivery to wrong recipient: %+v", got)
	}
}

func TestCollectionDiscoveryVisitsEveryPage(t *testing.T) {
	if os.Getenv("KREDIT_INTEGRATION") != "1" {
		t.Skip("integration database required")
	}
	ctx := context.Background()
	database, err := db.Open(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	client, err := jobs.NewEnqueueClient(database.Raw())
	if err != nil {
		t.Fatal(err)
	}
	runtime := &Runtime{Database: database, WebhookJobs: client}
	prefix := "page-test-" + identifier.New() + ":"
	query := `SELECT lpad(n::text,3,'0'),'` + prefix + `'||n::text FROM generate_series(1,205) n WHERE lpad(n::text,3,'0')>$1 ORDER BY 1 LIMIT 100`
	defer func() {
		_, _ = database.Raw().Exec(ctx, `DELETE FROM jobs.river_job WHERE args->>'resource_id' LIKE $1`, prefix+"%")
	}()
	for range 2 {
		if err = runtime.enqueueCollectionPages(ctx, query, jobs.OpReconcileProvider); err != nil {
			t.Fatal(err)
		}
	}
	var count int
	if err = database.Raw().QueryRow(ctx, `SELECT count(*) FROM jobs.river_job WHERE args->>'resource_id' LIKE $1`, prefix+"%").Scan(&count); err != nil || count != 205 {
		t.Fatalf("enqueued=%d %v", count, err)
	}
}
