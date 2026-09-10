package organizations

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"kredit/internal/access"
	"kredit/internal/db"
)

func TestTeamInvitationLifecycle(t *testing.T) {
	s := NewStore()
	now := time.Now().UTC()
	s.now = func() time.Time { return now }
	org, _, err := s.Create("owner", CreateInput{LegalName: "Team", BusinessType: "limited_company", BusinessAddress: "Lagos", Industry: "retail"})
	if err != nil {
		t.Fatal(err)
	}
	for _, outcome := range []string{"accepted", "revoked", "expired"} {
		invite, _, err := s.Invite("owner", org.ID, outcome+"@example.test", "email", outcome, access.RoleViewer)
		if err != nil {
			t.Fatal(err)
		}
		switch outcome {
		case "accepted":
			if len(s.ActivateInvitations(outcome)) != 1 {
				t.Fatal("invitation not activated")
			}
		case "revoked":
			if _, err = s.ChangeStatus(org.ID, "owner", outcome, "removed"); err != nil {
				t.Fatal(err)
			}
		case "expired":
			now = now.Add(8 * 24 * time.Hour)
			if len(s.ActivateInvitations(outcome)) != 0 {
				t.Fatal("expired invitation activated")
			}
		}
		if got := s.invitations[invite.ID].Status; got != outcome {
			t.Fatalf("got %s want %s", got, outcome)
		}
	}
}

func TestRestrictedTeamInvitationLifecycle(t *testing.T) {
	if os.Getenv("KREDIT_INTEGRATION") != "1" {
		t.Skip("isolated database required")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	app, err := db.OpenAsRole(ctx, os.Getenv("APP_DATABASE_URL"), "kredit_app")
	if err != nil {
		t.Fatal(err)
	}
	defer app.Close()
	createUser := func(label string) string {
		t.Helper()
		var id string
		if err := pool.QueryRow(ctx, `INSERT INTO app.users(normalized_email) VALUES($1) RETURNING id::text`, fmt.Sprintf("team-%s-%d@example.test", label, time.Now().UnixNano())).Scan(&id); err != nil {
			t.Fatal(err)
		}
		return id
	}
	owner := createUser("owner")
	s := NewPostgresStore(app.Raw(), "isolated-team-test")
	org, _, err := s.Create(owner, CreateInput{LegalName: "Team lifecycle", BusinessType: "limited_company", BusinessAddress: "Lagos", Industry: "retail"})
	if err != nil {
		t.Fatal(err)
	}
	for _, outcome := range []string{"accepted", "revoked", "expired"} {
		user := createUser(outcome)
		invite, m, err := s.Invite(owner, org.ID, outcome+"@example.test", "email", user, access.RoleViewer)
		if err != nil {
			t.Fatal(err)
		}
		switch outcome {
		case "accepted":
			if got := s.ActivateInvitations(user); len(got) != 1 {
				t.Fatalf("activation failed: %+v", got)
			}
		case "revoked":
			if _, err = s.ChangeStatus(org.ID, owner, user, "removed"); err != nil {
				t.Fatal(err)
			}
		case "expired":
			if _, err = pool.Exec(ctx, `UPDATE app.memberships SET invited_at=now()-interval '8 days' WHERE id=$1`, m.ID); err != nil {
				t.Fatal(err)
			}
			if _, err = pool.Exec(ctx, `UPDATE app.organization_invitations SET expires_at=now()-interval '1 day' WHERE id=$1`, invite.ID); err != nil {
				t.Fatal(err)
			}
			if len(s.ActivateInvitations(user)) != 0 {
				t.Fatal("expired invitation activated")
			}
		}
		var status, linked string
		var accepted bool
		if err = pool.QueryRow(ctx, `SELECT status,membership_id::text,accepted_at IS NOT NULL FROM app.organization_invitations WHERE id=$1`, invite.ID).Scan(&status, &linked, &accepted); err != nil {
			t.Fatal(err)
		}
		if status != outcome || linked != m.ID || accepted != (outcome == "accepted") {
			t.Fatalf("incorrect lifecycle: %s %s %v", status, linked, accepted)
		}
		if len(s.ActivateInvitations(user)) != 0 {
			t.Fatal("invitation activated twice")
		}
	}
}
