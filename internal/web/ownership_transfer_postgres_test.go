package web

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestOwnershipTransferRechecksAuthorityAndPreservesEffectiveSuccessor(t *testing.T) {
	if os.Getenv("KREDIT_INTEGRATION") != "1" || os.Getenv("DATABASE_URL") == "" {
		t.Skip("isolated integration database required")
	}
	pool, err := pgxpool.New(t.Context(), os.Getenv("DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	tx, err := pool.Begin(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	actor, target, other := uuid.NewString(), uuid.NewString(), uuid.NewString()
	for _, id := range []string{actor, target, other} {
		if _, err := tx.Exec(t.Context(), `INSERT INTO app.users(id,normalized_email,status) VALUES($1::uuid,$2,'active')`, id, id+"@example.test"); err != nil {
			t.Fatal(err)
		}
	}
	// Keep authority fixtures inside this rollback-only transaction; this test
	// must also work on a freshly migrated database with no seeded owner.
	if _, err := tx.Exec(t.Context(), `INSERT INTO app.platform_role_assignments(user_id,role,granted_by,reason) VALUES($1::uuid,'platform_owner',$1::uuid,'Synthetic transfer actor')`, actor); err != nil {
		t.Fatal(err)
	}
	// A pre-existing expiring role must not make the successor's ownership
	// disappear later or leave the promised admin access ineffective.
	for _, role := range []string{"platform_owner", "platform_admin"} {
		if _, err := tx.Exec(t.Context(), `INSERT INTO app.platform_role_assignments(user_id,role,granted_by,reason,expires_at) VALUES($1::uuid,$2,$3::uuid,'Synthetic transfer fixture',now()+interval '1 hour')`, target, role, actor); err != nil {
			t.Fatal(err)
		}
	}
	if err := transferOwnershipTx(t.Context(), tx, actor, target, "Synthetic ownership transfer"); err != nil {
		t.Fatal(err)
	}
	var effective int
	if err := tx.QueryRow(t.Context(), `SELECT count(*) FROM app.platform_role_assignments WHERE user_id=$1::uuid AND role IN('platform_owner','platform_admin') AND revoked_at IS NULL AND expires_at IS NULL`, target).Scan(&effective); err != nil || effective != 2 {
		t.Fatalf("successor lacks permanent effective access: %d %v", effective, err)
	}
	// Models a second request that authenticated before the first committed,
	// then reached its transaction after ownership had already moved.
	if err := transferOwnershipTx(t.Context(), tx, actor, other, "Stale ownership transfer attempt"); !errors.Is(err, errOwnershipChanged) {
		t.Fatalf("former owner could transfer again: %v", err)
	}
	if err := tx.QueryRow(t.Context(), `SELECT count(*) FROM app.platform_role_assignments WHERE user_id=$1::uuid`, other).Scan(&effective); err != nil || effective != 0 {
		t.Fatalf("stale caller granted access: %d %v", effective, err)
	}
}
