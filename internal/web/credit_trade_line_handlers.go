package web

import (
	"kredit/internal/access"
	"kredit/internal/agreementdocs"
	"kredit/internal/ledger"
	"kredit/internal/mandates"
	"kredit/internal/notifications"
	"kredit/internal/tradelines"
	"net/http"
	"net/url"
)

func (s *Server) createTradeLine(w http.ResponseWriter, r *http.Request) {
	if !s.isFeatureEnabled(r.Context(), "features.trade_lines", false) {
		writeProblem(w, http.StatusForbidden, "feature_disabled", "Trade lines are currently unavailable")
		return
	}
	orgID, _ := pathID(r, "organizationID")
	_, user, _, ok := s.requireOrganizationAccess(w, r, orgID, access.PermissionManageFinancial)
	if !ok {
		return
	}
	if !s.requireCSRF(w, r) {
		return
	}
	var in tradeLineInput
	if err := decodeJSON(w, r, &in); err != nil {
		writeProblem(w, 400, "invalid_request", err.Error())
		return
	}
	if !s.requireSaleRiskClear(w, r, orgID, in.BuyerUserID, "credit") {
		return
	}
	resolver, ok := s.runtime.Mandates.(mandates.TradeLineResolver)
	if !ok {
		writeProblem(w, http.StatusServiceUnavailable, "mandate_verification_unavailable", "trade-line mandate verification is unavailable")
		return
	}
	mandate, err := resolver.ResolveTradeLineMandate(r.Context(), in.MandateID, in.BuyerUserID, in.BuyerBusinessID, orgID)
	if err != nil || mandate.Status != mandates.Active {
		writeProblem(w, http.StatusUnprocessableEntity, "mandate_inactive", "an active mandate owned by the selected buyer business is required")
		return
	}
	if mandate.AmountCeiling < in.ApprovedLimitKobo {
		writeProblem(w, http.StatusUnprocessableEntity, "mandate_ceiling_insufficient", "the mandate ceiling does not cover the proposed trade-line limit")
		return
	}
	line, err := s.runtime.ScopedTradeLines(r.Context()).CreateLine(tradelines.CreateLineInput{SupplierOrganizationID: orgID, BuyerUserID: in.BuyerUserID, BuyerBusinessID: in.BuyerBusinessID, ApprovedLimitKobo: ledger.Money(in.ApprovedLimitKobo), Cadence: in.Cadence, DefaultGraceHours: in.DefaultGraceHours, StartAt: in.StartAt, EndAt: in.EndAt, MandateID: mandate.ID, MandateActive: true, MandateVerified: true, TermsVersion: in.TermsVersion})
	if err != nil {
		writeProblem(w, 422, "trade_line_invalid", err.Error())
		return
	}
	s.auditCredit(user.ID, orgID, "trade_line.created", line.ID)
	writeJSON(w, 201, map[string]any{"trade_line": line})
}
func (s *Server) listTradeLines(w http.ResponseWriter, r *http.Request) {
	orgID, _ := pathID(r, "organizationID")
	if _, _, _, ok := s.requireOrganizationAccess(w, r, orgID, access.PermissionReadOrganization); !ok {
		return
	}
	financialRows, readErr := s.runtime.readTradeLinesForSupplier(r.Context(), orgID)
	if financialReadError(w, readErr) {
		return
	}
	writeJSON(w, 200, map[string]any{"trade_lines": financialRows})
}
func (s *Server) getTradeLine(w http.ResponseWriter, r *http.Request) {
	orgID, _ := pathID(r, "organizationID")
	lineID, _ := pathID(r, "lineID")
	if _, _, _, ok := s.requireOrganizationAccess(w, r, orgID, access.PermissionReadOrganization); !ok {
		return
	}
	line, ok := s.runtime.ScopedTradeLines(r.Context()).Get(lineID)
	if !ok || line.SupplierOrganizationID != orgID {
		writeProblem(w, 404, "trade_line_not_found", "We could not find that customer limit.")
		return
	}
	writeJSON(w, 200, map[string]any{"trade_line": line})
}
func (s *Server) reduceTradeLineLimit(w http.ResponseWriter, r *http.Request) {
	orgID, _ := pathID(r, "organizationID")
	lineID, _ := pathID(r, "lineID")
	_, user, _, ok := s.requireOrganizationAccess(w, r, orgID, access.PermissionManageFinancial)
	if !ok {
		return
	}
	if !s.requireCSRF(w, r) {
		return
	}
	line, exists := s.runtime.ScopedTradeLines(r.Context()).Get(lineID)
	if !exists || line.SupplierOrganizationID != orgID {
		writeProblem(w, 404, "trade_line_not_found", "We could not find that customer limit.")
		return
	}
	var input tradeLineLimitInput
	if err := decodeJSON(w, r, &input); err != nil {
		writeProblem(w, 400, "invalid_request", err.Error())
		return
	}
	updated, err := s.runtime.ScopedTradeLines(r.Context()).ReduceLimit(lineID, ledger.Money(input.ApprovedLimitKobo), input.ExpectedVersion)
	if err != nil {
		writeProblem(w, 409, "trade_line_limit_change_failed", err.Error())
		return
	}
	s.auditCredit(user.ID, orgID, "trade_line.limit_reduced", lineID)
	writeJSON(w, 200, map[string]any{"trade_line": updated})
}
func (s *Server) reserveDrawdown(w http.ResponseWriter, r *http.Request) {
	if !s.isFeatureEnabled(r.Context(), "features.drawdowns", false) {
		writeProblem(w, http.StatusForbidden, "feature_disabled", "Drawdowns are currently unavailable")
		return
	}
	orgID, _ := pathID(r, "organizationID")
	lineID, _ := pathID(r, "lineID")
	_, user, _, ok := s.requireOrganizationAccess(w, r, orgID, access.PermissionManageFinancial)
	if !ok {
		return
	}
	if !s.requireCSRF(w, r) {
		return
	}
	line, exists := s.runtime.ScopedTradeLines(r.Context()).Get(lineID)
	if !exists || line.SupplierOrganizationID != orgID {
		writeProblem(w, 404, "trade_line_not_found", "We could not find that customer limit.")
		return
	}
	if !s.requireSaleRiskClear(w, r, orgID, line.BuyerUserID, "credit") {
		return
	}
	var in drawdownInput
	if err := decodeJSON(w, r, &in); err != nil {
		writeProblem(w, 400, "invalid_request", err.Error())
		return
	}
	if in.IdempotencyKey == "" {
		in.IdempotencyKey = r.Header.Get("Idempotency-Key")
	}
	drawdown, reservation, updated, err := s.runtime.ScopedTradeLines(r.Context()).ReserveDrawdown(tradelines.CreateDrawdownInput{LineID: lineID, PrincipalKobo: ledger.Money(in.PrincipalKobo), GoodsDescription: in.GoodsDescription, InvoiceReference: in.InvoiceReference, InvoiceDocumentHash: in.InvoiceDocumentHash, DueDate: in.DueDate, CollectionAt: in.CollectionAt, IdempotencyKey: in.IdempotencyKey, ExpiresAt: in.ExpiresAt})
	if err != nil {
		writeProblem(w, 409, "drawdown_reservation_failed", err.Error())
		return
	}
	s.auditCredit(user.ID, orgID, "trade_line.drawdown_reserved", drawdown.ID)
	_, _ = s.runtime.EmitNotification(r.Context(), notifications.Event{ID: "drawdown-confirmation-required:" + drawdown.ID, Type: "TradeLineDrawdownConfirmationRequired", RecipientID: updated.BuyerUserID, Priority: notifications.PriorityCritical, AmountKobo: int64(drawdown.PrincipalKobo), Currency: "NGN", Reference: drawdown.ID, NextAction: "Review and confirm the exact purchase terms", SecurePath: "/workspace/purchases/trade-lines?business_id=" + url.QueryEscape(updated.BuyerBusinessID)})
	writeJSON(w, 201, map[string]any{"drawdown": drawdown, "reservation": reservation, "trade_line": updated})
}
func (s *Server) confirmDrawdown(w http.ResponseWriter, r *http.Request) {
	_, user, ok := s.requireAuth(w, r)
	if !ok {
		return
	}
	if !s.requireCSRF(w, r) {
		return
	}
	lineID, _ := pathID(r, "lineID")
	drawdownID, _ := pathID(r, "drawdownID")
	var in struct {
		AgreementHash string `json:"agreement_hash"`
	}
	if err := decodeJSON(w, r, &in); err != nil {
		writeProblem(w, 400, "invalid_request", err.Error())
		return
	}
	if !s.requireDrawdownScope(w, r, lineID, drawdownID, user.ID, "") {
		return
	}
	selected, exists := s.runtime.ScopedTradeLines(r.Context()).Get(lineID)
	if !exists {
		writeProblem(w, 404, "trade_line_not_found", "Customer limit was not found.")
		return
	}
	if !s.requireSaleRiskClear(w, r, selected.SupplierOrganizationID, user.ID, "credit") {
		return
	}
	drawdown, line, err := s.runtime.ScopedTradeLines(r.Context()).ConfirmDrawdown(drawdownID, user.ID, in.AgreementHash)
	if err != nil || line.ID != lineID {
		writeProblem(w, 409, "drawdown_confirmation_failed", errString(err, "drawdown confirmation failed"))
		return
	}
	s.auditCredit(user.ID, "", "trade_line.drawdown_confirmed", drawdown.ID)
	_, _ = s.runtime.EmitNotification(r.Context(), notifications.Event{ID: "drawdown-confirmed:" + drawdown.ID, Type: "TradeLineDrawdownConfirmed", OrganizationID: line.SupplierOrganizationID, Priority: notifications.PriorityCritical, AmountKobo: int64(drawdown.PrincipalKobo), Currency: "NGN", Reference: drawdown.ID, NextAction: "Release the goods only when ready", SecurePath: "/workspace/sales/limits/" + line.ID})
	_, _ = s.runtime.EmitNotification(r.Context(), notifications.Event{ID: "drawdown-safe-to-release:" + drawdown.ID, Type: "TradeLineDrawdownSafeToRelease", OrganizationID: line.SupplierOrganizationID, Priority: notifications.PriorityCritical, AmountKobo: int64(drawdown.PrincipalKobo), Currency: "NGN", Reference: drawdown.ID, NextAction: "Record release evidence when the goods leave", SecurePath: "/workspace/sales/limits/" + line.ID})
	writeJSON(w, 200, map[string]any{"drawdown": drawdown, "trade_line": line})
}
func (s *Server) releaseDrawdown(w http.ResponseWriter, r *http.Request) {
	orgID, _ := pathID(r, "organizationID")
	lineID, _ := pathID(r, "lineID")
	drawdownID, _ := pathID(r, "drawdownID")
	_, user, _, ok := s.requireOrganizationAccess(w, r, orgID, access.PermissionReleaseGoods)
	if !ok {
		return
	}
	if !s.requireSupplierReady(w, orgID, user.ID, "releasing goods") {
		return
	}
	if !s.requireCSRF(w, r) {
		return
	}
	var in struct {
		DeliveryMethod    string `json:"delivery_method"`
		Notes             string `json:"notes"`
		EvidenceReference string `json:"evidence_reference"`
	}
	if err := decodeJSON(w, r, &in); err != nil {
		writeProblem(w, 400, "invalid_request", err.Error())
		return
	}
	line, exists := s.runtime.ScopedTradeLines(r.Context()).Get(lineID)
	if !exists || line.SupplierOrganizationID != orgID {
		writeProblem(w, 404, "trade_line_not_found", "We could not find that customer limit.")
		return
	}
	if !s.requireSaleRiskClear(w, r, orgID, line.BuyerUserID, "release") {
		return
	}
	if !s.requireDrawdownScope(w, r, lineID, drawdownID, "", orgID) {
		return
	}
	drawdown, updated, err := s.runtime.ScopedTradeLines(r.Context()).ReleaseDrawdown(tradelines.ReleaseInput{DrawdownID: drawdownID, SupplierOrganizationID: orgID, ActorID: user.ID, DeliveryMethod: in.DeliveryMethod, Notes: in.Notes, EvidenceReference: in.EvidenceReference})
	if err != nil {
		writeProblem(w, 409, "drawdown_release_failed", err.Error())
		return
	}
	s.auditCredit(user.ID, orgID, "trade_line.drawdown_goods_released", drawdown.ID)
	_, _ = s.runtime.EmitNotification(r.Context(), notifications.Event{ID: "drawdown-released:" + drawdown.ID, Type: "TradeLineDrawdownGoodsReleased", RecipientID: updated.BuyerUserID, Priority: notifications.PriorityCritical, AmountKobo: int64(drawdown.PrincipalKobo), Currency: "NGN", Reference: drawdown.ID, NextAction: "Confirm receipt or report an issue", SecurePath: "/workspace/purchases/trade-lines?business_id=" + url.QueryEscape(updated.BuyerBusinessID)})
	_, _ = s.runtime.EmitNotification(r.Context(), notifications.Event{ID: "drawdown-receipt-required:" + drawdown.ID, Type: "TradeLineDrawdownReceiptRequired", RecipientID: updated.BuyerUserID, Priority: notifications.PriorityCritical, AmountKobo: int64(drawdown.PrincipalKobo), Currency: "NGN", Reference: drawdown.ID, NextAction: "Record whether the goods arrived without an issue", SecurePath: "/workspace/purchases/trade-lines?business_id=" + url.QueryEscape(updated.BuyerBusinessID)})
	writeJSON(w, 200, map[string]any{"drawdown": drawdown, "trade_line": updated})
}

