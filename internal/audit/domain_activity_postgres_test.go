package audit

import (
	"context"
	"os"
	"testing"

	"kredit/internal/db"
	"kredit/internal/identifier"
)

func TestDomainActivityCommitsAndRollsBackWithPrivateRequest(t *testing.T) {
	if os.Getenv("KREDIT_INTEGRATION") != "1" {
		t.Skip("isolated database required")
	}
	ctx := context.Background()
	root, err := db.Open(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	app, err := db.OpenAsRole(ctx, os.Getenv("APP_DATABASE_URL"), "kredit_app")
	if err != nil {
		t.Fatal(err)
	}
	defer app.Close()
	user := identifier.New()
	if _, err = root.Raw().Exec(ctx, `INSERT INTO app.users(id,normalized_email) VALUES($1,$2)`, user, "activity-"+user+"@example.test"); err != nil {
		t.Fatal(err)
	}
	for _, commit := range []bool{false, true} {
		id := identifier.New()
		tx, err := app.Raw().Begin(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = tx.Exec(ctx, `SELECT set_config('app.current_user_id',$1,true)`, user); err != nil {
			t.Fatal(err)
		}
		if _, err = tx.Exec(ctx, `INSERT INTO app.privacy_requests(id,requester_user_id,request_type,details) VALUES($1,$2,'ACCESS','private details must never enter a notice')`, id, user); err != nil {
			_ = tx.Rollback(ctx)
			t.Fatal(err)
		}
		if commit {
			err = tx.Commit(ctx)
		} else {
			err = tx.Rollback(ctx)
		}
		if err != nil {
			t.Fatal(err)
		}
		var audits, notices, leaks int
		if err = root.Raw().QueryRow(ctx, `SELECT (SELECT count(*) FROM app.audit_events WHERE resource_id=$1),(SELECT count(*) FROM app.notifications n JOIN app.audit_events a ON n.event_reference='activity:'||a.id::text||':'||$2::text WHERE a.resource_id=$1),(SELECT count(*) FROM app.notifications n JOIN app.audit_events a ON n.event_reference LIKE 'activity:'||a.id::text||':%' WHERE a.resource_id=$1 AND (n.recipient_id<>$2::uuid OR n.body LIKE '%private details%'))`, id, user).Scan(&audits, &notices, &leaks); err != nil {
			t.Fatal(err)
		}
		expected := 0
		if commit {
			expected = 1
		}
		if audits != expected || notices != expected || leaks != 0 {
			t.Fatalf("activity not atomic/private: committed=%v audits=%d notices=%d leaks=%d", commit, audits, notices, leaks)
		}
	}
}
