package collections

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"kredit/internal/operations"
	"kredit/internal/outbox"
	"kredit/internal/payments"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func workflowActors(t *testing.T, f collectionFixture) (string, string, *operations.PostgresStore) {
	t.Helper()
	ctx := context.Background()
	actors := []string{uuid.NewString(), uuid.NewString()}
	for i, id := range actors {
		role := "finance_operator"
		if i == 1 {
			role = "approver"
		}
		if _, err := f.pool.Exec(ctx, `INSERT INTO app.users(id,normalized_email,display_name) VALUES($1::uuid,$2,$3)`, id, id+"@admin-workflow.test", role); err != nil {
			t.Fatal(err)
		}
		if _, err := f.pool.Exec(ctx, `INSERT INTO app.platform_role_assignments(user_id,role,granted_by,reason) VALUES($1::uuid,$2,$1::uuid,'Financial workflow test')`, id, role); err != nil {
			t.Fatal(err)
		}
	}
	// Exercise the application role, not the fixture administrator's RLS bypass.
	cfg, err := pgxpool.ParseConfig(f.pool.Config().ConnString())
	if err != nil {
		t.Fatal(err)
	}
	cfg.ConnConfig.RuntimeParams["role"] = "kredit_app"
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	return actors[0], actors[1], operations.NewPostgresStore(pool, outbox.NewStore(pool), nil)
}
func proposal(f collectionFixture, kind string) operations.ChangeProposal {
	return operations.ChangeProposal{ID: uuid.NewString(), ObligationID: f.id, Kind: kind, Reason: "Documented financial correction request", Values: operations.ChangeValues{Amount: 2000000}, ExpiresAt: time.Now().Add(7 * 24 * time.Hour).Truncate(time.Microsecond)}
}
func TestAdminWorkflowIndependentCorrectionAndConcurrentApproval(t *testing.T) {
	f := financialFixture(t)
	maker, checker, s := workflowActors(t, f)
	ctx := context.Background()
	in := proposal(f, "write_off")
	if err := s.ProposeChange(ctx, checker, in); err == nil {
		t.Fatal("approver created their own proposal")
	}
	if err := s.ProposeChange(ctx, maker, in); err != nil {
		t.Fatal(err)
	}
	if err := s.ProposeChange(ctx, maker, in); err != nil {
		t.Fatal("exact replay", err)
	}
	bad := in
	bad.Values.Amount++
	if err := s.ProposeChange(ctx, maker, bad); err == nil {
		t.Fatal("conflicting id accepted")
	}
	if err := s.DecideChange(ctx, in.ID, maker, "approve", "I approve my own change", false); err == nil {
		t.Fatal("maker self-approved")
	}
	if err := s.DecideChange(ctx, in.ID, f.user, "approve", "Unauthorized buyer approval", false); err == nil {
		t.Fatal("buyer approved correction")
	}
	var wg sync.WaitGroup
	success := make(chan bool, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			success <- s.DecideChange(ctx, in.ID, checker, "approve", "Independently verified supporting evidence", false) == nil
		}()
	}
	wg.Wait()
	close(success)
	n := 0
	for ok := range success {
		if ok {
			n++
		}
	}
	if n != 1 {
		t.Fatalf("approved %d times", n)
	}
	var outstanding int64
	var journals, actions, events int
	if err := f.pool.QueryRow(ctx, `SELECT outstanding_kobo,(SELECT count(*) FROM ledger.transactions WHERE idempotency_key=$2),(SELECT count(*) FROM app.operation_actions WHERE id=$3::uuid),(SELECT count(*) FROM app.admin_change_events WHERE change_id=$3::uuid) FROM app.obligations WHERE id=$1::uuid`, f.id, "operation:"+in.ID, in.ID).Scan(&outstanding, &journals, &actions, &events); err != nil {
		t.Fatal(err)
	}
	if outstanding != 48000000 || journals != 1 || actions != 1 || events != 2 {
		t.Fatalf("incorrect atomic result: %d %d %d %d", outstanding, journals, actions, events)
	}
	var schedule int64
	if err := f.pool.QueryRow(ctx, `SELECT sum(i.principal_due_kobo-i.allocated_kobo) FROM app.schedule_items i JOIN app.repayment_schedules s ON s.id=i.schedule_id WHERE s.obligation_id=$1::uuid AND i.state<>'CANCELLED'`, f.id).Scan(&schedule); err != nil || schedule != outstanding {
		t.Fatalf("schedule mismatch %d %v", schedule, err)
	}
	if _, err := f.pool.Exec(ctx, `UPDATE app.admin_change_requests SET reason='tampered intent' WHERE id=$1::uuid`, in.ID); err == nil {
		t.Fatal("approved intent was mutable")
	}
}
func TestAdminWorkflowStaleAndReservedCorrections(t *testing.T) {
	for _, scenario := range []string{"payment", "reserved", "revoked"} {
		t.Run(scenario, func(t *testing.T) {
			f := financialFixture(t)
			maker, checker, s := workflowActors(t, f)
			ctx := context.Background()
			in := proposal(f, "write_off")
			if err := s.ProposeChange(ctx, maker, in); err != nil {
				t.Fatal(err)
			}
			switch scenario {
			case "payment":
				if _, _, err := f.payments.Record(payments.RecordInput{ObligationID: f.id, AmountKobo: 1000, SourceType: payments.SourceSupplierTransfer, RecordedBy: f.user, IdempotencyKey: "stale-approval:" + f.id}); err != nil {
					t.Fatal(err)
				}
			case "reserved":
				if _, err := f.engine(&observedProvider{MockProvider: NewMockProvider("test")}).Start(ctx, f.id, "pending:"+f.id, time.Now()); err != nil {
					t.Fatal(err)
				}
			case "revoked":
				if _, err := f.pool.Exec(ctx, `UPDATE app.platform_role_assignments SET revoked_at=now() WHERE user_id=$1::uuid`, maker); err != nil {
					t.Fatal(err)
				}
			}
			if err := s.DecideChange(ctx, in.ID, checker, "approve", "Independent correction review", false); err == nil {
				t.Fatal("unsafe proposal applied")
			}
		})
	}
}
func TestAdminWorkflowScheduleNeedsExactBuyerConsent(t *testing.T) {
	f := financialFixture(t)
	maker, checker, s := workflowActors(t, f)
	ctx := context.Background()
	before, err := s.ChangeContext(ctx, f.id)
	if err != nil {
		t.Fatal(err)
	}
	var items []struct {
		ID  string    `json:"id"`
		Due time.Time `json:"due_at"`
	}
	if err = json.Unmarshal(before.Items, &items); err != nil {
		t.Fatal(err)
	}
	in := proposal(f, "schedule_amendment")
	in.Values = operations.ChangeValues{Dates: []operations.DateChange{{ItemID: items[0].ID, DueAt: time.Now().Add(10 * 24 * time.Hour).Truncate(time.Microsecond)}}}
	if err = s.ProposeChange(ctx, maker, in); err != nil {
		t.Fatal(err)
	}
	if err = s.DecideChange(ctx, in.ID, checker, "approve", "Independent date-change review", false); err != nil {
		t.Fatal(err)
	}
	var date time.Time
	if err = f.pool.QueryRow(ctx, `SELECT due_at FROM app.schedule_items WHERE id=$1::uuid`, items[0].ID).Scan(&date); err != nil || !date.Equal(items[0].Due) {
		t.Fatal("dates changed before buyer accepted", err)
	}
	if err = s.DecideChange(ctx, in.ID, checker, "accept", "Pretend to be the buyer", true); err == nil {
		t.Fatal("different user accepted")
	}
	if err = s.DecideChange(ctx, in.ID, f.user, "accept", "I accept these exact repayment dates", true); err != nil {
		t.Fatal(err)
	}
	if err = f.pool.QueryRow(ctx, `SELECT due_at FROM app.schedule_items WHERE id=$1::uuid`, items[0].ID).Scan(&date); err != nil || !date.Equal(in.Values.Dates[0].DueAt) {
		t.Fatal("buyer acceptance did not apply dates", err)
	}
	var amount int64
	var state, buyer string
	if err = f.pool.QueryRow(ctx, `SELECT o.outstanding_kobo,c.state,c.buyer_decided_by::text FROM app.obligations o JOIN app.admin_change_requests c ON c.obligation_id=o.id WHERE c.id=$1::uuid`, in.ID).Scan(&amount, &state, &buyer); err != nil || amount != 50000000 || state != "applied" || buyer != f.user {
		t.Fatal("incorrect consent evidence", amount, state, buyer, err)
	}
	var n int
	if err = f.pool.QueryRow(ctx, `SELECT count(*) FROM app.outbox_events WHERE idempotency_key LIKE $1`, "schedule-amendment:"+in.ID+":%").Scan(&n); err != nil || n != 2 {
		t.Fatal("missing durable notices", n, err)
	}
}

