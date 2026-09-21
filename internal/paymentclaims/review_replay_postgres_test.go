package paymentclaims

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"kredit/internal/db"
	"kredit/internal/ledger"
	"kredit/internal/outbox"
	"kredit/internal/payments"
	"kredit/internal/schedules"
)

// This is an actual restricted-role database regression. The fixture writer
// uses the isolated integration database, but both claim confirmation and its
// payment journal run as kredit_app. No bank or production service is called.
func TestPostgresConfirmationReplayPreservesReviewAndAccounting(t *testing.T) {
	if os.Getenv("KREDIT_INTEGRATION") != "1" || os.Getenv("DATABASE_URL") == "" {
		t.Skip("integration database required")
	}
	ctx := context.Background()
	admin, err := pgxpool.New(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close()
	prefix := "claim-review-" + uuid.NewString()
	var buyer, reviewer, otherReviewer, org, buyerOrg, business, request, agreement, activation, obligation string
	for index, target := range []*string{&buyer, &reviewer, &otherReviewer} {
		if err := admin.QueryRow(ctx, `INSERT INTO app.users(normalized_email,status) VALUES($1,'active') RETURNING id::text`, prefix+string(rune('a'+index))+"@example.test").Scan(target); err != nil {
			t.Fatal(err)
		}
	}
	if err := admin.QueryRow(ctx, `INSERT INTO app.organizations(legal_name,business_type,business_address,industry) VALUES($1,'limited_company','Synthetic address','Test') RETURNING id::text`, prefix).Scan(&org); err != nil {
		t.Fatal(err)
	}
	for _, actor := range []string{reviewer, otherReviewer} {
		if _, err := admin.Exec(ctx, `INSERT INTO app.memberships(organization_id,user_id,role,status) VALUES($1::uuid,$2::uuid,'finance','active')`, org, actor); err != nil {
			t.Fatal(err)
		}
	}
	// A random business UUID is not a purchasing identity. Exercise the current
	// branch/purchasing policies with a real buying workspace and active owner.
	if err := admin.QueryRow(ctx, `INSERT INTO app.organizations(legal_name,business_type,business_address,industry) VALUES($1,'limited_company','Synthetic address','Test') RETURNING id::text`, prefix+"-buyer").Scan(&buyerOrg); err != nil {
		t.Fatal(err)
	}
	if _, err := admin.Exec(ctx, `INSERT INTO app.memberships(organization_id,user_id,role,status) VALUES($1::uuid,$2::uuid,'owner','active')`, buyerOrg, buyer); err != nil {
		t.Fatal(err)
	}
	if err := admin.QueryRow(ctx, `INSERT INTO app.businesses(organization_id,owner_user_id,legal_name,business_type,business_address,industry,status) VALUES($1::uuid,$2::uuid,$3,'limited_company','Synthetic address','Test','verified') RETURNING id::text`, buyerOrg, buyer, prefix+"-buyer").Scan(&business); err != nil {
		t.Fatal(err)
	}
	if err := admin.QueryRow(ctx, `INSERT INTO app.credit_requests(supplier_organization_id,buyer_user_id,buyer_business_id,principal_kobo,goods_description,due_date,collection_at,state,created_by) VALUES($1::uuid,$2::uuid,$4::uuid,10000,'Synthetic goods',current_date+7,now()+interval '7 days','ACTIVE',$3::uuid) RETURNING id::text`, org, buyer, reviewer, business).Scan(&request); err != nil {
		t.Fatal(err)
	}
	if err := admin.QueryRow(ctx, `INSERT INTO app.agreement_versions(credit_request_id,version,canonical_json,document_hash,terms_version,privacy_version,created_by) VALUES($1::uuid,1,'{}',$2,'v1','v1',$3::uuid) RETURNING id::text`, request, prefix, reviewer).Scan(&agreement); err != nil {
		t.Fatal(err)
	}
	if err := admin.QueryRow(ctx, `INSERT INTO ledger.transactions(event_type,reference_type,reference_id,idempotency_key,effective_at) VALUES('test','credit_request',$1,$2,now()) RETURNING id::text`, request, prefix+"-activation").Scan(&activation); err != nil {
		t.Fatal(err)
	}
	if err := admin.QueryRow(ctx, `INSERT INTO app.obligations(credit_request_id,agreement_version_id,supplier_organization_id,buyer_business_id,principal_kobo,currency,lifecycle_status,payment_status,outstanding_kobo,base_fee_kobo,ledger_transaction_id,activated_at) SELECT $1::uuid,$2::uuid,$3::uuid,buyer_business_id,principal_kobo,'NGN','ACTIVE','UNPAID',principal_kobo,50,$4::uuid,now() FROM app.credit_requests WHERE id=$1::uuid RETURNING id::text`, request, agreement, org, activation).Scan(&obligation); err != nil {
		t.Fatal(err)
	}
	payload, err := json.Marshal(map[string]any{"request": map[string]any{"id": request, "version": 1}, "obligation": map[string]any{"id": obligation, "outstanding_kobo": 10000, "payment_status": "UNPAID"}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := admin.Exec(ctx, `INSERT INTO app.credit_aggregate_snapshots(credit_request_id,supplier_organization_id,buyer_user_id,aggregate,version) VALUES($1,$2,$3,$4,1)`, request, org, buyer, payload); err != nil {
		t.Fatal(err)
	}
	due := time.Now().UTC().AddDate(0, 0, 7)
	if _, _, err := schedules.NewPostgresStore(admin).CreateDefault(obligation, ledger.Money(10000), due.Format("2006-01-02"), due, 0); err != nil {
		t.Fatal(err)
	}
	app, err := db.OpenAsRole(ctx, os.Getenv("APP_DATABASE_URL"), "kredit_app")
	if err != nil {
		t.Fatal(err)
	}
	defer app.Close()
	claims := NewPostgresStore(app.Raw())
	buyerContext := db.WithTenantContext(ctx, buyer, "")
	probe, err := app.Raw().Begin(buyerContext)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = probe.Rollback(ctx) }()
	if err := db.SetTenantContext(buyerContext, probe); err != nil {
		t.Fatal(err)
	}
	var canRead, visible bool
	if err := probe.QueryRow(ctx, `SELECT app.can_purchase($1::uuid),EXISTS(SELECT 1 FROM app.obligations WHERE id=$2::uuid)`, business, obligation).Scan(&canRead, &visible); err != nil || !canRead || !visible {
		t.Fatalf("buyer fixture lacks current purchasing/read authority: permission=%t visible=%t error=%v", canRead, visible, err)
	}
	if err := probe.Rollback(ctx); err != nil {
		t.Fatal(err)
	}
	claim, err := claims.Create(buyerContext, CreateInput{ObligationID: obligation, BuyerUserID: buyer, AmountKobo: 2500, PaidAt: time.Now().UTC(), TransferReference: prefix + "-bank", IdempotencyKey: prefix + "-claim"})
	if err != nil {
		t.Fatalf("create claim for authorized buyer: %v", err)
	}
	paymentStore := payments.NewPostgresStore(app.Raw(), outbox.NewStore(app.Raw()), nil)
	reviewContext := db.WithTenantContext(ctx, reviewer, org)
	first, err := claims.Confirm(reviewContext, claim.ID, reviewer, "Matched original bank evidence", paymentStore)
	if err != nil {
		t.Fatal(err)
	}
	if first.State != Confirmed || first.PaymentID == "" || first.ReviewedAt == nil {
		t.Fatal("confirmation did not retain its payment and review evidence")
	}
	for _, reason := range []string{"Matched original bank evidence", "  Matched original bank evidence  "} {
		retry, err := NewPostgresStore(app.Raw()).Confirm(reviewContext, claim.ID, reviewer, reason, paymentStore)
		if err != nil || retry.PaymentID != first.PaymentID || retry.ReviewedBy != first.ReviewedBy || retry.ReviewReason != first.ReviewReason || retry.ReviewedAt == nil || !retry.ReviewedAt.Equal(*first.ReviewedAt) {
			t.Fatalf("restart-safe exact replay changed evidence: %+v %v", retry, err)
		}
	}
	for _, tc := range []struct{ actor, reason string }{
		{otherReviewer, "Matched original bank evidence"},
		{reviewer, "Different bank evidence"},
	} {
		if _, err := claims.Confirm(db.WithTenantContext(ctx, tc.actor, org), claim.ID, tc.actor, tc.reason, paymentStore); !errors.Is(err, ErrReviewConflict) {
			t.Fatalf("different review was accepted: %v", err)
		}
	}
	stored, err := claims.Get(reviewContext, claim.ID)
	if err != nil || stored.PaymentID != first.PaymentID || stored.ReviewedBy != first.ReviewedBy || stored.ReviewReason != first.ReviewReason || stored.ReviewedAt == nil || !stored.ReviewedAt.Equal(*first.ReviewedAt) {
		t.Fatalf("conflicting retry altered the recorded review: %+v %v", stored, err)
	}
	var outstanding, allocated int64
	var paymentCount, journalCount int
	if err := admin.QueryRow(ctx, `SELECT outstanding_kobo FROM app.obligations WHERE id=$1::uuid`, obligation).Scan(&outstanding); err != nil {
		t.Fatal(err)
	}
	if err := admin.QueryRow(ctx, `SELECT COALESCE(sum(allocated_kobo),0) FROM app.schedule_items WHERE schedule_id IN (SELECT id FROM app.repayment_schedules WHERE obligation_id=$1::uuid)`, obligation).Scan(&allocated); err != nil {
		t.Fatal(err)
	}
	if err := admin.QueryRow(ctx, `SELECT count(*) FROM app.payments WHERE obligation_id=$1::uuid`, obligation).Scan(&paymentCount); err != nil {
		t.Fatal(err)
	}
	if err := admin.QueryRow(ctx, `SELECT count(*) FROM ledger.transactions WHERE idempotency_key=$1`, "payment:payment-claim:"+claim.ID).Scan(&journalCount); err != nil {
		t.Fatal(err)
	}
	if outstanding != 7500 || allocated != 2500 || paymentCount != 1 || journalCount != 1 {
		t.Fatalf("replay changed financial state: outstanding=%d allocated=%d payments=%d journals=%d", outstanding, allocated, paymentCount, journalCount)
	}
	// Keep append-only synthetic evidence in the disposable database. Do not
	// disable immutable-history triggers merely to clean up a test fixture.
}
