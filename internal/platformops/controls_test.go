package platformops

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"kredit/internal/db"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestEveryCommandRequiresSafetyEnvelope(t *testing.T) {
	for commandType := range commandTypes {
		input := CommandInput{Type: commandType, TargetType: "collection", TargetID: "target", ExpectedVersion: 1, Reason: "Documented operator reason", IdempotencyKey: "safe-key"}
		if target := map[string]string{"retry_document_scan": "document", "retry_job": "job", "suspend_user": "user", "restore_user": "user", "suspend_organization": "organization", "restore_organization": "organization", "lift_risk_hold": "buyer"}[commandType]; target != "" {
			input.TargetType = target
		}
		if commandType == "place_risk_hold" {
			input.TargetType = "buyer"
			input.Scope = "collection"
			input.ExpiresAt = time.Now().Add(time.Hour)
		}
		if err := validateCommand(input, true); err != nil {
			t.Fatalf("valid %s rejected: %v", commandType, err)
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
	if _, err = pool.Exec(ctx, `INSERT INTO app.platform_role_assignments(user_id,role,granted_by,reason) VALUES($1::uuid,'platform_admin',$1::uuid,'Isolated controls fixture')`, actor); err != nil {
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
	saved, found, err := store.ReplayCommand(ctx, actor, base)
	if err != nil || !found || saved.ID != applied.ID || saved.Reason != base.Reason {
		t.Fatalf("saved command unavailable after target changed: found=%v command=%+v err=%v", found, saved, err)
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
	var document string
	if err = pool.QueryRow(ctx, `INSERT INTO app.documents(uploaded_by,purpose,object_key,file_name,content_type,size_bytes,sha256,scan_state,retention_class,upload_completed_at,scan_attempts) VALUES($1::uuid,'evidence',$2,'proof.pdf','application/pdf',4,repeat('a',64),'QUARANTINED','dispute',now(),5) RETURNING id::text`, actor, "isolated-scan-"+suffix).Scan(&document); err != nil {
		t.Fatal(err)
	}
	retryScan := CommandInput{Type: "retry_document_scan", TargetType: "document", TargetID: document, Reason: "Scanner restored after provider outage", ExpectedVersion: 1, IdempotencyKey: "scan-" + suffix}
	runtimeConfig := poolConfig.Copy()
	runtimeConfig.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
		_, err := conn.Exec(ctx, `SET ROLE kredit_app`)
		return err
	}
	runtimePool, err := pgxpool.NewWithConfig(ctx, runtimeConfig)
	if err != nil {
		t.Fatal(err)
	}
	defer runtimePool.Close()
	runtimeStore := NewStore(runtimePool)
	if _, err := runtimeStore.PreviewCommand(db.WithTenantContext(ctx, actor, ""), retryScan); err != nil {
		t.Fatalf("admin could not preview document recovery: %v", err)
	}
	if results, err := runtimeStore.Search(db.WithTenantContext(ctx, actor, ""), document); err != nil || len(results) != 1 || results[0].Type != "document" {
		t.Fatalf("admin document discovery: %+v %v", results, err)
	}
	if results, err := runtimeStore.Search(db.WithTenantContext(ctx, target, ""), document); err != nil || len(results) != 0 {
		t.Fatalf("document metadata crossed account boundary: %+v %v", results, err)
	}
	if _, err = runtimeStore.ExecuteCommand(ctx, actor, retryScan); err != nil {
		t.Fatal(err)
	}
	var scanState string
	var attempts, reviewVersion int
	if err = pool.QueryRow(ctx, `SELECT scan_state,scan_attempts,scan_review_version FROM app.documents WHERE id=$1::uuid`, document).Scan(&scanState, &attempts, &reviewVersion); err != nil || scanState != "PENDING" || attempts != 0 || reviewVersion != 2 {
		t.Fatalf("scan recovery must queue, never approve: %s %d %d %v", scanState, attempts, reviewVersion, err)
	}
	retryScan.IdempotencyKey += "-duplicate"
	if _, err = store.ExecuteCommand(ctx, actor, retryScan); err == nil {
		t.Fatal("stale document recovery version accepted")
	}
	if _, err = pool.Exec(ctx, `UPDATE app.platform_role_assignments SET revoked_at=now() WHERE user_id=$1::uuid`, actor); err != nil {
		t.Fatal(err)
	}
	base.IdempotencyKey += "-revoked"
	base.ExpectedVersion = 3
	if _, err = store.ExecuteCommand(ctx, actor, base); err == nil {
		t.Fatal("revoked operator changed a target")
	}
}

func TestCommandTargetCannotMisdirectAuditOrNotification(t *testing.T) {
	for _, command := range []string{"suspend_user", "restore_user", "suspend_organization", "retry_collection", "cancel_collection", "lift_risk_hold"} {
		if err := validateCommand(CommandInput{Type: command, TargetType: "unrelated", TargetID: "target"}, false); err == nil {
			t.Fatalf("%s accepted unrelated target type", command)
		}
	}
}
