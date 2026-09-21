package buyers

import (
	"context"
	"errors"
	"kredit/internal/identity"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

func importContacts() []ImportContact {
	return []ImportContact{{SourceReference: "roster:DIST-001", Target: "distributor@example.test", TargetType: "email", LegalName: "Distributor", BusinessType: "limited_company", BusinessAddress: "Lagos", Industry: "food"}, {SourceReference: "roster:DIST-002", Target: "other@example.test", TargetType: "email", LegalName: "Other distributor", BusinessType: "sole_proprietor", BusinessAddress: "Ibadan", Industry: "food"}}
}
func TestImportValidationRejectsAmbiguousRoster(t *testing.T) {
	hash := strings.Repeat("a", 64)
	for _, kind := range []string{"duplicate_reference", "duplicate_contact", "bad_phone", "unknown_business", "empty_name", "oversize", "invalid_hash"} {
		t.Run(kind, func(t *testing.T) {
			rows := importContacts()
			h := hash
			switch kind {
			case "duplicate_reference":
				rows[1].SourceReference = rows[0].SourceReference
			case "duplicate_contact":
				rows[1].Target = strings.ToUpper(rows[0].Target)
			case "bad_phone":
				rows[0].TargetType = "phone"
				rows[0].Target = "08012345678"
			case "unknown_business":
				rows[0].BusinessType = "unverified"
			case "empty_name":
				rows[0].LegalName = " "
			case "oversize":
				rows[0].BusinessAddress = strings.Repeat("x", 501)
			case "invalid_hash":
				h = "not-a-hash"
			}
			if validateImport(h, rows) == nil {
				t.Fatal("accepted invalid roster")
			}
		})
	}
	if err := validateImport(hash, importContacts()); err != nil {
		t.Fatal(err)
	}
}
func TestDurableImportAtomicReplayCancellationAndAuthority(t *testing.T) {
	if os.Getenv("KREDIT_INTEGRATION") != "1" {
		t.Skip("isolated database required")
	}
	ctx := context.Background()
	admin, err := pgxpool.New(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close()
	actor, other, org := uuid.NewString(), uuid.NewString(), uuid.NewString()
	for _, id := range []string{actor, other} {
		if _, err = admin.Exec(ctx, `INSERT INTO app.users(id,normalized_email) VALUES($1::uuid,$2)`, id, id+"@import.test"); err != nil {
			t.Fatal(err)
		}
	}
	if _, err = admin.Exec(ctx, `INSERT INTO app.organizations(id,legal_name,business_type,business_address,industry) VALUES($1::uuid,'Import test','limited_company','Lagos','food')`, org); err != nil {
		t.Fatal(err)
	}
	if _, err = admin.Exec(ctx, `INSERT INTO app.memberships(organization_id,user_id,role,status) VALUES($1::uuid,$2::uuid,'sales','active')`, org, actor); err != nil {
		t.Fatal(err)
	}
	cfg, err := pgxpool.ParseConfig(os.Getenv("DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	cfg.ConnConfig.RuntimeParams["role"] = "kredit_app"
	runtime, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer runtime.Close()
	s := NewPostgresStore(runtime, "import-test", identity.NewMockProvider())
	hash := strings.Repeat("a", 64)
	batch, err := s.SaveImport(ctx, actor, org, hash, importContacts())
	if err != nil {
		t.Fatal(err)
	}
	same, err := s.SaveImport(ctx, actor, org, hash, importContacts())
	if err != nil || same.ID != batch.ID {
		t.Fatalf("replay: %#v %v", same, err)
	}
	changed := importContacts()
	changed[0].LegalName = "Changed"
	if _, err = s.SaveImport(ctx, actor, org, hash, changed); !errors.Is(err, ErrImportConflict) {
		t.Fatalf("changed payload accepted: %v", err)
	}
	if _, _, err = s.CreateImportInvitation(ctx, actor, org, batch.ID, 1); !errors.Is(err, ErrImportConflict) {
		t.Fatalf("unapproved import accepted: %v", err)
	}
	if _, err = s.ReadImport(ctx, other, org, batch.ID); !errors.Is(err, ErrImportAuthority) {
		t.Fatalf("unrelated actor: %v", err)
	}
	if _, err = s.TransitionImport(ctx, actor, org, batch.ID, "approve"); err != nil {
		t.Fatal(err)
	}
	results := make(chan CreateInvitationResult, 2)
	errs := make(chan error, 2)
	var wg sync.WaitGroup
	for range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			v, _, e := s.CreateImportInvitation(ctx, actor, org, batch.ID, 1)
			results <- v
			errs <- e
		}()
	}
	wg.Wait()
	close(results)
	close(errs)
	for e := range errs {
		if e != nil {
			t.Fatal(e)
		}
	}
	first := <-results
	second := <-results
	if first.Invitation.ID != second.Invitation.ID || first.Replayed == second.Replayed {
		t.Fatal("concurrent row duplicated or lost replay state")
	}
	reopened, err := s.ReadImport(ctx, actor, org, batch.ID)
	if err != nil || len(reopened.CompletedRows) != 1 || reopened.CompletedRows[0] != 1 {
		t.Fatalf("resume lost progress: %#v %v", reopened, err)
	}
	if _, err = s.Accept(ctx, first.RawToken, other, AcceptInput{FullName: "Distributor Owner", ConsentsAccepted: true, TermsVersion: "terms-v1", PrivacyVersion: "privacy-v1", IdentityNoticeVersion: IdentityNoticeVersion}); err != nil {
		t.Fatal(err)
	}
	accepted, _, err := s.CreateImportInvitation(ctx, actor, org, batch.ID, 1)
	if err != nil || accepted.Invitation.Status != "accepted" || accepted.RawToken != "" || !accepted.Replayed {
		t.Fatalf("accepted import recovery: %#v %v", accepted, err)
	}

	otherOrg := uuid.NewString()
	if _, err = admin.Exec(ctx, `INSERT INTO app.organizations(id,legal_name,business_type,business_address,industry) VALUES($1::uuid,'Second supplier','limited_company','Lagos','food')`, otherOrg); err != nil {
		t.Fatal(err)
	}
	if _, err = admin.Exec(ctx, `INSERT INTO app.memberships(organization_id,user_id,role,status) VALUES($1::uuid,$2::uuid,'sales','active')`, otherOrg, actor); err != nil {
		t.Fatal(err)
	}
	otherBatch, err := s.SaveImport(ctx, actor, otherOrg, hash, importContacts())
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.TransitionImport(ctx, actor, otherOrg, otherBatch.ID, "approve"); err != nil {
		t.Fatal(err)
	}
	forged, err := runtime.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = forged.Exec(ctx, `SELECT set_config('app.current_user_id',$1,true),set_config('app.current_organization_id',$2,true)`, actor, otherOrg); err != nil {
		t.Fatal(err)
	}
	_, err = forged.Exec(ctx, `INSERT INTO app.distributor_import_rows(batch_id,row_number,invitation_id,created_by) VALUES($1::uuid,1,$2::uuid,$3::uuid)`, otherBatch.ID, first.Invitation.ID, actor)
	_ = forged.Rollback(ctx)
	var denied *pgconn.PgError
	if !errors.As(err, &denied) || denied.Code != "42501" {
		t.Fatalf("expected database policy rejection for foreign invitation, got: %v", err)
	}
	if _, err = s.TransitionImport(ctx, actor, org, batch.ID, "cancel"); err != nil {
		t.Fatal(err)
	}
	if _, _, err = s.CreateImportInvitation(ctx, actor, org, batch.ID, 2); !errors.Is(err, ErrImportConflict) {
		t.Fatalf("cancelled batch continued: %v", err)
	}
	var count int
	if err = admin.QueryRow(ctx, `SELECT count(*) FROM app.buyer_invitations WHERE organization_id=$1::uuid`, org).Scan(&count); err != nil || count != 1 {
		t.Fatalf("cancellation erased/duplicated invitation: %d %v", count, err)
	}
	tx, err := runtime.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = tx.Exec(ctx, `SELECT set_config('app.current_user_id',$1,true),set_config('app.current_organization_id',$2,true)`, other, org); err != nil {
		t.Fatal(err)
	}
	for _, table := range []string{"distributor_import_batches", "distributor_import_rows"} {
		if err = tx.QueryRow(ctx, `SELECT count(*) FROM app.`+table).Scan(&count); err != nil || count != 0 {
			t.Fatalf("%s exposed: %d %v", table, count, err)
		}
	}
	_ = tx.Rollback(ctx)
	if _, err = admin.Exec(ctx, `UPDATE app.memberships SET status='removed' WHERE organization_id=$1::uuid AND user_id=$2::uuid`, org, actor); err != nil {
		t.Fatal(err)
	}
	if _, err = s.ReadImport(ctx, actor, org, batch.ID); !errors.Is(err, ErrImportAuthority) {
		t.Fatalf("revoked authority retained: %v", err)
	}
}
