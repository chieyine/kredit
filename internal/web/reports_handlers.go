package web

import (
	"net/http"
	"strings"

	"kredit/internal/access"
	"kredit/internal/audit"
	"kredit/internal/corrections"
	"kredit/internal/notifications"
)

type correctionInput struct {
	SubjectType   string   `json:"subject_type"`
	SubjectID     string   `json:"subject_id"`
	SourceEventID string   `json:"source_event_id"`
	Reason        string   `json:"reason"`
	Evidence      []string `json:"evidence"`
}
type correctionDecisionInput struct {
	Outcome string `json:"outcome"`
	Reason  string `json:"reason"`
}

func (s *Server) reportReceivables(w http.ResponseWriter, r *http.Request) {
	orgID, err := pathID(r, "organizationID")
	if err != nil {
		writeProblem(w, 400, "invalid_path", err.Error())
		return
	}
	if _, _, _, ok := s.requireOrganizationAccess(w, r, orgID, access.PermissionReadFinancial); !ok {
		return
	}
	_, _ = s.runtime.Reports.Track("report.receivables.viewed", orgID, "supplier receivables reporting", nil)
	report, err := s.runtime.Reports.ReceivablesForSupplier(r.Context(), orgID)
	if err != nil {
		writeProblem(w, 503, "report_unavailable", "Financial report could not be loaded")
		return
	}
	writeJSON(w, 200, report)
}

func (s *Server) providerStatus(w http.ResponseWriter, r *http.Request) {
	orgID, err := pathID(r, "organizationID")
	if err != nil {
		writeProblem(w, 400, "invalid_path", err.Error())
		return
	}
	if _, _, _, ok := s.requireOrganizationAccess(w, r, orgID, access.PermissionReadFinancial); !ok {
		return
	}
	writeJSON(w, 200, s.runtime.Collections.ProviderStatus())
}

