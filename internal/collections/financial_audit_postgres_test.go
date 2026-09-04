package collections

import (
	"context"
	"sync"
	"testing"
	"time"

	"kredit/internal/disputes"
	"kredit/internal/operations"
	"kredit/internal/outbox"
	"kredit/internal/paymentclaims"
	"kredit/internal/payments"
)

func TestAuditFullPaymentReplayAndReversal(t *testing.T) {
	f := financialFixture(t)
	in := payments.RecordInput{ObligationID: f.id, AmountKobo: 50000000, Currency: "USD", SourceType: payments.SourceSupplierTransfer, RecordedBy: f.user, IdempotencyKey: "audit-payment:" + f.id}
	if _, _, err := f.payments.Record(in); err == nil {
		t.Fatal("foreign currency reduced NGN debt")
	}
	in.Currency = "NGN"
	var wg sync.WaitGroup
	ids := make(chan string, 6)
	for i := 0; i < 6; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			p, _, err := f.payments.Record(in)
			if err != nil {
				t.Error(err)
				return
			}
			ids <- p.ID
		}()
	}
	wg.Wait()
	close(ids)
	var id string
	for got := range ids {
		if id != "" && id != got {
			t.Fatal("duplicate payment")
		}
		id = got
	}
	if id == "" {
		t.Fatal("no payment recorded")
	}
	if _, err := f.payments.Reverse(id, f.user, "confirmed bank return"); err != nil {
		t.Fatal(err)
	}
	if outstanding, err := f.payments.Rebuild(f.id); err != nil || outstanding != 50000000 {
		t.Fatalf("reversal: %d %v", outstanding, err)
	}
}

func TestAuditForgivenessPreservedAndOperationKeysScoped(t *testing.T) {
	f, g := financialFixture(t), financialFixture(t)
	ops := operations.NewPostgresStore(f.pool, outbox.NewStore(f.pool), nil)
	key := "audit-shared-key:" + f.id
	a, err := ops.WriteOffWithKey(f.user, f.organization, f.id, 1000, "approved small write-off", "", key)
	if err != nil {
		t.Fatal(err)
	}
	b, err := ops.WriteOffWithKey(g.user, g.organization, g.id, 1000, "approved small write-off", "", key)
	if err != nil || a.LedgerTransactionID == b.LedgerTransactionID {
		t.Fatalf("different debts shared a journal: %+v %v", b, err)
	}
	if _, err := ops.WriteOffWithKey(f.user, f.organization, f.id, 2000, "approved small write-off", "", key); err == nil {
		t.Fatal("changed operation accepted under same key")
	}
	ds := disputes.NewPostgresStore(f.pool, nil)
	d, err := ds.Open(disputes.OpenInput{ObligationID: f.id, OpenedBy: f.user, DisputedAmountKobo: 2000, Reason: "damaged goods"})
	if err != nil {
		t.Fatal(err)
	}
	in := disputes.DecideInput{DisputeID: d.ID, ReviewerID: f.user, Outcome: "partial", AdjustmentKobo: 1000, RemainingDisputedKobo: 2000, Reason: "documented return"}
	if _, _, err := ds.Decide(in); err == nil {
		t.Fatal("adjustment retained disputed principal")
	}
	in.RemainingDisputedKobo = 1000
	if _, _, err := ds.Decide(in); err != nil {
		t.Fatal(err)
	}
	in.RemainingDisputedKobo = 0
	if _, _, err := ds.Decide(in); err != nil {
		t.Fatal(err)
	}
	if outstanding, err := f.payments.Rebuild(f.id); err != nil || outstanding != 49997000 {
		t.Fatalf("forgiven debt resurrected: %d %v", outstanding, err)
	}
}

func TestAuditWriteOffCannotConsumePendingDebit(t *testing.T) {
	f := financialFixture(t)
	p := &observedProvider{MockProvider: NewMockProvider("secret")}
	if _, err := f.engine(p).Start(context.Background(), f.id, "held:"+f.id, time.Now()); err != nil {
		t.Fatal(err)
	}
	ops := operations.NewPostgresStore(f.pool, nil, nil)
	if _, err := ops.WriteOffWithKey(f.user, f.organization, f.id, 1000, "small adjustment", "", "blocked:"+f.id); err == nil {
		t.Fatal("write-off consumed pending bank debit")
	}
}

