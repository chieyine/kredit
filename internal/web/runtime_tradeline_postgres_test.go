package web

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"kredit/internal/config"
	"kredit/internal/credit"
	"kredit/internal/db"
	"kredit/internal/schedules"
	"kredit/internal/tradelines"

	"github.com/jackc/pgx/v5"
)

func TestPostgresTradeLineActivationCommitsAsOneFinancialTransaction(t *testing.T) {
	url := os.Getenv("DATABASE_URL")
	if url == "" || os.Getenv("KREDIT_INTEGRATION") != "1" {
		t.Skip("integration database required")
	}
	ctx := context.Background()
	database, err := db.Open(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	pool := database.Raw()
	var userID, organizationID, businessID, mandateID string
	if err := pool.QueryRow(ctx, `INSERT INTO app.users(normalized_email) VALUES($1) RETURNING id::text`, fmt.Sprintf("atomic-drawdown-%d@example.test", time.Now().UnixNano())).Scan(&userID); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO app.organizations(legal_name,business_type,business_address,industry) VALUES('Atomic Drawdown Supplier','limited_company','test','test') RETURNING id::text`).Scan(&organizationID); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO app.businesses(owner_user_id,legal_name,business_type,business_address,industry,status) VALUES($1::uuid,'Atomic Buyer','limited_company','test','test','verified') RETURNING id::text`, userID).Scan(&businessID); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO app.payment_mandates(buyer_subject_type,buyer_subject_id,provider,provider_mandate_id,mandate_type,amount_ceiling_kobo,state,accepted_disclosure_version) VALUES('business',$1::uuid,'mock','atomic-'||$1,'recurring',500000,'active','v1') RETURNING id::text`, businessID).Scan(&mandateID); err != nil {
		t.Fatal(err)
	}
	var lineID, drawdownID, obligationID string
	defer func() {
		if obligationID != "" {
			_, _ = pool.Exec(ctx, `DELETE FROM app.schedule_items WHERE schedule_id IN(SELECT id FROM app.repayment_schedules WHERE obligation_id=$1::uuid)`, obligationID)
			_, _ = pool.Exec(ctx, `DELETE FROM app.repayment_schedules WHERE obligation_id=$1::uuid`, obligationID)
		}
		if drawdownID != "" {
			_, _ = pool.Exec(ctx, `DELETE FROM app.outbox_events WHERE aggregate_id IN($1,$2) OR idempotency_key LIKE '%'||$1||'%'`, drawdownID, obligationID)
			_, _ = pool.Exec(ctx, `DELETE FROM ledger.postings WHERE transaction_id IN(SELECT id FROM ledger.transactions WHERE reference_id=$1)`, obligationID)
			_, _ = pool.Exec(ctx, `DELETE FROM app.drawdown_reservations WHERE drawdown_id=$1::uuid`, drawdownID)
			_, _ = pool.Exec(ctx, `DELETE FROM app.drawdowns WHERE id=$1::uuid`, drawdownID)
			_, _ = pool.Exec(ctx, `DELETE FROM app.obligations WHERE credit_request_id=$1::uuid`, drawdownID)
			_, _ = pool.Exec(ctx, `DELETE FROM app.receipt_confirmations WHERE credit_request_id=$1::uuid`, drawdownID)
			_, _ = pool.Exec(ctx, `DELETE FROM app.goods_releases WHERE credit_request_id=$1::uuid`, drawdownID)
			_, _ = pool.Exec(ctx, `DELETE FROM app.agreement_acceptances WHERE credit_request_id=$1::uuid`, drawdownID)
			_, _ = pool.Exec(ctx, `UPDATE app.credit_requests SET agreement_version_id=NULL,acceptance_id=NULL,release_id=NULL,receipt_id=NULL,obligation_id=NULL WHERE id=$1::uuid`, drawdownID)
			_, _ = pool.Exec(ctx, `DELETE FROM app.agreement_versions WHERE credit_request_id=$1::uuid`, drawdownID)
			_, _ = pool.Exec(ctx, `DELETE FROM app.credit_aggregate_snapshots WHERE credit_request_id=$1::uuid`, drawdownID)
			_, _ = pool.Exec(ctx, `DELETE FROM app.credit_requests WHERE id=$1::uuid`, drawdownID)
			_, _ = pool.Exec(ctx, `DELETE FROM ledger.transactions WHERE reference_id=$1`, obligationID)
		}
		if lineID != "" {
			_, _ = pool.Exec(ctx, `DELETE FROM app.trade_lines WHERE id=$1::uuid`, lineID)
		}
		_, _ = pool.Exec(ctx, `DELETE FROM app.payment_mandates WHERE id=$1::uuid`, mandateID)
		_, _ = pool.Exec(ctx, `DELETE FROM app.businesses WHERE id=$1::uuid`, businessID)
		_, _ = pool.Exec(ctx, `DELETE FROM app.organizations WHERE id=$1::uuid`, organizationID)
		_, _ = pool.Exec(ctx, `DELETE FROM app.users WHERE id=$1::uuid`, userID)
	}()
	cfg := config.Config{Environment: "development", Currency: "NGN", MoneyUnit: "kobo", CollectionProvider: "mock", TokenHashKey: "atomic-drawdown-test", Timezone: "Africa/Lagos"}
	runtime := NewRuntimeWithDB(cfg, database)
	line, err := runtime.TradeLines.CreateLine(tradelines.CreateLineInput{SupplierOrganizationID: organizationID, BuyerUserID: userID, BuyerBusinessID: businessID, ApprovedLimitKobo: 500000, Cadence: "monthly", StartAt: time.Now().Add(-time.Hour), EndAt: time.Now().AddDate(1, 0, 0), MandateID: mandateID, MandateActive: true, MandateVerified: true})
	if err != nil {
		t.Fatal(err)
	}
	lineID = line.ID
	drawdown, _, _, err := runtime.TradeLines.ReserveDrawdown(tradelines.CreateDrawdownInput{LineID: line.ID, PrincipalKobo: 125000, GoodsDescription: "atomic inventory", DueDate: "2026-09-30", CollectionAt: time.Date(2026, 10, 1, 9, 0, 0, 0, time.UTC), IdempotencyKey: "atomic-" + line.ID})
	if err != nil {
		t.Fatal(err)
	}
	drawdownID = drawdown.ID
	if _, _, err := runtime.TradeLines.ConfirmDrawdown(drawdown.ID, userID, drawdown.AgreementHash); err != nil {
		t.Fatal(err)
	}
	if _, _, err := runtime.TradeLines.ReleaseDrawdown(tradelines.ReleaseInput{DrawdownID: drawdown.ID, SupplierOrganizationID: organizationID, ActorID: userID, DeliveryMethod: "courier", EvidenceReference: "ATOMIC-TRACK"}); err != nil {
		t.Fatal(err)
	}
	tradeStore := runtime.TradeLines.(*tradelines.PostgresStore)
	creditStore := runtime.Credit.(*credit.PostgresStore)
	scheduleStore := schedules.NewPostgresStore(pool)
	tradeStore.SetTransactionalActivationHandler(func(ctx context.Context, tx pgx.Tx, input tradelines.ActivationInput) (string, func(), error) {
		view, _, _, err := creditStore.ActivateTradeLineDrawdownTx(ctx, tx, credit.TradeLineActivationInput{DrawdownID: input.Drawdown.ID, TradeLineID: input.Line.ID, SupplierOrganizationID: input.Line.SupplierOrganizationID, BuyerUserID: input.Line.BuyerUserID, BuyerBusinessID: input.Line.BuyerBusinessID, MandateID: input.Line.MandateID, PrincipalKobo: input.Drawdown.PrincipalKobo, GoodsDescription: input.Drawdown.GoodsDescription, DueDate: input.Drawdown.DueDate, GraceHours: input.Drawdown.GraceHours, CollectionAt: input.Drawdown.CollectionAt, TermsVersion: input.Drawdown.TermsVersion, DrawdownAgreementHash: input.Drawdown.AgreementHash, BuyerConfirmedAt: input.Drawdown.BuyerConfirmedAt, ReleaseActorID: input.Drawdown.ReleaseActorID, DeliveryMethod: input.Drawdown.DeliveryMethod, ReleasedAt: input.Drawdown.ReleasedAt, ReceiptActorID: input.Drawdown.ReceiptActorID, ReceiptAt: input.Drawdown.ReceiptAt})
		if err != nil {
			return "", nil, err
		}
		if _, _, err := scheduleStore.CreateTx(ctx, tx, schedules.CreateInput{ObligationID: view.Obligation.ID, PrincipalKobo: view.Obligation.PrincipalKobo, ScheduleType: schedules.TypeEqual, Count: 1, StartDate: time.Date(2026, 9, 30, 0, 0, 0, 0, time.UTC), DueHour: 9, Timezone: "Africa/Lagos", GraceHours: 24, Cadence: schedules.CadenceCustom}); err != nil {
			return "", nil, err
		}
		return "", nil, errors.New("forced rollback after every financial write")
	})
	if _, _, err := runtime.TradeLines.RecordDrawdownReceipt(tradelines.ReceiptInput{DrawdownID: drawdown.ID, BuyerUserID: userID, State: "no_issue"}); err == nil {
		t.Fatal("expected forced atomic rollback")
	}
	var creditCount, ledgerCount, scheduleCount int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM app.credit_requests WHERE id=$1::uuid`, drawdown.ID).Scan(&creditCount); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM ledger.transactions WHERE idempotency_key=$1`, "trade-line-drawdown:"+drawdown.ID+":activation").Scan(&ledgerCount); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM app.repayment_schedules s JOIN app.obligations o ON o.id=s.obligation_id WHERE o.credit_request_id=$1::uuid`, drawdown.ID).Scan(&scheduleCount); err != nil {
		t.Fatal(err)
	}
	if creditCount != 0 || ledgerCount != 0 || scheduleCount != 0 {
		t.Fatalf("rollback leaked financial state: credit=%d ledger=%d schedule=%d", creditCount, ledgerCount, scheduleCount)
	}
	var state string
	if err := pool.QueryRow(ctx, `SELECT state FROM app.drawdowns WHERE id=$1::uuid`, drawdown.ID).Scan(&state); err != nil || state != tradelines.DrawdownGoodsReleased {
		t.Fatalf("drawdown advanced during rollback: state=%s err=%v", state, err)
	}

	committingRuntime := NewRuntimeWithDB(cfg, database)
	activated, _, err := committingRuntime.TradeLines.RecordDrawdownReceipt(tradelines.ReceiptInput{DrawdownID: drawdown.ID, BuyerUserID: userID, State: "no_issue"})
	if err != nil {
		t.Fatal(err)
	}
	obligationID = activated.ObligationID
	if obligationID == "" {
		t.Fatal("committed activation did not create an obligation")
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM app.repayment_schedules WHERE obligation_id=$1::uuid`, obligationID).Scan(&scheduleCount); err != nil || scheduleCount != 1 {
		t.Fatalf("schedule count=%d err=%v", scheduleCount, err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM ledger.postings p JOIN ledger.transactions t ON t.id=p.transaction_id WHERE t.reference_id=$1`, obligationID).Scan(&ledgerCount); err != nil || ledgerCount != 4 {
		t.Fatalf("activation postings=%d err=%v", ledgerCount, err)
	}
	var obligationCount, activationEventCount int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM app.obligations WHERE credit_request_id=$1::uuid`, drawdown.ID).Scan(&obligationCount); err != nil || obligationCount != 1 {
		t.Fatalf("resulting obligations=%d err=%v", obligationCount, err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM app.outbox_events WHERE idempotency_key IN($1,$2)`, "trade-line-drawdown:"+drawdown.ID+":TradeLineDrawdownActivated", "ledger:trade-line-drawdown:"+drawdown.ID+":activation").Scan(&activationEventCount); err != nil || activationEventCount != 2 {
		t.Fatalf("atomic activation outbox events=%d err=%v", activationEventCount, err)
	}
	current, found := committingRuntime.TradeLines.Get(lineID)
	if !found {
		t.Fatal("trade line not found")
	}
	if _, err = committingRuntime.TradeLines.ReduceLimit(lineID, 400000, current.Version); err != nil {
		t.Fatal(err)
	}
	current, found = committingRuntime.TradeLines.Get(lineID)
	if !found || current.ApprovedLimitKobo != 400000 {
		t.Fatalf("reduced limit did not persist: %+v found=%v", current, found)
	}

}
