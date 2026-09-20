package web

import (
	"errors"
	"net/http"
	"time"

	"kredit/internal/access"
	"kredit/internal/audit"
	"kredit/internal/erp"
)

func (s *Server) erpReconcile(w http.ResponseWriter, r *http.Request) {
	orgID, err := pathID(r, "organizationID")
	if err != nil {
		writeProblem(w, 400, "invalid_business", err.Error())
		return
	}
	_, user, _, ok := s.requireOrganizationAccess(w, r, orgID, access.PermissionReadFinancial)
	if !ok {
		return
	}
	if !s.requireCSRF(w, r) {
		return
	}

	var in struct {
		PeriodStart time.Time            `json:"period_start"`
		PeriodEnd   time.Time            `json:"period_end"`
		External    []erp.ExternalRecord `json:"external_records"`
	}
	if err := decodeJSON(w, r, &in); err != nil {
		writeProblem(w, 400, "invalid_payload", err.Error())
		return
	}
	if in.PeriodStart.IsZero() {
		in.PeriodStart = time.Now().UTC().AddDate(0, -1, 0)
	}
	if in.PeriodEnd.IsZero() {
		in.PeriodEnd = time.Now().UTC()
	}

	// Validate submitted evidence before reading the database. No fallback may
	// manufacture ledger entries from principal balances or due dates.
	if _, err := erp.ReconcileExternal(orgID, in.PeriodStart, in.PeriodEnd, nil, in.External); err != nil {
		writeProblem(w, 422, "invalid_reconciliation", "Use unique transaction references, signed amounts and dates within the selected period.")
		return
	}
	if s.runtime.Database == nil {
		writeProblem(w, 503, "ledger_unavailable", "Authoritative ledger data is required for reconciliation.")
		return
	}
	internal, err := erp.LoadLedgerMovements(r.Context(), s.runtime.Database.Raw(), orgID, in.PeriodStart, in.PeriodEnd)
	if err != nil {
		status := http.StatusServiceUnavailable
		if errors.Is(err, erp.ErrInvalidRecords) {
			status = http.StatusUnprocessableEntity
		}
		writeProblem(w, status, "ledger_unavailable", "Ledger movements could not be loaded; verify the period and retry.")
		return
	}

	report, err := erp.ReconcileExternal(orgID, in.PeriodStart, in.PeriodEnd, internal, in.External)
	if err != nil {
		writeProblem(w, 422, "reconciliation_failed", "The submitted movements cannot be reconciled safely.")
		return
	}

	s.runtime.Audit.Append(audit.Event{
		ActorUserID:    user.ID,
		OrganizationID: orgID,
		Action:         "erp.reconciled",
		ResourceType:   "erp_reconciliation",
		ResourceID:     report.ReportHash,
		Outcome:        "success",
		RequestID:      requestIDFromContext(r.Context()),
	})

	writeJSON(w, 200, report)
}
