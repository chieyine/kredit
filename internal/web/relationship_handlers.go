package web

import (
	"net/http"
	"sort"

	"kredit/internal/audit"
	"kredit/internal/relationships"
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
	consent, err := s.runtime.Relationships.Record(r.Context(), user.ID, input.SupplierOrganizationID, input.ConsentType, input.Version, input.EvidenceHash, input.Granted)
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
	consents, err := s.runtime.Relationships.List(r.Context(), user.ID)
	if err != nil {
		writeProblem(w, http.StatusServiceUnavailable, "consent_history_unavailable", "We could not load your seller permissions. Please try again.")
		return
	}
	suppliers, err := s.buyerPermissionSuppliers(r, user.ID, consents)
	if err != nil {
		writeProblem(w, http.StatusServiceUnavailable, "supplier_directory_unavailable", "Your seller permissions could not be fully loaded. Try again.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"consents": consents, "suppliers": suppliers})
}

func (s *Server) buyerPermissionSuppliers(r *http.Request, buyerID string, consents []relationships.Consent) ([]relationships.Supplier, error) {
	if source, ok := s.runtime.Relationships.(relationships.SupplierReader); ok {
		return source.Suppliers(r.Context(), buyerID)
	}
	// Development adapters assemble the same directory from owned local records.
	byID := map[string]relationships.Supplier{}
	views, err := s.runtime.readCreditForBuyer(r.Context(), buyerID)
	if err != nil {
		return nil, err
	}
	for _, view := range views {
		v := view.Request
		byID[v.SupplierOrganizationID] = relationships.Supplier{ID: v.SupplierOrganizationID, LegalName: v.SupplierLegalName, TradingName: v.SupplierTradingName}
	}
	lines, err := s.runtime.readTradeLinesForBuyer(r.Context(), buyerID)
	if err != nil {
		return nil, err
	}
	include := func(id string) {
		if _, ok := byID[id]; ok {
			return
		}
		org, ok := s.runtime.Organizations.Get(id)
		if ok {
			byID[id] = relationships.Supplier{ID: id, LegalName: org.LegalName, TradingName: org.TradingName}
		}
	}
	for _, line := range lines {
		include(line.SupplierOrganizationID)
	}
	for _, consent := range consents {
		include(consent.SupplierOrgID)
	}
	out := make([]relationships.Supplier, 0, len(byID))
	for _, supplier := range byID {
		out = append(out, supplier)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].LegalName == out[j].LegalName {
			return out[i].ID < out[j].ID
		}
		return out[i].LegalName < out[j].LegalName
	})
	return out, nil
}
