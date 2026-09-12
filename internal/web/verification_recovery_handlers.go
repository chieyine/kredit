package web

import (
	"context"
	"kredit/internal/access"
	"kredit/internal/buyers"
	"net/http"
)

type verificationRecovery interface {
	ListVerificationAttempts(context.Context, string) ([]buyers.VerificationAttempt, error)
	ResolveVerificationAttempt(context.Context, string, string, string, string, string) error
}

func (s *Server) listVerificationRequests(w http.ResponseWriter, r *http.Request) {
	_, user, _, ok := s.requirePlatformAccess(w, r, access.PermissionProviderOperations)
	if !ok {
		return
	}
	store, ok := s.runtime.Buyers.(verificationRecovery)
	if !ok {
		writeProblem(w, 503, "recovery_unavailable", "Verification recovery is unavailable.")
		return
	}
	items, err := store.ListVerificationAttempts(r.Context(), user.ID)
	if err != nil {
		writeProblem(w, 503, "recovery_unavailable", "Verification requests could not be loaded.")
		return
	}
	writeJSON(w, 200, map[string]any{"registrations": items})
}
func (s *Server) resolveVerificationRequest(w http.ResponseWriter, r *http.Request) {
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
	store, ok := s.runtime.Buyers.(verificationRecovery)
	if !ok {
		writeProblem(w, 503, "recovery_unavailable", "Verification recovery is unavailable.")
		return
	}
	if err := store.ResolveVerificationAttempt(r.Context(), user.ID, r.PathValue("attemptID"), input.Action, input.Reference, input.Reason); err != nil {
		writeProblem(w, 409, "recovery_unconfirmed", err.Error())
		return
	}
	writeJSON(w, 200, map[string]bool{"resolved": true})
}
