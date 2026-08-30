package operations

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	"kredit/internal/outbox"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPostgresWriteOffIsAtomicIdempotentAndRestartSafe(t *testing.T) {
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
	var actor, org, request, agreement, activation, obligation string
	if err := pool.QueryRow(ctx, `INSERT INTO app.users(normalized_email) VALUES($1) RETURNING id::text`, fmt.Sprintf("operation-%d@example.test", time.Now().UnixNano())).Scan(&actor); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO app.organizations(legal_name,business_type,business_address,industry) VALUES('Operation Test','limited_company','test','test') RETURNING id::text`).Scan(&org); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO app.credit_requests(supplier_organization_id,buyer_user_id,buyer_business_id,principal_kobo,goods_description,due_date,collection_at,state,created_by) VALUES($1::uuid,$2::uuid,gen_random_uuid(),10000,'goods',current_date+7,now()+interval '7 days','ACTIVE',$2::uuid) RETURNING id::text`, org, actor).Scan(&request); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO app.agreement_versions(credit_request_id,version,canonical_json,document_hash,terms_version,privacy_version,created_by) VALUES($1::uuid,1,'{}',$2,'v1','v1',$3::uuid) RETURNING id::text`, request, "operation-"+request, actor).Scan(&agreement); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO ledger.transactions(event_type,reference_type,reference_id,idempotency_key,effective_at) VALUES('test','credit_request',$1,$2,now()) RETURNING id::text`, request, "operation-activation-"+request).Scan(&activation); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO app.obligations(credit_request_id,agreement_version_id,supplier_organization_id,buyer_business_id,principal_kobo,currency,lifecycle_status,payment_status,outstanding_kobo,base_fee_kobo,ledger_transaction_id,activated_at) SELECT $1::uuid,$2::uuid,$3::uuid,buyer_business_id,principal_kobo,'NGN','ACTIVE','UNPAID',principal_kobo,50,$4::uuid,now() FROM app.credit_requests WHERE id=$1::uuid RETURNING id::text`, request, agreement, org, activation).Scan(&obligation); err != nil {
		t.Fatal(err)
	}
	payload, _ := json.Marshal(map[string]any{"request": map[string]any{"id": request, "version": 1}, "obligation": map[string]any{"id": obligation, "outstanding_kobo": 10000, "payment_status": "UNPAID"}})
	if _, err := pool.Exec(ctx, `INSERT INTO app.credit_aggregate_snapshots(credit_request_id,supplier_organization_id,buyer_user_id,aggregate,version) VALUES($1,$2,$3,$4,1)`, request, org, actor, payload); err != nil {
		t.Fatal(err)
	}
	key := "operation-key-" + request
	defer func() {
		_, _ = pool.Exec(ctx, `DELETE FROM app.outbox_events WHERE idempotency_key=$1`, `operation:write_off:`+key)
		_, _ = pool.Exec(ctx, `DELETE FROM app.operation_actions WHERE resource_id=$1::uuid`, obligation)
		_, _ = pool.Exec(ctx, `DELETE FROM ledger.postings WHERE transaction_id IN(SELECT id FROM ledger.transactions WHERE idempotency_key=$1 OR id=$2::uuid)`, `operation:write_off:`+key, activation)
		_, _ = pool.Exec(ctx, `DELETE FROM ledger.transactions WHERE idempotency_key=$1 OR id=$2::uuid`, `operation:write_off:`+key, activation)
		_, _ = pool.Exec(ctx, `DELETE FROM app.credit_aggregate_snapshots WHERE credit_request_id=$1`, request)
		_, _ = pool.Exec(ctx, `DELETE FROM app.obligations WHERE id=$1::uuid`, obligation)
		_, _ = pool.Exec(ctx, `DELETE FROM app.agreement_versions WHERE id=$1::uuid`, agreement)
		_, _ = pool.Exec(ctx, `DELETE FROM app.credit_requests WHERE id=$1::uuid`, request)
		_, _ = pool.Exec(ctx, `DELETE FROM app.organizations WHERE id=$1::uuid`, org)
		_, _ = pool.Exec(ctx, `DELETE FROM app.users WHERE id=$1::uuid`, actor)
	}()
	store := NewPostgresStore(pool, outbox.NewStore(pool), nil)
	action, err := store.WriteOffWithKey(actor, org, obligation, 2000, "Unrecoverable balance", "", key)
	if err != nil {
		t.Fatal(err)
	}
	duplicate, err := NewPostgresStore(pool, outbox.NewStore(pool), nil).WriteOffWithKey(actor, org, obligation, 2000, "Unrecoverable balance", "", key)
	if err != nil || duplicate.ID != action.ID {
		t.Fatalf("idempotent retry failed: %v %+v", err, duplicate)
	}
	var normalized, snapshot, actions int64
	if err := pool.QueryRow(ctx, `SELECT outstanding_kobo FROM app.obligations WHERE id=$1::uuid`, obligation).Scan(&normalized); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT (aggregate#>>'{obligation,outstanding_kobo}')::bigint FROM app.credit_aggregate_snapshots WHERE credit_request_id=$1`, request).Scan(&snapshot); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM app.operation_actions WHERE resource_id=$1::uuid`, obligation).Scan(&actions); err != nil {
		t.Fatal(err)
	}
	if normalized != 8000 || snapshot != 8000 || actions != 1 {
		t.Fatalf("write-off not exactly once: normalized=%d snapshot=%d actions=%d", normalized, snapshot, actions)
	}
}
