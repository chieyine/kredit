package web

import (
	"kredit/internal/access"
	"kredit/internal/consumer"
	"net/http"
)

func (s *Server) consumerSummary(w http.ResponseWriter, r *http.Request) {
	org, err := pathID(r, "organizationID")
	if err != nil {
		writeProblem(w, 400, "invalid_business", "Choose a valid business.")
		return
	}
	_, user, _, ok := s.requireOrganizationAccess(w, r, org, access.PermissionReadConsumerSales)
	if !ok {
		return
	}
	if s.runtime.Database == nil {
		writeProblem(w, 503, "summary_unavailable", "Personal sale balances are unavailable.")
		return
	}
	store := consumer.Store{Pool: s.runtime.Database.Raw()}
	items, err := store.Summaries(r.Context(), user.ID, org)
	if err != nil {
		writeProblem(w, 503, "summary_unavailable", "Personal sale balances could not be confirmed. Try again.")
		return
	}
	writeJSON(w, 200, map[string]any{"items": items})
}
