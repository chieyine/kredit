package web

import (
	"kredit/internal/access"
	"kredit/internal/billing"
	"net/http"
	"strings"
)

func (s *Server) feeAuthorizations(w http.ResponseWriter, r *http.Request) {
	org, err := pathID(r, "organizationID")
	if err != nil {
		writeProblem(w, 400, "invalid_business", "Invalid business reference.")
		return
	}
	admin := strings.HasPrefix(r.URL.Path, "/api/v1/ops/")
	var actor string
	if admin {
		session, user, _, ok := s.requirePlatformAccess(w, r, access.PermissionPlatformOwner)
		if !ok {
			return
		}
		actor = user.ID
		if r.Method != "GET" && (!s.requireFreshMFA(w, session) || !s.requireCSRF(w, r)) {
			return
		}
	} else {
		session, user, _, ok := s.requireOrganizationAccess(w, r, org, access.PermissionManageFinancial)
		if !ok {
			return
		}
		actor = user.ID
		if r.Method != "GET" && (!s.requireFreshMFA(w, session) || !s.requireCSRF(w, r)) {
			return
		}
	}
	service := s.runtime.FeeBilling
	if service == nil {
		writeProblem(w, 503, "billing_unavailable", "Fee bank setup is unavailable.")
		return
	}
	if r.Method == "GET" {
		items, err := service.List(r.Context(), org, actor)
		if err != nil {
			writeProblem(w, 503, "billing_unavailable", "Fee permissions could not be loaded.")
			return
		}
		writeJSON(w, 200, map[string]any{"authorizations": items, "available": service.Active != ""})
		return
	}
	var in struct {
		Action string `json:"action"`
		ID     string `json:"id"`
		billing.FeeCustomer
		Ceiling   int64  `json:"ceiling_kobo"`
		Consent   string `json:"consent_version"`
		Reference string `json:"reference"`
		Evidence  string `json:"evidence"`
	}
	if !decodeJSONRequest(w, r, &in) {
		return
	}
	var a billing.SavedAuthorization
	if admin {
		a, err = service.Review(r.Context(), org, actor, in.ID, in.Action, in.Reference, in.Evidence)
	} else {
		switch in.Action {
		case "start":
			a, err = service.Start(r.Context(), org, actor, in.FeeCustomer, in.Ceiling, in.Consent)
		case "authorize":
			a, err = service.Authorize(r.Context(), org, actor, in.ID)
		case "resume":
			err = service.Resume(r.Context(), org, actor, in.ID)
		case "pause":
			err = service.Pause(r.Context(), org, actor, in.ID)
		default:
			writeProblem(w, 400, "invalid_action", "Choose a supported fee setup action.")
			return
		}
	}
	if err != nil {
		writeProblem(w, 409, "fee_setup_unconfirmed", err.Error())
		return
	}
	writeJSON(w, 200, map[string]any{"authorization": a, "saved": true})
}