func (s *Server) receiptDrawdown(w http.ResponseWriter, r *http.Request) {
	_, user, ok := s.requireAuth(w, r)
	if !ok {
		return
	}
	if !s.requireCSRF(w, r) {
		return
	}
	lineID, _ := pathID(r, "lineID")
	drawdownID, _ := pathID(r, "drawdownID")
	var in struct {
		State       string `json:"state"`
		IssueReason string `json:"issue_reason"`
	}
	if err := decodeJSON(w, r, &in); err != nil {
		writeProblem(w, 400, "invalid_request", err.Error())
		return
	}
	if !s.requireDrawdownScope(w, r, lineID, drawdownID, user.ID, "") {
		return
	}
	drawdown, line, err := s.runtime.ScopedTradeLines(r.Context()).RecordDrawdownReceipt(tradelines.ReceiptInput{DrawdownID: drawdownID, BuyerUserID: user.ID, State: in.State, IssueReason: in.IssueReason})
	if err != nil || line.ID != lineID {
		writeProblem(w, 409, "drawdown_receipt_failed", errString(err, "drawdown receipt failed"))
		return
	}
	action := "trade_line.drawdown_receipt_confirmed"
	eventType := "TradeLineDrawdownActivated"
	if in.State == "issue_reported" {
		action = "trade_line.drawdown_receipt_issue_reported"
		eventType = "TradeLineDrawdownReceiptIssueReported"
	}
	s.auditCredit(user.ID, line.SupplierOrganizationID, action, drawdown.ID)
	_, _ = s.runtime.EmitNotification(r.Context(), notifications.Event{ID: "drawdown-receipt:" + drawdown.ID + ":" + in.State, Type: eventType, OrganizationID: line.SupplierOrganizationID, Priority: notifications.PriorityCritical, AmountKobo: int64(drawdown.PrincipalKobo), Currency: "NGN", Reference: drawdown.ID, NextAction: "Review the drawdown status", SecurePath: "/workspace/sales/limits/" + line.ID})
	if in.State == "no_issue" {
		s.auditCredit(user.ID, line.SupplierOrganizationID, "trade_line.drawdown_obligation_activated", drawdown.ID)
		_, _ = s.runtime.EmitNotification(r.Context(), notifications.Event{ID: "drawdown-receipt-confirmed:" + drawdown.ID, Type: "TradeLineDrawdownReceiptConfirmed", OrganizationID: line.SupplierOrganizationID, Priority: notifications.PriorityCritical, AmountKobo: int64(drawdown.PrincipalKobo), Currency: "NGN", Reference: drawdown.ID, NextAction: "The obligation and schedule are now active", SecurePath: "/workspace/sales/limits/" + line.ID})
	}
	writeJSON(w, 200, map[string]any{"drawdown": drawdown, "trade_line": line})
}

