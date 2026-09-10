package web

import (
	"context"
	"net/http/httptest"
	"testing"

	"kredit/internal/reports"
	"kredit/internal/usercontrol"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestOptionalActivityHonorsActorRestrictionAndReadFailure(t *testing.T) {
	ctx := context.Background()
	privacy := usercontrol.NewStore("test-secret")
	events := reports.NewStore(reports.Source{})
	server := &Server{runtime: &Runtime{UserControl: privacy, Reports: events}}
	r, err := privacy.CreatePrivacyRequest(ctx, "restricted-actor", "", "OBJECTION", "Stop optional processing")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := privacy.DecidePrivacy(ctx, r.ID, "reviewer", "APPROVED", "Verified optional processing objection", r.Version); err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest("GET", "/", nil)
	server.trackOptionalActivity(request, "restricted-actor", "report.fees.viewed", "shared-business", "product_improvement", nil)
	server.trackOptionalActivity(request, "other-actor", "report.fees.viewed", "shared-business", "product_improvement", nil)
	pool, err := pgxpool.New(ctx, "postgres://unused@127.0.0.1:1/unused?sslmode=disable")
	if err != nil {
		t.Fatal(err)
	}
	pool.Close()
	server.runtime.UserControl = usercontrol.NewPostgresStore(pool, "test-secret")
	server.trackOptionalActivity(request, "other-actor", "history.viewed", "other-actor", "product_improvement", nil)
	server.runtime.UserControl = nil
	server.trackOptionalActivity(request, "other-actor", "history.viewed", "other-actor", "product_improvement", nil)
	stored, err := events.ListAnalytics()
	if err != nil || len(stored) != 1 || stored[0].Name != "report.fees.viewed" {
		t.Fatalf("only unrestricted actor activity should be recorded: %#v, %v", stored, err)
	}
}