func TestPostgresReservationRejectsNewHoldAfterStaleSnapshot(t *testing.T) {
	for _, kind := range []string{"dispute", "claim", "schedule"} {
		t.Run(kind, func(t *testing.T) {
			f := financialFixture(t)
			ctx := context.Background()
			var err error
			switch kind {
			case "dispute":
				_, err = f.pool.Exec(ctx, `INSERT INTO app.disputes(obligation_id,supplier_organization_id,buyer_user_id,opened_by,total_disputed_kobo,remaining_disputed_kobo,reason,state,collection_effect) VALUES($1::uuid,$2::uuid,$3::uuid,$3::uuid,100,100,'Late dispute','OPEN','CONTESTED_ONLY')`, f.id, f.organization, f.user)
			case "claim":
				_, err = f.pool.Exec(ctx, `INSERT INTO app.payment_claims(obligation_id,supplier_organization_id,buyer_user_id,amount_kobo,currency,paid_at,transfer_reference,hold_expires_at,idempotency_key) VALUES($1::uuid,$2::uuid,$3::uuid,100,'NGN',now(),'late claim',now()+interval '1 hour',$1)`, f.id, f.organization, f.user)
			case "schedule":
				_, err = f.pool.Exec(ctx, `UPDATE app.schedule_items SET collection_at=now()+interval '1 day' WHERE schedule_id IN (SELECT id FROM app.repayment_schedules WHERE obligation_id=$1::uuid)`, f.id)
			}
			if err != nil {
				t.Fatal(err)
			}
			p := &observedProvider{MockProvider: NewMockProvider("secret")}
			if _, err := f.engine(p).Start(ctx, f.id, "stale-eligibility", time.Now()); err == nil {
				t.Fatal("stale eligibility accepted")
			}
			if p.count.Load() != 0 {
				t.Fatal("provider contacted despite current collection hold")
			}
		})
	}
}

func TestClaimConfirmationCommitsPaymentAndDecisionTogether(t *testing.T) {
	for _, scenario := range []string{"empty_reason", "rejected", "valid_retry"} {
		t.Run(scenario, func(t *testing.T) {
			f := financialFixture(t)
			ctx := context.Background()
			store := paymentclaims.NewPostgresStore(f.pool)
			in := paymentclaims.CreateInput{ObligationID: f.id, BuyerUserID: f.user, AmountKobo: 2500, TransferReference: "transfer", IdempotencyKey: "claim:" + f.id}
			claim, err := store.Create(ctx, in)
			if err != nil {
				t.Fatal(err)
			}
			changed := in
			changed.AmountKobo++
			if _, err = store.Create(ctx, changed); err == nil {
				t.Fatal("changed payment claim accepted")
			}
			if scenario == "rejected" {
				if _, err = store.Decide(ctx, claim.ID, f.user, paymentclaims.Rejected, "No matching transfer", ""); err != nil {
					t.Fatal(err)
				}
			}
			reason := "Bank transfer checked"
			if scenario == "empty_reason" {
				reason = ""
			}
			result, err := store.Confirm(ctx, claim.ID, f.user, reason, f.payments)
			var expected int64 = 50000000
			var count int
			if scenario == "valid_retry" {
				if err != nil || result.State != paymentclaims.Confirmed || result.PaymentID == "" {
					t.Fatalf("claim=%+v %v", result, err)
				}
				expected -= 2500
				again, err := store.Confirm(ctx, claim.ID, f.user, reason, f.payments)
				if err != nil || again.PaymentID != result.PaymentID {
					t.Fatalf("retry=%+v %v", again, err)
				}
			} else if err == nil {
				t.Fatal("invalid confirmation accepted")
			}
			var outstanding int64
			if err = f.pool.QueryRow(ctx, `SELECT outstanding_kobo FROM app.obligations WHERE id=$1::uuid`, f.id).Scan(&outstanding); err != nil || outstanding != expected {
				t.Fatalf("balance=%d want=%d %v", outstanding, expected, err)
			}
			if err = f.pool.QueryRow(ctx, `SELECT count(*) FROM app.payments WHERE obligation_id=$1::uuid`, f.id).Scan(&count); err != nil {
				t.Fatal(err)
			}
			if scenario == "valid_retry" && count != 1 || scenario != "valid_retry" && count != 0 {
				t.Fatalf("payments=%d", count)
			}
		})
	}
}