// Validate the route's parent before a command can change a drawdown. Checking
// the returned line afterwards would report failure after committing the change.
func (s *Server) requireDrawdownScope(w http.ResponseWriter, r *http.Request, lineID, drawdownID, buyerID, organizationID string) bool {
	line, exists := s.runtime.ScopedTradeLines(r.Context()).Get(lineID)
	if !exists || (buyerID == "" && organizationID == "") || (buyerID != "" && line.BuyerUserID != buyerID) || (organizationID != "" && line.SupplierOrganizationID != organizationID) {
		writeProblem(w, http.StatusNotFound, "trade_line_not_found", "We could not find that customer limit.")
		return false
	}
	statement, err := s.runtime.ScopedTradeLines(r.Context()).Statement(lineID)
	if err != nil {
		writeProblem(w, http.StatusServiceUnavailable, "drawdown_unavailable", "We could not verify that purchase. Please try again.")
		return false
	}
	for _, drawdown := range statement.Drawdowns {
		if drawdown.ID == drawdownID && drawdown.TradeLineID == lineID {
			return true
		}
	}
	writeProblem(w, http.StatusNotFound, "drawdown_not_found", "We could not find that purchase under this customer limit.")
	return false
}

func (s *Server) cancelDrawdown(w http.ResponseWriter, r *http.Request) {
	orgID, _ := pathID(r, "organizationID")
	lineID, _ := pathID(r, "lineID")
	drawdownID, _ := pathID(r, "drawdownID")
	_, user, _, ok := s.requireOrganizationAccess(w, r, orgID, access.PermissionCreateCredit)
	if !ok {
		return
	}
	if !s.requireCSRF(w, r) {
		return
	}
	line, exists := s.runtime.ScopedTradeLines(r.Context()).Get(lineID)
	if !exists || line.SupplierOrganizationID != orgID {
		writeProblem(w, 404, "trade_line_not_found", "We could not find that customer limit.")
		return
	}
	drawdown, updated, err := s.runtime.ScopedTradeLines(r.Context()).CancelDrawdown(line.ID, drawdownID, user.ID)
	if err != nil {
		writeProblem(w, 409, "drawdown_cancel_failed", err.Error())
		return
	}
	s.auditCredit(user.ID, orgID, "trade_line.drawdown_cancelled", drawdown.ID)
	_, _ = s.runtime.EmitNotification(r.Context(), notifications.Event{ID: "drawdown-cancelled:" + drawdown.ID, Type: "TradeLineDrawdownCancelled", RecipientID: updated.BuyerUserID, Priority: notifications.PriorityCritical, AmountKobo: int64(drawdown.PrincipalKobo), Currency: "NGN", Reference: drawdown.ID, NextAction: "Review the released trade-line capacity", SecurePath: "/workspace/purchases/trade-lines?business_id=" + url.QueryEscape(updated.BuyerBusinessID)})
	writeJSON(w, 200, map[string]any{"drawdown": drawdown, "trade_line": updated})
}

