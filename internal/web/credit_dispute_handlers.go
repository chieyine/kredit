package web

import (
	"kredit/internal/access"
	"kredit/internal/disputes"
	"kredit/internal/ledger"
	"kredit/internal/operations"
	"net/http"
)

func (s *Server) openDispute(w http.ResponseWriter, r *http.Request) {
	if !s.isFeatureEnabled(r.Context(), "features.disputes", true) {
		writeProblem(w, http.StatusForbidden, "feature_disabled", "Disputes are currently unavailable")
		return
	}
	orgID, _ := pathID(r, "organizationID")
	requestID, _ := pathID(r, "requestID")
	_, user, _, ok := s.requireOrganizationAccess(w, r, orgID, access.PermissionManageDisputes)
	if !ok {
		return
	}
	if !s.requireCSRF(w, r) {
		return
	}
	v, err := s.runtime.getCreditForSupplier(r.Context(), requestID, orgID)
	if err != nil || v.Obligation == nil {
		writeProblem(w, 404, "obligation_not_found", "We could not find that sale.")
		return
	}
	var in disputeInput
	if err := decodeJSON(w, r, &in); err != nil {
		writeProblem(w, 400, "invalid_request", err.Error())
		return
	}
	dispute, err := s.runtime.ScopedDisputes(r.Context()).Open(disputes.OpenInput{ObligationID: v.Obligation.ID, OpenedBy: user.ID, SupplierOrganizationID: orgID, DisputedAmountKobo: ledger.Money(in.DisputedAmountKobo), Reason: in.Reason, Explanation: in.Explanation, CollectionEffect: in.CollectionEffect})
	if err != nil {
		writeProblem(w, 422, "dispute_invalid", err.Error())
		return
	}
	s.auditCredit(user.ID, orgID, "dispute.opened", dispute.ID)
	writeJSON(w, 201, map[string]any{"dispute": dispute})
}
func (s *Server) openBuyerDispute(w http.ResponseWriter, r *http.Request) {
	if !s.isFeatureEnabled(r.Context(), "features.disputes", true) {
		writeProblem(w, http.StatusForbidden, "feature_disabled", "Disputes are currently unavailable")
		return
	}
	_, user, ok := s.requireAuth(w, r)
	if !ok {
		return
	}
	if !s.requireCSRF(w, r) {
		return
	}
	requestID, _ := pathID(r, "requestID")
	v, err := s.runtime.Credit.GetForBuyer(requestID, user.ID)
	if err != nil || v.Obligation == nil || v.Request.BuyerUserID != user.ID {
		writeProblem(w, 404, "obligation_not_found", "We could not find that sale.")
		return
	}
	var in disputeInput
	if err := decodeJSON(w, r, &in); err != nil {
		writeProblem(w, 400, "invalid_request", err.Error())
		return
	}
	dispute, err := s.runtime.ScopedDisputes(r.Context()).Open(disputes.OpenInput{ObligationID: v.Obligation.ID, OpenedBy: user.ID, SupplierOrganizationID: v.Request.SupplierOrganizationID, DisputedAmountKobo: ledger.Money(in.DisputedAmountKobo), Reason: in.Reason, Explanation: in.Explanation, CollectionEffect: in.CollectionEffect})
	if err != nil {
		writeProblem(w, 422, "dispute_invalid", err.Error())
		return
	}
	s.auditCredit(user.ID, "", "dispute.opened", dispute.ID)
	writeJSON(w, 201, map[string]any{"dispute": dispute})
}
func (s *Server) getBuyerDispute(w http.ResponseWriter, r *http.Request) {
	_, user, ok := s.requireAuth(w, r)
	if !ok {
		return
	}
	disputeID, _ := pathID(r, "disputeID")
	dispute, evidence, decisions, err := s.runtime.ScopedDisputes(r.Context()).Get(disputeID)
	if err != nil || dispute.BuyerUserID != user.ID {
		writeProblem(w, 404, "dispute_not_found", "We could not find that problem.")
		return
	}
	writeJSON(w, 200, map[string]any{"dispute": dispute, "evidence": evidence, "decisions": decisions})
}
func (s *Server) addBuyerDisputeEvidence(w http.ResponseWriter, r *http.Request) {
	_, user, ok := s.requireAuth(w, r)
	if !ok {
		return
	}
	if !s.requireCSRF(w, r) {
		return
	}
	disputeID, _ := pathID(r, "disputeID")
	dispute, _, _, err := s.runtime.ScopedDisputes(r.Context()).Get(disputeID)
	if err != nil || dispute.BuyerUserID != user.ID {
		writeProblem(w, 404, "dispute_not_found", "We could not find that problem.")
		return
	}
	var in evidenceInput
	if err := decodeJSON(w, r, &in); err != nil {
		writeProblem(w, 400, "invalid_request", err.Error())
		return
	}
	if dispute.State == disputes.StateResolved || dispute.State == disputes.StateWithdrawn {
		writeProblem(w, 409, "dispute_closed", "This dispute is closed.")
		return
	}
	if err := s.validateDisputeDocument(r, dispute, user.ID, in.DocumentID); err != nil {
		writeProblem(w, 422, "document_not_ready", "Upload a document for this dispute and wait for its safety check before attaching it.")
		return
	}
	evidence, err := s.runtime.ScopedDisputes(r.Context()).AddEvidence(disputeID, user.ID, in.DocumentID, in.Statement)
	if err != nil {
		writeProblem(w, 422, "evidence_invalid", err.Error())
		return
	}
	s.auditCredit(user.ID, dispute.SupplierOrganizationID, "dispute.evidence_added", disputeID)
	writeJSON(w, 201, map[string]any{"evidence": evidence})
}
func (s *Server) listDisputes(w http.ResponseWriter, r *http.Request) {
	orgID, _ := pathID(r, "organizationID")
	if _, _, _, ok := s.requireOrganizationAccess(w, r, orgID, access.PermissionReadFinancial); !ok {
		return
	}
	financialRows, readErr := s.runtime.readDisputesForOrganization(r.Context(), orgID)
	if financialReadError(w, readErr) {
		return
	}
	writeJSON(w, 200, map[string]any{"disputes": financialRows})
}
func (s *Server) getDispute(w http.ResponseWriter, r *http.Request) {
	orgID, _ := pathID(r, "organizationID")
	disputeID, _ := pathID(r, "disputeID")
	if _, _, _, ok := s.requireOrganizationAccess(w, r, orgID, access.PermissionReadFinancial); !ok {
		return
	}
	dispute, evidence, decisions, err := s.runtime.ScopedDisputes(r.Context()).Get(disputeID)
	if err != nil || dispute.SupplierOrganizationID != orgID {
		writeProblem(w, 404, "dispute_not_found", "We could not find that problem.")
		return
	}
	writeJSON(w, 200, map[string]any{"dispute": dispute, "evidence": evidence, "decisions": decisions})
}
func (s *Server) addDisputeEvidence(w http.ResponseWriter, r *http.Request) {
	orgID, _ := pathID(r, "organizationID")
	disputeID, _ := pathID(r, "disputeID")
	_, user, _, ok := s.requireOrganizationAccess(w, r, orgID, access.PermissionManageDisputes)
	if !ok {
		return
	}
	if !s.requireCSRF(w, r) {
		return
	}
	dispute, _, _, err := s.runtime.ScopedDisputes(r.Context()).Get(disputeID)
	if err != nil || dispute.SupplierOrganizationID != orgID {
		writeProblem(w, 404, "dispute_not_found", "We could not find that problem.")
		return
	}
	var in evidenceInput
	if err := decodeJSON(w, r, &in); err != nil {
		writeProblem(w, 400, "invalid_request", err.Error())
		return
	}
	if dispute.State == disputes.StateResolved || dispute.State == disputes.StateWithdrawn {
		writeProblem(w, 409, "dispute_closed", "This dispute is closed.")
		return
	}
	if err := s.validateDisputeDocument(r, dispute, user.ID, in.DocumentID); err != nil {
		writeProblem(w, 422, "document_not_ready", "Upload a document for this dispute and wait for its safety check before attaching it.")
		return
	}
	evidence, err := s.runtime.ScopedDisputes(r.Context()).AddEvidence(disputeID, user.ID, in.DocumentID, in.Statement)
	if err != nil {
		writeProblem(w, 422, "evidence_invalid", err.Error())
		return
	}
	s.auditCredit(user.ID, orgID, "dispute.evidence_added", disputeID)
	writeJSON(w, 201, map[string]any{"evidence": evidence})
}
func (s *Server) decideDispute(w http.ResponseWriter, r *http.Request) {
	orgID, _ := pathID(r, "organizationID")
	_, _, _, ok := s.requireOrganizationAccess(w, r, orgID, access.PermissionManageDisputes)
	if !ok {
		return
	}
	if !s.requireCSRF(w, r) {
		return
	}
	writeProblem(w, http.StatusForbidden, "independent_review_required", "buyer disputes can only be decided by an independent dispute reviewer")
}

