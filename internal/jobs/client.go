package jobs

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"kredit/internal/observability"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/riverqueue/river/rivertype"
	"go.opentelemetry.io/otel/attribute"
)

const (
	QueueCriticalFinancial = "critical-financial"
	QueueFinancial         = QueueCriticalFinancial
	QueueProvider          = "provider-webhooks"
	QueueCollections       = "collections"
	QueueReconciliation    = "reconciliation"
	QueueNotifications     = "notifications"
	QueueDocuments         = "documents"
	QueueReports           = "reports"
	QueueMaintenance       = "maintenance"
	KindMaintenance        = "kredit.maintenance"
	KindFinancial          = "kredit.financial"
	KindReconciliation     = "kredit.reconciliation"
	KindProviderWebhook    = "kredit.provider_webhook"
	KindCollection         = "kredit.collection"
	KindNotification       = "kredit.notification"
	KindDocument           = "kredit.document"
	KindReport             = "kredit.report"

	OpExpireReservations          = "expire_drawdown_reservations"
	OpReconcileSupplierOnboarding = "reconcile_supplier_onboarding"
	OpEvaluateSchedules           = "evaluate_repayment_schedules"
	OpReconcileLedger             = "reconcile_ledger"
	OpProcessWebhook              = "process_webhook"
	OpReconcileProvider           = "reconcile_provider"
	OpDeliver                     = "deliver"
	OpScan                        = "scan"
	OpExport                      = "export"
)

type MaintenanceArgs struct {
	Operation      string `json:"operation"`
	OrganizationID string `json:"organization_id,omitempty"`
	ResourceID     string `json:"resource_id,omitempty"`
}

func (MaintenanceArgs) Kind() string { return KindMaintenance }
func (MaintenanceArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: QueueMaintenance, MaxAttempts: 5, UniqueOpts: river.UniqueOpts{ByArgs: true, ByState: []rivertype.JobState{rivertype.JobStateAvailable, rivertype.JobStatePending, rivertype.JobStateRunning, rivertype.JobStateRetryable, rivertype.JobStateScheduled}}}
}

type FinancialArgs struct {
	Operation  string `json:"operation"`
	ResourceID string `json:"resource_id,omitempty"`
}

func (FinancialArgs) Kind() string { return KindFinancial }
func (FinancialArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: QueueFinancial, MaxAttempts: 12, UniqueOpts: river.UniqueOpts{ByArgs: true}}
}

type ReconciliationArgs struct {
	Operation  string `json:"operation"`
	ResourceID string `json:"resource_id,omitempty"`
}

func (ReconciliationArgs) Kind() string { return KindReconciliation }
func (ReconciliationArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: QueueReconciliation, MaxAttempts: 12, UniqueOpts: river.UniqueOpts{ByArgs: true, ByState: []rivertype.JobState{rivertype.JobStateAvailable, rivertype.JobStatePending, rivertype.JobStateRunning, rivertype.JobStateRetryable, rivertype.JobStateScheduled}}}
}

type ProviderWebhookArgs struct {
	Provider       string `json:"provider"`
	EventID        string `json:"event_id"`
	EventType      string `json:"event_type"`
	Payload        []byte `json:"payload"`
	SignatureValid bool   `json:"signature_valid"`
}

func (ProviderWebhookArgs) Kind() string { return KindProviderWebhook }
func (ProviderWebhookArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: QueueProvider, MaxAttempts: 10, UniqueOpts: river.UniqueOpts{ByArgs: true}}
}

type CollectionArgs struct {
	Operation  string `json:"operation"`
	ResourceID string `json:"resource_id"`
}

func (CollectionArgs) Kind() string { return KindCollection }
func (CollectionArgs) InsertOpts() river.InsertOpts {
	// A completed lookup must not suppress the next periodic reconciliation.
	return river.InsertOpts{Queue: QueueCollections, MaxAttempts: 10, UniqueOpts: river.UniqueOpts{ByArgs: true, ByState: []rivertype.JobState{rivertype.JobStateAvailable, rivertype.JobStatePending, rivertype.JobStateRunning, rivertype.JobStateRetryable, rivertype.JobStateScheduled}}}
}

type NotificationArgs struct {
	Operation      string `json:"operation"`
	NotificationID string `json:"notification_id"`
	Channel        string `json:"channel,omitempty"`
}

func (NotificationArgs) Kind() string { return KindNotification }
func (NotificationArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: QueueNotifications, MaxAttempts: 8, UniqueOpts: river.UniqueOpts{ByArgs: true}}
}

