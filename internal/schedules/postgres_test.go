package schedules

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPostgresStoreRoundTrip(t *testing.T) {
	url := os.Getenv("DATABASE_URL")
	if url == "" || os.Getenv("KREDIT_INTEGRATION") != "1" {
		t.Skip("KREDIT_INTEGRATION=1 and DATABASE_URL are required")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	var userID, organizationID, requestID, agreementID, transactionID, obligationID string
	if err := pool.QueryRow(ctx, `INSERT INTO app.users (normalized_email) VALUES ($1) RETURNING id::text`, fmt.Sprintf("schedule-test-%d@example.test", time.Now().UnixNano())).Scan(&userID); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO app.organizations (legal_name, business_type, business_address, industry) VALUES ('Schedule Test', 'limited_company', 'test', 'test') RETURNING id::text`).Scan(&organizationID); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO app.credit_requests (supplier_organization_id, buyer_user_id, buyer_business_id, principal_kobo, goods_description, due_date, collection_at, state, created_by) VALUES ($1::uuid, $2::uuid, gen_random_uuid(), 3000, 'test', current_date + 30, now() + interval '30 days', 'ACTIVE', $2::uuid) RETURNING id::text`, organizationID, userID).Scan(&requestID); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO app.agreement_versions (credit_request_id, version, canonical_json, document_hash, terms_version, privacy_version, created_by) VALUES ($1::uuid, 1, '{}'::jsonb, $1, 'v1', 'v1', $2::uuid) RETURNING id::text`, requestID, userID).Scan(&agreementID); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO ledger.transactions (event_type, reference_type, reference_id, idempotency_key, effective_at) VALUES ('test', 'credit_request', $1::uuid, $2, now()) RETURNING id::text`, requestID, "schedule-test-"+requestID).Scan(&transactionID); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO app.obligations (credit_request_id, agreement_version_id, supplier_organization_id, buyer_business_id, principal_kobo, currency, lifecycle_status, payment_status, outstanding_kobo, base_fee_kobo, ledger_transaction_id, activated_at) SELECT $1::uuid, $2::uuid, $3::uuid, buyer_business_id, principal_kobo, 'NGN', 'ACTIVE', 'CURRENT', principal_kobo, 0, $4::uuid, now() FROM app.credit_requests WHERE id = $1::uuid RETURNING id::text`, requestID, agreementID, organizationID, transactionID).Scan(&obligationID); err != nil {
		t.Fatal(err)
	}
	defer func() {
		_, _ = pool.Exec(ctx, `DELETE FROM app.schedule_items WHERE schedule_id IN (SELECT id FROM app.repayment_schedules WHERE obligation_id = $1::uuid)`, obligationID)
		_, _ = pool.Exec(ctx, `DELETE FROM app.repayment_schedules WHERE obligation_id = $1::uuid`, obligationID)
		_, _ = pool.Exec(ctx, `DELETE FROM app.obligations WHERE id = $1::uuid`, obligationID)
		_, _ = pool.Exec(ctx, `DELETE FROM app.agreement_versions WHERE id = $1::uuid`, agreementID)
		_, _ = pool.Exec(ctx, `DELETE FROM app.credit_requests WHERE id = $1::uuid`, requestID)
		_, _ = pool.Exec(ctx, `DELETE FROM ledger.transactions WHERE id = $1::uuid`, transactionID)
		_, _ = pool.Exec(ctx, `DELETE FROM app.organizations WHERE id = $1::uuid`, organizationID)
		_, _ = pool.Exec(ctx, `DELETE FROM app.users WHERE id = $1::uuid`, userID)
	}()

	store := NewPostgresStore(pool)
	schedule, items, err := store.Create(CreateInput{ObligationID: obligationID, PrincipalKobo: 3000, ScheduleType: TypeEqual, Count: 2, StartDate: time.Now().UTC().AddDate(0, 0, 1), DueHour: 10, Cadence: CadenceMonthly, Timezone: "UTC", GraceHours: 24})
	if err != nil || len(items) != 2 {
		t.Fatalf("create schedule: %v %+v", err, items)
	}
	if got, loaded, err := store.GetForObligation(obligationID); err != nil || got.ID != schedule.ID || len(loaded) != 2 {
		t.Fatalf("load schedule: %v %+v %+v", err, got, loaded)
	}
	targets, err := store.Allocate(obligationID, 1000)
	if err != nil || len(targets) != 1 {
		t.Fatalf("allocate: %v %+v", err, targets)
	}
	if err := store.ReverseAllocations(targets); err != nil {
		t.Fatal(err)
	}
}
