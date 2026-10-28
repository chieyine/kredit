package web

import (
	"context"
	"errors"
	"kredit/internal/access"
	"kredit/internal/agreementdocs"
	"kredit/internal/audit"
	"kredit/internal/buyers"
	"kredit/internal/credit"
	"kredit/internal/documents"
	"kredit/internal/ledger"
	"kredit/internal/mandates"
	"net/http"
	"strings"
	"time"
)

type creditRequestInput struct {
	TimingMode          string    `json:"timing_mode,omitempty"`
	CollectionLocal     string    `json:"collection_local,omitempty"`
	CollectionPolicy    string    `json:"collection_policy,omitempty"`
	BuyerUserID         string    `json:"buyer_user_id"`
	BuyerBusinessID     string    `json:"buyer_business_id"`
	BuyerLegalName      string    `json:"buyer_legal_name"`
	BuyerTradingName    string    `json:"buyer_trading_name"`
	PrincipalKobo       int64     `json:"principal_kobo"`
	GoodsDescription    string    `json:"goods_description"`
	InvoiceReference    string    `json:"invoice_reference"`
	InvoiceDocumentID   string    `json:"invoice_document_id"`
	InvoiceDocumentHash string    `json:"invoice_document_hash"`
	DueDate             string    `json:"due_date"`
	GraceHours          int       `json:"grace_hours"`
	CollectionAt        time.Time `json:"collection_at"`
	ScheduleType        string    `json:"schedule_type"`
	ScheduleCount       int       `json:"schedule_count"`
	ScheduleCadence     string    `json:"schedule_cadence"`
	MonthEndPolicy      string    `json:"month_end_policy"`
	CustomScheduleItems []struct {
		AmountKobo int64  `json:"amount_kobo"`
		DueDate    string `json:"due_date"`
	} `json:"custom_schedule_items"`
}
type creditDraftUpdateInput struct {
	CollectionPolicy    string    `json:"collection_policy,omitempty"`
	ExpectedVersion     int64     `json:"expected_version"`
	PrincipalKobo       int64     `json:"principal_kobo"`
	GoodsDescription    string    `json:"goods_description"`
	InvoiceDocumentID   *string   `json:"invoice_document_id"`
	InvoiceReference    *string   `json:"invoice_reference"`
	InvoiceDocumentHash *string   `json:"invoice_document_hash"`
	DueDate             string    `json:"due_date"`
	GraceHours          int       `json:"grace_hours"`
	CollectionAt        time.Time `json:"collection_at"`
	ScheduleType        string    `json:"schedule_type"`
	ScheduleCount       int       `json:"schedule_count"`
	ScheduleCadence     string    `json:"schedule_cadence"`
	MonthEndPolicy      string    `json:"month_end_policy"`
	CustomScheduleItems []struct {
		AmountKobo int64  `json:"amount_kobo"`
		DueDate    string `json:"due_date"`
	} `json:"custom_schedule_items"`
}
type releaseInput struct {
	DeliveryMethod string `json:"delivery_method"`
	Notes          string `json:"notes"`
}
type acceptInput struct {
	AgreementVersionID string `json:"agreement_version_id"`
	AgreementHash      string `json:"agreement_hash"`
	MandateProviderID  string `json:"mandate_provider_id"`
}
type receiptInput struct {
	State       string `json:"state"`
	IssueReason string `json:"issue_reason"`
}
type paymentInput struct {
	SourceType        string    `json:"source_type"`
	AmountKobo        int64     `json:"amount_kobo"`
	Currency          string    `json:"currency"`
	Provider          string    `json:"provider"`
	ProviderReference string    `json:"provider_reference"`
	PaidAt            time.Time `json:"paid_at"`
	IdempotencyKey    string    `json:"idempotency_key"`
}
type tradeLineLimitInput struct {
	ExpectedVersion   int64 `json:"expected_version"`
	ApprovedLimitKobo int64 `json:"approved_limit_kobo"`
}
type tradeLineInput struct {
	BuyerUserID       string    `json:"buyer_user_id"`
	BuyerBusinessID   string    `json:"buyer_business_id"`
	ApprovedLimitKobo int64     `json:"approved_limit_kobo"`
	Cadence           string    `json:"cadence"`
	DefaultGraceHours int       `json:"default_grace_hours"`
	StartAt           time.Time `json:"start_at"`
	EndAt             time.Time `json:"end_at"`
	MandateID         string    `json:"mandate_id"`
	TermsVersion      string    `json:"terms_version"`
}
type drawdownInput struct {
	PrincipalKobo       int64     `json:"principal_kobo"`
	GoodsDescription    string    `json:"goods_description"`
	InvoiceReference    string    `json:"invoice_reference"`
	InvoiceDocumentHash string    `json:"invoice_document_hash"`
	DueDate             string    `json:"due_date"`
	CollectionAt        time.Time `json:"collection_at"`
	IdempotencyKey      string    `json:"idempotency_key"`
	ExpiresAt           time.Time `json:"expires_at"`
}