type operationInput struct {
	AmountKobo int64  `json:"amount_kobo"`
	Reason     string `json:"reason"`
	ApprovedBy string `json:"approved_by"`
}

func (s *Server) writeOffObligation(w http.ResponseWriter, r *http.Request) {
	orgID, _ := pathID(r, "organizationID")
	requestID, _ := pathID(r, "requestID")
	_, user, _, ok := s.requireOrganizationAccess(w, r, orgID, access.PermissionManageFinancial)
	if !ok {
		return
	}
	if !s.requireCSRF(w, r) {
		return
	}
	v, err := s.runtime.getCreditForSupplier(r.Context(), requestID, orgID)
	if err != nil || v.Obligation == nil {
		writeProblem(w, 404, "obligation_not_found", "We could not find that sale.")
		return
	}
	var in operationInput
	if err := decodeJSON(w, r, &in); err != nil {
		writeProblem(w, 400, "invalid_request", err.Error())
		return
	}
	if ledger.Money(in.AmountKobo) > v.Obligation.OutstandingKobo {
		writeProblem(w, 422, "operation_invalid", "You cannot write off more than the customer still owes.")
		return
	}
	var action operations.Action
	if keyed, ok := s.runtime.Operations.(operations.IdempotentService); ok {
		action, err = keyed.WriteOffWithKey(user.ID, orgID, v.Obligation.ID, ledger.Money(in.AmountKobo), in.Reason, in.ApprovedBy, r.Header.Get("Idempotency-Key"))
	} else {
		action, err = s.runtime.Operations.WriteOff(user.ID, orgID, v.Obligation.ID, ledger.Money(in.AmountKobo), in.Reason, in.ApprovedBy)
	}
	if err != nil {
		writeProblem(w, 409, "write_off_failed", err.Error())
		return
	}
	s.auditCredit(user.ID, orgID, "operation.write_off", action.ID)
	writeJSON(w, 200, map[string]any{"action": action})
}
func (s *Server) waiveObligationFee(w http.ResponseWriter, r *http.Request) {
	orgID, _ := pathID(r, "organizationID")
	requestID, _ := pathID(r, "requestID")
	_, user, _, ok := s.requireOrganizationAccess(w, r, orgID, access.PermissionManageFinancial)
	if !ok {
		return
	}
	if !s.requireCSRF(w, r) {
		return
	}
	v, err := s.runtime.getCreditForSupplier(r.Context(), requestID, orgID)
	if err != nil || v.Obligation == nil {
		writeProblem(w, 404, "obligation_not_found", "We could not find that sale.")
		return
	}
	var in operationInput
	if err := decodeJSON(w, r, &in); err != nil {
		writeProblem(w, 400, "invalid_request", err.Error())
		return
	}
	var action operations.Action
	if keyed, ok := s.runtime.Operations.(operations.IdempotentService); ok {
		action, err = keyed.WaiveFeeWithKey(user.ID, orgID, v.Obligation.ID, ledger.Money(in.AmountKobo), in.Reason, in.ApprovedBy, r.Header.Get("Idempotency-Key"))
	} else {
		action, err = s.runtime.Operations.WaiveFee(user.ID, orgID, v.Obligation.ID, ledger.Money(in.AmountKobo), in.Reason, in.ApprovedBy)
	}
	if err != nil {
		writeProblem(w, 409, "fee_waiver_failed", err.Error())
		return
	}
	s.auditCredit(user.ID, orgID, "operation.fee_waiver", action.ID)
	writeJSON(w, 200, map[string]any{"action": action})
}
