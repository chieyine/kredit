package web

import (
	"kredit/internal/access"
	"kredit/internal/settlement"
	"net/http"
)

func (s *Server) sellerSettlements(w http.ResponseWriter, r *http.Request) {
	org := r.PathValue("organizationID")
	var actor string
	if org == "" {
		_, user, _, ok := s.requirePlatformAccess(w, r, access.PermissionPlatformOwner)
		if !ok {
			return
		}
		actor = user.ID
	} else {
		_, user, _, ok := s.requireOrganizationAccess(w, r, org, access.PermissionReadFinancial)
		if !ok {
			return
		}
		actor = user.ID
	}
	if s.runtime.Database == nil {
		writeProblem(w, 503, "settlements_unavailable", "Settlement records are unavailable.")
		return
	}
	items, err := settlement.NewReceiptStore(s.runtime.Database.Raw()).List(r.Context(), actor, org)
	if err != nil {
		writeProblem(w, 503, "settlements_unavailable", "Settlement records could not be loaded.")
		return
	}
	writeJSON(w, 200, map[string]any{"settlements": items})
}
func (s *Server) recordSellerSettlement(w http.ResponseWriter, r *http.Request) {
	session, user, _, ok := s.requirePlatformAccess(w, r, access.PermissionPlatformOwner)
	if !ok || !s.requireFreshMFA(w, session) || !s.requireCSRF(w, r) {
		return
	}
	if s.runtime.Database == nil {
		writeProblem(w, 503, "settlements_unavailable", "Settlement records are unavailable.")
		return
	}
	var in settlement.BankReceipt
	if !decodeJSONRequest(w, r, &in) {
		return
	}
	item, err := settlement.NewReceiptStore(s.runtime.Database.Raw()).Record(r.Context(), user.ID, r.PathValue("organizationID"), r.PathValue("attemptID"), in)
	if err != nil {
		writeProblem(w, 409, "settlement_unconfirmed", err.Error())
		return
	}
	writeJSON(w, 200, map[string]any{"settlement": item})
}