func (s *Server) cancelBuyerDrawdown(w http.ResponseWriter, r *http.Request) {
	_, user, ok := s.requireAuth(w, r)
	if !ok {
		return
	}
	if !s.requireCSRF(w, r) {
		return
	}
	lineID, _ := pathID(r, "lineID")
	drawdownID, _ := pathID(r, "drawdownID")
	line, exists := s.runtime.ScopedTradeLines(r.Context()).Get(lineID)
	if !exists || line.BuyerUserID != user.ID {
		writeProblem(w, http.StatusNotFound, "trade_line_not_found", "We could not find that customer limit.")
		return
	}
	drawdown, updated, err := s.runtime.ScopedTradeLines(r.Context()).CancelDrawdown(line.ID, drawdownID, user.ID)
	if err != nil {
		writeProblem(w, http.StatusConflict, "drawdown_cancel_failed", err.Error())
		return
	}
	s.auditCredit(user.ID, line.SupplierOrganizationID, "trade_line.drawdown_cancelled", drawdown.ID)
	_, _ = s.runtime.EmitNotification(r.Context(), notifications.Event{ID: "drawdown-cancelled:" + drawdown.ID, Type: "TradeLineDrawdownCancelled", OrganizationID: line.SupplierOrganizationID, Priority: notifications.PriorityCritical, AmountKobo: int64(drawdown.PrincipalKobo), Currency: "NGN", Reference: drawdown.ID, NextAction: "Review the released capacity", SecurePath: "/workspace/sales/limits/" + line.ID})
	writeJSON(w, http.StatusOK, map[string]any{"drawdown": drawdown, "trade_line": updated})
}
func (s *Server) suspendTradeLine(w http.ResponseWriter, r *http.Request) {
	orgID, _ := pathID(r, "organizationID")
	lineID, _ := pathID(r, "lineID")
	_, user, _, ok := s.requireOrganizationAccess(w, r, orgID, access.PermissionManageFinancial)
	if !ok {
		return
	}
	if !s.requireCSRF(w, r) {
		return
	}
	var in struct {
		Reason string `json:"reason"`
	}
	if err := decodeJSON(w, r, &in); err != nil {
		writeProblem(w, 400, "invalid_request", err.Error())
		return
	}
	line, exists := s.runtime.ScopedTradeLines(r.Context()).Get(lineID)
	if !exists || line.SupplierOrganizationID != orgID {
		writeProblem(w, 404, "trade_line_not_found", "We could not find that customer limit.")
		return
	}
	line, err := s.runtime.ScopedTradeLines(r.Context()).Suspend(lineID, in.Reason)
	if err != nil {
		writeProblem(w, 409, "trade_line_suspend_failed", err.Error())
		return
	}
	s.auditCredit(user.ID, orgID, "trade_line.suspended", lineID)
	writeJSON(w, 200, map[string]any{"trade_line": line})
}
func (s *Server) resumeTradeLine(w http.ResponseWriter, r *http.Request) {
	orgID, _ := pathID(r, "organizationID")
	lineID, _ := pathID(r, "lineID")
	_, user, _, ok := s.requireOrganizationAccess(w, r, orgID, access.PermissionManageFinancial)
	if !ok {
		return
	}
	if !s.requireCSRF(w, r) {
		return
	}
	line, exists := s.runtime.ScopedTradeLines(r.Context()).Get(lineID)
	if !exists || line.SupplierOrganizationID != orgID {
		writeProblem(w, 404, "trade_line_not_found", "We could not find that customer limit.")
		return
	}
	line, err := s.runtime.ScopedTradeLines(r.Context()).Resume(lineID)
	if err != nil {
		writeProblem(w, 409, "trade_line_resume_failed", err.Error())
		return
	}
	s.auditCredit(user.ID, orgID, "trade_line.resumed", lineID)
	writeJSON(w, 200, map[string]any{"trade_line": line})
}
func (s *Server) tradeLineStatement(w http.ResponseWriter, r *http.Request) {
	orgID, _ := pathID(r, "organizationID")
	lineID, _ := pathID(r, "lineID")
	if _, _, _, ok := s.requireOrganizationAccess(w, r, orgID, access.PermissionReadFinancial); !ok {
		return
	}
	line, exists := s.runtime.ScopedTradeLines(r.Context()).Get(lineID)
	if !exists || line.SupplierOrganizationID != orgID {
		writeProblem(w, 404, "trade_line_not_found", "We could not find that customer limit.")
		return
	}
	statement, err := s.runtime.ScopedTradeLines(r.Context()).Statement(lineID)
	if err != nil {
		writeProblem(w, 404, "statement_not_found", err.Error())
		return
	}
	writeJSON(w, 200, statement)
}

