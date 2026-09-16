//go:build integration

package bankdebit

import (
	"context"
	"errors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"kredit/internal/mandates"
	"os"
	"strings"
	"testing"
)

// A disposable database verifies the real RLS policy and committed send fence.
// No bank is contacted and no real customer data is used.
func TestEnrollmentIsolationAndLateResponseFence(t *testing.T) {
	dsn := os.Getenv("KREDIT_NATIVE_TEST_DB")
	if !strings.Contains(dsn, "/kredit_native_audit?") {
		t.Fatal("requires the disposable kredit_native_audit database")
	}
	ctx := context.Background()
	admin, e := pgxpool.New(ctx, dsn)
	if e != nil {
		t.Fatal(e)
	}
	defer admin.Close()
	user, other := uuid.NewString(), uuid.NewString()
	for _, id := range []string{user, other} {
		if _, e = admin.Exec(ctx, `INSERT INTO app.users(id,normalized_email,status) VALUES($1::uuid,$2,'active')`, id, id+"@example.test"); e != nil {
			t.Fatal(e)
		}
	}
	cfg, e := pgxpool.ParseConfig(dsn)
	if e != nil {
		t.Fatal(e)
	}
	cfg.AfterConnect = func(ctx context.Context, c *pgx.Conn) error { _, e := c.Exec(ctx, "SET ROLE kredit_app"); return e }
	runtime, e := pgxpool.NewWithConfig(ctx, cfg)
	if e != nil {
		t.Fatal(e)
	}
	defer runtime.Close()
	store := NewStore(runtime, strings.Repeat("k", 32))
	ref := strings.ReplaceAll(uuid.NewString(), "-", "")
	_, e = store.Create(ctx, "audit-native", mandates.AuthorizationInput{Reference: ref, UserID: user, BusinessID: uuid.NewString(), AmountCeiling: 100000})
	if e != nil {
		t.Fatal(e)
	}
	details := Details{Name: "Synthetic Buyer", Email: "buyer@example.test", Phone: "08012345678", Address: "Synthetic address", BankCode: "058", AccountNumber: "0123456789", Consent: true}
	if _, e = store.Begin(ctx, "audit-native", ref, other, details); e == nil {
		t.Fatal("another buyer could submit this permission")
	}
	first, e := store.Begin(ctx, "audit-native", ref, user, details)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = store.Begin(ctx, "audit-native", ref, user, details); e == nil {
		t.Fatal("duplicate bank submission was not blocked")
	}
	var cipher string
	if e = admin.QueryRow(ctx, `SELECT details_ciphertext FROM app.bank_debit_enrollments WHERE provider='audit-native' AND reference=$1`, ref).Scan(&cipher); e != nil {
		t.Fatal(e)
	}
	if strings.Contains(cipher, details.AccountNumber) || strings.Contains(cipher, details.Email) {
		t.Fatal("sensitive bank details stored in plaintext")
	}
	// Simulate an operator-confirmed reset, followed by a new customer request.
	// A late response to the old version must not attach to the newer attempt.
	if _, e = admin.Exec(ctx, `UPDATE app.bank_debit_enrollments SET state='DRAFT',version=version+1 WHERE provider='audit-native' AND reference=$1`, ref); e != nil {
		t.Fatal(e)
	}
	second, e := store.Begin(ctx, "audit-native", ref, user, details)
	if e != nil {
		t.Fatal(e)
	}
	if e = store.Confirm(ctx, first, Result{Reference: "old-provider-reference"}); e == nil {
		t.Fatal("stale provider response overwrote the replacement request")
	}
	if e = store.Confirm(ctx, second, Result{Reference: "correct-provider-reference"}); e != nil {
		t.Fatal(e)
	}
	tx, e := store.tx(ctx, other)
	if e != nil {
		t.Fatal(e)
	}
	defer tx.Rollback(ctx)
	var count int
	if e = tx.QueryRow(ctx, `SELECT count(*) FROM app.bank_debit_enrollments WHERE reference=$1`, ref).Scan(&count); e != nil {
		t.Fatal(e)
	}
	if count != 0 {
		t.Fatal("another buyer can read bank enrollment")
	}
}

type delayedCancellation struct{ mandates.Provider }

func (p delayedCancellation) CancelMandate(context.Context, string, string) (mandates.Mandate, error) {
	return mandates.Mandate{}, errors.New("synthetic bank timeout")
}
func TestCancellationStaysPausedWhenBankStillReportsActive(t *testing.T) {
	dsn := os.Getenv("KREDIT_NATIVE_TEST_DB")
	if !strings.Contains(dsn, "/kredit_native_audit?") {
		t.Fatal("requires disposable audit database")
	}
	ctx := context.Background()
	admin, e := pgxpool.New(ctx, dsn)
	if e != nil {
		t.Fatal(e)
	}
	defer admin.Close()
	user := uuid.NewString()
	var business string
	if _, e = admin.Exec(ctx, `INSERT INTO app.users(id,normalized_email,status) VALUES($1::uuid,$2,'active')`, user, user+"@example.test"); e != nil {
		t.Fatal(e)
	}
	if e = admin.QueryRow(ctx, `INSERT INTO app.businesses(owner_user_id,legal_name,business_type,business_address,industry) VALUES($1::uuid,'Synthetic Buyer','limited_company','Synthetic address','Test') RETURNING id::text`, user).Scan(&business); e != nil {
		t.Fatal(e)
	}
	cfg, e := pgxpool.ParseConfig(dsn)
	if e != nil {
		t.Fatal(e)
	}
	cfg.AfterConnect = func(ctx context.Context, c *pgx.Conn) error { _, e := c.Exec(ctx, "SET ROLE kredit_app"); return e }
	pool, e := pgxpool.NewWithConfig(ctx, cfg)
	if e != nil {
		t.Fatal(e)
	}
	defer pool.Close()
	provider := mandates.NewPostgresProviderWithRemote(pool, delayedCancellation{Provider: mandates.NewMockProvider()})
	m, e := provider.CreateAuthorizationSession(ctx, mandates.AuthorizationInput{UserID: user, BusinessID: business, AmountCeiling: 100000})
	if e != nil {
		t.Fatal(e)
	}
	if _, e = provider.CancelMandate(ctx, m.ProviderID, "Buyer withdrew permission"); e == nil {
		t.Fatal("timeout should remain visible")
	}
	current, e := provider.GetMandate(ctx, m.ProviderID)
	if e != nil {
		t.Fatal(e)
	}
	if current.Status != mandates.Paused || !current.CancellationRequested {
		t.Fatalf("bank response revived withdrawn permission: %+v", current)
	}
}
