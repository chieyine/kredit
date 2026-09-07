package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/google/uuid"
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

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to connect to database: %v\n", err)
		os.Exit(1)
	}
	defer pool.Close()

	normID := strings.TrimSpace(strings.ToLower(identifier))
	var userID string
	err = pool.QueryRow(ctx, `SELECT id::text FROM app.users WHERE normalized_email=$1 OR normalized_phone=$1`, normID).Scan(&userID)
	if err != nil {
		if err == pgx.ErrNoRows {
			// Create user if not exists
			newID := uuid.New().String()
			isEmail := strings.Contains(normID, "@")
			var emailCol, phoneCol any
			if isEmail {
				emailCol = normID
				phoneCol = nil
			} else {
				emailCol = nil
				phoneCol = normID
			}
			err = pool.QueryRow(ctx, `
				INSERT INTO app.users(id, normalized_email, normalized_phone, display_name, status)
				VALUES($1::uuid, $2, $3, 'Platform Owner', 'active')
				RETURNING id::text
			`, newID, emailCol, phoneCol).Scan(&userID)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to create user for owner: %v\n", err)
				os.Exit(1)
			}
			fmt.Printf("Created active user %s for identifier %s\n", userID, normID)
		} else {
			fmt.Fprintf(os.Stderr, "Failed to query user: %v\n", err)
			os.Exit(1)
		}
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to start transaction: %v\n", err)
		os.Exit(1)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Assign platform_owner role
	_, err = tx.Exec(ctx, `
		INSERT INTO app.platform_role_assignments(user_id, role, granted_by, reason)
		VALUES($1::uuid, 'platform_owner', $1::uuid, $2)
		ON CONFLICT(user_id, role) WHERE revoked_at IS NULL
		DO UPDATE SET reason = EXCLUDED.reason
	`, userID, reason)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to assign platform_owner role: %v\n", err)
		os.Exit(1)
	}

	// Also assign platform_admin role for base administrative workflows
	_, err = tx.Exec(ctx, `
		INSERT INTO app.platform_role_assignments(user_id, role, granted_by, reason)
		VALUES($1::uuid, 'platform_admin', $1::uuid, $2)
		ON CONFLICT(user_id, role) WHERE revoked_at IS NULL
		DO UPDATE SET reason = EXCLUDED.reason
	`, userID, reason)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to assign platform_admin role: %v\n", err)
		os.Exit(1)
	}

	// Ensure platform governance is set to solo_owner initially
	_, err = tx.Exec(ctx, `
		INSERT INTO app.platform_governance(id, mode, updated_by, reason)
		VALUES('singleton', 'solo_owner', $1::uuid, 'Initial owner bootstrap')
		ON CONFLICT (id) DO NOTHING
	`, userID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to ensure platform governance: %v\n", err)
		os.Exit(1)
	}

	// Record audit event
	_, err = tx.Exec(ctx, `
		INSERT INTO app.audit_events(actor_user_id, action, resource_type, resource_id, outcome, severity, metadata)
		VALUES($1::uuid, 'platform_owner.bootstrapped', 'user', $1::uuid, 'success', 'high', jsonb_build_object('identifier', $2::text, 'reason', $3::text))
	`, userID, normID, reason)
	if err != nil {
		// Log but continue if audit_events schema details vary
		fmt.Fprintf(os.Stderr, "Notice: could not record audit event directly: %v\n", err)
	}

	if err := tx.Commit(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to commit owner bootstrap: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Successfully bootstrapped platform owner %s (%s) in solo_owner mode.\n", userID, normID)
}