func (s *Server) createCreditRequest(w http.ResponseWriter, r *http.Request) {
	orgID, err := pathID(r, "organizationID")
	if err != nil {
		writeProblem(w, 400, "invalid_path", err.Error())
		return
	}
	_, user, _, ok := s.requireOrganizationAccess(w, r, orgID, access.PermissionCreateCredit)
	if !ok {
		return
	}
	if !s.requireCSRF(w, r) {
		return
	}
	var in creditRequestInput
	if err := decodeJSON(w, r, &in); err != nil {
		writeProblem(w, 400, "invalid_request", err.Error())
		return
	}
	if in.TimingMode != "" {
		if in.TimingMode != "lagos_end_of_day" && in.TimingMode != "lagos_explicit" {
			writeProblem(w, 422, "credit_terms_invalid", "That payment timing option is not supported.")
			return
		}
		canonical, timingErr := credit.CollectionInstant(in.DueDate, in.GraceHours)
		if timingErr == nil && in.TimingMode == "lagos_explicit" {
			earliest := canonical
			canonical, timingErr = credit.ExplicitCollectionInstant(in.CollectionLocal)
			if timingErr == nil && canonical.Before(earliest) {
				writeProblem(w, 422, "credit_terms_invalid", "Bank collection must be after the agreed payment day and extra hours.")
				return
			}
		}
		if timingErr != nil || (!in.CollectionAt.IsZero() && !in.CollectionAt.Equal(canonical)) {
			writeProblem(w, 422, "credit_terms_changed", "Review the payment date again before saving.")
			return
		}
		in.CollectionAt = canonical
	}
	org, exists := s.runtime.Organizations.Get(orgID)
	if !exists {
		writeProblem(w, 404, "organization_not_found", "We could not find that business.")
		return
	}
	customSchedule := make([]credit.ScheduleTerm, 0, len(in.CustomScheduleItems))
	for _, item := range in.CustomScheduleItems {
		customSchedule = append(customSchedule, credit.ScheduleTerm{AmountKobo: ledger.Money(item.AmountKobo), DueDate: item.DueDate})
	}
	if !s.requireSaleRiskClear(w, r, orgID, in.BuyerUserID, "credit") {
		return
	}
	if err := s.validateCreditInvoice(r, orgID, user.ID, in.InvoiceDocumentID, in.InvoiceDocumentHash); err != nil {
		writeProblem(w, 422, "invalid_invoice", err.Error())
		return
	}
	req, err := s.runtime.Credit.Create(credit.CreateInput{CollectionPolicy: in.CollectionPolicy, SupplierOrganizationID: orgID, SupplierLegalName: org.LegalName, SupplierTradingName: org.TradingName, BuyerUserID: in.BuyerUserID, BuyerBusinessID: in.BuyerBusinessID, BuyerLegalName: in.BuyerLegalName, BuyerTradingName: in.BuyerTradingName, PrincipalKobo: creditMoney(in.PrincipalKobo), GoodsDescription: in.GoodsDescription, InvoiceReference: in.InvoiceReference, InvoiceDocumentHash: in.InvoiceDocumentHash, InvoiceDocumentID: in.InvoiceDocumentID, DueDate: in.DueDate, GraceHours: in.GraceHours, CollectionAt: in.CollectionAt, ScheduleType: in.ScheduleType, ScheduleCount: in.ScheduleCount, ScheduleCadence: in.ScheduleCadence, MonthEndPolicy: in.MonthEndPolicy, CustomScheduleItems: customSchedule, CreatedBy: user.ID})
	if err != nil {
		writeProblem(w, 422, "credit_request_invalid", err.Error())
		return
	}
	s.auditCredit(user.ID, orgID, "credit.request.created", req.ID)
	writeJSON(w, 201, map[string]any{"request": req})
}
func (s *Server) listCreditRequests(w http.ResponseWriter, r *http.Request) {
	orgID, err := pathID(r, "organizationID")
	if err != nil {
		writeProblem(w, 400, "invalid_path", err.Error())
		return
	}
	if _, _, _, ok := s.requireOrganizationAccess(w, r, orgID, access.PermissionReadOrganization); !ok {
		return
	}
	financialRows, readErr := s.runtime.readCreditForSupplier(r.Context(), orgID)
	if financialReadError(w, readErr) {
		return
	}
	writeJSON(w, 200, map[string]any{"requests": financialRows})
}
func (s *Server) getCreditRequest(w http.ResponseWriter, r *http.Request) {
	orgID, _ := pathID(r, "organizationID")
	id, _ := pathID(r, "requestID")
	if _, _, _, ok := s.requireOrganizationAccess(w, r, orgID, access.PermissionReadOrganization); !ok {
		return
	}
	v, err := s.runtime.getCreditForSupplier(r.Context(), id, orgID)
	if err != nil {
		writeProblem(w, 404, "credit_request_not_found", err.Error())
		return
	}
	writeJSON(w, 200, v)
}
func (s *Server) updateDraftCreditRequest(w http.ResponseWriter, r *http.Request) {
	orgID, _ := pathID(r, "organizationID")
	id, _ := pathID(r, "requestID")
	_, user, _, ok := s.requireOrganizationAccess(w, r, orgID, access.PermissionCreateCredit)
	if !ok {
		return
	}
	if !s.requireCSRF(w, r) {
		return
	}
	current, err := s.runtime.getCreditForSupplier(r.Context(), id, orgID)
	if err != nil {
		writeProblem(w, http.StatusNotFound, "credit_request_not_found", "credit request was not found")
		return
	}
	var in creditDraftUpdateInput
	if err := decodeJSON(w, r, &in); err != nil {
		writeProblem(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	invoiceReference, invoiceHash, invoiceID := current.Request.InvoiceReference, current.Request.InvoiceDocumentHash, current.Request.InvoiceDocumentID
	if in.InvoiceReference != nil {
		invoiceReference = *in.InvoiceReference
	}
	if in.InvoiceDocumentHash != nil {
		invoiceHash = *in.InvoiceDocumentHash
	}
	if in.InvoiceDocumentID != nil {
		invoiceID = *in.InvoiceDocumentID
	}
	if in.InvoiceDocumentID != nil || in.InvoiceDocumentHash != nil {
		if err := s.validateCreditInvoice(r, orgID, user.ID, invoiceID, invoiceHash); err != nil {
			writeProblem(w, 422, "invalid_invoice", err.Error())
			return
		}
	}
	customSchedule := make([]credit.ScheduleTerm, 0, len(in.CustomScheduleItems))
	for _, item := range in.CustomScheduleItems {
		customSchedule = append(customSchedule, credit.ScheduleTerm{AmountKobo: ledger.Money(item.AmountKobo), DueDate: item.DueDate})
	}
	request, err := s.runtime.Credit.UpdateDraft(id, user.ID, credit.UpdateDraftInput{CollectionPolicy: in.CollectionPolicy, ExpectedVersion: in.ExpectedVersion, PrincipalKobo: ledger.Money(in.PrincipalKobo), GoodsDescription: in.GoodsDescription, ReplaceInvoice: true, InvoiceReference: invoiceReference, InvoiceDocumentHash: invoiceHash, InvoiceDocumentID: invoiceID, DueDate: in.DueDate, GraceHours: in.GraceHours, CollectionAt: in.CollectionAt, ScheduleType: in.ScheduleType, ScheduleCount: in.ScheduleCount, ScheduleCadence: in.ScheduleCadence, MonthEndPolicy: in.MonthEndPolicy, CustomScheduleItems: customSchedule})
	if err != nil {
		writeProblem(w, http.StatusConflict, "credit_update_failed", err.Error())
		return
	}
	s.auditCredit(user.ID, orgID, "credit.request.updated", id)
	writeJSON(w, http.StatusOK, map[string]any{"request": request})
}
func (s *Server) sendCreditRequest(w http.ResponseWriter, r *http.Request) {
	orgID, _ := pathID(r, "organizationID")
	id, _ := pathID(r, "requestID")
	_, user, _, ok := s.requireOrganizationAccess(w, r, orgID, access.PermissionCreateCredit)
	if !ok {
		return
	}
	if !s.requireSupplierReady(w, r, orgID, user.ID, "sending credit") {
		return
	}
	if !s.requireCSRF(w, r) {
		return
	}
	existing, readErr := s.runtime.getCreditForSupplier(r.Context(), id, orgID)
	if readErr != nil {
		writeProblem(w, http.StatusNotFound, "credit_request_not_found", "credit request was not found")
		return
	}
	if !s.requireSaleRiskClear(w, r, orgID, existing.Request.BuyerUserID, "credit") {
		return
	}
	v, err := s.runtime.Credit.Send(id, user.ID)
	if err != nil {
		writeProblem(w, 409, "credit_send_failed", err.Error())
		return
	}
	s.auditCredit(user.ID, orgID, "credit.request.sent", id)
	writeJSON(w, 200, v)
}
func (s *Server) cancelCreditRequest(w http.ResponseWriter, r *http.Request) {
	orgID, _ := pathID(r, "organizationID")
	id, _ := pathID(r, "requestID")
	_, user, _, ok := s.requireOrganizationAccess(w, r, orgID, access.PermissionCreateCredit)
	if !ok {
		return
	}
	if !s.requireCSRF(w, r) {
		return
	}
	if _, err := s.runtime.getCreditForSupplier(r.Context(), id, orgID); err != nil {
		writeProblem(w, http.StatusNotFound, "credit_request_not_found", "credit request was not found")
		return
	}
	view, err := s.runtime.Credit.Cancel(id, user.ID)
	if err != nil {
		writeProblem(w, http.StatusConflict, "credit_cancel_failed", err.Error())
		return
	}
	s.auditCredit(user.ID, orgID, "credit.request.cancelled", id)
	writeJSON(w, http.StatusOK, view)
}
func (s *Server) releaseCreditRequest(w http.ResponseWriter, r *http.Request) {
	orgID, _ := pathID(r, "organizationID")
	id, _ := pathID(r, "requestID")
	_, user, _, ok := s.requireOrganizationAccess(w, r, orgID, access.PermissionReleaseGoods)
	if !ok {
		return
	}
	if !s.requireSupplierReady(w, r, orgID, user.ID, "releasing goods") {
		return
	}
	if !s.requireCSRF(w, r) {
		return
	}
	var in releaseInput
	if err := decodeJSON(w, r, &in); err != nil {
		writeProblem(w, 400, "invalid_request", err.Error())
		return
	}
	existing, readErr := s.runtime.getCreditForSupplier(r.Context(), id, orgID)
	if readErr != nil {
		writeProblem(w, http.StatusNotFound, "credit_request_not_found", "credit request was not found")
		return
	}
	if !s.requireSaleRiskClear(w, r, orgID, existing.Request.BuyerUserID, "release") {
		return
	}
	v, err := s.runtime.Credit.Release(id, orgID, user.ID, in.DeliveryMethod, in.Notes)
	if err != nil {
		writeProblem(w, 409, "goods_release_failed", err.Error())
		return
	}
	s.auditCredit(user.ID, orgID, "goods.released", id)
	writeJSON(w, 200, v)
}
func (s *Server) getBuyerCreditRequest(w http.ResponseWriter, r *http.Request) {
	_, user, ok := s.requireAuth(w, r)
	if !ok {
		return
	}
	id, _ := pathID(r, "requestID")
	v, err := s.runtime.Credit.GetForBuyer(id, user.ID)
	if err != nil {
		writeProblem(w, 404, "credit_request_not_found", err.Error())
		return
	}
	writeJSON(w, 200, v)
}
func (s *Server) listBuyerCreditRequests(w http.ResponseWriter, r *http.Request) {
	_, user, ok := s.requireAuth(w, r)
	if !ok {
		return
	}
	businessID, scopedOK := s.buyerWorkspaceScope(w, r, user.ID)
	if !scopedOK {
		return
	}
	financialRows, readErr := s.runtime.readCreditForBuyer(r.Context(), user.ID)
	if financialReadError(w, readErr) {
		return
	}
	if businessID != "" {
		financialRows = purchasingRows(financialRows, func(item credit.View) bool { return item.Request.BuyerBusinessID == businessID })
	}
	writeJSON(w, http.StatusOK, map[string]any{"requests": financialRows})
}
func (s *Server) getBuyerAgreement(w http.ResponseWriter, r *http.Request) {
	_, user, ok := s.requireAuth(w, r)
	if !ok {
		return
	}
	id, _ := pathID(r, "requestID")
	v, err := s.runtime.Credit.GetForBuyer(id, user.ID)
	if err != nil {
		writeProblem(w, 404, "credit_request_not_found", err.Error())
		return
	}
	document, err := credit.PrintableAgreement(v)
	if err != nil {
		writeProblem(w, http.StatusConflict, "agreement_document_unavailable", "The agreed terms could not be verified. Please try again or contact support.")
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("X-Agreement-Hash", v.Agreement.DocumentHash)
	w.WriteHeader(200)
	_, _ = w.Write([]byte(document))
}

func (s *Server) getSupplierAgreementDocument(w http.ResponseWriter, r *http.Request) {
	organizationID, err := pathID(r, "organizationID")
	if err != nil {
		writeProblem(w, http.StatusBadRequest, "invalid_path", err.Error())
		return
	}
	requestID, err := pathID(r, "requestID")
	if err != nil {
		writeProblem(w, http.StatusBadRequest, "invalid_path", err.Error())
		return
	}
	if _, _, _, ok := s.requireOrganizationAccess(w, r, organizationID, access.PermissionReadOrganization); !ok {
		return
	}
	view, err := s.runtime.getCreditForSupplier(r.Context(), requestID, organizationID)
	if err != nil {
		writeProblem(w, http.StatusNotFound, "credit_request_not_found", "credit request was not found")
		return
	}
	s.renderAgreementDocument(w, r, view)
}

func (s *Server) getBuyerAgreementDocument(w http.ResponseWriter, r *http.Request) {
	_, user, ok := s.requireAuth(w, r)
	if !ok {
		return
	}
	requestID, err := pathID(r, "requestID")
	if err != nil {
		writeProblem(w, http.StatusBadRequest, "invalid_path", err.Error())
		return
	}
	view, err := s.runtime.Credit.GetForBuyer(requestID, user.ID)
	if err != nil {
		writeProblem(w, http.StatusNotFound, "credit_request_not_found", "credit request was not found")
		return
	}
	s.renderAgreementDocument(w, r, view)
}

func (s *Server) renderAgreementDocument(w http.ResponseWriter, r *http.Request, view credit.View) {
	if view.Obligation == nil {
		writeProblem(w, http.StatusConflict, "agreement_not_activated", "the printable agreement is available after the obligation is activated")
		return
	}
	schedule, items, err := s.runtime.Schedules.ForContext(r.Context()).GetForObligation(view.Obligation.ID)
	if err != nil {
		writeProblem(w, http.StatusConflict, "agreement_schedule_unavailable", "the activated agreement schedule is unavailable")
		return
	}
	document, err := agreementdocs.RenderHTML(agreementdocs.DocumentData{View: view, Schedule: schedule, Items: items})
	if err != nil {
		writeProblem(w, http.StatusConflict, "agreement_document_unavailable", err.Error())
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Content-Security-Policy", agreementdocs.ContentSecurityPolicy)
	w.Header().Set("Content-Disposition", "inline; filename=\"kredit-agreement-"+view.Request.ID+".html\"")
	w.Header().Set("X-Agreement-Hash", view.Agreement.DocumentHash)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(document)
}
func (s *Server) authorizeCreditMandate(w http.ResponseWriter, r *http.Request) {
	_, user, ok := s.requireAuth(w, r)
	if !ok {
		return
	}
	if !s.requireCSRF(w, r) {
		return
	}
	id, _ := pathID(r, "requestID")
	if _, err := s.runtime.Credit.GetForBuyer(id, user.ID); err != nil {
		writeProblem(w, http.StatusNotFound, "credit_request_not_found", "credit request was not found")
		return
	}
	var options mandates.AuthorizationOptions
	if r.ContentLength != 0 {
		if err := decodeJSON(w, r, &options); err != nil {
			writeProblem(w, 400, "invalid_mandate_options", "Those bank debit settings are not valid.")
			return
		}
	}
	v, err := s.runtime.Credit.AuthorizeMandate(r.Context(), id, user.ID, options)
	if err != nil {
		writeProblem(w, 409, "mandate_authorization_failed", err.Error())
		return
	}
	s.auditCredit(user.ID, "", "mandate.authorization_requested", id)
	writeJSON(w, 200, v)
}
func (s *Server) acceptCreditRequest(w http.ResponseWriter, r *http.Request) {
	session, user, ok := s.requireAuth(w, r)
	if !ok {
		return
	}
	if !s.requireCSRF(w, r) {
		return
	}
	id, _ := pathID(r, "requestID")
	var in acceptInput
	if err := decodeJSON(w, r, &in); err != nil {
		writeProblem(w, 400, "invalid_request", err.Error())
		return
	}
	request, err := s.runtime.Credit.GetForBuyer(id, user.ID)
	if err != nil {
		writeProblem(w, http.StatusNotFound, "credit_request_not_found", "credit request was not found")
		return
	}
	if !s.requireSaleRiskClear(w, r, request.Request.SupplierOrganizationID, user.ID, "credit") {
		return
	}
	if request.Request.BuyerUserID != user.ID && !s.requireSaleRiskClear(w, r, request.Request.SupplierOrganizationID, request.Request.BuyerUserID, "credit") {
		return
	}
	portal, err := s.runtime.Buyers.ReadBusinessPortal(r.Context(), user.ID, request.Request.BuyerBusinessID)
	if err != nil {
		writeProblem(w, 403, "buyer_profile_required", err.Error())
		return
	}
	if portal.Business.ID != request.Request.BuyerBusinessID || portal.Representative.BusinessID != request.Request.BuyerBusinessID || portal.Representative.PersonID != portal.Person.ID {
		writeProblem(w, http.StatusForbidden, "buyer_authority_required", "Verified authority for this buyer business is required.")
		return
	}
	v, err := s.runtime.Credit.Accept(id, user.ID, in.AgreementVersionID, in.AgreementHash, in.MandateProviderID, session.AuthenticationLevel, buyers.VerificationCurrent(portal, time.Now()), portal.Representative.AuthorityStatus == "verified")
	if err != nil {
		writeProblem(w, 409, "credit_acceptance_failed", err.Error())
		return
	}
	s.auditCredit(user.ID, "", "credit.accepted", id)
	writeJSON(w, 200, v)
}
func (s *Server) declineCreditRequest(w http.ResponseWriter, r *http.Request) {
	_, user, ok := s.requireAuth(w, r)
	if !ok {
		return
	}
	if !s.requireCSRF(w, r) {
		return
	}
	id, _ := pathID(r, "requestID")
	if _, err := s.runtime.Credit.GetForBuyer(id, user.ID); err != nil {
		writeProblem(w, http.StatusNotFound, "credit_request_not_found", "credit request was not found")
		return
	}
	view, err := s.runtime.Credit.Decline(id, user.ID)
	if err != nil {
		writeProblem(w, http.StatusConflict, "credit_decline_failed", err.Error())
		return
	}
	s.auditCredit(user.ID, "", "credit.request.declined", id)
	writeJSON(w, http.StatusOK, view)
}
func (s *Server) recordCreditReceipt(w http.ResponseWriter, r *http.Request) {
	_, user, ok := s.requireAuth(w, r)
	if !ok {
		return
	}
	if !s.requireCSRF(w, r) {
		return
	}
	id, _ := pathID(r, "requestID")
	var in receiptInput
	if err := decodeJSON(w, r, &in); err != nil {
		writeProblem(w, 400, "invalid_request", err.Error())
		return
	}
	if _, err := s.runtime.Credit.GetForBuyer(id, user.ID); err != nil {
		writeProblem(w, http.StatusNotFound, "credit_request_not_found", "credit request was not found")
		return
	}
	v, tx, err := s.runtime.Credit.RecordReceipt(id, user.ID, strings.ToLower(in.State), in.IssueReason)
	if err != nil {
		writeProblem(w, 409, "receipt_failed", err.Error())
		return
	}
	s.auditCredit(user.ID, "", "receipt.recorded", id)
	response := map[string]any{"view": v}
	if tx != nil {
		response["ledger_transaction"] = tx
	}
	writeJSON(w, 200, response)
}
func (s *Server) auditCredit(actor, org, action, id string) {
	s.runtime.Audit.Append(audit.Event{ActorUserID: actor, OrganizationID: org, Action: action, ResourceType: "credit_request", ResourceID: id, Outcome: "success"})
}
func creditMoney(v int64) ledger.Money { return ledger.Money(v) }

func errString(err error, fallback string) string {
	if err != nil {
		return err.Error()
	}
	return fallback
}

type collectionStartInput struct {
	IdempotencyKey string `json:"idempotency_key"`
}
type disputeInput struct {
	DisputedAmountKobo int64  `json:"disputed_amount_kobo"`
	Reason             string `json:"reason"`
	Explanation        string `json:"explanation"`
	CollectionEffect   string `json:"collection_effect"`
}
type evidenceInput struct {
	DocumentID string `json:"document_id"`
	Statement  string `json:"statement"`
}

func (s *Server) requireSaleRiskClear(w http.ResponseWriter, r *http.Request, organizationID, buyerID, scope string) bool {
	if s.runtime.PlatformOps == nil {
		return true
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	for _, target := range []struct{ kind, id string }{{"supplier", organizationID}, {"buyer", buyerID}} {
		if target.id == "" {
			continue
		}
		blocked, err := s.runtime.PlatformOps.ActiveHold(ctx, target.kind, target.id, scope)
		if err != nil {
			writeProblem(w, http.StatusServiceUnavailable, "risk_hold_unavailable", "Risk hold status could not be verified. Try again.")
			return false
		}
		if blocked {
			writeProblem(w, http.StatusLocked, "risk_hold_active", "This action is blocked by an active risk hold.")
			return false
		}
	}
	return true
}

func (s *Server) validateCreditInvoice(r *http.Request, organization, actor, id, hash string) error {
	if id == "" {
		if hash != "" {
			return errors.New("upload the invoice before attaching it to this sale")
		}
		return nil
	}
	document, err := s.runtime.Documents.ReadForTenant(r.Context(), id, actor, organization)
	if err != nil || document.OrganizationID != organization || document.Purpose != "credit_invoice" || document.SHA256 != hash || document.UploadCompletedAt.IsZero() || document.ScanState == documents.ScanRejected || document.ScanState == documents.ScanQuarantine {
		return errors.New("the invoice is incomplete or unavailable for this business")
	}
	return nil
}
