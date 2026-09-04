//go:build integration

package auth

import (
	"context"
	"github.com/jackc/pgx/v5/pgxpool"
	"os"
	"testing"
	"time"
)

func TestPostgresAbandonedTOTPEnrollmentCanRestart(t *testing.T) {
	if os.Getenv("KREDIT_INTEGRATION") != "1" {
		t.Skip("integration database required")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	s := NewPostgresStore(pool, "audit-totp")
	u, err := s.FindOrCreateUser("audit-totp-"+time.Now().Format("150405.000000000")+"@example.test", "email")
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Exec(ctx, `DELETE FROM app.mfa_methods WHERE user_id=$1::uuid;DELETE FROM app.users WHERE id=$1::uuid`, u.ID)
	runtimeCfg, err := pgxpool.ParseConfig(os.Getenv("DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	runtimeCfg.ConnConfig.RuntimeParams["role"] = "kredit_app"
	runtimePool, err := pgxpool.NewWithConfig(ctx, runtimeCfg)
	if err != nil {
		t.Fatal(err)
	}
	defer runtimePool.Close()
	got, err := NewPostgresStore(runtimePool, "audit-totp").UserByID(u.ID)
	if err != nil || got.ID != u.ID {
		t.Fatalf("runtime role could not load recovery recipient: user=%+v err=%v", got, err)
	}
	first, err := s.BeginTOTPEnrollment(u.ID)
	if err != nil {
		t.Fatal(err)
	}
	second, err := s.BeginTOTPEnrollment(u.ID)
	if err != nil || first.Secret == second.Secret {
		t.Fatalf("restart=%v", err)
	}
	if err = s.VerifyTOTP(u.ID, TOTPCode(first.Secret, time.Now())); err == nil {
		t.Fatal("abandoned secret remained valid")
	}
	if err = s.VerifyTOTP(u.ID, TOTPCode(second.Secret, time.Now())); err != nil {
		t.Fatal(err)
	}
	if _, err = s.BeginTOTPEnrollment(u.ID); err == nil {
		t.Fatal("verified authenticator was replaced")
	}
}
