package web

import (
	"net/http"

	"kredit/internal/audit"
	"kredit/internal/credit"
	"kredit/internal/mandates"
	"kredit/internal/notifications"
)

type mandateCancellationInput struct {
	Reason string `json:"reason"`
}

func (s *Server) cancelBuyerMandate(w http.ResponseWriter, r *http.Request) {
	_, user, ok := s.requireAuth(w, r)
	if !ok {
		return
	}
	if !s.requireCSRF(w, r) {
		return
	}
	mandateID, _ := pathID(r, "mandateID")
	var input mandateCancellationInput
	if err := decodeJSON(w, r, &input); err != nil {
		writeProblem(w, 400, "invalid_request", err.Error())
		return
	}
	current, views, found := s.findBuyerMandate(user.ID, mandateID)
	if !found {
		writeProblem(w, 404, "mandate_not_found", "Mandate was not found")
		return
	}
	cancelled, err := s.runtime.Mandates.CancelMandate(r.Context(), current.ProviderID, input.Reason)
	if err != nil {
		writeProblem(w, 409, "mandate_cancellation_failed", err.Error())
		return
	}
	s.applyMandateToBuyerResources(user.ID, current, cancelled, views)
	for _, view := range views {
		s.runtime.Audit.Append(audit.Event{ActorUserID: user.ID, OrganizationID: view.Request.SupplierOrganizationID, Action: "mandate.cancelled", ResourceType: "payment_mandate", ResourceID: cancelled.ID, Outcome: "success"})
		_, _ = s.runtime.Notifications.Emit(r.Context(), notifications.Event{ID: "mandate-cancelled:" + cancelled.ID + ":" + view.Request.SupplierOrganizationID, Type: "MandateCancelled", RecipientID: view.Request.SupplierOrganizationID, Priority: notifications.PriorityCritical, AmountKobo: int64(cancelled.AmountCeiling), Currency: "NGN", Reference: cancelled.ID, NextAction: "Review outstanding credit and suspend new release", SecurePath: "/app/collections"})
	}
	writeJSON(w, 200, map[string]any{"mandate": cancelled, "affected_credit_requests": len(views)})
}

func (s *Server) restoreBuyerMandate(w http.ResponseWriter, r *http.Request) {
	_, user, ok := s.requireAuth(w, r)
	if !ok {
		return
	}
	if !s.requireCSRF(w, r) {
		return
	}
	mandateID, _ := pathID(r, "mandateID")
	current, views, found := s.findBuyerMandate(user.ID, mandateID)
	if !found {
		writeProblem(w, 404, "mandate_not_found", "Mandate was not found")
		return
	}
	restored, err := s.runtime.Mandates.RestoreAuthorization(r.Context(), current.ProviderID)
	if err != nil {
		writeProblem(w, 409, "mandate_restore_failed", err.Error())
		return
	}
	s.applyMandateToBuyerResources(user.ID, current, restored, views)
	for _, view := range views {
		s.runtime.Audit.Append(audit.Event{ActorUserID: user.ID, OrganizationID: view.Request.SupplierOrganizationID, Action: "mandate.restored", ResourceType: "payment_mandate", ResourceID: restored.ID, Outcome: "success"})
	}
	writeJSON(w, 201, map[string]any{"mandate": restored, "affected_credit_requests": len(views)})
}

func (s *Server) findBuyerMandate(userID, id string) (mandates.Mandate, []credit.View, bool) {
	views := s.runtime.Credit.ListForBuyer(userID)
	matching := []credit.View{}
	var current mandates.Mandate
	for _, view := range views {
		if view.Mandate == nil || (view.Mandate.ID != id && view.Mandate.ProviderID != id) {
			continue
		}
		if current.ID == "" {
			current = *view.Mandate
		}
		matching = append(matching, view)
	}
	return current, matching, current.ID != ""
}

func (s *Server) applyMandateToBuyerResources(userID string, previous, next mandates.Mandate, views []credit.View) {
	for _, view := range views {
		_, _ = s.runtime.Credit.SetMandate(view.Request.ID, userID, next)
	}
	for _, line := range s.runtime.TradeLines.ListForBuyer(userID) {
		if line.MandateID != previous.ID && line.MandateID != previous.ProviderID {
			continue
		}
		_, _ = s.runtime.TradeLines.SetMandateState(line.ID, next.ID, next.Status == mandates.Active)
	}
}
