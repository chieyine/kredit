package reports

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPaymentMetricsRespectCompletionAndChronology(t *testing.T) {
	if os.Getenv("KREDIT_INTEGRATION") != "1" || os.Getenv("DATABASE_URL") == "" {
		t.Skip("isolated integration database required")
	}
	pool, err := pgxpool.New(t.Context(), os.Getenv("DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	// CTE fixtures exercise the production SQL without changing persistent data.
	const fixtures = `WITH
 obligations(id,credit_request_id,principal_kobo,activated_at,payment_status,supplier_organization_id) AS (VALUES
 ('timely','c1',100::bigint,'2026-08-01'::timestamptz,'PAID','00000000-0000-7000-8000-000000000010'::uuid),
 ('late','c2',100,'2026-08-01','PAID','00000000-0000-7000-8000-000000000010'),
 ('future','c3',100,'2026-08-01','PAID','00000000-0000-7000-8000-000000000010'),
 ('written-off','c4',100,'2026-08-01','PAID','00000000-0000-7000-8000-000000000010')),
 payments(id,obligation_id,state,amount_kobo,paid_at) AS (VALUES
 ('p1','timely','recognized',50::bigint,'2026-09-05'::timestamptz),
 ('p2','timely','recognized',50,'2026-09-20'),
 ('p3','late','recognized',50,'2026-09-05'),
 ('p4','late','recognized',50,'2026-09-20'),
 ('p5','future','recognized',50,'2026-09-05'),
 ('p6','future','recognized',50,'2026-10-05'),
 ('p7','written-off','recognized',50,'2026-09-05')),
 credit_requests(id,collection_at) AS (VALUES
 ('c1','2026-09-06'::timestamptz),('c2','2026-09-04'),('c3','2026-09-06'),('c4','2026-09-06')),
 schedule_items(id,collection_at) AS (VALUES
 ('s1','2026-09-06'::timestamptz),('s2','2026-09-21'),('s3','2026-09-04'),('s4','2026-09-21')),
 payment_allocations(payment_id,schedule_item_id,amount_kobo) AS (VALUES
 ('p1','s1',50::bigint),('p2','s2',50),('p3','s3',50),('p4','s4',50)),
 collection_attempts(obligation_id,state,requested_at,final_at,succeeded_amount_kobo) AS (VALUES
 ('timely','FAILED','2026-09-05'::timestamptz,'2026-09-05'::timestamptz,0::bigint),
 ('timely','PARTIAL','2026-09-06','2026-09-06',50),
 ('late','SUCCEEDED','2026-09-04','2026-09-04',50),
 ('late','FAILED','2026-09-05','2026-09-05',0),
 ('late','PARTIAL','2026-09-06','2026-09-06',0),
 ('late','SUCCEEDED','2026-10-02','2026-10-02',50)), `
	replacer := strings.NewReplacer("app.obligations", "obligations", "app.payments", "payments",
		"app.credit_requests", "credit_requests", "app.schedule_items", "schedule_items",
		"app.payment_allocations", "payment_allocations", "app.collection_attempts", "collection_attempts")
	from := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	to := from.AddDate(0, 1, 0)
	for _, test := range []struct {
		name, query string
		want        float64
	}{
		{"final payment cohort excludes future completion and write-offs", daysToPaymentSQL, 50},
		{"instalment deadlines distinguish timely and late payments", onTimePaymentSQL, 50},
		{"recovery requires positive later collection inside window", failedCollectionRecoverySQL, 50},
	} {
		t.Run(test.name, func(t *testing.T) {
			query := fixtures + strings.TrimPrefix(replacer.Replace(test.query), "WITH ")
			var got float64
			if err := pool.QueryRow(t.Context(), query, from, to, "").Scan(&got); err != nil {
				t.Fatal(err)
			}
			if got != test.want {
				t.Fatalf("metric = %v, want %v", got, test.want)
			}
			if err := pool.QueryRow(t.Context(), query, from, to, "00000000-0000-7000-8000-000000000099").Scan(&got); err != nil {
				t.Fatal(err)
			}
			if got != 0 {
				t.Fatalf("unrelated supplier metric = %v, want 0", got)
			}
		})
	}
}
