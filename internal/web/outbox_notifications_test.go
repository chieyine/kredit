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
	if _, err = pool.Exec(ctx, `INSERT INTO app.credit_requests(id,supplier_organization_id,buyer_user_id,buyer_business_id,principal_kobo,goods_description,due_date,collection_at,state,created_by) VALUES($1::uuid,$2::uuid,$3::uuid,$4::uuid,1000,'Notification fixture',current_date+1,now()+interval '2 days','DRAFT',$3::uuid)`, request, org, user, identifier.New()); err != nil {
		t.Fatal(err)
	}
	defer func() {
		_, _ = pool.Exec(ctx, `DELETE FROM app.notifications WHERE event_reference=$1`, "outbox:"+eventID)
	}()
	defer func() {
		_, _ = pool.Exec(ctx, `DELETE FROM app.credit_requests WHERE id=$1::uuid`, request)
	}()
	workerDB, err := db.OpenAsRole(ctx, os.Getenv("RIVER_DATABASE_URL"), "kredit_worker")
	if err != nil {
		t.Fatal(err)
	}
	defer workerDB.Close()
	store := notifications.NewPostgresStore(workerDB.Raw(), "notification-test-secret")
	provider := notifications.NewMockProvider(notifications.ChannelEmail)
	store.RegisterProvider(provider)
	runtime := &Runtime{Database: workerDB, Notifications: store}
	payload, _ := json.Marshal(map[string]any{"event": "OBLIGATION_ACCEPTED"})
	event := outbox.Event{ID: eventID, AggregateType: "credit_request", AggregateID: request, EventType: "notification.requested", Payload: payload}
	if _, err = pool.Exec(ctx, `INSERT INTO app.outbox_events(id,aggregate_type,aggregate_id,event_type,payload,idempotency_key) VALUES($1::uuid,$2,$3,$4,$5::jsonb,$6)`, eventID, event.AggregateType, request, event.EventType, payload, "notice-scope:"+eventID); err != nil {
		t.Fatal(err)
	}
	changed := event
	changed.Payload = json.RawMessage(`{"event":"GOODS_RELEASED"}`)
	if err = runtime.QueueOutboxNotification(ctx, changed); err == nil {
		t.Fatal("notification lookup accepted changed outbox content")
	}
	var appMayRead, workerMayRead bool
	if err = pool.QueryRow(ctx, `SELECT has_function_privilege('kredit_app','app.notification_event_identity(uuid,text,text,jsonb)','EXECUTE'),has_function_privilege('kredit_worker','app.notification_event_identity(uuid,text,text,jsonb)','EXECUTE')`).Scan(&appMayRead, &workerMayRead); err != nil || appMayRead || !workerMayRead {
		t.Fatalf("notification discovery grants app=%v worker=%v error=%v", appMayRead, workerMayRead, err)
	}
	for range 2 {
		if err = runtime.QueueOutboxNotification(ctx, event); err != nil {
			t.Fatal(err)
		}
	}
	all := store.ListDeliveries(user)
	list := []notifications.Delivery{}
	for _, delivery := range all {
		if delivery.EventID == "outbox:"+eventID {
			list = append(list, delivery)
		}
	}
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