type DocumentArgs struct {
	Operation  string `json:"operation"`
	DocumentID string `json:"document_id"`
}

func (DocumentArgs) Kind() string { return KindDocument }
func (DocumentArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: QueueDocuments, MaxAttempts: 5, UniqueOpts: river.UniqueOpts{ByArgs: true}}
}

type ReportArgs struct {
	Operation string `json:"operation"`
	ExportID  string `json:"export_id"`
}

func (ReportArgs) Kind() string { return KindReport }
func (ReportArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: QueueReports, MaxAttempts: 4, UniqueOpts: river.UniqueOpts{ByArgs: true}}
}

type OperationHandler func(context.Context, string, string) error

type Handlers struct {
	ProviderWebhook func(context.Context, ProviderWebhookArgs) error
	Collection      OperationHandler
	Notification    OperationHandler
	Document        OperationHandler
	Report          OperationHandler
	Tracer          *observability.Tracer
	Metrics         *observability.Store
}

type TelemetryMiddleware struct {
	Tracer  *observability.Tracer
	Metrics *observability.Store
}

func (m *TelemetryMiddleware) Work(ctx context.Context, job *rivertype.JobRow, doInner func(context.Context) error) error {
	tracer := m.Tracer
	if tracer == nil {
		tracer = observability.NewNoopTracer()
	}
	attributes := []attribute.KeyValue{}
	if job != nil {
		attributes = append(attributes, attribute.String("river.queue", job.Queue), attribute.String("river.kind", job.Kind))
	}
	ctx, span := tracer.Start(ctx, "river.job", attributes...)
	started := time.Now()
	err := doInner(ctx)
	if m.Metrics != nil {
		m.Metrics.Inc("river_jobs_total")
		if err != nil {
			m.Metrics.Inc("river_jobs_failed")
		}
		m.Metrics.ObserveDuration("river_job_duration", time.Since(started))
	}
	outcome := "success"
	if err != nil {
		outcome = "error"
	}
	span.SetAttributes(attribute.String("river.outcome", outcome))
	span.End()
	return err
}

func (*TelemetryMiddleware) IsMiddleware() bool { return true }

type MaintenanceWorker struct {
	river.WorkerDefaults[MaintenanceArgs]
	Pool   *pgxpool.Pool
	Logger *slog.Logger
}

// sharedRateLimitWindow must match internal/web.sensitiveRateLimitWindow. The
// prune keeps four windows of history so a bucket is never dropped while it is
// still counting.
const sharedRateLimitWindow = 10 * time.Minute

func (w *MaintenanceWorker) Work(ctx context.Context, job *river.Job[MaintenanceArgs]) error {
	if w.Pool == nil {
		return errors.New("maintenance worker database is not configured")
	}
	switch job.Args.Operation {
	case OpExpireReservations:
		if err := ExpireDrawdownReservations(ctx, w.Pool); err != nil {
			return err
		}
		// The shared authentication rate-limit buckets are keyed by client and
		// route, so the table would otherwise grow with every distinct caller.
		// Pruning here keeps it bounded without a separate scheduled job.
		if _, err := w.Pool.Exec(ctx, `SELECT app.prune_rate_limits($1)`, sharedRateLimitWindow); err != nil {
			return fmt.Errorf("prune shared rate limits: %w", err)
		}
	case OpEvaluateSchedules:
		_, err := w.Pool.Exec(ctx, `
			UPDATE app.schedule_items
			SET state = CASE
				WHEN now() < due_at THEN 'OPEN'
				WHEN now() < collection_at THEN 'IN_GRACE'
				ELSE 'OVERDUE'
			END
			WHERE state NOT IN ('PAID','CANCELLED','PARTIALLY_PAID')
			  AND allocated_kobo = 0`)
		if err != nil {
			return fmt.Errorf("evaluate repayment schedules: %w", err)
		}
	case OpReconcileSupplierOnboarding:
		if _, err := w.Pool.Exec(ctx, `SELECT app.reconcile_supplier_onboarding(now())`); err != nil {
			return fmt.Errorf("reconcile supplier onboarding: %w", err)
		}
	default:
		return fmt.Errorf("unsupported maintenance operation %q", job.Args.Operation)
	}
	if w.Logger != nil {
		w.Logger.Info("maintenance job completed", "operation", job.Args.Operation)
	}
	return nil
}

