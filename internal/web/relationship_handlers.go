package web

import (
	"net/http"

	"kredit/internal/audit"
)

type consentRequest struct {
	SupplierOrganizationID string `json:"supplier_organization_id"`
	ConsentType            string `json:"consent_type"`
	Version                string `json:"version"`
	EvidenceHash           string `json:"evidence_hash"`
	Granted                bool   `json:"granted"`
}

func (s *Server) recordBuyerConsent(w http.ResponseWriter, r *http.Request) {
	_, user, ok := s.requireAuth(w, r)
	if !ok || !s.requireCSRF(w, r) {
		return
	}
	var input consentRequest
	if err := decodeJSON(w, r, &input); err != nil {
		writeProblem(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	consent, err := s.runtime.Relationships.Record(user.ID, input.SupplierOrganizationID, input.ConsentType, input.Version, input.EvidenceHash, input.Granted)
	if err != nil {
		writeProblem(w, http.StatusUnprocessableEntity, "consent_invalid", err.Error())
		return
	}
	s.runtime.Audit.Append(audit.Event{ActorUserID: user.ID, OrganizationID: input.SupplierOrganizationID, Action: "relationship.consent_recorded", ResourceType: "relationship_consent", ResourceID: consent.ID, Outcome: "success", RequestID: requestIDFromContext(r.Context())})
	writeJSON(w, http.StatusCreated, map[string]any{"consent": consent})
}

func (s *Server) listBuyerConsents(w http.ResponseWriter, r *http.Request) {
	_, user, ok := s.requireAuth(w, r)
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"consents": s.runtime.Relationships.List(user.ID)})
}
