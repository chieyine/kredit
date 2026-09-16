package web

import (
	"context"
	"kredit/internal/access"
	"kredit/internal/audit"
	"kredit/internal/mandates"
	"kredit/internal/providers/bankdebit"
	"net/http"
	"strings"
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
	if s.runtime.BankEnrollments != nil {
		native, err := s.runtime.BankEnrollments.Pending(r.Context(), user.ID)
		if err != nil {
			writeProblem(w, 503, "recovery_unavailable", "Bank authorization recovery could not be loaded.")
			return
		}
		items = append(items, native...)
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
	if strings.HasPrefix(r.PathValue("attemptID"), "native:") {
		parts := strings.SplitN(r.PathValue("attemptID"), ":", 3)
		if len(parts) != 3 {
			writeProblem(w, 400, "invalid_request", "Invalid authorization reference.")
			return
		}
		provider, ok := s.runtime.NativeBankAccounts[parts[1]].(bankdebit.Recoverer)
		if !ok {
			writeProblem(w, 503, "provider_unavailable", "Restore the original bank provider connection.")
			return
		}
		if err := provider.Recover(r.Context(), user.ID, parts[2], input.Action, input.Reference, input.Reason); err != nil {
			writeProblem(w, 409, "recovery_unconfirmed", err.Error())
			return
		}
		s.runtime.Audit.Append(audit.Event{ActorUserID: user.ID, Action: "bank_authorization.recovered", ResourceType: "bank_authorization", ResourceID: parts[2], Outcome: "success", Metadata: map[string]string{"provider": parts[1], "action": input.Action}})
		writeJSON(w, 200, map[string]bool{"resolved": true})
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
