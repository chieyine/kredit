package buyers

import (
	"context"
	"fmt"
	"kredit/internal/identity"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPostgresInvitationHashAndSecondSupplierReuse(t *testing.T) {
	if os.Getenv("KREDIT_INTEGRATION") != "1" {
		t.Skip("integration database required")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	var buyer, owner string
	for n, target := range []*string{&buyer, &owner} {
		if err = pool.QueryRow(ctx, `INSERT INTO app.users(normalized_email) VALUES($1) RETURNING id::text`, fmt.Sprintf("audit-invite-%d-%d@example.test", time.Now().UnixNano(), n)).Scan(target); err != nil {
			t.Fatal(err)
		}
	}
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
	s := NewPostgresStore(runtimePool, "audit-invitation", identity.NewMockProvider())
	var count int64
	if err = pool.QueryRow(ctx, `SELECT app.business_count()`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	s.SetBusinessLimit(count + 1)
	var first Portal
	for i := 0; i < 2; i++ {
		var org string
		if err = pool.QueryRow(ctx, `INSERT INTO app.organizations(legal_name,business_type,business_address,industry) VALUES('Audit supplier','limited_company','Lagos','retail') RETURNING id::text`).Scan(&org); err != nil {
			t.Fatal(err)
		}
		invitation, err := s.CreateInvitation(owner, org, CreateInvitationInput{Target: "buyer@example.test", TargetType: "email", LegalName: "Buyer Ltd", BusinessType: "limited_company", BusinessAddress: "Lagos", Industry: "retail"})
		if err != nil {
			t.Fatal(err)
		}
		if _, err = s.Preview(invitation.RawToken); err != nil {
			t.Fatal(err)
		}
		portal, err := s.Accept(ctx, invitation.RawToken, buyer, AcceptInput{FullName: "Buyer Name"})
		if err != nil {
			t.Fatal(err)
		}
		if i == 0 {
			first = portal
		} else if portal.Person.ID != first.Person.ID || portal.Business.ID != first.Business.ID || portal.Representative.ID != first.Representative.ID {
			t.Fatal("duplicate identity on second invitation")
		}
		replay, err := s.Accept(ctx, invitation.RawToken, buyer, AcceptInput{FullName: "Buyer Name"})
		if err != nil || replay.Business.ID != portal.Business.ID {
			t.Fatalf("same-buyer acceptance recovery failed: %v", err)
		}
		if _, err = s.Accept(ctx, invitation.RawToken, owner, AcceptInput{FullName: "Other person"}); err == nil {
			t.Fatal("another user replayed accepted invitation")
		}
		if i == 1 {
			next, err := s.CreateInvitation(owner, org, CreateInvitationInput{Target: "buyer@example.test", TargetType: "email", LegalName: "Another Buyer Ltd", BusinessType: "limited_company", BusinessAddress: "Lagos", Industry: "retail"})
			if err != nil {
				t.Fatal(err)
			}
			if _, err = s.Accept(ctx, next.RawToken, buyer, AcceptInput{FullName: "Buyer Name"}); err == nil {
				t.Fatal("new business exceeded the persisted limit")
			}
		}
	}
}
