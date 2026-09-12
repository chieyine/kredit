package web

import (
	"context"
	"kredit/internal/access"
	"kredit/internal/mandates"
	"net/http"
)

type authorizationRecovery interface {
	ListAuthorizationAttempts(context.Context, string) ([]mandates.AuthorizationAttempt, error)
	ResolveAuthorizationAttempt(context.Context, string, string, string, string, string) error
}

func (s *Server) listMandateAuthorizations(w http.ResponseWriter, r *http.Request) {
	_, user, _, ok := s.requirePlatformAccess(w, r, access.PermissionProviderOperations)
	if !ok {
		return
	}
	store, ok := s.runtime.Mandates.(authorizationRecovery)
	if !ok {
		writeProblem(w, 503, "recovery_unavailable", "Authorization recovery is unavailable.")
		return
	}
	items, err := store.ListAuthorizationAttempts(r.Context(), user.ID)
	if err != nil {
		writeProblem(w, 503, "recovery_unavailable", "Authorization requests could not be loaded.")
		return
	}
	writeJSON(w, 200, map[string]any{"registrations": items})
}
func (s *Server) resolveMandateAuthorization(w http.ResponseWriter, r *http.Request) {
	session, user, _, ok := s.requirePlatformAccess(w, r, access.PermissionProviderOperations)
	if !ok || !s.requireFreshMFA(w, session) || !s.requireCSRF(w, r) {
		return
	}
	var input struct {
		Action    string `json:"action"`
		Reference string `json:"provider_reference"`
		Reason    string `json:"reason"`
	}
	if !decodeJSONRequest(w, r, &input) {
		return
	}
	store, ok := s.runtime.Mandates.(authorizationRecovery)
	if !ok {
		writeProblem(w, 503, "recovery_unavailable", "Authorization recovery is unavailable.")
		return
	}
	if err := store.ResolveAuthorizationAttempt(r.Context(), user.ID, r.PathValue("attemptID"), input.Action, input.Reference, input.Reason); err != nil {
		writeProblem(w, 409, "recovery_unconfirmed", err.Error())
		return
	}
	writeJSON(w, 200, map[string]bool{"resolved": true})
}
