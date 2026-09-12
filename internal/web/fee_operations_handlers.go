package web

import (
	"kredit/internal/access"
	"kredit/internal/billing"
	"net/http"
	"strings"
)

func (s *Server) feeOperations(w http.ResponseWriter, r *http.Request) {
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
		_, user, _, ok := s.requireOrganizationAccess(w, r, org, access.PermissionReadFinancial)
		if !ok {
			return
		}
		actor = user.ID
	}
	service := s.runtime.FeeBilling
	if service == nil {
		writeProblem(w, 503, "billing_unavailable", "Fee reconciliation is unavailable.")
		return
	}
	if r.Method == "GET" {
		banks, debits, err := service.Operations(r.Context(), org, actor)
		if err != nil {
			writeProblem(w, 503, "billing_unavailable", "Fee reconciliation could not be loaded.")
			return
		}
		writeJSON(w, 200, map[string]any{"banks": banks, "debits": debits})
		return
	}
	var in struct {
		Action   string `json:"action"`
		ID       string `json:"id"`
		Provider string `json:"provider"`
		billing.Receipt
	}
	if !admin {
		writeProblem(w, 403, "owner_required", "Super-admin authority is required.")
		return
	}
	if !decodeJSONRequest(w, r, &in) {
		return
	}
	if in.Action == "bank" {
		err = service.BankReceipt(r.Context(), org, actor, in.Provider, in.Receipt)
	} else {
		err = service.ReviewDebit(r.Context(), org, actor, in.ID, in.Action, in.Evidence)
	}
	if err != nil {
		writeProblem(w, 409, "fee_reconciliation_unconfirmed", err.Error())
		return
	}
	writeJSON(w, 200, map[string]bool{"saved": true})
}
