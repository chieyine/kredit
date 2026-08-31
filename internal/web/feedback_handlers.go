package web

import (
	"net/http"

	"kredit/internal/audit"
	"kredit/internal/feedback"
)

func (s *Server) submitProductFeedback(w http.ResponseWriter, r *http.Request) {
	_, user, ok := s.requireAuth(w, r)
	if !ok || !s.requireCSRF(w, r) {
		return
	}
	var input feedback.Input
	if !decodeJSONRequest(w, r, &input) {
		return
	}
	input.UserID = user.ID
	if input.Area == "seller" {
		membership, found := s.runtime.Organizations.Membership(input.OrganizationID, user.ID)
		if !found || membership.Status == "removed" || membership.Status == "suspended" {
			writeProblem(w, http.StatusForbidden, "organization_forbidden", "you do not have access to this business")
			return
		}
	}
	entry, err := s.runtime.Feedback.Submit(r.Context(), input)
	if err != nil {
		writeProblem(w, http.StatusBadRequest, "invalid_feedback", err.Error())
		return
	}
	s.runtime.Audit.Append(audit.Event{ActorUserID: user.ID, OrganizationID: input.OrganizationID, Action: "product.feedback_submitted", ResourceType: "product_feedback", Outcome: "success", Severity: "notice", RequestID: requestIDFromContext(r.Context()), Metadata: map[string]string{"area": entry.Area, "screen": entry.Screen}})
	writeJSON(w, http.StatusCreated, map[string]any{"feedback": entry, "message": "Thank you. Your answer will help us make Kredit easier to use."})
}