// ExpireDrawdownReservations releases capacity, updates the drawdown, and
// records its event atomically. Confirmed reservations also expire until goods
// are released; released reservations are intentionally preserved.
func ExpireDrawdownReservations(ctx context.Context, pool *pgxpool.Pool) error {
	if pool == nil {
		return errors.New("maintenance worker database is not configured")
	}
	_, err := pool.Exec(ctx, `
		WITH expired AS (
			UPDATE app.drawdown_reservations r
			SET state = 'EXPIRED'
			WHERE r.state IN ('PENDING','CONFIRMED')
			  AND r.expires_at <= now()
			  AND EXISTS (
				SELECT 1 FROM app.drawdowns d
				WHERE d.id = r.drawdown_id
				  AND d.state IN ('PENDING_BUYER_CONFIRMATION','BUYER_CONFIRMED')
			  )
			RETURNING r.drawdown_id, r.trade_line_id, r.amount_kobo
		), expired_drawdowns AS (
			UPDATE app.drawdowns d
			SET state = 'EXPIRED'
			FROM expired e
			WHERE d.id = e.drawdown_id
			RETURNING d.id, d.trade_line_id, d.principal_kobo
		), totals AS (
			SELECT trade_line_id, sum(amount_kobo)::bigint AS amount_kobo
			FROM expired
			GROUP BY trade_line_id
		), updated_lines AS (
			UPDATE app.trade_lines l
			SET reserved_pending_kobo = greatest(0, l.reserved_pending_kobo - t.amount_kobo),
				available_limit_kobo = greatest(0, l.approved_limit_kobo - l.current_exposure_kobo - greatest(0, l.reserved_pending_kobo - t.amount_kobo)),
				updated_at = now(),
				version = l.version + 1
			FROM totals t
			WHERE l.id = t.trade_line_id
			RETURNING l.id
		)
		INSERT INTO app.outbox_events(aggregate_type, aggregate_id, event_type, payload, idempotency_key)
		SELECT 'trade_line_drawdown', e.drawdown_id::text, 'TradeLineDrawdownExpired',
			jsonb_build_object('trade_line_id', e.trade_line_id, 'drawdown_id', e.drawdown_id, 'state', 'EXPIRED', 'principal_kobo', e.amount_kobo),
			'trade-line-drawdown:' || e.drawdown_id::text || ':TradeLineDrawdownExpired'
		FROM expired e
		ON CONFLICT (idempotency_key) DO NOTHING`)
	if err != nil {
		return fmt.Errorf("expire drawdown reservations atomically: %w", err)
	}
	return nil
}

type FinancialWorker struct {
	river.WorkerDefaults[FinancialArgs]
	Pool   *pgxpool.Pool
	Logger *slog.Logger
}

func (w *FinancialWorker) Work(ctx context.Context, job *river.Job[FinancialArgs]) error {
	if w.Pool == nil {
		return errors.New("financial worker database is not configured")
	}
	if job.Args.Operation != OpReconcileLedger {
		return fmt.Errorf("unsupported financial operation %q", job.Args.Operation)
	}
	return reconcileLedger(ctx, w.Pool, w.Logger)
}

type ReconciliationWorker struct {
	river.WorkerDefaults[ReconciliationArgs]
	Pool   *pgxpool.Pool
	Logger *slog.Logger
}

func (w *ReconciliationWorker) Work(ctx context.Context, job *river.Job[ReconciliationArgs]) error {
	if w.Pool == nil {
		return errors.New("reconciliation worker database is not configured")
	}
	if job.Args.Operation != OpReconcileLedger {
		return fmt.Errorf("unsupported reconciliation operation %q", job.Args.Operation)
	}
	return reconcileLedger(ctx, w.Pool, w.Logger)
}

