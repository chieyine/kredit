package web

import (
	"net/http"
	"strings"
)

// Purchasing scope narrows an authenticated owner's records. A query parameter
// never grants authority; the profile must be present in the owned directory.
func (s *Server) buyerWorkspaceScope(w http.ResponseWriter, r *http.Request, userID string) (string, bool) {
	businessID := strings.TrimSpace(r.URL.Query().Get("business_id"))
	workspaceID := strings.TrimSpace(r.URL.Query().Get("organization"))
	if businessID == "" && workspaceID == "" {
		return "", true
	}
	profiles, err := s.runtime.Buyers.ListBusinessProfiles(r.Context(), userID)
	if err != nil {
		writeProblem(w, 503, "business_scope_unavailable", "Your purchasing business could not be checked.")
		return "", false
	}
	for _, profile := range profiles {
		if profile.OwnerUserID == userID && (businessID == "" || profile.ID == businessID) && (workspaceID == "" || profile.WorkspaceID == workspaceID) {
			return profile.ID, true
		}
	}
	writeProblem(w, 404, "business_not_found", "This purchasing business is not available to your account.")
	return "", false
}

func (s *Server) buyerWorkspaceObligations(w http.ResponseWriter, r *http.Request, userID, businessID string) (map[string]bool, bool) {
	result := map[string]bool{}
	if businessID == "" {
		return result, true
	}
	views, err := s.runtime.readCreditForBuyer(r.Context(), userID)
	if financialReadError(w, err) {
		return nil, false
	}
	for _, view := range views {
		if view.Request.BuyerBusinessID == businessID && view.Obligation != nil {
			result[view.Obligation.ID] = true
		}
	}
	return result, true
}

func purchasingRows[T any](items []T, keep func(T) bool) []T {
	result := make([]T, 0, len(items))
	for _, item := range items {
		if keep(item) {
			result = append(result, item)
		}
	}
	return result
}
