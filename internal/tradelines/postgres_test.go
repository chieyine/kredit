package tradelines

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"kredit/internal/jobs"
	"kredit/internal/outbox"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPostgresStoreRoundTrip(t *testing.T) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" || os.Getenv("KREDIT_INTEGRATION") != "1" {
		t.Skip("KREDIT_INTEGRATION=1 and DATABASE_URL are required")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	var userID, organizationID, businessID, mandateID string
	if err := pool.QueryRow(ctx, `INSERT INTO app.users(normalized_email) VALUES($1) RETURNING id::text`, fmt.Sprintf("trade-line-%d@example.test", time.Now().UnixNano())).Scan(&userID); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO app.organizations(legal_name,business_type,business_address,industry) VALUES('Trade Line Test','limited_company','test','test') RETURNING id::text`).Scan(&organizationID); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO app.businesses(owner_user_id,legal_name,business_type,business_address,industry,status) VALUES($1::uuid,'Buyer Test','limited_company','test','test','verified') RETURNING id::text`, userID).Scan(&businessID); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO app.payment_mandates(buyer_subject_type,buyer_subject_id,provider,provider_mandate_id,mandate_type,amount_ceiling_kobo,state,accepted_disclosure_version) VALUES('business',$1::uuid,'test','provider-'||$1,'recurring',100000,'active','v1') RETURNING id::text`, businessID).Scan(&mandateID); err != nil {
		t.Fatal(err)
	}
	defer func() {
		_, _ = pool.Exec(ctx, `DELETE FROM app.outbox_events WHERE aggregate_type='trade_line_drawdown' AND aggregate_id IN(SELECT id::text FROM app.drawdowns WHERE trade_line_id IN(SELECT id FROM app.trade_lines WHERE supplier_organization_id=$1::uuid))`, organizationID)
		_, _ = pool.Exec(ctx, `DELETE FROM app.drawdown_reservations WHERE trade_line_id IN(SELECT id FROM app.trade_lines WHERE supplier_organization_id=$1::uuid)`, organizationID)
		_, _ = pool.Exec(ctx, `DELETE FROM app.drawdowns WHERE trade_line_id IN(SELECT id FROM app.trade_lines WHERE supplier_organization_id=$1::uuid)`, organizationID)
		_, _ = pool.Exec(ctx, `DELETE FROM app.trade_lines WHERE supplier_organization_id=$1::uuid`, organizationID)
		_, _ = pool.Exec(ctx, `DELETE FROM app.payment_mandates WHERE id=$1::uuid`, mandateID)
		_, _ = pool.Exec(ctx, `DELETE FROM app.businesses WHERE id=$1::uuid`, businessID)
		_, _ = pool.Exec(ctx, `DELETE FROM app.organizations WHERE id=$1::uuid`, organizationID)
		_, _ = pool.Exec(ctx, `DELETE FROM app.users WHERE id=$1::uuid`, userID)
	}()
	store := NewPostgresStoreWithOutbox(pool, outbox.NewStore(pool))
	line, err := store.CreateLine(CreateLineInput{SupplierOrganizationID: organizationID, BuyerUserID: userID, BuyerBusinessID: businessID, ApprovedLimitKobo: 100000, Cadence: "monthly", StartAt: time.Now().UTC(), EndAt: time.Now().UTC().AddDate(1, 0, 0), MandateID: mandateID, MandateActive: true, MandateVerified: true})
	if err != nil {
		t.Fatal(err)
	}
	drawdown, reservation, updated, err := store.ReserveDrawdown(CreateDrawdownInput{LineID: line.ID, PrincipalKobo: 25000, GoodsDescription: "inventory", IdempotencyKey: "drawdown-" + line.ID})
	if err != nil {
		t.Fatal(err)
	}
	if reservation.State != ReservationPending || updated.AvailableLimitKobo != 75000 {
		t.Fatalf("unexpected reservation: %+v %+v", reservation, updated)
	}
	confirmed, _, err := store.ConfirmDrawdown(drawdown.ID, userID, drawdown.AgreementHash)
	if err != nil || confirmed.State != DrawdownConfirmed {
		t.Fatalf("confirm failed: %v %+v", err, confirmed)
	}
	if _, _, err := store.ReleaseDrawdown(ReleaseInput{DrawdownID: drawdown.ID, SupplierOrganizationID: organizationID, ActorID: userID, DeliveryMethod: "courier", EvidenceReference: "tracking-test"}); err != nil {
		t.Fatal(err)
	}
	issued, _, err := store.RecordDrawdownReceipt(ReceiptInput{DrawdownID: drawdown.ID, BuyerUserID: userID, State: "issue_reported", IssueReason: "quantity mismatch"})
	if err != nil || issued.ReceiptDisputeID == "" || issued.ObligationID != "" {
		t.Fatalf("receipt dispute was not opened safely: drawdown=%+v err=%v", issued, err)
	}
	restarted := NewPostgresStore(pool)
	loaded, ok := restarted.Get(line.ID)
	if !ok || loaded.ReservedPendingKobo != 25000 {
		t.Fatalf("restart-safe line load failed: %+v", loaded)
	}
	statement, err := restarted.Statement(line.ID)
	if err != nil || len(statement.Drawdowns) != 1 || statement.Drawdowns[0].State != DrawdownReceiptIssue || statement.Drawdowns[0].ReceiptDisputeID == "" {
		t.Fatalf("restart-safe statement failed: %v %+v", err, statement)
	}
	var disputeCount, eventCount int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM app.drawdown_receipt_disputes WHERE drawdown_id=$1::uuid`, drawdown.ID).Scan(&disputeCount); err != nil || disputeCount != 1 {
		t.Fatalf("receipt dispute count=%d err=%v", disputeCount, err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM app.outbox_events WHERE aggregate_type='trade_line_drawdown' AND aggregate_id=$1`, drawdown.ID).Scan(&eventCount); err != nil || eventCount != 6 {
		t.Fatalf("drawdown outbox count=%d err=%v", eventCount, err)
	}
	expiring, _, _, err := store.ReserveDrawdown(CreateDrawdownInput{LineID: line.ID, PrincipalKobo: 10000, GoodsDescription: "expiring stock", IdempotencyKey: "expiring-" + line.ID})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `UPDATE app.drawdown_reservations SET expires_at=now()-interval '1 minute' WHERE drawdown_id=$1::uuid`, expiring.ID); err != nil {
		t.Fatal(err)
	}
	if err := jobs.ExpireDrawdownReservations(ctx, pool); err != nil {
		t.Fatal(err)
	}
	postExpiry := NewPostgresStore(pool)
	expiredStatement, err := postExpiry.Statement(line.ID)
	if err != nil {
		t.Fatal(err)
	}
	foundExpired := false
	for _, item := range expiredStatement.Drawdowns {
		if item.ID == expiring.ID && item.State == DrawdownExpired {
			foundExpired = true
		}
	}
	expiredLine, _ := postExpiry.Get(line.ID)
	if !foundExpired || expiredLine.ReservedPendingKobo != 25000 || expiredLine.AvailableLimitKobo != 75000 {
		t.Fatalf("expiry did not atomically release capacity: line=%+v drawdowns=%+v", expiredLine, expiredStatement.Drawdowns)
	}
}
