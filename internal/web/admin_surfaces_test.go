package web

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"kredit/internal/config"
)

// The probe paths below are deliberately not routes the product registers.
// Re-registering a live pattern on the same ServeMux panics, and the point here
// is the reachability middleware rather than any particular handler.
const (
	enabledProbeSurface  = "surface-under-test"
	disabledProbeSurface = "surface-not-enabled"
)

func adminSurfaceServer(t *testing.T, surfaces []string) http.Handler {
	t.Helper()
	cfg := config.Config{Environment: "development", Version: "test", APIListenAddr: ":0", Currency: "NGN", MoneyUnit: "kobo", CollectionProvider: "mock", AdminSurfaces: surfaces}
	server := NewServer(cfg, slog.New(slog.NewTextHandler(io.Discard, nil)))
	reached := func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"reached": "yes"})
	}
	server.mux.HandleFunc("GET /api/v1/ops/"+enabledProbeSurface, reached)
	server.mux.HandleFunc("GET /api/v1/ops/"+disabledProbeSurface, reached)
	server.mux.HandleFunc("GET /api/v1/ops/"+disabledProbeSurface+"/{itemID}/detail", reached)
	server.mux.HandleFunc("GET /api/v1/surface-probe", reached)
	return server.Handler()
}

func statusForPath(handler http.Handler, path string) int {
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
	return response.Code
}

func TestUnenabledAdminSurfacesAreUnreachable(t *testing.T) {
	handler := adminSurfaceServer(t, []string{enabledProbeSurface})

	if code := statusForPath(handler, "/api/v1/ops/"+enabledProbeSurface); code != http.StatusOK {
		t.Fatalf("an enabled surface must stay reachable, got %d", code)
	}
	// A caller with no reason to know the surface exists is not told that it
	// does, so this is 404 rather than 403.
	if code := statusForPath(handler, "/api/v1/ops/"+disabledProbeSurface); code != http.StatusNotFound {
		t.Fatalf("a surface this deployment does not operate must be unreachable, got %d", code)
	}
	// Sub-paths belong to the same surface and follow it.
	if code := statusForPath(handler, "/api/v1/ops/"+disabledProbeSurface+"/item-1/detail"); code != http.StatusNotFound {
		t.Fatalf("a disabled surface must not be reachable through a sub-path, got %d", code)
	}
	// Nothing outside the operations namespace is affected.
	if code := statusForPath(handler, "/api/v1/surface-probe"); code != http.StatusOK {
		t.Fatalf("non-operations routes must be untouched, got %d", code)
	}
}

func TestAdminSurfacesAllowListSupportsExplicitAll(t *testing.T) {
	handler := adminSurfaceServer(t, []string{"all"})
	if code := statusForPath(handler, "/api/v1/ops/"+disabledProbeSurface); code != http.StatusOK {
		t.Fatalf("an explicit 'all' must enable every surface, got %d", code)
	}
}

// An unset list means the surface set was never enumerated. Production refuses
// that in configuration validation; everywhere else it must keep the full
// surface available so development and the test suites are unaffected.
func TestUnsetAdminSurfacesKeepsEveryOperationsRouteAvailable(t *testing.T) {
	handler := adminSurfaceServer(t, nil)
	if code := statusForPath(handler, "/api/v1/ops/"+disabledProbeSurface); code != http.StatusOK {
		t.Fatalf("an unset allow-list must not disable surfaces, got %d", code)
	}
}

func TestAdminSurfaceIsTheFirstSegmentAfterTheOperationsPrefix(t *testing.T) {
	cases := map[string]string{
		"/api/v1/ops/team":                     "team",
		"/api/v1/ops/team/user-1/roles":        "team",
		"/api/v1/ops/analytics/scorecard":      "analytics",
		"/api/v1/ops/account-recovery":         "account-recovery",
		"/api/v1/organizations/org-1/payments": "",
		"/api/v1/me":                           "",
	}
	for path, expected := range cases {
		if surface := adminSurface(path); surface != expected {
			t.Fatalf("adminSurface(%q)=%q, want %q", path, surface, expected)
		}
	}
}