func (s *Server) buyerTradeLineStatement(w http.ResponseWriter, r *http.Request) {
	_, user, ok := s.requireAuth(w, r)
	if !ok {
		return
	}
	lineID, _ := pathID(r, "lineID")
	line, exists := s.runtime.ScopedTradeLines(r.Context()).Get(lineID)
	if !exists || line.BuyerUserID != user.ID {
		writeProblem(w, 404, "trade_line_not_found", "We could not find that customer limit.")
		return
	}
	statement, err := s.runtime.ScopedTradeLines(r.Context()).Statement(lineID)
	if err != nil {
		writeProblem(w, 404, "statement_not_found", err.Error())
		return
	}
	writeJSON(w, 200, statement)
}

func (s *Server) getSupplierDrawdownAgreementDocument(w http.ResponseWriter, r *http.Request) {
	organizationID, _ := pathID(r, "organizationID")
	lineID, _ := pathID(r, "lineID")
	if _, _, _, ok := s.requireOrganizationAccess(w, r, organizationID, access.PermissionReadFinancial); !ok {
		return
	}
	line, exists := s.runtime.ScopedTradeLines(r.Context()).Get(lineID)
	if !exists || line.SupplierOrganizationID != organizationID {
		writeProblem(w, http.StatusNotFound, "trade_line_not_found", "We could not find that customer limit.")
		return
	}
	s.renderDrawdownAgreementDocument(w, r, line, r.PathValue("drawdownID"))
}

