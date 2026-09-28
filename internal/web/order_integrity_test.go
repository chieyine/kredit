package web

import (
	"context"
	"errors"
	"io"
	"kredit/internal/config"
	"kredit/internal/orders"
	"log/slog"
	"net/http/httptest"
	"strings"
	"testing"
)

func auditMemoryServer(t *testing.T) *Server {
	t.Helper()
	t.Setenv("APP_ENV", "development")
	t.Setenv("DATABASE_URL", "")
	t.Setenv("DATABASE_DIRECT_URL", "")
	cfg, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	return NewServer(cfg, slog.New(slog.NewTextHandler(io.Discard, nil)))
}
func TestAuditServicesPersistAcrossRequestsWithoutCrossServerState(t *testing.T) {
	s := auditMemoryServer(t)
	ctx := context.Background()
	first := s.getOrdersService()
	if err := first.CreateLineItems(ctx, "o", []orders.LineItem{{Description: "carton", Quantity: 1, UnitPriceKobo: 100}}); err != nil {
		t.Fatal(err)
	}
	items, err := s.getOrdersService().ListLineItems(ctx, "o")
	if err != nil || len(items) != 1 {
		t.Fatalf("memory reset across request: %+v %v", items, err)
	}
	other := auditMemoryServer(t)
	items, err = other.getOrdersService().ListLineItems(ctx, "o")
	if err != nil || len(items) != 0 {
		t.Fatal("unrelated servers share memory state")
	}
}

func TestAuditOrderOutageReturnsUnavailableWithoutDatabaseDetails(t *testing.T) {
	s := auditMemoryServer(t)
	r := httptest.NewRequest("GET", "/orders", nil)
	w := httptest.NewRecorder()
	s.writeOrderProblem(w, r, errors.New("postgres password secret and private table diagnostics"))
	if w.Code != 503 || strings.Contains(w.Body.String(), "password") || strings.Contains(w.Body.String(), "private table") {
		t.Fatalf("outage hidden or leaked: %d %s", w.Code, w.Body.String())
	}
}
