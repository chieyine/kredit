package platformops

import (
	"context"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestExternalIntentIsDurableAndClaimedOnce(t *testing.T) {
	if os.Getenv("KREDIT_INTEGRATION") != "1" || os.Getenv("DATABASE_URL") == "" {
		t.Skip("isolated seeded integration database required")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	// Immutable command evidence is intentionally retained only in the isolated
	// audit database. This fixture does not invoke a provider or change balances.
	key := fmt.Sprintf("external-intent-%d", time.Now().UnixNano())
	var actor, obligation, reservation, attempt string
	if err = pool.QueryRow(ctx, `INSERT INTO app.users(normalized_email) VALUES($1) RETURNING id::text`, key+"@example.test").Scan(&actor); err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, `INSERT INTO app.platform_role_assignments(user_id,role,granted_by,reason) VALUES($1::uuid,'platform_admin',$1::uuid,'Isolated command fixture')`, actor); err != nil {
		t.Fatal(err)
	}
	if err = pool.QueryRow(ctx, `SELECT id::text FROM app.obligations ORDER BY activated_at,id LIMIT 1`).Scan(&obligation); err != nil {
		t.Fatal("seed the isolated audit database before running this test: ", err)
	}
	if err = pool.QueryRow(ctx, `INSERT INTO app.collection_reservations(obligation_id,outstanding_snapshot_version,reserved_amount_kobo,state,expires_at,idempotency_key) VALUES($1::uuid,1,1,'RELEASED',now(),$2) RETURNING id::text`, obligation, key).Scan(&reservation); err != nil {
		t.Fatal(err)
	}
	if err = pool.QueryRow(ctx, `INSERT INTO app.collection_attempts(reservation_id,obligation_id,provider,external_reference,requested_amount_kobo,state) VALUES($1::uuid,$2::uuid,'isolated-fixture',$3,1,'FAILED') RETURNING id::text`, reservation, obligation, key).Scan(&attempt); err != nil {
		t.Fatal(err)
	}
	store := NewStore(pool)
	in := CommandInput{Type: "retry_collection", TargetType: "collection", TargetID: attempt, Reason: "Isolated retry safety fixture", ExpectedVersion: 1, IdempotencyKey: key}
	var claimed atomic.Int32
	var wg sync.WaitGroup
	for range 2 {
		wg.Go(func() {
			_, won, err := store.BeginExternalCommand(ctx, actor, in)
			if err != nil {
				t.Error(err)
			}
			if won {
				claimed.Add(1)
			}
		})
	}
	wg.Wait()
	if claimed.Load() != 1 {
		t.Fatalf("provider action claimed %d times", claimed.Load())
	}
	saved, found, err := NewStore(pool).ReplayCommand(ctx, actor, in)
	if err != nil || !found || saved.State != "PREVIEWED" {
		t.Fatalf("intent did not survive restart: %+v %v", saved, err)
	}
	other := in
	other.IdempotencyKey += "-new"
	if _, won, err := store.BeginExternalCommand(ctx, actor, other); err == nil || won {
		t.Fatal("new key repeated unresolved external action")
	}
	changed := in
	changed.Reason = "Changed operation must be refused"
	if _, won, err := store.BeginExternalCommand(ctx, actor, changed); err == nil || won {
		t.Fatal("changed intent reused old key")
	}
	in.ExternalResult = map[string]any{"id": attempt, "state": "FAILED"}
	completed, err := store.FinishExternalCommand(ctx, actor, in)
	if err != nil || completed.State != "APPLIED" || completed.ID != saved.ID {
		t.Fatalf("result was not attached to original intent: %+v %v", completed, err)
	}
	if _, err = store.FinishExternalCommand(ctx, actor, in); err != nil {
		t.Fatal(err)
	}
	var outcomes int
	if err = pool.QueryRow(ctx, `SELECT count(*) FROM app.operations_command_events WHERE command_id=$1::uuid AND state='APPLIED'`, saved.ID).Scan(&outcomes); err != nil || outcomes != 1 {
		t.Fatalf("duplicate terminal outcomes: %d %v", outcomes, err)
	}
	if _, err = pool.Exec(ctx, `UPDATE app.platform_role_assignments SET revoked_at=now() WHERE user_id=$1::uuid`, actor); err != nil {
		t.Fatal(err)
	}
	if _, won, err := store.BeginExternalCommand(ctx, actor, other); err == nil || won {
		t.Fatal("revoked operator claimed a provider action")
	}
}
