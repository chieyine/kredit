package web

import (
	"kredit/internal/access"
	"net/http"
)

func (s *Server) financialReviews(w http.ResponseWriter, r *http.Request) {
	if _, _, _, ok := s.requirePlatformAccess(w, r, access.PermissionProviderOperations); !ok {
		return
	}
	if s.runtime.PlatformOps == nil {
		writeProblem(w, 503, "reconciliation_unavailable", "Reconciliation persistence is required")
		return
	}
	data, err := s.runtime.PlatformOps.FinancialReviews(r.Context())
	if err != nil {
		writeProblem(w, 503, "reconciliation_unavailable", "Reconciliation cases could not be loaded")
		return
	}
	writeJSON(w, 200, map[string]any{"cases": data})
}
func (s *Server) decideFinancialReview(w http.ResponseWriter, r *http.Request) {
	session, user, _, ok := s.requirePlatformAccess(w, r, access.PermissionProviderOperations)
	if !ok {
		return
	}
	if !s.requireFreshMFA(w, session) || !s.requireCSRF(w, r) {
		return
	}
	if s.runtime.PlatformOps == nil {
		writeProblem(w, 503, "reconciliation_unavailable", "Reconciliation persistence is required")
		return
	}
	id, err := pathID(r, "caseID")
	if err != nil {
		writeProblem(w, 400, "invalid_path", "Invalid case identifier")
		return
	}
	var in struct {
		Action string `json:"action"`
		Reason string `json:"reason"`
	}
	if err = decodeJSON(w, r, &in); err != nil {
		writeProblem(w, 400, "invalid_request", err.Error())
		return
	}
	if err = s.runtime.PlatformOps.DecideFinancialReview(r.Context(), id, user.ID, in.Action, in.Reason); err != nil {
		writeProblem(w, 409, "review_conflict", err.Error())
		return
	}
	writeJSON(w, 200, map[string]any{"status": "applied"})
}
