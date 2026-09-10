package paymentclaims

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"kredit/internal/db"
	"kredit/internal/ledger"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPostgresClaimIsRestartSafeAndExpiresHold(t *testing.T) {
	if os.Getenv("KREDIT_INTEGRATION") != "1" || os.Getenv("DATABASE_URL") == "" {
		t.Skip("integration database required")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	var userID, orgID, requestID, agreementID, transactionID, obligationID string
	unique := fmt.Sprint(time.Now().UnixNano())
	if err = pool.QueryRow(ctx, `INSERT INTO app.users(normalized_email) VALUES($1) RETURNING id::text`, `claim-`+unique+`@example.test`).Scan(&userID); err != nil {
		t.Fatal(err)
	}
	if err = pool.QueryRow(ctx, `INSERT INTO app.organizations(legal_name,business_type,business_address,industry) VALUES($1,'limited_company','test','test') RETURNING id::text`, `Claim `+unique).Scan(&orgID); err != nil {
		t.Fatal(err)
	}
	if err = pool.QueryRow(ctx, `INSERT INTO app.credit_requests(supplier_organization_id,buyer_user_id,buyer_business_id,principal_kobo,goods_description,due_date,collection_at,state,created_by) VALUES($1::uuid,$2::uuid,gen_random_uuid(),10000,'goods',current_date+7,now()+interval '7 days','ACTIVE',$2::uuid) RETURNING id::text`, orgID, userID).Scan(&requestID); err != nil {
		t.Fatal(err)
	}
	if err = pool.QueryRow(ctx, `INSERT INTO app.agreement_versions(credit_request_id,version,canonical_json,document_hash,terms_version,privacy_version,created_by) VALUES($1::uuid,1,'{}',$2,'v1','v1',$3::uuid) RETURNING id::text`, requestID, "claim-"+unique, userID).Scan(&agreementID); err != nil {
		t.Fatal(err)
	}
	if err = pool.QueryRow(ctx, `INSERT INTO ledger.transactions(event_type,reference_type,reference_id,idempotency_key,effective_at) VALUES('test','credit_request',$1,$2,now()) RETURNING id::text`, requestID, "claim-activation-"+unique).Scan(&transactionID); err != nil {
		t.Fatal(err)
	}
	if err = pool.QueryRow(ctx, `INSERT INTO app.obligations(credit_request_id,agreement_version_id,supplier_organization_id,buyer_business_id,principal_kobo,currency,lifecycle_status,payment_status,outstanding_kobo,base_fee_kobo,ledger_transaction_id,activated_at) SELECT $1::uuid,$2::uuid,$3::uuid,buyer_business_id,principal_kobo,'NGN','ACTIVE','UNPAID',principal_kobo,50,$4::uuid,now() FROM app.credit_requests WHERE id=$1::uuid RETURNING id::text`, requestID, agreementID, orgID, transactionID).Scan(&obligationID); err != nil {
		t.Fatal(err)
	}
	defer func() {
		_, _ = pool.Exec(ctx, `DELETE FROM app.payment_claims WHERE obligation_id=$1::uuid`, obligationID)
		_, _ = pool.Exec(ctx, `DELETE FROM app.obligations WHERE id=$1::uuid`, obligationID)
		_, _ = pool.Exec(ctx, `DELETE FROM app.agreement_versions WHERE id=$1::uuid`, agreementID)
		_, _ = pool.Exec(ctx, `DELETE FROM app.credit_requests WHERE id=$1::uuid`, requestID)
		_, _ = pool.Exec(ctx, `DELETE FROM ledger.transactions WHERE id=$1::uuid`, transactionID)
		_, _ = pool.Exec(ctx, `DELETE FROM app.organizations WHERE id=$1::uuid`, orgID)
		_, _ = pool.Exec(ctx, `DELETE FROM app.users WHERE id=$1::uuid`, userID)
	}()
	ctx = db.WithTenantContext(ctx, userID, orgID)
	store := NewPostgresStore(pool)
	claim, err := store.Create(ctx, CreateInput{ObligationID: obligationID, BuyerUserID: userID, AmountKobo: ledger.Money(2500), PaidAt: time.Now().UTC(), TransferReference: "BANK-" + unique, IdempotencyKey: "claim-" + unique})
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := NewPostgresStore(pool).Get(ctx, claim.ID)
	if err != nil || loaded.TransferReference != claim.TransferReference {
		t.Fatalf("loaded=%+v err=%v", loaded, err)
	}
	if _, err = store.Decide(ctx, claim.ID, userID, Confirmed, "Unverified payment reference", transactionID); err == nil {
		t.Fatal("legacy decision path accepted an arbitrary payment ID")
	}
	after, err := store.Get(ctx, claim.ID)
	if err != nil || after.State != Pending || after.PaymentID != "" {
		t.Fatalf("rejected confirmation changed the claim: %+v %v", after, err)
	}
	if hold := store.ActiveHold(ctx, obligationID, time.Now().UTC()); hold != 2500 {
		t.Fatalf("hold=%d", hold)
	}
	if hold := store.ActiveHold(ctx, obligationID, claim.HoldExpiresAt.Add(time.Second)); hold != 0 {
		t.Fatalf("expired hold=%d", hold)
	}

	app, err := db.OpenAsRole(ctx, os.Getenv("APP_DATABASE_URL"), "kredit_app")
	if err != nil {
		t.Fatal(err)
	}
	defer app.Close()
	restricted := NewPostgresStore(app.Raw())
	for _, scope := range []string{"buyer", "supplier", "obligation"} {
		var list []Claim
		switch scope {
		case "buyer":
			list, err = restricted.ReadForBuyer(ctx, userID)
		case "supplier":
			list, err = restricted.ReadForSupplier(ctx, orgID)
		case "obligation":
			list, err = restricted.ReadForObligation(ctx, obligationID)
		}
		if err != nil || len(list) != 1 || list[0].ID != claim.ID {
			t.Fatalf("restricted %s read=%+v error=%v", scope, list, err)
		}
	}
	if _, err = restricted.Get(ctx, claim.ID); err != nil {
		t.Fatalf("restricted claim read: %v", err)
	}
	var otherOrg string
	if err = pool.QueryRow(ctx, `SELECT gen_random_uuid()::text`).Scan(&otherOrg); err != nil {
		t.Fatal(err)
	}
	if _, err = restricted.Get(db.WithTenantContext(ctx, userID, otherOrg), claim.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("claim leaked across explicit organization: %v", err)
	}
	if hold := restricted.ActiveHold(ctx, obligationID, time.Now()); hold != 2500 {
		t.Fatalf("restricted role ignored payment hold: %d", hold)
	}
	if hold := restricted.ActiveHold(context.Background(), obligationID, time.Now()); hold != ledger.Money(1<<63-1) {
		t.Fatalf("missing identity did not block collection: %d", hold)
	}
	if _, err = restricted.Decide(ctx, claim.ID, userID, Rejected, "Transfer was not received", ""); err != nil {
		t.Fatal(err)
	}
	if _, err = restricted.Decide(ctx, claim.ID, userID, Rejected, "Transfer was not received", ""); err != nil {
		t.Fatalf("same decision retry failed: %v", err)
	}
	if _, err = restricted.Decide(ctx, claim.ID, userID, Rejected, "Changed review details", ""); err == nil {
		t.Fatal("changed rejection replay was accepted")
	}
	app.Close()
	if _, err = restricted.Get(ctx, claim.ID); err == nil || errors.Is(err, ErrNotFound) {
		t.Fatalf("outage was reported as absence: %v", err)
	}
}

func TestUnavailablePaymentClaimCheckBlocksCollection(t *testing.T) {
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, "postgres://test@localhost/test")
	if err != nil {
		t.Fatal(err)
	}
	pool.Close()
	if hold := NewPostgresStore(pool).ActiveHold(ctx, "unavailable", time.Now()); hold != ledger.Money(1<<63-1) {
		t.Fatalf("failed hold lookup allowed collection: %d", hold)
	}
}
