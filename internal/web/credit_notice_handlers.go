package web

import (
	"kredit/internal/access"
	"kredit/internal/whatsapp"
	"net/http"
)

func (s *Server) listOperationActions(w http.ResponseWriter, r *http.Request) {
	orgID, _ := pathID(r, "organizationID")
	if _, _, _, ok := s.requireOrganizationAccess(w, r, orgID, access.PermissionReadAudit); !ok {
		return
	}
	actions, err := s.runtime.Operations.ListForOrganization(r.Context(), orgID)
	if err != nil {
		writeProblem(w, http.StatusServiceUnavailable, "operation_history_unavailable", "We could not load your business activity. Please try again.")
		return
	}
	writeJSON(w, 200, map[string]any{"actions": actions})
}

func (s *Server) whatsappWebhook(w http.ResponseWriter, r *http.Request) {
	var event whatsapp.Event
	if err := decodeJSON(w, r, &event); err != nil {
		writeProblem(w, 400, "invalid_webhook", err.Error())
		return
	}
	command, err := s.runtime.WhatsApp.Handle(r.Context(), event)
	if err != nil {
		writeProblem(w, 401, "whatsapp_webhook_rejected", err.Error())
		return
	}
	response := map[string]any{"command": command}
	if command.RequiresConfirmation {
		response["confirmation_summary"] = whatsapp.ConfirmationSummary(command)
		response["secure_action_required"] = true
	}
	writeJSON(w, 200, response)
}
func (s *Server) myNotifications(w http.ResponseWriter, r *http.Request) {
	_, user, ok := s.requireAuth(w, r)
	if !ok {
		return
	}
	writeJSON(w, 200, map[string]any{"notifications": s.runtime.Notifications.ListDeliveries(user.ID)})
}

// Credit and release holds apply to both parties. Reporting payments, declining
// offers and recording delivery problems remain available while a hold is active.
