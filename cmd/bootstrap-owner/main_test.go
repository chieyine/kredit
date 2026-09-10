package main

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/jackc/pgx/v5"
)

func TestEligibleOwnerRequiresActiveAccountAndFreshSession(t *testing.T) {
	if os.Getenv("KREDIT_INTEGRATION") != "1" || os.Getenv("DATABASE_URL") == "" {
		t.Skip("KREDIT_INTEGRATION=1 and DATABASE_URL required")
	}
	conn, err := pgx.Connect(t.Context(), os.Getenv("DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = conn.Close(context.Background()) }()
	tx, err := conn.Begin(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	var userID, sessionID string
	err = tx.QueryRow(t.Context(), `INSERT INTO app.users(normalized_phone) VALUES('+2348099990198') RETURNING id::text`).Scan(&userID)
	if err != nil {
		t.Fatal(err)
	}
	err = tx.QueryRow(t.Context(), `INSERT INTO app.sessions(user_id,token_hash,authentication_level,expires_at,mfa_verified_at) VALUES($1::uuid,decode(replace(uuidv7()::text,'-',''),'hex'),'AAL2',now()+interval '1 hour',now()) RETURNING id::text`, userID).Scan(&sessionID)
	if err != nil {
		t.Fatal(err)
	}
	got, err := eligibleOwner(t.Context(), tx, "0809 999 0198")
	if err != nil || got != userID {
		t.Fatalf("normalized phone lookup: got %q, %v", got, err)
	}
	for _, change := range []string{
		`UPDATE app.users SET status='suspended' WHERE id=$1::uuid`,
		`UPDATE app.sessions SET revoked_at=now() WHERE user_id=$1::uuid`,
		`UPDATE app.sessions SET expires_at=now()-interval '1 second' WHERE user_id=$1::uuid`,
		`UPDATE app.sessions SET mfa_verified_at=now()-interval '11 minutes' WHERE user_id=$1::uuid`,
		`UPDATE app.sessions SET authentication_level='AAL1' WHERE user_id=$1::uuid`,
	} {
		if _, err := tx.Exec(t.Context(), "SAVEPOINT eligibility"); err != nil {
			t.Fatal(err)
		}
		if _, err := tx.Exec(t.Context(), change, userID); err != nil {
			t.Fatal(err)
		}
		if _, err := eligibleOwner(t.Context(), tx, "+2348099990198"); !errors.Is(err, pgx.ErrNoRows) {
			t.Fatalf("ineligible account accepted after %s: %v", change, err)
		}
		if _, err := tx.Exec(t.Context(), "ROLLBACK TO SAVEPOINT eligibility"); err != nil {
			t.Fatal(err)
		}
	}
}
