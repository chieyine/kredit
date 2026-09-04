package platformops

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestEveryCommandRequiresSafetyEnvelope(t *testing.T) {
	for commandType := range commandTypes {
		input := CommandInput{Type: commandType, TargetType: "collection", TargetID: "target", ExpectedVersion: 1, Reason: "Documented operator reason", IdempotencyKey: "safe-key"}
		if commandType == "place_risk_hold" {
			input.TargetType = "buyer"
			input.Scope = "collection"
			input.ExpiresAt = time.Now().Add(time.Hour)
		}
		for name, mutate := range map[string]func(*CommandInput){
			"reason":      func(v *CommandInput) { v.Reason = "short" },
			"version":     func(v *CommandInput) { v.ExpectedVersion = 0 },
			"idempotency": func(v *CommandInput) { v.IdempotencyKey = "" },
		} {
			candidate := input
			mutate(&candidate)
			if err := validateCommand(candidate, true); err == nil {
				t.Fatalf("%s accepted without %s", commandType, name)
			}
		}
	}
}

func TestControlledSuspendRestoreHoldAndIdempotency(t *testing.T) {
	if os.Getenv("KREDIT_INTEGRATION") != "1" || os.Getenv("DATABASE_URL") == "" {
		t.Skip("integration database required")
	}
	ctx := context.Background()
	poolConfig, err := pgxpool.ParseConfig(os.Getenv("DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	poolConfig.MaxConns = 1
	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	suffix := fmt.Sprint(time.Now().UnixNano())
	var actor, target string
	if err = pool.QueryRow(ctx, `INSERT INTO app.users(normalized_email) VALUES($1) RETURNING id::text`, "ops-actor-"+suffix+"@example.test").Scan(&actor); err != nil {
		t.Fatal(err)
	}
	if err = pool.QueryRow(ctx, `INSERT INTO app.users(normalized_email) VALUES($1) RETURNING id::text`, "ops-target-"+suffix+"@example.test").Scan(&target); err != nil {
		t.Fatal(err)
	}
	store := NewStore(pool)
	base := CommandInput{Type: "suspend_user", TargetType: "user", TargetID: target, Reason: "Confirmed account compromise", ExpectedVersion: 1, IdempotencyKey: "suspend-" + suffix, CorrelationID: "correlation-" + suffix}
	preview, err := store.PreviewCommand(ctx, base)
	if err != nil || preview.CurrentVersion != 1 {
		t.Fatalf("preview=%+v err=%v", preview, err)
	}
	bad := base
	bad.Reason = "short"
	bad.IdempotencyKey = "bad-" + suffix
	if _, err = store.ExecuteCommand(ctx, actor, bad); err == nil {
		t.Fatal("short reason accepted")
	}
	applied, err := store.ExecuteCommand(ctx, actor, base)
	if err != nil {
		t.Fatal(err)
	}
	replayed, err := store.ExecuteCommand(ctx, actor, base)
	if err != nil || replayed.ID != applied.ID {
		t.Fatalf("idempotency failed: %+v %v", replayed, err)
	}
	changed := base
	changed.TargetID = actor
	if _, err = store.ExecuteCommand(ctx, actor, changed); err == nil {
		t.Fatal("same key authorized a different target")
	}
	var status string
	if err = pool.QueryRow(ctx, `SELECT status FROM app.users WHERE id=$1::uuid`, target).Scan(&status); err != nil || status != "suspended" {
		t.Fatalf("status=%s err=%v", status, err)
	}
	stale := base
	stale.IdempotencyKey = "stale-" + suffix
	if _, err = store.ExecuteCommand(ctx, actor, stale); err == nil {
		t.Fatal("stale version accepted")
	}
	var suspension string
	if err = pool.QueryRow(ctx, `SELECT id::text FROM app.platform_suspensions WHERE target_id=$1::uuid AND lifted_at IS NULL`, target).Scan(&suspension); err != nil {
		t.Fatal(err)
	}
	restore := CommandInput{Type: "restore_user", TargetType: "user", TargetID: suspension, Reason: "Investigation safely completed", ExpectedVersion: 1, IdempotencyKey: "restore-" + suffix, CorrelationID: "correlation-" + suffix}
	if _, err = store.ExecuteCommand(ctx, actor, restore); err != nil {
		t.Fatal(err)
	}
	if err = pool.QueryRow(ctx, `SELECT status FROM app.users WHERE id=$1::uuid`, target).Scan(&status); err != nil || status != "active" {
		t.Fatalf("restored status=%s err=%v", status, err)
	}
	hold := CommandInput{Type: "place_risk_hold", TargetType: "buyer", TargetID: target, Scope: "collection", ExpiresAt: time.Now().Add(time.Hour), Reason: "Provider anomaly under review", ExpectedVersion: 1, IdempotencyKey: "hold-" + suffix, CorrelationID: "correlation-" + suffix}
	if _, err = store.ExecuteCommand(ctx, actor, hold); err != nil {
		t.Fatal(err)
	}
	blocked, err := store.ActiveHold(ctx, "buyer", target, "collection")
	if err != nil || !blocked {
		t.Fatalf("blocked=%v err=%v", blocked, err)
	}
	if _, err = pool.Exec(ctx, `UPDATE app.operations_commands SET reason='tampered record' WHERE id=$1::uuid`, applied.ID); err == nil {
		t.Fatal("immutable command was mutable")
	}
	diagnostics, err := store.Diagnostics(ctx, 60, "correlation-sensitive-1234")
	if err != nil {
		t.Fatal(err)
	}
	if diagnostics.CorrelationID == "correlation-sensitive-1234" || diagnostics.CorrelationID == "" {
		t.Fatal("correlation id was not redacted")
	}
}