func TestAdminWorkflowBuyerRejectsAndStaleAcceptanceIsBlocked(t *testing.T) {
	for _, scenario := range []string{"reject", "paid", "revoked_approver"} {
		t.Run(scenario, func(t *testing.T) {
			f := financialFixture(t)
			maker, checker, s := workflowActors(t, f)
			ctx := context.Background()
			before, err := s.ChangeContext(ctx, f.id)
			if err != nil {
				t.Fatal(err)
			}
			var items []struct {
				ID  string    `json:"id"`
				Due time.Time `json:"due_at"`
			}
			_ = json.Unmarshal(before.Items, &items)
			in := proposal(f, "schedule_amendment")
			in.Values = operations.ChangeValues{Dates: []operations.DateChange{{ItemID: items[0].ID, DueAt: time.Now().Add(10 * 24 * time.Hour).Truncate(time.Microsecond)}}}
			if err = s.ProposeChange(ctx, maker, in); err != nil {
				t.Fatal(err)
			}
			if err = s.DecideChange(ctx, in.ID, checker, "approve", "Independent amendment review", false); err != nil {
				t.Fatal(err)
			}
			switch scenario {
			case "paid":
				if _, _, err = f.payments.Record(payments.RecordInput{ObligationID: f.id, AmountKobo: 1000, SourceType: payments.SourceSupplierTransfer, RecordedBy: f.user, IdempotencyKey: "amendment-payment:" + f.id}); err != nil {
					t.Fatal(err)
				}
			case "revoked_approver":
				if _, err = f.pool.Exec(ctx, `UPDATE app.platform_role_assignments SET revoked_at=now() WHERE user_id=$1::uuid`, checker); err != nil {
					t.Fatal(err)
				}
			}
			if scenario == "reject" {
				if err = s.DecideChange(ctx, in.ID, f.user, "reject", "I reject these repayment dates", true); err != nil {
					t.Fatal(err)
				}
			} else if err = s.DecideChange(ctx, in.ID, f.user, "accept", "I accept these repayment dates", true); err == nil {
				t.Fatal("stale or unauthorized approval applied")
			}
			var date time.Time
			if err = f.pool.QueryRow(ctx, `SELECT due_at FROM app.schedule_items WHERE id=$1::uuid`, items[0].ID).Scan(&date); err != nil || !date.Equal(items[0].Due) {
				t.Fatal("nonaccepted proposal changed date", err)
			}
		})
	}
}

func TestAdminWorkflowClosedObligationCanCancelButNotApply(t *testing.T) {
	f := financialFixture(t)
	maker, checker, s := workflowActors(t, f)
	ctx := context.Background()
	in := proposal(f, "write_off")
	if err := s.ProposeChange(ctx, maker, in); err != nil {
		t.Fatal(err)
	}
	if _, err := f.pool.Exec(ctx, `UPDATE app.obligations SET lifecycle_status='CLOSED' WHERE id=$1::uuid`, f.id); err != nil {
		t.Fatal(err)
	}
	if err := s.DecideChange(ctx, in.ID, checker, "approve", "Independent review of closed debt", false); err == nil {
		t.Fatal("closed debt changed")
	}
	if err := s.DecideChange(ctx, in.ID, maker, "cancel", "Withdraw because the obligation closed", false); err != nil {
		t.Fatal("closed obligation left an unresolvable proposal", err)
	}
}