func (s *Server) readinessStatus(w http.ResponseWriter, r *http.Request) {
	orgID, err := pathID(r, "organizationID")
	if err != nil {
		writeProblem(w, 400, "invalid_path", err.Error())
		return
	}
	if _, _, _, ok := s.requireOrganizationAccess(w, r, orgID, access.PermissionReadAudit); !ok {
		return
	}
	writeJSON(w, 200, s.runtime.Readiness)
}
func (s *Server) reportAgeing(w http.ResponseWriter, r *http.Request) {
	orgID, err := pathID(r, "organizationID")
	if err != nil {
		writeProblem(w, 400, "invalid_path", err.Error())
		return
	}
	if _, _, _, ok := s.requireOrganizationAccess(w, r, orgID, access.PermissionReadFinancial); !ok {
		return
	}
	_, _ = s.runtime.Reports.Track("report.ageing.viewed", orgID, "supplier ageing reporting", nil)
	report, err := s.runtime.Reports.AgeingForSupplier(r.Context(), orgID)
	if err != nil {
		writeProblem(w, 503, "report_unavailable", "Financial report could not be loaded")
		return
	}
	writeJSON(w, 200, report)
}
func (s *Server) reportFees(w http.ResponseWriter, r *http.Request) {
	orgID, err := pathID(r, "organizationID")
	if err != nil {
		writeProblem(w, 400, "invalid_path", err.Error())
		return
	}
	if _, _, _, ok := s.requireOrganizationAccess(w, r, orgID, access.PermissionReadFinancial); !ok {
		return
	}
	_, _ = s.runtime.Reports.Track("report.fees.viewed", orgID, "supplier fee reporting", nil)
	report, err := s.runtime.Reports.FeesForSupplier(r.Context(), orgID)
	if err != nil {
		writeProblem(w, 503, "fee_report_unavailable", "Fee report could not be loaded")
		return
	}
	writeJSON(w, 200, report)
}
func (s *Server) exportReceivables(w http.ResponseWriter, r *http.Request) {
	orgID, err := pathID(r, "organizationID")
	if err != nil {
		writeProblem(w, 400, "invalid_path", err.Error())
		return
	}
	_, user, _, ok := s.requireOrganizationAccess(w, r, orgID, access.PermissionReadFinancial)
	if !ok {
		return
	}
	if !s.requireCSRF(w, r) {
		return
	}
	format := strings.ToLower(r.URL.Query().Get("format"))
	if format == "" {
		format = "csv"
	}
	if format != "csv" {
		writeProblem(w, 422, "export_format_invalid", "only csv exports are supported")
		return
	}
	data, err := s.runtime.Reports.ExportReceivablesCSV(r.Context(), orgID)
	if err != nil {
		writeProblem(w, 503, "export_failed", "Financial export could not be prepared")
		return
	}
	s.runtime.Audit.Append(audit.Event{ActorUserID: user.ID, OrganizationID: orgID, Action: "report.exported", ResourceType: "report", ResourceID: "receivables", Outcome: "success", RequestID: requestIDFromContext(r.Context())})
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", "attachment; filename=receivables.csv")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

func (s *Server) buyerHistory(w http.ResponseWriter, r *http.Request) {
	_, user, ok := s.requireAuth(w, r)
	if !ok {
		return
	}
	h, err := s.runtime.Reports.HistoryForBuyer(r.Context(), user.ID)
	if err != nil {
		writeProblem(w, 503, "report_unavailable", "Financial history could not be loaded")
		return
	}
	h.Shareable = false
	_, _ = s.runtime.Reports.Track("history.viewed", user.ID, "buyer factual history", nil)
	writeJSON(w, 200, h)
}

func (s *Server) supplierCustomerHistory(w http.ResponseWriter, r *http.Request) {
	orgID, err := pathID(r, "organizationID")
	if err != nil {
		writeProblem(w, 400, "invalid_path", err.Error())
		return
	}
	buyerID, err := pathID(r, "buyerUserID")
	if err != nil {
		writeProblem(w, 400, "invalid_path", err.Error())
		return
	}
	if _, _, _, ok := s.requireOrganizationAccess(w, r, orgID, access.PermissionReadFinancial); !ok {
		return
	}
	h, err := s.runtime.Reports.HistoryForSupplierBuyer(r.Context(), orgID, buyerID)
	if err != nil {
		writeProblem(w, 503, "report_unavailable", "Financial history could not be loaded")
		return
	}
	h.Shareable = false
	writeJSON(w, 200, h)
}

func (s *Server) supplierCustomerStatement(w http.ResponseWriter, r *http.Request) {
	orgID, err := pathID(r, "organizationID")
	if err != nil {
		writeProblem(w, 400, "invalid_path", err.Error())
		return
	}
	buyerID, err := pathID(r, "buyerUserID")
	if err != nil {
		writeProblem(w, 400, "invalid_path", err.Error())
		return
	}
	if _, _, _, ok := s.requireOrganizationAccess(w, r, orgID, access.PermissionReadFinancial); !ok {
		return
	}
	report, err := s.runtime.Reports.CustomerStatement(r.Context(), orgID, buyerID)
	if err != nil {
		writeProblem(w, 503, "report_unavailable", "Financial report could not be loaded")
		return
	}
	writeJSON(w, 200, report)
}

func (s *Server) openCorrection(w http.ResponseWriter, r *http.Request) {
	_, user, ok := s.requireAuth(w, r)
	if !ok {
		return
	}
	if !s.requireCSRF(w, r) {
		return
	}
	var in correctionInput
	if err := decodeJSON(w, r, &in); err != nil {
		writeProblem(w, 400, "invalid_request", err.Error())
		return
	}
	orgID := ""
	lookupType, lookupID := in.SubjectType, in.SubjectID
	if lookupType == "payment" {
		if payment, paymentErr := s.runtime.getPayment(r.Context(), lookupID); paymentErr == nil && payment.BuyerUserID == user.ID {
			lookupType, lookupID = "obligation", payment.ObligationID
		}
	}
	if lookupType == "obligation" || lookupType == "credit_request" {
		financialRows1, readErr1 := s.runtime.readCreditForBuyer(r.Context(), user.ID)
		if financialReadError(w, readErr1) {
			return
		}
		for _, v := range financialRows1 {
			if (lookupType == "credit_request" && v.Request.ID == lookupID) || (lookupType == "obligation" && v.Obligation != nil && v.Obligation.ID == lookupID) {
				orgID = v.Request.SupplierOrganizationID
				break
			}
		}
	}
	if orgID == "" {
		writeProblem(w, 404, "subject_not_found", "the subject was not found for this buyer")
		return
	}
	c, err := s.runtime.Corrections.Open(orgID, in.SubjectType, in.SubjectID, in.SourceEventID, user.ID, in.Reason, in.Evidence)
	if err != nil {
		writeProblem(w, 422, "correction_invalid", err.Error())
		return
	}
	s.runtime.Audit.Append(audit.Event{ActorUserID: user.ID, OrganizationID: orgID, Action: "history.correction.requested", ResourceType: "correction", ResourceID: c.ID, Outcome: "success", RequestID: requestIDFromContext(r.Context())})
	writeJSON(w, 201, map[string]any{"correction": c})
}

func (s *Server) listCorrections(w http.ResponseWriter, r *http.Request) {
	orgID, err := pathID(r, "organizationID")
	if err != nil {
		writeProblem(w, 400, "invalid_path", err.Error())
		return
	}
	if _, _, _, ok := s.requireOrganizationAccess(w, r, orgID, access.PermissionReadAudit); !ok {
		return
	}
	writeJSON(w, 200, map[string]any{"corrections": s.runtime.Corrections.ListForOrganization(orgID)})
}

func (s *Server) decideCorrection(w http.ResponseWriter, r *http.Request) {
	orgID, err := pathID(r, "organizationID")
	if err != nil {
		writeProblem(w, 400, "invalid_path", err.Error())
		return
	}
	id, err := pathID(r, "correctionID")
	if err != nil {
		writeProblem(w, 400, "invalid_path", err.Error())
		return
	}
	_, user, _, ok := s.requireOrganizationAccess(w, r, orgID, access.PermissionManageFinancial)
	if !ok {
		return
	}
	if !s.requireCSRF(w, r) {
		return
	}
	var in correctionDecisionInput
	if err := decodeJSON(w, r, &in); err != nil {
		writeProblem(w, 400, "invalid_request", err.Error())
		return
	}
	correction, _, err := s.runtime.Corrections.Get(id)
	if err != nil || correction.OrganizationID != orgID {
		writeProblem(w, 404, "correction_not_found", "correction was not found")
		return
	}
	if in.Outcome == corrections.StateReview {
		updated, err := s.runtime.Corrections.StartReview(id, user.ID)
		if err != nil {
			writeProblem(w, 409, "correction_review_failed", err.Error())
			return
		}
		writeJSON(w, 200, map[string]any{"correction": updated})
		return
	}
	updated, decision, err := s.runtime.Corrections.Decide(id, user.ID, in.Outcome, in.Reason)
	if err != nil {
		writeProblem(w, 422, "correction_decision_invalid", err.Error())
		return
	}
	s.runtime.Audit.Append(audit.Event{ActorUserID: user.ID, OrganizationID: orgID, Action: "history.correction.decided", ResourceType: "correction", ResourceID: id, Outcome: in.Outcome, RequestID: requestIDFromContext(r.Context())})
	lookupType, lookupID := correction.SubjectType, correction.SubjectID
	if lookupType == "payment" {
		if payment, paymentErr := s.runtime.getPayment(r.Context(), lookupID); paymentErr == nil {
			lookupType, lookupID = "obligation", payment.ObligationID
		}
	}
	financialRows2, readErr2 := s.runtime.readCreditForSupplier(r.Context(), orgID)
	if financialReadError(w, readErr2) {
		return
	}
	for _, view := range financialRows2 {
		if (lookupType == "credit_request" && view.Request.ID == lookupID) || (lookupType == "obligation" && view.Obligation != nil && view.Obligation.ID == lookupID) {
			_, _ = s.runtime.EmitNotification(r.Context(), notifications.Event{ID: "correction-notice-" + id, Type: "history.correction.updated", RecipientID: view.Request.BuyerUserID, OrganizationID: orgID, Priority: notifications.PriorityRoutine, Reference: id, NextAction: "Review your factual history"})
			break
		}
	}
	writeJSON(w, 200, map[string]any{"correction": updated, "decision": decision})
}