func reconcileLedger(ctx context.Context, pool *pgxpool.Pool, logger *slog.Logger) error {
	rows, err := pool.Query(ctx, `SELECT transaction_id::text, COALESCE(SUM(debit_kobo),0), COALESCE(SUM(credit_kobo),0) FROM ledger.postings GROUP BY transaction_id HAVING COALESCE(SUM(debit_kobo),0) <> COALESCE(SUM(credit_kobo),0)`)
	if err != nil {
		return fmt.Errorf("reconcile ledger: %w", err)
	}
	defer rows.Close()
	if rows.Next() {
		var transactionID string
		var debits, credits int64
		if err := rows.Scan(&transactionID, &debits, &credits); err != nil {
			return err
		}
		return fmt.Errorf("ledger imbalance transaction=%s debits=%d credits=%d", transactionID, debits, credits)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if logger != nil {
		logger.Info("financial reconciliation passed")
	}
	return nil
}

type ProviderWebhookWorker struct {
	river.WorkerDefaults[ProviderWebhookArgs]
	Pool    *pgxpool.Pool
	Handler func(context.Context, ProviderWebhookArgs) error
}

func (w *ProviderWebhookWorker) Work(ctx context.Context, job *river.Job[ProviderWebhookArgs]) error {
	if w.Pool == nil {
		return errors.New("provider webhook worker database is not configured")
	}
	if job.Args.Provider == "" || job.Args.EventID == "" || job.Args.EventType == "" || len(job.Args.Payload) == 0 {
		return errors.New("provider, event, type, and payload are required")
	}
	if !job.Args.SignatureValid {
		return river.JobCancel(errors.New("provider webhook signature is invalid"))
	}
	tx, err := w.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, `INSERT INTO app.provider_webhook_inbox (provider,event_id,event_type,payload,signature_valid,provider_sequence) VALUES ($1,$2,$3,$4::jsonb,true,CASE WHEN ($4::jsonb->>'sequence') ~ '^[0-9]+$' THEN ($4::jsonb->>'sequence')::bigint END) ON CONFLICT (provider,event_id) DO UPDATE SET duplicate_count=app.provider_webhook_inbox.duplicate_count+1 WHERE app.provider_webhook_inbox.state='processed'`, job.Args.Provider, job.Args.EventID, job.Args.EventType, string(job.Args.Payload)); err != nil {
		return err
	}
	var state string
	if err := tx.QueryRow(ctx, `SELECT state FROM app.provider_webhook_inbox WHERE provider = $1 AND event_id = $2 FOR UPDATE`, job.Args.Provider, job.Args.EventID).Scan(&state); err != nil {
		return err
	}
	if state == "processed" {
		return tx.Commit(ctx)
	}
	if _, err := tx.Exec(ctx, `UPDATE app.provider_webhook_inbox SET state = 'processing', attempts = attempts + 1 WHERE provider = $1 AND event_id = $2`, job.Args.Provider, job.Args.EventID); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}
	if w.Handler == nil {
		return errors.New("provider webhook handler is not configured")
	}
	if err := w.Handler(ctx, job.Args); err != nil {
		_, _ = w.Pool.Exec(ctx, `UPDATE app.provider_webhook_inbox SET state = 'failed', last_error = $3 WHERE provider = $1 AND event_id = $2`, job.Args.Provider, job.Args.EventID, err.Error())
		return err
	}
	_, err = w.Pool.Exec(ctx, `UPDATE app.provider_webhook_inbox SET state = 'processed', processed_at = NOW(), last_error = NULL WHERE provider = $1 AND event_id = $2`, job.Args.Provider, job.Args.EventID)
	return err
}

type CollectionWorker struct {
	river.WorkerDefaults[CollectionArgs]
	Handler OperationHandler
}

func (w *CollectionWorker) Work(ctx context.Context, job *river.Job[CollectionArgs]) error {
	if w.Handler == nil {
		return errors.New("collection operation handler is not configured")
	}
	return w.Handler(ctx, job.Args.Operation, job.Args.ResourceID)
}

type NotificationWorker struct {
	river.WorkerDefaults[NotificationArgs]
	Handler OperationHandler
}

func (w *NotificationWorker) Work(ctx context.Context, job *river.Job[NotificationArgs]) error {
	if w.Handler == nil {
		return errors.New("notification operation handler is not configured")
	}
	return w.Handler(ctx, job.Args.Operation, job.Args.NotificationID)
}

type DocumentWorker struct {
	river.WorkerDefaults[DocumentArgs]
	Handler OperationHandler
}

func (w *DocumentWorker) Work(ctx context.Context, job *river.Job[DocumentArgs]) error {
	if w.Handler == nil {
		return errors.New("document operation handler is not configured")
	}
	return w.Handler(ctx, job.Args.Operation, job.Args.DocumentID)
}

type ReportWorker struct {
	river.WorkerDefaults[ReportArgs]
	Handler OperationHandler
}

func (w *ReportWorker) Work(ctx context.Context, job *river.Job[ReportArgs]) error {
	if w.Handler == nil {
		return errors.New("report operation handler is not configured")
	}
	return w.Handler(ctx, job.Args.Operation, job.Args.ExportID)
}

type Client struct{ inner *river.Client[pgx.Tx] }

func NewClient(pool *pgxpool.Pool, logger *slog.Logger) (*Client, error) {
	return NewClientWithHandlers(pool, logger, Handlers{})
}

