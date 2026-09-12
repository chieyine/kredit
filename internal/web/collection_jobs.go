package web

import (
	"context"
	"errors"
	"kredit/internal/mandates"
	"strings"
	"time"

	"kredit/internal/businesspolicy"
	"kredit/internal/collections"
	"kredit/internal/config"
	"kredit/internal/db"
	"kredit/internal/jobs"
)

// EnqueueCollectionWork discovers work independently of frontend activity.
// Reconciliation always runs, even when new collection/retry flags are off.
func (r *Runtime) EnqueueCollectionWork(ctx context.Context, cfg config.Config) error {
	if r.Database == nil || r.WebhookJobs == nil {
		return nil
	}
	if r.FeeBilling != nil {
		rows, err := r.Database.Raw().Query(ctx, `SELECT organization_id::text FROM app.fee_billing_work()`)
		if err != nil {
			return err
		}
		orgs := []string{}
		for rows.Next() {
			var org string
			if err = rows.Scan(&org); err != nil {
				break
			}
			orgs = append(orgs, org)
		}
		if err == nil {
			err = rows.Err()
		}
		rows.Close()
		if err != nil {
			return err
		}
		for _, org := range orgs {
			if err = r.WebhookJobs.EnqueueCollection(ctx, jobs.CollectionArgs{Operation: "fee_billing", ResourceID: org, OrganizationID: org}); err != nil {
				return err
			}
		}
	}
	policy := businesspolicy.Defaults(cfg)
	if r.BusinessPolicies != nil {
		snapshot, err := r.BusinessPolicies.Read(ctx)
		if err != nil {
			return err
		}
		policy = snapshot.Values
	}
	if r.PlatformOps != nil {
		if err := r.PlatformOps.RefreshFinancialReviews(ctx); err != nil {
			return err
		}
	}
	if cfg.RealCollections || cfg.MonoSweepEnabled {
		if _, err := r.Database.Raw().Exec(ctx, `SELECT app.enqueue_pre_debit_notices()`); err != nil {
			return err
		}
	}

	err := r.enqueueTenantCollectionPages(ctx, `SELECT resource_id,organization_id::text FROM app.collection_attempt_work_page($1,100)`, jobs.OpReconcileProvider)
	if err != nil {
		return err
	}
	if _, err = r.Database.Raw().Exec(ctx, `SELECT app.enqueue_due_payment_notices($1)`, int(policy.UpcomingNoticeDays)); err != nil {
		return err
	}
	if r.Mandates != nil {
		if _, err = r.Database.Raw().Exec(ctx, `INSERT INTO app.outbox_events(aggregate_type,aggregate_id,event_type,payload,idempotency_key) SELECT 'payment_mandate',id::text,'notification.requested',jsonb_build_object('event','MANDATE_EXPIRING','ends_at',ends_at),'mandate-expiring:'||id::text||':'||ends_at::text FROM app.payment_mandates WHERE state='active' AND ends_at>now() AND ends_at<=now()+make_interval(days=>$1) ON CONFLICT(idempotency_key) DO NOTHING`, int(policy.MandateNoticeDays)); err != nil {
			return err
		}
		if err := r.enqueueCollectionPages(ctx, `SELECT id::text,provider||':'||provider_mandate_id FROM app.payment_mandates WHERE state IN ('pending','active') AND (provider_updated_at IS NULL OR provider_updated_at<now()-interval '5 minutes') AND id::text>$1 ORDER BY id::text LIMIT 100`, "reconcile_mandate"); err != nil {
			return err
		}
	}
	if !policy.CollectionsEnabled || !policy.AutomaticCollection {
		return nil
	}
	return r.enqueueTenantCollectionPages(ctx, `SELECT resource_id,organization_id::text FROM app.collection_due_work_page($1,100)`, "collect_due")
}

// Page through the entire eligible set so unresolved early rows cannot starve
// later invoices. Release the read connection before enqueueing each batch.
func (r *Runtime) enqueueCollectionPages(ctx context.Context, query, operation string) error {
	cursor := ""
	for {
		rows, err := r.Database.Raw().Query(ctx, query, cursor)
		if err != nil {
			return err
		}
		type item struct{ id, resource string }
		batch := []item{}
		for rows.Next() {
			var v item
			if err = rows.Scan(&v.id, &v.resource); err != nil {
				rows.Close()
				return err
			}
			batch = append(batch, v)
		}
		err = rows.Err()
		rows.Close()
		if err != nil {
			return err
		}
		for _, v := range batch {
			if err = r.WebhookJobs.EnqueueCollection(ctx, jobs.CollectionArgs{Operation: operation, ResourceID: v.resource}); err != nil {
				return err
			}
			cursor = v.id
		}
		if len(batch) < 100 {
			return nil
		}
	}
}

