package web

import (
	"kredit/internal/access"
	"kredit/internal/identity"
	"net/http"
)

func (s *Server) listNativeIdentity(w http.ResponseWriter, r *http.Request) {
	_, user, ok := s.requireAuth(w, r)
	if !ok {
		return
	}
	review := r.URL.Query().Get("review") == "true"
	if review {
		if _, _, _, ok = s.requirePlatformAccess(w, r, access.PermissionReviewCompliance); !ok {
			return
		}
	}
	items := []identity.NativeCase{}
	for _, provider := range identity.NativeAccounts(s.runtime.Identity) {
		cases, err := provider.List(identity.WithActor(r.Context(), user.ID), review)
		if err != nil {
			writeProblem(w, 503, "verification_unavailable", "Saved verification checks could not be loaded.")
			return
		}
		items = append(items, cases...)
	}
	writeJSON(w, 200, map[string]any{"cases": items, "consent_version": identity.NativeConsentVersion})
}
func (s *Server) actNativeIdentity(w http.ResponseWriter, r *http.Request) {
	session, user, ok := s.requireAuth(w, r)
	if !ok || !s.requireCSRF(w, r) {
		return
	}
	var input identity.NativeAction
	if !decodeJSONRequest(w, r, &input) {
		return
	}
	if input.Action == "approve" || input.Action == "reject" || input.Action == "retry" {
		if _, _, _, ok = s.requirePlatformAccess(w, r, access.PermissionReviewCompliance); !ok || !s.requireFreshMFA(w, session) {
			return
		}
	}
	selected, err := identity.Resolve(s.runtime.Identity, r.PathValue("provider"))
	if err != nil {
		writeProblem(w, 409, "verification_account_unavailable", err.Error())
		return
	}
	native, ok := selected.(*identity.NativeLookup)
	if !ok {
		writeProblem(w, 409, "verification_action_unavailable", "This account uses its provider's verification flow.")
		return
	}
	if err = native.Act(identity.WithActor(r.Context(), user.ID), r.PathValue("caseID"), input); err != nil {
		writeProblem(w, 409, "verification_incomplete", err.Error())
		return
	}
	writeJSON(w, 200, map[string]bool{"saved": true})
}
