package bankdebit

import (
	"context"
	"errors"
	"fmt"
	"net"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"kredit/internal/mandates"
)

func TestEnrollmentLookupErrorsDoNotMasqueradeAsMissing(t *testing.T) {
	if err := classifyEnrollmentLookupError(nil); err != nil {
		t.Fatalf("successful lookup became an error: %v", err)
	}
	for _, err := range []error{pgx.ErrNoRows, fmt.Errorf("lookup: %w", pgx.ErrNoRows)} {
		if got := classifyEnrollmentLookupError(err); got != ErrEnrollmentNotFound {
			t.Fatalf("missing row classification: %v", got)
		}
	}
	driver := &pgconn.PgError{Code: "08006", Message: "synthetic private database diagnostics"}
	for _, cause := range []error{context.Canceled, context.DeadlineExceeded, driver, fmt.Errorf("driver: %w", driver)} {
		got := classifyEnrollmentLookupError(cause)
		if !errors.Is(got, ErrEnrollmentUnavailable) || !errors.Is(got, cause) || errors.Is(got, ErrEnrollmentNotFound) {
			t.Fatalf("unavailable lookup lost its classification or cause: %v", got)
		}
		if got.Error() != ErrEnrollmentUnavailable.Error() {
			t.Fatalf("lookup exposed private diagnostics: %q", got.Error())
		}
	}
	var recovered *pgconn.PgError
	if !errors.As(classifyEnrollmentLookupError(driver), &recovered) || recovered != driver {
		t.Fatal("structured internal diagnostics were lost")
	}
}

func TestEnrollmentAbsentStorageReturnsUnavailable(t *testing.T) {
	ctx := context.Background()
	input := mandates.AuthorizationInput{Reference: "synthetic-authorization", UserID: "11111111-1111-4111-8111-111111111111", AmountCeiling: 100}
	enrollment := Enrollment{Provider: "synthetic", Reference: input.Reference, Input: input}
	details := Details{Name: "Synthetic Buyer", Email: "buyer@example.test", Phone: "+2348000000000", Address: "Synthetic test address", BankCode: "058", AccountNumber: "0123456789", Consent: true}
	if err := details.Validate(); err != nil {
		t.Fatalf("invalid test input: %v", err)
	}
	for _, tc := range []struct {
		name  string
		store *Store
	}{
		{name: "nil receiver"},
		{name: "zero store", store: &Store{}},
		{name: "constructor without pool", store: NewStore(nil, "")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			checks := []struct {
				name string
				run  func() error
			}{
				{name: "create", run: func() error { _, err := tc.store.Create(ctx, "synthetic", input); return err }},
				{name: "load", run: func() error { _, err := tc.store.Load(ctx, "synthetic", input.Reference); return err }},
				{name: "begin", run: func() error {
					_, err := tc.store.Begin(ctx, "synthetic", input.Reference, input.UserID, details)
					return err
				}},
				{name: "confirm", run: func() error {
					return tc.store.Confirm(ctx, enrollment, Result{Reference: "synthetic-provider-reference"})
				}},
				{name: "cancel draft", run: func() error { return tc.store.CancelDraft(ctx, enrollment) }},
			}
			for _, check := range checks {
				t.Run(check.name, func(t *testing.T) {
					if err := check.run(); !errors.Is(err, ErrEnrollmentUnavailable) {
						t.Fatalf("expected unavailable storage, got %v", err)
					}
				})
			}
		})
	}
}

func TestEnrollmentLoadClosedPoolIsNotReportedMissing(t *testing.T) {
	ctx := context.Background()
	cfg, err := pgxpool.ParseConfig("postgres://synthetic@127.0.0.1:1/kredit_unreachable_test?sslmode=disable")
	if err != nil {
		t.Fatal(err)
	}
	// A closed-pool test must never contact a database or a payment provider.
	cfg.MinConns = 0
	cfg.MinIdleConns = 0
	cfg.ConnConfig.DialFunc = func(context.Context, string, string) (net.Conn, error) {
		return nil, errors.New("network access forbidden in storage-boundary test")
	}
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	pool.Close()
	_, err = NewStore(pool, "").Load(ctx, "synthetic", "synthetic-reference")
	if !errors.Is(err, ErrEnrollmentUnavailable) || errors.Is(err, ErrEnrollmentNotFound) {
		t.Fatalf("closed pool was not classified as unavailable: %v", err)
	}
	if err.Error() != ErrEnrollmentUnavailable.Error() {
		t.Fatalf("closed-pool lookup exposed diagnostics: %q", err.Error())
	}
}
