package collections

import (
	"context"
	"errors"
	"kredit/internal/credit"
	"kredit/internal/notifications"
	"kredit/internal/observability"
	"kredit/internal/operations"
	"kredit/internal/outbox"
	"kredit/internal/payments"
	"kredit/internal/platformops"
	"kredit/internal/reports"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestCompletionReportsReadDurableBalancesAndFailOnReadErrors(t *testing.T) {
	f := financialFixture(t)
	pg := credit.NewPostgresStore(f.pool, credit.NewStore(nil, nil))
	if _, err := pg.ReadForSupplier(context.Background(), f.organization); err != nil {
		t.Fatal(err)
	}
	reportStore := reports.NewPostgresStore(f.pool, reports.Source{})
	if _, _, err := f.payments.Record(payments.RecordInput{ObligationID: f.id, AmountKobo: 1000, SourceType: payments.SourceSupplierTransfer, RecordedBy: f.user, IdempotencyKey: "report-check:" + f.id}); err != nil {
		t.Fatal(err)
	}
	report, err := reportStore.ReceivablesForSupplier(context.Background(), f.organization)
	if err != nil || report.Summary.OutstandingKobo != 49999000 || report.Summary.VoluntaryPaidKobo != 1000 {
		t.Fatalf("report=%+v error=%v", report, err)
	}
	ops := operations.NewPostgresStore(f.pool, outbox.NewStore(f.pool), nil)
	if _, err = ops.WriteOffWithKey(f.user, f.organization, f.id, 1000, "confirmed small write off", "", "report-forgiven:"+f.id); err != nil {
		t.Fatal(err)
	}
	report, err = reportStore.ReceivablesForSupplier(context.Background(), f.organization)
	if err != nil || report.Summary.OutstandingKobo != 49998000 {
		t.Fatalf("report failed after forgiveness: %+v %v", report, err)
	}
	f.pool.Close()
	if _, err = pg.ReadForSupplier(context.Background(), f.organization); err == nil {
		t.Fatal("closed database returned cached credit views")
	}
	if _, err = reportStore.ReceivablesForSupplier(context.Background(), f.organization); err == nil {
		t.Fatal("database failure became an empty report")
	}
	if _, err = reportStore.HistoryForBuyer(context.Background(), f.user); err == nil {
		t.Fatal("database failure became empty history")
	}
	if _, err = reportStore.ExportReceivablesCSV(context.Background(), f.organization); err == nil {
		t.Fatal("database failure produced a financial export")
	}
	if _, err = f.payments.Read(f.id); err == nil {
		t.Fatal("database failure became an empty payment list")
	}
}

func TestCompletionNoticeRequiresDeliveryAndWaitingPeriod(t *testing.T) {
	f := financialFixture(t)
	ctx := context.Background()
	var eventID string
	err := f.pool.QueryRow(ctx, `INSERT INTO app.outbox_events(aggregate_type,aggregate_id,event_type,payload,idempotency_key) SELECT 'obligation',$1::uuid::text,'notification.requested','{}',app.collection_notice_key(i) FROM app.schedule_items i JOIN app.repayment_schedules s ON s.id=i.schedule_id WHERE s.obligation_id=$1::uuid RETURNING id::text`, f.id).Scan(&eventID)
	if err != nil {
		t.Fatal(err)
	}
	store := notifications.NewPostgresStore(f.pool, "test-only-notice-key")
	provider := notifications.NewMockProvider(notifications.ChannelEmail)
	store.RegisterProvider(provider)
	deliveries, err := store.Emit(ctx, notifications.Event{ID: "outbox:" + eventID, Type: "PriorDebitNotice", RecipientID: f.user, Email: "synthetic@example.test", Priority: notifications.PriorityCritical, DeferDelivery: true})
	if err != nil || len(deliveries) != 1 {
		t.Fatalf("queue: %+v %v", deliveries, err)
	}
	p := &observedProvider{MockProvider: NewMockProvider("secret")}
	engine := f.engine(p)
	engine.RequirePriorNotice(time.Second)
	start := func() error { _, err := engine.Start(ctx, f.id, "notice-gate:"+f.id, time.Now()); return err }
	if err = start(); err == nil || p.count.Load() != 0 {
		t.Fatal("queued notice authorized a debit")
	}
	if err = store.DeliverScheduled(ctx, deliveries[0].ID); err != nil {
		t.Fatal(err)
	}
	if err = start(); err == nil || p.count.Load() != 0 {
		t.Fatal("send acknowledgment authorized a debit")
	}
	var messageID string
	if err = f.pool.QueryRow(ctx, `SELECT provider_message_id FROM app.notifications WHERE id=$1::uuid`, deliveries[0].ID).Scan(&messageID); err != nil {
		t.Fatal(err)
	}
	receipt := notifications.DeliveryReceipt{EventID: "receipt:" + eventID, NotificationEventID: "outbox:" + eventID, MessageID: messageID, DeliveredAt: time.Now().Add(-48 * time.Hour)}
	if err = store.RecordDeliveryReceipt(ctx, "email", receipt); err != nil {
		t.Fatal(err)
	}
	if err = store.RecordDeliveryReceipt(ctx, "email", receipt); err != nil {
		t.Fatal("receipt replay failed", err)
	}
	receipt.MessageID = "wrong-message"
	if err = store.RecordDeliveryReceipt(ctx, "email", receipt); err == nil {
		t.Fatal("changed receipt accepted")
	}
	if err = start(); err == nil || p.count.Load() != 0 {
		t.Fatal("backdated receipt bypassed waiting period")
	}
	// Historical provider receipt fixture: advance past any configured notice floor
	// without weakening production waits or backdating through the receipt API.
	if _, err = f.pool.Exec(ctx, `INSERT INTO app.notification_delivery_receipts(channel,event_id,payload_hash,notification_id,received_at) SELECT channel,event_id||':aged-fixture',payload_hash,notification_id,now()-interval '32 days' FROM app.notification_delivery_receipts WHERE notification_id=$1::uuid`, deliveries[0].ID); err != nil {
		t.Fatal(err)
	}
	if err = start(); err == nil || p.count.Load() != 0 {
		t.Fatal("delivered notice without buyer acknowledgement authorized a debit")
	}
	// Aged buyer fixture must refer to the exact independently delivered notice.
	if _, err = f.pool.Exec(ctx, `INSERT INTO app.collection_notice_acknowledgements(schedule_item_id,buyer_user_id,notification_id,receipt_channel,receipt_event_id,acknowledged_at)
		SELECT i.id,$2::uuid,$3::uuid,'email',$4,now()-interval '32 days'
		FROM app.schedule_items i JOIN app.repayment_schedules s ON s.id=i.schedule_id WHERE s.obligation_id=$1::uuid`, f.id, f.user, deliveries[0].ID, "receipt:"+eventID+":aged-fixture"); err != nil {
		t.Fatal(err)
	}
	if _, err = f.pool.Exec(ctx, `UPDATE app.schedule_items SET collection_at=collection_at+interval '1 minute' WHERE schedule_id IN (SELECT id FROM app.repayment_schedules WHERE obligation_id=$1::uuid)`, f.id); err != nil {
		t.Fatal(err)
	}
	if err = start(); err == nil {
		t.Fatal("changed schedule reused a stale notice")
	}
	if _, err = f.pool.Exec(ctx, `UPDATE app.schedule_items SET collection_at=collection_at-interval '1 minute' WHERE schedule_id IN (SELECT id FROM app.repayment_schedules WHERE obligation_id=$1::uuid)`, f.id); err != nil {
		t.Fatal(err)
	}
	if err = start(); err != nil || p.count.Load() != 1 {
		t.Fatalf("valid delivered notice did not authorize single debit: %v", err)
	}
}

func TestCompletionReconciliationCannotCloseUnresolvedDifference(t *testing.T) {
	f := financialFixture(t)
	ctx := context.Background()
	store := platformops.NewStore(f.pool)
	if _, err := f.pool.Exec(ctx, `UPDATE app.obligations SET outstanding_kobo=outstanding_kobo-1 WHERE id=$1::uuid`, f.id); err != nil {
		t.Fatal(err)
	}
	if err := store.RefreshFinancialReviews(ctx); err != nil {
		t.Fatal(err)
	}
	var id string
	if err := f.pool.QueryRow(ctx, `SELECT id::text FROM app.financial_review_cases WHERE kind='balance' AND target_id=$1`, f.id).Scan(&id); err != nil {
		t.Fatal(err)
	}
	if err := store.DecideFinancialReview(ctx, id, f.user, "claim", "Investigating financial difference"); err != nil {
		t.Fatal(err)
	}
	if err := store.DecideFinancialReview(ctx, id, f.user, "resolve", "Claiming everything agrees now"); err == nil {
		t.Fatal("unresolved discrepancy closed")
	}
	if _, err := f.pool.Exec(ctx, `UPDATE app.obligations SET outstanding_kobo=principal_kobo WHERE id=$1::uuid`, f.id); err != nil {
		t.Fatal(err)
	}
	if err := store.DecideFinancialReview(ctx, id, f.user, "resolve", "Verified authoritative records agree"); err != nil {
		t.Fatal(err)
	}
	if _, err := f.pool.Exec(ctx, `UPDATE app.financial_review_events SET reason='rewritten' WHERE case_id=$1::uuid`, id); err == nil {
		t.Fatal("review history was editable")
	}
	metrics, err := observability.DurableFinancialMetrics(ctx, f.pool)
	if err != nil || !strings.Contains(metrics, "kredit_balance_discrepancies") {
		t.Fatalf("metrics unavailable %s %v", metrics, err)
	}
	if _, err := f.pool.Exec(ctx, `UPDATE app.obligations SET outstanding_kobo=outstanding_kobo-1 WHERE id=$1::uuid`, f.id); err != nil {
		t.Fatal(err)
	}
	if err := store.RefreshFinancialReviews(ctx); err != nil {
		t.Fatal(err)
	}
	var state string
	if err := f.pool.QueryRow(ctx, `SELECT state FROM app.financial_review_cases WHERE id=$1::uuid`, id).Scan(&state); err != nil || state != "OPEN" {
		t.Fatalf("recurrent case failed to reopen: %s %v", state, err)
	}
}

func TestCompletionReadsUnderRuntimeRole(t *testing.T) {
	f := financialFixture(t)
	ctx := context.Background()
	cfg, err := pgxpool.ParseConfig(os.Getenv("DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	cfg.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
		_, err := conn.Exec(ctx, `SET ROLE kredit_app`)
		return err
	}
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	report, err := reports.NewPostgresStore(pool, reports.Source{}).ReceivablesForSupplier(context.Background(), f.organization)
	if err != nil || report.Summary.OutstandingKobo != 50000000 {
		t.Fatalf("runtime report=%+v %v", report, err)
	}
	if _, err = observability.DurableFinancialMetrics(ctx, pool); err != nil {
		t.Fatal("runtime metrics", err)
	}
	if err = platformops.NewStore(pool).RefreshFinancialReviews(ctx); err != nil {
		t.Fatal("runtime reconciliation", err)
	}
}

func TestCompletionPaymentAndNotificationIntentRollBackTogether(t *testing.T) {
	f := financialFixture(t)
	ctx := context.Background()
	tx, err := f.pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	payment, _, err := f.payments.RecordTx(ctx, tx, payments.RecordInput{ObligationID: f.id, AmountKobo: 1000, SourceType: payments.SourceSupplierTransfer, RecordedBy: f.user, IdempotencyKey: "rollback-notice:" + f.id})
	if err != nil {
		_ = tx.Rollback(ctx)
		t.Fatal(err)
	}
	var count int
	if err = tx.QueryRow(ctx, `SELECT count(*) FROM app.outbox_events WHERE idempotency_key=$1`, "notice:payment-recorded:"+payment.ID+":"+f.user).Scan(&count); err != nil || count != 1 {
		_ = tx.Rollback(ctx)
		t.Fatalf("notice was not in payment transaction: %d %v", count, err)
	}
	if err = tx.Rollback(ctx); err != nil {
		t.Fatal(err)
	}
	if err = f.pool.QueryRow(ctx, `SELECT count(*) FROM app.outbox_events WHERE idempotency_key=$1`, "notice:payment-recorded:"+payment.ID+":"+f.user).Scan(&count); err != nil || count != 0 {
		t.Fatalf("rolled-back payment left a notification: %d %v", count, err)
	}
	if err = f.pool.QueryRow(ctx, `SELECT count(*) FROM app.payments WHERE id=$1::uuid`, payment.ID).Scan(&count); err != nil || count != 0 {
		t.Fatal("payment did not roll back")
	}
}

func TestCompletionProviderReversalCreatesFinancialReview(t *testing.T) {
	f := financialFixture(t)
	ctx := context.Background()
	p := NewMockProvider("reversal-check")
	engine := f.engine(p)
	attempt, err := engine.Start(ctx, f.id, "reversal-check:"+f.id, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	p.mu.Lock()
	p.responses[attempt.ProviderCollectionID] = Response{State: ProviderReversed, ProviderCollectionID: attempt.ProviderCollectionID, SettlementReference: "returned-funds"}
	p.mu.Unlock()
	if _, err = engine.Reconcile(ctx, attempt.ID); err != nil {
		t.Fatal(err)
	}
	var count int
	if err = f.pool.QueryRow(ctx, `SELECT count(*) FROM app.financial_discrepancies WHERE kind='provider_reversal' AND target_id=$1`, attempt.ID).Scan(&count); err != nil || count != 1 {
		t.Fatalf("unrecognized reversal was not flagged: %d %v", count, err)
	}
	var paymentID string
	if err = f.pool.QueryRow(ctx, `SELECT id::text FROM app.payments WHERE idempotency_key=$1`, "collection-attempt:"+attempt.ID).Scan(&paymentID); err != nil {
		t.Fatal(err)
	}
	if _, err = f.payments.Reverse(paymentID, f.user, "Verified bank return evidence"); err != nil {
		t.Fatal(err)
	}
	if err = f.pool.QueryRow(ctx, `SELECT count(*) FROM app.financial_discrepancies WHERE kind='provider_reversal' AND target_id=$1`, attempt.ID).Scan(&count); err != nil || count != 0 {
		t.Fatalf("recognized reversal still flagged: %d %v", count, err)
	}
}

func TestFinancialReadsRespectCancelledRequest(t *testing.T) {
	f := financialFixture(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	pg := credit.NewPostgresStore(f.pool, credit.NewStore(nil, nil))
	started := time.Now()
	if _, err := pg.ReadForSupplier(ctx, f.organization); !errors.Is(err, context.Canceled) {
		t.Fatalf("credit read: %v", err)
	}
	reportStore := reports.NewPostgresStore(f.pool, reports.Source{})
	if _, err := reportStore.ReceivablesForSupplier(ctx, f.organization); !errors.Is(err, context.Canceled) {
		t.Fatalf("report read: %v", err)
	}
	if time.Since(started) > time.Second {
		t.Fatal("cancelled reads waited for the database timeout")
	}
}
