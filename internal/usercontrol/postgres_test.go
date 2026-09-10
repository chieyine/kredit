//go:build integration

package usercontrol

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"
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
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	owner, finance, buyer := uuid.NewString(), uuid.NewString(), uuid.NewString()
	for _, id := range []string{owner, finance, buyer} {
		if _, err := tx.Exec(ctx, `INSERT INTO app.users(id,normalized_email) VALUES($1::uuid,$2)`, id, id+"@recovery.test"); err != nil {
			t.Fatal(err)
		}
	}
	for _, actor := range []string{owner, finance} {
		if _, err := tx.Exec(ctx, `INSERT INTO app.platform_role_assignments(user_id,role,granted_by,reason) VALUES($1::uuid,'platform_admin',$1::uuid,'Synthetic privacy reviewer')`, actor); err != nil {
			t.Fatal(err)
		}
	}
	s := NewPostgresStore(pool, "integration-secret")
	s.pool = tx
	org := uuid.NewString()
	if _, err := tx.Exec(ctx, `INSERT INTO app.organizations(id,legal_name,business_type,business_address,industry) VALUES($1::uuid,'Privacy scope fixture','limited_company','Test address','Test')`, org); err != nil {
		t.Fatal(err)
	}
	if _, err := s.CreatePrivacyRequest(ctx, buyer, org, "ACCESS", "Unrelated business scope"); !errors.Is(err, ErrPrivacyOrganizationForbidden) {
		t.Fatalf("unrelated business privacy request accepted: %v", err)
	}
	if _, err := tx.Exec(ctx, `INSERT INTO app.memberships(organization_id,user_id,role,status) VALUES($1::uuid,$2::uuid,'viewer','active')`, org, buyer); err != nil {
		t.Fatal(err)
	}
	if _, err := s.CreatePrivacyRequest(ctx, buyer, org, "ACCESS", "My personal data at this business"); err != nil {
		t.Fatalf("own active membership scope rejected: %v", err)
	}
	var deliveredToken string
	s.SetRecoveryDelivery(func(_ context.Context, r RecoveryRequest, token string) error {
		if r.TargetUserID != owner {
			t.Fatalf("recovery sent to wrong user: %s", r.TargetUserID)
		}
		deliveredToken = token
		return nil
	})
	codes, err := s.GenerateRecoveryCodes(ctx, owner)
	if err != nil {
		t.Fatal(err)
	}
	id, err := s.RequestRecovery(ctx, owner+"@recovery.test", "email", "integration-device")
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
	_, err = tx.Exec(ctx, `UPDATE app.account_recovery_requests SET cooling_off_until=now()-interval '1 minute' WHERE id=$1::uuid`, id)
	if err != nil {
		t.Fatal(err)
	}
	if token == "" || deliveredToken != token {
		t.Fatal("continuation token was not privately delivered")
	}
	userID, err := s.CompleteRecovery(ctx, id, deliveredToken)
	if err != nil || userID != owner {
		t.Fatalf("complete=%q err=%v", userID, err)
	}
	var active int
	if err = tx.QueryRow(ctx, `SELECT count(*) FROM app.account_recovery_codes WHERE user_id=$1::uuid AND state='ACTIVE'`, owner).Scan(&active); err != nil || active != 0 {
		t.Fatalf("active=%d err=%v", active, err)
	}

	if _, err := tx.Exec(ctx, `SET LOCAL ROLE kredit_app`); err != nil {
		t.Fatal(err)
	}
	_, err = tx.Exec(ctx, `INSERT INTO app.legal_holds(user_id,scope,reason,created_by) VALUES($1::uuid,'financial-records','statutory financial retention',$2::uuid)`, buyer, owner)
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
	p, err = s.CompletePrivacyWithReason(ctx, p.ID, owner, finance, p.Version, "Verified non-essential processing restriction; held financial records retained")
	if err != nil || p.State != "COMPLETED" {
		t.Fatalf("complete privacy=%+v err=%v", p, err)
	}
	var restrictions int
	if err = tx.QueryRow(ctx, `SELECT count(*) FROM app.processing_restrictions WHERE user_id=$1::uuid AND active`, buyer).Scan(&restrictions); err != nil || restrictions != 1 {
		t.Fatalf("restrictions=%d err=%v", restrictions, err)
	}

	if allowed, err := s.AllowsOptionalProcessing(ctx, buyer); err != nil || allowed {
		t.Fatalf("runtime role did not apply processing restriction: %v %v", allowed, err)
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
	if err != nil || !strings.Contains(string(payload), buyer+"@recovery.test") {
		t.Fatalf("authoritative export=%s err=%v", payload, err)
	}
	var exported map[string]json.RawMessage
	if err = json.Unmarshal(payload, &exported); err != nil {
		t.Fatal(err)
	}
	for _, section := range []string{"payments", "seller_consents", "notification_preferences", "bank_permissions", "uploaded_files", "correction_decisions"} {
		if _, ok := exported[section]; !ok {
			t.Fatalf("missing export section %s", section)
		}
	}
	if strings.Contains(string(payload), "token_hash") || strings.Contains(string(payload), "destination_ciphertext") {
		t.Fatal("credential data entered privacy export")
	}
	if _, err = s.PrivacyExport(ctx, ex.ID, owner); err == nil {
		t.Fatal("another user accessed the privacy export")
	}

	// A real owner can finish work alone only under explicit solo governance.
	if _, err := tx.Exec(ctx, `INSERT INTO app.platform_role_assignments(user_id,role,granted_by,reason) VALUES($1::uuid,'platform_owner',$1::uuid,'Synthetic privacy owner')`, owner); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(ctx, `UPDATE app.platform_governance SET mode='solo_owner'`); err != nil {
		t.Fatal(err)
	}
	sole, err := s.CreatePrivacyRequest(ctx, buyer, "", "ACCESS", "Solo owner export fixture")
	if err != nil {
		t.Fatal(err)
	}
	sole, err = s.DecidePrivacy(ctx, sole.ID, owner, "APPROVED", "Verify scope before preparing export", sole.Version)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.CompletePrivacy(ctx, sole.ID, owner, owner, sole.Version); err == nil {
		t.Fatal("solo completion accepted without a reason")
	}
	if _, err = s.CompletePrivacyWithReason(ctx, sole.ID, owner, owner, sole.Version, "Verified scope and completed the protected export"); err != nil {
		t.Fatalf("solo owner could not complete privacy work: %v", err)
	}
	delegated, err := s.CreatePrivacyRequest(ctx, buyer, "", "ACCESS", "Delegated completion fixture")
	if err != nil {
		t.Fatal(err)
	}
	delegated, err = s.DecidePrivacy(ctx, delegated.ID, owner, "APPROVED", "Verify scope under delegated governance", delegated.Version)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(ctx, `UPDATE app.platform_governance SET mode='delegated_team'`); err != nil {
		t.Fatal(err)
	}
	if _, err = s.CompletePrivacyWithReason(ctx, delegated.ID, owner, owner, delegated.Version, "Attempt same reviewer completion in delegated mode"); err == nil {
		t.Fatal("delegated governance permitted self-completion")
	}
}