func (s *Server) getBuyerDrawdownAgreementDocument(w http.ResponseWriter, r *http.Request) {
	_, user, ok := s.requireAuth(w, r)
	if !ok {
		return
	}
	lineID, _ := pathID(r, "lineID")
	line, exists := s.runtime.ScopedTradeLines(r.Context()).Get(lineID)
	if !exists || line.BuyerUserID != user.ID {
		writeProblem(w, http.StatusNotFound, "trade_line_not_found", "We could not find that customer limit.")
		return
	}
	s.renderDrawdownAgreementDocument(w, r, line, r.PathValue("drawdownID"))
}

func (s *Server) renderDrawdownAgreementDocument(w http.ResponseWriter, r *http.Request, line tradelines.TradeLine, drawdownID string) {
	statement, err := s.runtime.ScopedTradeLines(r.Context()).Statement(line.ID)
	if err != nil {
		writeProblem(w, http.StatusNotFound, "statement_not_found", "trade-line statement was not found")
		return
	}
	var drawdown *tradelines.Drawdown
	for i := range statement.Drawdowns {
		if statement.Drawdowns[i].ID == drawdownID {
			drawdown = &statement.Drawdowns[i]
			break
		}
	}
	if drawdown == nil {
		writeProblem(w, http.StatusNotFound, "drawdown_not_found", "drawdown was not found")
		return
	}
	document, err := agreementdocs.RenderDrawdownHTML(agreementdocs.DrawdownDocumentData{Line: line, Drawdown: *drawdown})
	if err != nil {
		writeProblem(w, http.StatusConflict, "agreement_document_unavailable", err.Error())
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Content-Security-Policy", agreementdocs.ContentSecurityPolicy)
	w.Header().Set("Content-Disposition", "inline; filename=\"kredit-drawdown-agreement-"+drawdown.ID+".html\"")
	w.Header().Set("X-Agreement-Hash", drawdown.AgreementHash)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(document)
}
