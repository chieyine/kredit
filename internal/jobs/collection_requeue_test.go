package jobs

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
)

func TestPeriodicCollectionReconciliationCanRunAgainAfterCompletion(t *testing.T) {
	if os.Getenv("KREDIT_INTEGRATION") != "1" {
		t.Skip("integration database required")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	client, err := NewEnqueueClient(pool)
	if err != nil {
		t.Fatal(err)
	}
	args := CollectionArgs{Operation: OpReconcileProvider, ResourceID: fmt.Sprintf("requeue-test-%d", time.Now().UnixNano())}
	defer func() {
		_, _ = pool.Exec(ctx, `DELETE FROM jobs.river_job WHERE args->>'resource_id'=$1`, args.ResourceID)
	}()
	for range 2 {
		if err = client.EnqueueCollection(ctx, args); err != nil {
			t.Fatal(err)
		}
	}
	var count int
	if err = pool.QueryRow(ctx, `SELECT count(*) FROM jobs.river_job WHERE args->>'resource_id'=$1`, args.ResourceID).Scan(&count); err != nil || count != 1 {
		t.Fatalf("active reconciliation should deduplicate: count=%d err=%v", count, err)
	}
	if _, err = pool.Exec(ctx, `UPDATE jobs.river_job SET state='completed',finalized_at=now() WHERE args->>'resource_id'=$1`, args.ResourceID); err != nil {
		t.Fatal(err)
	}
	if err = client.EnqueueCollection(ctx, args); err != nil {
		t.Fatal(err)
	}
	if err = pool.QueryRow(ctx, `SELECT count(*) FROM jobs.river_job WHERE args->>'resource_id'=$1`, args.ResourceID).Scan(&count); err != nil || count != 2 {
		t.Fatalf("completed lookup blocked periodic recheck: count=%d err=%v", count, err)
	}
}

func TestAllPeriodicJobsCanRepeatAfterCompletion(t *testing.T) {
	if os.Getenv("KREDIT_INTEGRATION") != "1" {
		t.Skip("integration database required")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	client, err := NewEnqueueClient(pool)
	if err != nil {
		t.Fatal(err)
	}
	for _, kind := range []string{"maintenance", "reconciliation"} {
		t.Run(kind, func(t *testing.T) {
			resource := fmt.Sprintf("repeat-%s-%d", kind, time.Now().UnixNano())
			defer func() { _, _ = pool.Exec(ctx, `DELETE FROM jobs.river_job WHERE args->>'resource_id'=$1`, resource) }()
			var args river.JobArgs = MaintenanceArgs{Operation: OpEvaluateSchedules, ResourceID: resource}
			if kind == "reconciliation" {
				args = ReconciliationArgs{Operation: OpReconcileLedger, ResourceID: resource}
			}
			// Use the same insertion path/options as the enqueue client.
			for range 2 {
				if _, err := client.inner.Insert(ctx, args, nil); err != nil {
					t.Fatal(err)
				}
			}
			var count int
			if err := pool.QueryRow(ctx, `SELECT count(*) FROM jobs.river_job WHERE args->>'resource_id'=$1`, resource).Scan(&count); err != nil || count != 1 {
				t.Fatalf("active jobs=%d %v", count, err)
			}
			if _, err := pool.Exec(ctx, `UPDATE jobs.river_job SET state='completed',finalized_at=now() WHERE args->>'resource_id'=$1`, resource); err != nil {
				t.Fatal(err)
			}
			if _, err := client.inner.Insert(ctx, args, nil); err != nil {
				t.Fatal(err)
			}
			if err := pool.QueryRow(ctx, `SELECT count(*) FROM jobs.river_job WHERE args->>'resource_id'=$1`, resource).Scan(&count); err != nil || count != 2 {
				t.Fatalf("recurrence jobs=%d %v", count, err)
			}
		})
	}
}
