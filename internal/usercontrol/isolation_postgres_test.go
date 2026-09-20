//go:build integration

package usercontrol

import (
	"context"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"kredit/internal/db"
	"os"
	"strings"
	"testing"
)

func TestRecoveryAndPrivacyThroughRestrictedLogin(t *testing.T) {
	if os.Getenv("KREDIT_INTEGRATION") != "1" || os.Getenv("KREDIT_TEST_APP_DATABASE_URL") == "" {
		t.Skip("isolated admin and runtime fixture connections required")
	}
	ctx := t.Context()
	admin, err := pgxpool.New(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close()
	runtime, err := db.OpenAsRole(ctx, os.Getenv("KREDIT_TEST_APP_DATABASE_URL"), "kredit_app")
	if err != nil {
		t.Fatal(err)
	}
	defer runtime.Close()
	user, reviewer, second := uuid.NewString(), uuid.NewString(), uuid.NewString()
	for _, id := range []string{user, reviewer, second} {
		if _, err = admin.Exec(ctx, `INSERT INTO app.users(id,normalized_email) VALUES($1::uuid,$2)`, id, id+"@isolated-recovery.test"); err != nil {
			t.Fatal(err)
		}
	}
	for _, id := range []string{reviewer, second} {
		if _, err = admin.Exec(ctx, `INSERT INTO app.platform_role_assignments(user_id,role,granted_by,reason) VALUES($1::uuid,'platform_admin',$1::uuid,'Synthetic isolation reviewer')`, id); err != nil {
			t.Fatal(err)
		}
	}
	s := NewPostgresStore(runtime.Raw(), "synthetic-isolation-secret")
	var delivered string
	s.SetRecoveryDelivery(func(_ context.Context, r RecoveryRequest, token string) error {
		if r.TargetUserID != user {
			t.Fatal("wrong recovery subject")
		}
		delivered = token
		return nil
	})
	codes, err := s.GenerateRecoveryCodes(ctx, user)
	if err != nil {
		t.Fatal(err)
	}
	id, err := s.RequestRecovery(ctx, user+"@isolated-recovery.test", "email", uuid.NewString())
	if err != nil || id == "" {
		t.Fatalf("request: %s %v", id, err)
	}
	if _, err = s.AddRecoveryEvidence(ctx, id, "recovery_code", codes[0]); err != nil {
		t.Fatal(err)
	}
	recovery, err := s.AddRecoveryEvidence(ctx, id, "verified_email", "synthetic verified challenge")
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err = s.ReviewRecovery(ctx, id, uuid.NewString(), "approve", "Unauthorized review", recovery.Version); err == nil {
		t.Fatal("unassigned reviewer accepted")
	}
	recovery, token, err := s.ReviewRecovery(ctx, id, reviewer, "approve", "Synthetic identity verified", recovery.Version)
	if err != nil || token == "" || token != delivered {
		t.Fatalf("review: %v", err)
	}
	if _, err = s.CompleteRecovery(ctx, id, token); err == nil {
		t.Fatal("cooling off bypassed")
	}
	if _, err = admin.Exec(ctx, `UPDATE app.account_recovery_requests SET cooling_off_until=now()-interval '1 minute' WHERE id=$1::uuid`, id); err != nil {
		t.Fatal(err)
	}
	if _, err = s.CompleteRecovery(ctx, id, "wrong token"); err == nil {
		t.Fatal("wrong recovery token accepted")
	}
	got, err := s.CompleteRecovery(ctx, id, token)
	if err != nil || got != user {
		t.Fatalf("complete recovery: %s %v", got, err)
	}
	privacy, err := s.CreatePrivacyRequest(ctx, user, "", "PORTABILITY", "Synthetic export request")
	if err != nil {
		t.Fatal(err)
	}
	privacy, err = s.DecidePrivacy(ctx, privacy.ID, reviewer, "APPROVED", "Synthetic identity verified", privacy.Version)
	if err != nil {
		t.Fatal(err)
	}
	privacy, err = s.CompletePrivacy(ctx, privacy.ID, reviewer, second, privacy.Version)
	if err != nil {
		t.Fatal(err)
	}
	payload, err := s.PrivacyExport(ctx, privacy.ID, user)
	if err != nil || !strings.Contains(string(payload), user+"@isolated-recovery.test") {
		t.Fatalf("subject export: %v", err)
	}
	if _, err = s.PrivacyExport(ctx, privacy.ID, second); err == nil {
		t.Fatal("another subject read private export")
	}
	if _, err = s.CreatePrivacyRequest(ctx, user, "", "ACCESS", "Pending synthetic review"); err != nil {
		t.Fatal(err)
	}
	if rows, err := s.ListPrivacyReview(db.WithTenantContext(ctx, second, "")); err != nil || len(rows) == 0 {
		t.Fatalf("active reviewer cannot read: %v", err)
	}
	if _, err = admin.Exec(ctx, `UPDATE app.platform_role_assignments SET revoked_at=now() WHERE user_id=$1::uuid`, second); err != nil {
		t.Fatal(err)
	}
	if rows, err := s.ListPrivacyReview(db.WithTenantContext(ctx, second, "")); err != nil || len(rows) != 0 {
		t.Fatalf("revoked reviewer retained access: %d %v", len(rows), err)
	}
}
