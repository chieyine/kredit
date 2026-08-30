package reports

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPostgresAnalyticsIsPrivacySafeAndRestartSafe(t *testing.T) {
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
	store := NewPostgresStore(pool, Source{})
	event, err := store.Track("report.viewed", "restricted-subject", "reporting", map[string]string{"surface": "supplier"})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _, _ = pool.Exec(ctx, `DELETE FROM app.analytics_events WHERE id=$1::uuid`, event.ID) }()
	if event.SubjectID == "restricted-subject" {
		t.Fatal("raw subject was retained")
	}
	loaded := NewPostgresStore(pool, Source{}).ListAnalytics()
	found := false
	for _, item := range loaded {
		if item.ID == event.ID {
			found = true
			if item.Metadata["surface"] != "supplier" {
				t.Fatalf("metadata lost: %+v", item)
			}
		}
	}
	if !found {
		t.Fatal("analytics event did not survive restart")
	}
}

func TestPostgresProductEventDeduplicationAndScorecardReconciliation(t *testing.T) {
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
	var organizationID string
	if err := pool.QueryRow(ctx, `SELECT id::text FROM app.organizations WHERE id='00000000-0000-7000-8000-000000000010'::uuid`).Scan(&organizationID); err != nil {
		t.Fatal(err)
	}
	key := "integration:product-event:dedupe"
	defer func() { _, _ = pool.Exec(ctx, `DELETE FROM app.analytics_events WHERE deduplication_key=$1`, key) }()
	for range 2 {
		if _, err := pool.Exec(ctx, `SELECT app.record_product_event('test.replayed',$1::uuid,$1::uuid,'operations_reliability',now(),$2,'{}')`, organizationID, key); err != nil {
			t.Fatal(err)
		}
	}
	var count int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM app.analytics_events WHERE deduplication_key=$1`, key).Scan(&count); err != nil || count != 1 {
		t.Fatalf("duplicate product event count=%d err=%v", count, err)
	}
	store := NewPostgresStore(pool, Source{})
	card, err := store.PilotScorecard(ctx, time.Now().UTC().Add(-300*24*time.Hour), time.Now().UTC().Add(24*time.Hour), organizationID)
	if err != nil {
		t.Fatal(err)
	}
	if len(card.KPIs) != 3 || len(card.Guardrails) != 8 || len(card.Reconciliation) == 0 || card.SourceOfTruth == "" || !card.ReconciliationOK {
		t.Fatalf("incomplete pilot scorecard: %+v", card)
	}
	for _, event := range []string{"credit.repeat_sale", "supplier.retained"} {
		if card.Funnel[event] == 0 {
			t.Fatalf("required authoritative event %s is missing from seeded funnel: %+v", event, card.Funnel)
		}
	}
	var mandateEvents int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM app.analytics_events WHERE name='mandate.activated'`).Scan(&mandateEvents); err != nil || mandateEvents == 0 {
		t.Fatalf("required authoritative mandate event is missing: count=%d err=%v", mandateEvents, err)
	}
	requiredGuardrails := map[string]bool{"recognized_loss_rate": false, "support_intervention_rate": false, "accessibility_defects": false}
	for _, metric := range card.Guardrails {
		if _, required := requiredGuardrails[metric.Key]; required {
			requiredGuardrails[metric.Key] = true
		}
	}
	for key, found := range requiredGuardrails {
		if !found {
			t.Fatalf("required weekly guardrail %s is missing", key)
		}
	}
}

func TestPostgresAuthoritativePaymentMandateEmitsCanonicalEvents(t *testing.T) {
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
	providerReference := fmt.Sprintf("analytics-mandate-%d", time.Now().UnixNano())
	var mandateID string
	if err := pool.QueryRow(ctx, `INSERT INTO app.payment_mandates(buyer_subject_type,buyer_subject_id,provider,provider_mandate_id,mandate_type,amount_ceiling_kobo,state,accepted_disclosure_version) VALUES('business','00000000-0000-7000-8000-000000000021'::uuid,'analytics-test',$1,'recurring',100000,'pending','v1') RETURNING id::text`, providerReference).Scan(&mandateID); err != nil {
		t.Fatal(err)
	}
	defer func() {
		_, _ = pool.Exec(ctx, `DELETE FROM app.analytics_events WHERE deduplication_key LIKE 'payment_mandate:'||$1||':%'`, mandateID)
		_, _ = pool.Exec(ctx, `DELETE FROM app.payment_mandates WHERE id=$1::uuid`, mandateID)
	}()
	if _, err := pool.Exec(ctx, `UPDATE app.payment_mandates SET state='active',provider_updated_at=now() WHERE id=$1::uuid`, mandateID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `UPDATE app.payment_mandates SET state='cancelled',provider_updated_at=now() WHERE id=$1::uuid`, mandateID); err != nil {
		t.Fatal(err)
	}
	rows, err := pool.Query(ctx, `SELECT name FROM app.analytics_events WHERE deduplication_key LIKE 'payment_mandate:'||$1||':%' ORDER BY occurred_at,name`, mandateID)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	got := []string{}
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			t.Fatal(err)
		}
		got = append(got, name)
	}
	joined := strings.Join(got, ",")
	for _, want := range []string{"mandate.started", "mandate.activated", "mandate.cancelled"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("canonical mandate event %s missing from %v", want, got)
		}
	}
}