func NewClientWithHandlers(pool *pgxpool.Pool, logger *slog.Logger, handlers Handlers) (*Client, error) {
	if pool == nil {
		return nil, errors.New("river database pool is required")
	}
	workers := river.NewWorkers()
	river.AddWorker(workers, &MaintenanceWorker{Pool: pool, Logger: logger})
	river.AddWorker(workers, &FinancialWorker{Pool: pool, Logger: logger})
	river.AddWorker(workers, &ReconciliationWorker{Pool: pool, Logger: logger})
	// Optional workers are registered only when their durable handler is
	// supplied. This prevents a production process from claiming jobs it cannot
	// safely complete and then converting them into retry/dead-letter churn.
	if handlers.ProviderWebhook != nil {
		river.AddWorker(workers, &ProviderWebhookWorker{Pool: pool, Handler: handlers.ProviderWebhook})
	}
	if handlers.Collection != nil {
		river.AddWorker(workers, &CollectionWorker{Handler: handlers.Collection})
	}
	if handlers.Notification != nil {
		river.AddWorker(workers, &NotificationWorker{Handler: handlers.Notification})
	}
	if handlers.Document != nil {
		river.AddWorker(workers, &DocumentWorker{Handler: handlers.Document})
	}
	if handlers.Report != nil {
		river.AddWorker(workers, &ReportWorker{Handler: handlers.Report})
	}
	client, err := river.NewClient(riverpgxv5.New(pool), &river.Config{
		Queues: map[string]river.QueueConfig{
			river.QueueDefault:  {MaxWorkers: 1},
			QueueFinancial:      {MaxWorkers: 8},
			QueueProvider:       {MaxWorkers: 8},
			QueueCollections:    {MaxWorkers: 12},
			QueueReconciliation: {MaxWorkers: 4},
			QueueNotifications:  {MaxWorkers: 12},
			QueueDocuments:      {MaxWorkers: 4},
			QueueReports:        {MaxWorkers: 4},
			QueueMaintenance:    {MaxWorkers: 2},
		},
		Workers:      workers,
		Schema:       "jobs",
		Middleware:   []rivertype.Middleware{&TelemetryMiddleware{Tracer: handlers.Tracer, Metrics: handlers.Metrics}},
		ErrorHandler: NewDeadLetterHandler(pool, logger),
	})
	if err != nil {
		return nil, fmt.Errorf("create river client: %w", err)
	}
	return &Client{inner: client}, nil
}

func (c *Client) Start(ctx context.Context) error {
	if c == nil || c.inner == nil {
		return errors.New("river client is not configured")
	}
	return c.inner.Start(ctx)
}

func (c *Client) Stop(ctx context.Context) error {
	if c == nil || c.inner == nil {
		return nil
	}
	return c.inner.Stop(ctx)
}

func (c *Client) EnqueueMaintenance(ctx context.Context, args MaintenanceArgs) error {
	return c.insert(ctx, args, nil)
}
func (c *Client) EnqueueFinancial(ctx context.Context, args FinancialArgs) error {
	return c.insert(ctx, args, nil)
}
func (c *Client) EnqueueReconciliation(ctx context.Context, args ReconciliationArgs) error {
	return c.insert(ctx, args, nil)
}
func (c *Client) EnqueueProviderWebhook(ctx context.Context, args ProviderWebhookArgs) error {
	return c.insert(ctx, args, nil)
}
func (c *Client) EnqueueCollection(ctx context.Context, args CollectionArgs) error {
	return c.insert(ctx, args, nil)
}
func (c *Client) EnqueueNotification(ctx context.Context, args NotificationArgs) error {
	return c.insert(ctx, args, nil)
}
func (c *Client) EnqueueDocument(ctx context.Context, args DocumentArgs) error {
	return c.insert(ctx, args, nil)
}
func (c *Client) EnqueueReport(ctx context.Context, args ReportArgs) error {
	return c.insert(ctx, args, nil)
}

func (c *Client) insert(ctx context.Context, args river.JobArgs, opts *river.InsertOpts) error {
	if c == nil || c.inner == nil {
		return errors.New("river client is not configured")
	}
	_, err := c.inner.Insert(ctx, args, opts)
	return err
}

func ScheduleMaintenance(now time.Time) time.Time { return now.Add(time.Minute) }

// NewEnqueueClient never runs workers in the API process.
func NewEnqueueClient(pool *pgxpool.Pool) (*Client, error) {
	inner, err := river.NewClient(riverpgxv5.New(pool), &river.Config{Schema: "jobs"})
	if err != nil {
		return nil, err
	}
	return &Client{inner: inner}, nil
}
