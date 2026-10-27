package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"time"

	"kredit/internal/auth"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	var identifier string
	var reason string
	flag.StringVar(&identifier, "identifier", "", "Email or phone of user to bootstrap as platform owner")
	flag.StringVar(&reason, "reason", "Initial platform owner deployment bootstrap", "Audit reason for bootstrap")
	flag.Parse()

	if identifier == "" {
		identifier = os.Getenv("OWNER_BOOTSTRAP_IDENTIFIER")
	}
	if identifier == "" {
		identifier = os.Getenv("OWNER_BOOTSTRAP_EMAIL")
	}
	if identifier == "" {
		fmt.Fprintf(os.Stderr, "Usage: bootstrap-owner -identifier <email|phone> [-reason <reason>]\nOr set OWNER_BOOTSTRAP_IDENTIFIER env var.\n")
		os.Exit(1)
	}
	databaseURL := os.Getenv("DATABASE_DIRECT_URL")
	if databaseURL == "" {
		databaseURL = os.Getenv("DATABASE_URL")
	}
	if databaseURL == "" {
		fmt.Fprintf(os.Stderr, "DATABASE_DIRECT_URL or DATABASE_URL is required\n")
		os.Exit(1)
	}
	// run owns the connection and transaction, so its deferred rollback and
	// close happen before the process exits.
	if err := run(databaseURL, identifier, reason); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(databaseURL, identifier, reason string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer pool.Close()

	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// One deployment bootstrap, including concurrent invocations. Recovery uses
	// the existing authenticated account recovery flow, never a second bootstrap.
	if _, err = tx.Exec(ctx, `LOCK TABLE app.platform_role_assignments IN SHARE ROW EXCLUSIVE MODE`); err != nil {
		return err
	}
	var used bool
	if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM app.platform_role_assignments WHERE role='platform_owner')`).Scan(&used); err != nil || used {
		return errors.New("owner bootstrap has already been used or could not be checked")
	}
	normID := auth.NormalizeIdentifier(identifier)
	userID, err := eligibleOwner(ctx, tx, normID)
	if err != nil {
		return errors.New("owner must first sign in and complete MFA within the last 10 minutes")
	}
	// The owner also holds platform_admin for the base administrative workflows.
	for _, role := range []string{"platform_owner", "platform_admin"} {
		if _, err = tx.Exec(ctx, `
		INSERT INTO app.platform_role_assignments(user_id, role, granted_by, reason)
		VALUES($1::uuid, $2, $1::uuid, $3)
		ON CONFLICT(user_id, role) WHERE revoked_at IS NULL
		DO UPDATE SET reason = EXCLUDED.reason
	`, userID, role, reason); err != nil {
			return fmt.Errorf("failed to assign %s role: %w", role, err)
		}
	}
	// Platform governance starts in solo_owner mode.
	if _, err = tx.Exec(ctx, `
		INSERT INTO app.platform_governance(id, mode, updated_by, reason)
		VALUES('singleton', 'solo_owner', $1::uuid, 'Initial owner bootstrap')
		ON CONFLICT (id) DO NOTHING
	`, userID); err != nil {
		return fmt.Errorf("failed to ensure platform governance: %w", err)
	}
	if _, err = tx.Exec(ctx, `
		INSERT INTO app.audit_events(actor_user_id, action, resource_type, resource_id, outcome, severity, metadata)
		VALUES($1::uuid, 'platform_owner.bootstrapped', 'user', $1::uuid, 'success', 'high', jsonb_build_object('identifier', $2::text, 'reason', $3::text))
	`, userID, normID, reason); err != nil {
		return fmt.Errorf("owner audit could not be recorded: %w", err)
	}
	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit owner bootstrap: %w", err)
	}
	fmt.Printf("Successfully bootstrapped platform owner %s (%s) in solo_owner mode.\n", userID, normID)
	return nil
}

func eligibleOwner(ctx context.Context, tx pgx.Tx, identifier string) (string, error) {
	var userID string
	// Lock the eligible account and session until the grant commits, so a
	// concurrent suspension or session revocation cannot invalidate this check.
	err := tx.QueryRow(ctx, `SELECT u.id::text FROM app.users u
      JOIN app.sessions s ON s.user_id=u.id
      WHERE (u.normalized_email=$1 OR u.normalized_phone=$1) AND u.status='active'
        AND s.authentication_level='AAL2' AND s.revoked_at IS NULL
        AND s.expires_at>clock_timestamp()
        AND s.mfa_verified_at>clock_timestamp()-interval '10 minutes'
      ORDER BY s.mfa_verified_at DESC LIMIT 1 FOR UPDATE OF u,s`, auth.NormalizeIdentifier(identifier)).Scan(&userID)
	return userID, err
}