func (r *Runtime) enqueueTenantCollectionPages(ctx context.Context, query, operation string) error {
	cursor := ""
	for {
		rows, err := r.Database.Raw().Query(ctx, query, cursor)
		if err != nil {
			return err
		}
		type item struct{ id, organizationID string }
		batch := []item{}
		for rows.Next() {
			var v item
			if err = rows.Scan(&v.id, &v.organizationID); err != nil {
				rows.Close()
				return err
			}
			batch = append(batch, v)
		}
		err = rows.Err()
		rows.Close()
		if err != nil {
			return err
		}
		for _, v := range batch {
			if err = r.WebhookJobs.EnqueueCollection(ctx, jobs.CollectionArgs{Operation: operation, ResourceID: v.id, OrganizationID: v.organizationID}); err != nil {
				return err
			}
			cursor = v.id
		}
		if len(batch) < 100 {
			return nil
		}
	}
}

func (r *Runtime) HandleCollectionJob(ctx context.Context, cfg config.Config, args jobs.CollectionArgs) error {
	if args.Operation == "fee_billing" {
		if r.FeeBilling == nil {
			return errors.New("fee billing is unavailable")
		}
		return r.FeeBilling.Run(ctx, args.OrganizationID)
	}
	operation, id := args.Operation, args.ResourceID
	if (operation == jobs.OpReconcileProvider || operation == "collect_due") && args.OrganizationID == "" {
		return errors.New("collection tenant context is required")
	}
	if args.OrganizationID != "" {
		ctx = db.WithTenantContext(ctx, "", args.OrganizationID)
	}
	if operation == "reconcile_mandate" {
		provider, reference, hasProvider := strings.Cut(id, ":")
		if !hasProvider {
			provider = "mono-sweep"
			reference = id
		}
		mandate, err := r.Mandates.GetMandate(mandates.WithProvider(ctx, provider), reference)
		if err != nil {
			return err
		}
		return r.applyVerifiedMandate(ctx, mandate, "mandate-reconciliation:"+id)
	}
	if operation == jobs.OpReconcileProvider {
		_, err := r.Collections.Reconcile(ctx, id)
		return err
	}
	if operation != "collect_due" {
		return errors.New("unsupported collection operation")
	}
	policy := businesspolicy.Defaults(cfg)
	if r.BusinessPolicies != nil {
		snapshot, err := r.BusinessPolicies.Read(ctx)
		if err != nil {
			return err
		}
		policy = snapshot.Values
	}
	if !policy.CollectionsEnabled || !policy.AutomaticCollection {
		return nil
	}
	var eligibility collections.Eligibility
	var err error
	if scoped, ok := r.Collections.(interface {
		EligibilityContext(context.Context, string, time.Time) (collections.Eligibility, error)
	}); ok {
		eligibility, err = scoped.EligibilityContext(ctx, id, time.Now().UTC())
	} else {
		eligibility, err = r.Collections.Eligibility(id, time.Now().UTC())
	}
	if err != nil {
		return err
	}
	if !eligibility.Eligible {
		return nil
	}
	attempts, err := r.readCollectionsAttemptsContext(ctx, id)
	if err != nil {
		return err
	}
	latest := -1
	for i := range attempts {
		if latest < 0 || attempts[i].RequestedAt.After(attempts[latest].RequestedAt) {
			latest = i
		}
	}
	if latest >= 0 && (attempts[latest].State == "FAILED" || attempts[latest].State == "PARTIAL") {
		a := attempts[latest]
		max := int(policy.MaxRetries)
		if max <= 0 {
			max = 3
		}
		if a.AttemptNumber >= max {
			return nil
		}
		if !policy.AutomaticRetry || a.RetryClassification == "final" || a.NextRetryAt.IsZero() || time.Now().Before(a.NextRetryAt) {
			return nil
		}
		_, err = r.Collections.Retry(ctx, a.ID, time.Now().UTC())
		return err
	}
	var itemID string
	tx, err := r.Database.Raw().Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err = db.SetObligationContext(ctx, tx, id); err != nil {
		return err
	}
	if err = tx.QueryRow(ctx, `SELECT i.id::text FROM app.schedule_items i JOIN app.repayment_schedules s ON s.id=i.schedule_id WHERE s.obligation_id=$1::uuid AND i.state NOT IN ('PAID','CANCELLED') AND i.collection_at<=now() AND i.principal_due_kobo>i.allocated_kobo ORDER BY i.sequence LIMIT 1`, id).Scan(&itemID); err != nil {
		return err
	}
	// Release the discovery connection before collection acquires its own
	// transaction; a one-connection pool must not deadlock here.
	if err = tx.Commit(ctx); err != nil {
		return err
	}
	_, err = r.Collections.Start(ctx, id, "due-schedule:"+itemID, time.Now().UTC())
	return err
}
