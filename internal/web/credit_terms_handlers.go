package web

import (
	"kredit/internal/access"
	"kredit/internal/credit"
	"net/http"
)

func (s *Server) previewCreditTerms(w http.ResponseWriter, r *http.Request) {
	organizationID, err := pathID(r, "organizationID")
	if err != nil {
		writeProblem(w, 400, "invalid_path", "Choose a business first.")
		return
	}
	if _, _, _, ok := s.requireOrganizationAccess(w, r, organizationID, access.PermissionCreateCredit); !ok {
		return
	}
	if !s.requireCSRF(w, r) {
		return
	}
	var input struct {
		DueDate         string `json:"due_date"`
		GraceHours      int    `json:"grace_hours"`
		CollectionLocal string `json:"collection_local,omitempty"`
	}
	if err := decodeJSON(w, r, &input); err != nil {
		writeProblem(w, 400, "invalid_request", "Check the payment date and extra hours.")
		return
	}
	instant, err := credit.CollectionInstant(input.DueDate, input.GraceHours)
	mode := "lagos_end_of_day"
	if err == nil && input.CollectionLocal != "" {
		earliest := instant
		instant, err = credit.ExplicitCollectionInstant(input.CollectionLocal)
		mode = "lagos_explicit"
		if err == nil && instant.Before(earliest) {
			writeProblem(w, 422, "credit_terms_invalid", "Bank collection must be after the agreed payment day and extra hours.")
			return
		}
	}
	if err != nil {
		writeProblem(w, 422, "credit_terms_invalid", "Check the payment date and extra hours.")
		return
	}
	writeJSON(w, 200, map[string]any{"due_date": input.DueDate, "grace_hours": input.GraceHours, "collection_at": instant, "timezone": "Africa/Lagos", "cutoff": "23:59", "timing_mode": mode})
}
