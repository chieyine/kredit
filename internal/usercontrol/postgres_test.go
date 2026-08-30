//go:build integration

package usercontrol

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPostgresRecoveryAndPrivacyControls(t *testing.T) {
	if os.Getenv("KREDIT_INTEGRATION") != "1" {
		t.Skip("set KREDIT_INTEGRATION=1")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	owner := "00000000-0000-7000-8000-000000000001"
	finance := "00000000-0000-7000-8000-000000000002"
	buyer := "00000000-0000-7000-8000-000000000004"
	_, _ = pool.Exec(ctx, `DELETE FROM app.account_recovery_events;DELETE FROM app.account_recovery_evidence;DELETE FROM app.account_recovery_requests;DELETE FROM app.account_recovery_codes;DELETE FROM app.privacy_exports;DELETE FROM app.processing_restrictions;DELETE FROM app.privacy_request_events;DELETE FROM app.privacy_requests;DELETE FROM app.legal_holds`)
	s := NewPostgresStore(pool, "integration-secret")
	codes, err := s.GenerateRecoveryCodes(ctx, owner)
	if err != nil {
		t.Fatal(err)
	}
	id, err := s.RequestRecovery(ctx, "owner@abc-pharmaceuticals.test", "email", "integration-device")
	if err != nil || id == "" {
		t.Fatalf("request=%q err=%v", id, err)
	}
	r, err := s.AddRecoveryEvidence(ctx, id, "recovery_code", codes[0])
	if err != nil {
		t.Fatal(err)
	}
	r, err = s.AddRecoveryEvidence(ctx, id, "verified_email", "verified-challenge")
	if err != nil || r.State != RecoveryPendingReview {
		t.Fatalf("evidence=%+v err=%v", r, err)
	}
	r, token, err := s.ReviewRecovery(ctx, id, finance, "approve", "verified business identity", r.Version)
	if err != nil || r.State != RecoveryCoolingOff {
		t.Fatalf("review=%+v err=%v", r, err)
	}
	_, err = pool.Exec(ctx, `UPDATE app.account_recovery_requests SET cooling_off_until=now()-interval '1 minute' WHERE id=$1::uuid`, id)
	if err != nil {
		t.Fatal(err)
	}
	userID, err := s.CompleteRecovery(ctx, id, token)
	if err != nil || userID != owner {
		t.Fatalf("complete=%q err=%v", userID, err)
	}
	var active int
	if err = pool.QueryRow(ctx, `SELECT count(*) FROM app.account_recovery_codes WHERE user_id=$1::uuid AND state='ACTIVE'`, owner).Scan(&active); err != nil || active != 0 {
		t.Fatalf("active=%d err=%v", active, err)
	}

	_, err = pool.Exec(ctx, `INSERT INTO app.legal_holds(user_id,scope,reason,created_by) VALUES($1::uuid,'financial-records','statutory financial retention',$2::uuid)`, buyer, owner)
	if err != nil {
		t.Fatal(err)
	}
	p, err := s.CreatePrivacyRequest(ctx, buyer, "", "DELETION", "Delete data that is not legally retained")
	if err != nil {
		t.Fatal(err)
	}
	p, err = s.DecidePrivacy(ctx, p.ID, owner, "APPROVED", "validated deletion request", p.Version)
	if err != nil || p.State != "PARTIALLY_APPROVED" || !p.LegalHoldApplies {
		t.Fatalf("decision=%+v err=%v", p, err)
	}
	p, err = s.CompletePrivacy(ctx, p.ID, owner, finance, p.Version)
	if err != nil || p.State != "COMPLETED" {
		t.Fatalf("complete privacy=%+v err=%v", p, err)
	}
	var restrictions int
	if err = pool.QueryRow(ctx, `SELECT count(*) FROM app.processing_restrictions WHERE user_id=$1::uuid AND active`, buyer).Scan(&restrictions); err != nil || restrictions != 1 {
		t.Fatalf("restrictions=%d err=%v", restrictions, err)
	}

	ex, err := s.CreatePrivacyRequest(ctx, buyer, "", "PORTABILITY", "Provide a portable copy of my account data")
	if err != nil {
		t.Fatal(err)
	}
	ex, err = s.DecidePrivacy(ctx, ex.ID, owner, "APPROVED", "identity and scope validated", ex.Version)
	if err != nil {
		t.Fatal(err)
	}
	ex, err = s.CompletePrivacy(ctx, ex.ID, owner, finance, ex.Version)
	if err != nil || ex.ExportReference == "" {
		t.Fatalf("export=%+v err=%v", ex, err)
	}
	payload, err := s.PrivacyExport(ctx, ex.ID, buyer)
	if err != nil || !strings.Contains(string(payload), "buyer@royal-pharmacy.test") {
		t.Fatalf("authoritative export=%s err=%v", payload, err)
	}
	if _, err = s.PrivacyExport(ctx, ex.ID, owner); err == nil {
		t.Fatal("another user accessed the privacy export")
	}
}
