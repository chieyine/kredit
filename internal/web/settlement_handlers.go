package web

import (
	"kredit/internal/access"
	"net/http"
	"sort"
)

func (s *Server) supplierSettlementBanks(w http.ResponseWriter, r *http.Request) {
	orgID, _ := pathID(r, "organizationID")
	if _, _, _, ok := s.requireOrganizationAccess(w, r, orgID, access.PermissionManageFinancial); !ok {
		return
	}
	if s.runtime.Settlement == nil {
		writeProblem(w, 503, "settlement_unavailable", "Bank registration is not configured. Contact support.")
		return
	}
	banks, err := s.runtime.Settlement.Banks(r.Context())
	if err != nil {
		writeProblem(w, 503, "banks_unavailable", "Banks could not be loaded. Try again later.")
		return
	}
	sort.Slice(banks, func(i, j int) bool { return banks[i].Name < banks[j].Name })
	writeJSON(w, 200, map[string]any{"banks": banks})
}
