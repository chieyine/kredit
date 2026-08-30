package web

import (
	"context"
	"net/http"
	"strings"
	"time"

	"kredit/internal/access"
	"kredit/internal/agreementdocs"
	"kredit/internal/audit"
	"kredit/internal/collections"
	"kredit/internal/credit"
	"kredit/internal/disputes"
	"kredit/internal/ledger"
	"kredit/internal/mandates"
	"kredit/internal/notifications"
	"kredit/internal/operations"
	"kredit/internal/payments"
	"kredit/internal/schedules"
	"kredit/internal/tradelines"
	"kredit/internal/whatsapp"
)

type creditRequestInput struct {
	BuyerUserID         string    `json:"buyer_user_id"`
	BuyerBusinessID     string    `json:"buyer_business_id"`
	BuyerLegalName      string    `json:"buyer_legal_name"`
	BuyerTradingName    string    `json:"buyer_trading_name"`
	PrincipalKobo       int64     `json:"principal_kobo"`
	GoodsDescription    string    `json:"goods_description"`
	InvoiceReference    string    `json:"invoice_reference"`
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
	ExpectedVersion     int64     `json:"expected_version"`
	PrincipalKobo       int64     `json:"principal_kobo"`
	GoodsDescription    string    `json:"goods_description"`
	InvoiceReference    string    `json:"invoice_reference"`
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
type scheduleInput struct {
	ScheduleType         string    `json:"schedule_type"`
	Count                int       `json:"count"`
	InstalmentAmountKobo int64     `json:"instalment_amount_kobo"`
	StartDate            time.Time `json:"start_date"`
	DueHour              int       `json:"due_hour"`
	DueMinute            int       `json:"due_minute"`
	Timezone             string    `json:"timezone"`
	GraceHours           int       `json:"grace_hours"`
	Cadence              string    `json:"cadence"`
	MonthEndPolicy       string    `json:"month_end_policy"`
	AllocationPolicy     string    `json:"allocation_policy"`
	CustomItems          []struct {
		AmountKobo int64     `json:"amount_kobo"`
		DueDate    time.Time `json:"due_date"`
	} `json:"custom_items"`
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
	org, exists := s.runtime.Organizations.Get(orgID)
	if !exists {
		writeProblem(w, 404, "organization_not_found", "organization was not found")
		return
	}
	customSchedule := make([]credit.ScheduleTerm, 0, len(in.CustomScheduleItems))
	for _, item := range in.CustomScheduleItems {
		customSchedule = append(customSchedule, credit.ScheduleTerm{AmountKobo: ledger.Money(item.AmountKobo), DueDate: item.DueDate})
	}
	req, err := s.runtime.Credit.Create(credit.CreateInput{SupplierOrganizationID: orgID, SupplierLegalName: org.LegalName, SupplierTradingName: org.TradingName, BuyerUserID: in.BuyerUserID, BuyerBusinessID: in.BuyerBusinessID, BuyerLegalName: in.BuyerLegalName, BuyerTradingName: in.BuyerTradingName, PrincipalKobo: creditMoney(in.PrincipalKobo), GoodsDescription: in.GoodsDescription, InvoiceReference: in.InvoiceReference, InvoiceDocumentHash: in.InvoiceDocumentHash, DueDate: in.DueDate, GraceHours: in.GraceHours, CollectionAt: in.CollectionAt, ScheduleType: in.ScheduleType, ScheduleCount: in.ScheduleCount, ScheduleCadence: in.ScheduleCadence, MonthEndPolicy: in.MonthEndPolicy, CustomScheduleItems: customSchedule, CreatedBy: user.ID})
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
	writeJSON(w, 200, map[string]any{"requests": s.runtime.Credit.ListForSupplier(orgID)})
}
func (s *Server) getCreditRequest(w http.ResponseWriter, r *http.Request) {
	orgID, _ := pathID(r, "organizationID")
	id, _ := pathID(r, "requestID")
	if _, _, _, ok := s.requireOrganizationAccess(w, r, orgID, access.PermissionReadOrganization); !ok {
		return
	}
	v, err := s.runtime.Credit.GetForSupplier(id, orgID)
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
	if _, err := s.runtime.Credit.GetForSupplier(id, orgID); err != nil {
		writeProblem(w, http.StatusNotFound, "credit_request_not_found", "credit request was not found")
		return
	}
	var in creditDraftUpdateInput
	if err := decodeJSON(w, r, &in); err != nil {
		writeProblem(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	customSchedule := make([]credit.ScheduleTerm, 0, len(in.CustomScheduleItems))
	for _, item := range in.CustomScheduleItems {
		customSchedule = append(customSchedule, credit.ScheduleTerm{AmountKobo: ledger.Money(item.AmountKobo), DueDate: item.DueDate})
	}
	request, err := s.runtime.Credit.UpdateDraft(id, user.ID, credit.UpdateDraftInput{ExpectedVersion: in.ExpectedVersion, PrincipalKobo: ledger.Money(in.PrincipalKobo), GoodsDescription: in.GoodsDescription, InvoiceReference: in.InvoiceReference, InvoiceDocumentHash: in.InvoiceDocumentHash, DueDate: in.DueDate, GraceHours: in.GraceHours, CollectionAt: in.CollectionAt, ScheduleType: in.ScheduleType, ScheduleCount: in.ScheduleCount, ScheduleCadence: in.ScheduleCadence, MonthEndPolicy: in.MonthEndPolicy, CustomScheduleItems: customSchedule})
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
	if !s.requireSupplierReady(w, orgID, user.ID, "sending credit") {
		return
	}
	if !s.requireCSRF(w, r) {
		return
	}
	if _, err := s.runtime.Credit.GetForSupplier(id, orgID); err != nil {
		writeProblem(w, http.StatusNotFound, "credit_request_not_found", "credit request was not found")
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
	if _, err := s.runtime.Credit.GetForSupplier(id, orgID); err != nil {
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
	if !s.requireSupplierReady(w, orgID, user.ID, "releasing goods") {
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
	if _, err := s.runtime.Credit.GetForSupplier(id, orgID); err != nil {
		writeProblem(w, http.StatusNotFound, "credit_request_not_found", "credit request was not found")
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
	writeJSON(w, http.StatusOK, map[string]any{"requests": s.runtime.Credit.ListForBuyer(user.ID)})
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
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("X-Agreement-Hash", v.Agreement.DocumentHash)
	w.WriteHeader(200)
	_, _ = w.Write([]byte(credit.PrintableAgreement(v)))
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
	view, err := s.runtime.Credit.GetForSupplier(requestID, organizationID)
	if err != nil {
		writeProblem(w, http.StatusNotFound, "credit_request_not_found", "credit request was not found")
		return
	}
	s.renderAgreementDocument(w, view)
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
	s.renderAgreementDocument(w, view)
}

func (s *Server) renderAgreementDocument(w http.ResponseWriter, view credit.View) {
	if view.Obligation == nil {
		writeProblem(w, http.StatusConflict, "agreement_not_activated", "the printable agreement is available after the obligation is activated")
		return
	}
	schedule, items, err := s.runtime.Schedules.GetForObligation(view.Obligation.ID)
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
	v, err := s.runtime.Credit.AuthorizeMandate(context.Background(), id, user.ID)
	if err != nil {
		writeProblem(w, 409, "mandate_authorization_failed", err.Error())
		return
	}
	s.auditCredit(user.ID, "", "mandate.authorized", id)
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
	if _, err := s.runtime.Credit.GetForBuyer(id, user.ID); err != nil {
		writeProblem(w, http.StatusNotFound, "credit_request_not_found", "credit request was not found")
		return
	}
	portal, err := s.runtime.Buyers.Portal(user.ID)
	if err != nil {
		writeProblem(w, 403, "buyer_profile_required", err.Error())
		return
	}
	v, err := s.runtime.Credit.Accept(id, user.ID, in.AgreementVersionID, in.AgreementHash, in.MandateProviderID, session.AuthenticationLevel, portal.Person.Status == "verified" && len(portal.VerificationCases) > 0, portal.Representative.AuthorityStatus == "verified")
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

func (s *Server) recordPayment(w http.ResponseWriter, r *http.Request) {
	orgID, _ := pathID(r, "organizationID")
	requestID, _ := pathID(r, "requestID")
	_, user, _, ok := s.requireOrganizationAccess(w, r, orgID, access.PermissionManageFinancial)
	if !ok {
		return
	}
	if !s.requireCSRF(w, r) {
		return
	}
	var in paymentInput
	if err := decodeJSON(w, r, &in); err != nil {
		writeProblem(w, 400, "invalid_request", err.Error())
		return
	}
	if in.IdempotencyKey == "" {
		in.IdempotencyKey = r.Header.Get("Idempotency-Key")
	}
	if in.SourceType == "" {
		in.SourceType = payments.SourceVoluntary
	}
	v, err := s.runtime.Credit.GetForSupplier(requestID, orgID)
	if err != nil || v.Obligation == nil {
		writeProblem(w, 404, "obligation_not_found", "active obligation was not found")
		return
	}
	p, a, err := s.runtime.Payments.Record(payments.RecordInput{ObligationID: v.Obligation.ID, SourceType: in.SourceType, AmountKobo: ledger.Money(in.AmountKobo), Currency: in.Currency, Provider: in.Provider, ProviderReference: in.ProviderReference, PaidAt: in.PaidAt, RecordedBy: user.ID, IdempotencyKey: in.IdempotencyKey})
	if err != nil {
		writeProblem(w, 409, "payment_record_failed", err.Error())
		return
	}
	s.auditCredit(user.ID, orgID, "payment.recorded", p.ID)
	_, _ = s.runtime.Notifications.Emit(r.Context(), notifications.Event{ID: "payment-recorded:" + p.ID, Type: "PaymentRecorded", RecipientID: p.BuyerUserID, Priority: notifications.PriorityRoutine, AmountKobo: int64(p.AmountKobo), Currency: p.Currency, Reference: p.ID, NextAction: "Review your payment history", SecurePath: "/buyer/credit-requests/" + requestID})
	response := map[string]any{"payment": p, "allocation": a}
	if token, issueErr := s.issuePublicToken("receipt", p.ID, 365*24*time.Hour); issueErr == nil {
		response["receipt_url"] = strings.TrimRight(s.config.PublicBaseURL, "/") + "/receipt/" + token
	}
	writeJSON(w, 201, response)
}

func (s *Server) listPayments(w http.ResponseWriter, r *http.Request) {
	orgID, _ := pathID(r, "organizationID")
	requestID, _ := pathID(r, "requestID")
	if _, _, _, ok := s.requireOrganizationAccess(w, r, orgID, access.PermissionReadFinancial); !ok {
		return
	}
	v, err := s.runtime.Credit.GetForSupplier(requestID, orgID)
	if err != nil || v.Obligation == nil {
		writeProblem(w, 404, "obligation_not_found", "obligation was not found")
		return
	}
	writeJSON(w, 200, map[string]any{"payments": s.runtime.Payments.List(v.Obligation.ID), "outstanding_kobo": v.Obligation.OutstandingKobo})
}

func (s *Server) reconcilePayments(w http.ResponseWriter, r *http.Request) {
	orgID, _ := pathID(r, "organizationID")
	requestID, _ := pathID(r, "requestID")
	if _, _, _, ok := s.requireOrganizationAccess(w, r, orgID, access.PermissionReadFinancial); !ok {
		return
	}
	v, err := s.runtime.Credit.GetForSupplier(requestID, orgID)
	if err != nil || v.Obligation == nil {
		writeProblem(w, 404, "obligation_not_found", "obligation was not found")
		return
	}
	rebuilt, err := s.runtime.Payments.Rebuild(v.Obligation.ID)
	if err != nil {
		writeProblem(w, 409, "reconciliation_failed", err.Error())
		return
	}
	if refreshed, refreshErr := s.runtime.Credit.GetForSupplier(requestID, orgID); refreshErr == nil && refreshed.Obligation != nil {
		v = refreshed
	}
	paymentsForObligation := s.runtime.Payments.List(v.Obligation.ID)
	transactions := []ledger.Transaction{}
	for _, payment := range paymentsForObligation {
		posted, err := s.runtime.Ledger.GetByReference(payment.ID)
		if err != nil {
			writeProblem(w, 503, "ledger_unavailable", "ledger history is temporarily unavailable")
			return
		}
		transactions = append(transactions, posted...)
	}
	writeJSON(w, 200, map[string]any{"obligation_id": v.Obligation.ID, "cached_outstanding_kobo": v.Obligation.OutstandingKobo, "rebuilt_outstanding_kobo": rebuilt, "payments": paymentsForObligation, "ledger_transactions": transactions})
}

func (s *Server) reversePayment(w http.ResponseWriter, r *http.Request) {
	orgID, _ := pathID(r, "organizationID")
	requestID, _ := pathID(r, "requestID")
	paymentID, _ := pathID(r, "paymentID")
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
	v, err := s.runtime.Credit.GetForSupplier(requestID, orgID)
	if err != nil || v.Obligation == nil {
		writeProblem(w, 404, "obligation_not_found", "obligation was not found")
		return
	}
	p, err := s.runtime.Payments.Get(paymentID)
	if err != nil || p.ObligationID != v.Obligation.ID {
		writeProblem(w, 404, "payment_not_found", "payment was not found")
		return
	}
	p, err = s.runtime.Payments.Reverse(paymentID, user.ID, in.Reason)
	if err != nil {
		writeProblem(w, 409, "payment_reversal_failed", err.Error())
		return
	}
	s.auditCredit(user.ID, orgID, "payment.reversed", paymentID)
	writeJSON(w, 200, map[string]any{"payment": p})
}

func (s *Server) listBuyerPayments(w http.ResponseWriter, r *http.Request) {
	_, user, ok := s.requireAuth(w, r)
	if !ok {
		return
	}
	requestID, _ := pathID(r, "requestID")
	v, err := s.runtime.Credit.GetForBuyer(requestID, user.ID)
	if err != nil || v.Obligation == nil {
		writeProblem(w, 404, "obligation_not_found", "obligation was not found")
		return
	}
	writeJSON(w, 200, map[string]any{"payments": s.runtime.Payments.List(v.Obligation.ID), "outstanding_kobo": v.Obligation.OutstandingKobo})
}
func (s *Server) getBuyerObligation(w http.ResponseWriter, r *http.Request) {
	_, user, ok := s.requireAuth(w, r)
	if !ok {
		return
	}
	obligationID, _ := pathID(r, "obligationID")
	view, err := s.runtime.Credit.GetByObligationForBuyer(obligationID, user.ID)
	if err != nil || view.Obligation == nil {
		writeProblem(w, 404, "obligation_not_found", "Obligation was not found")
		return
	}
	_, items, _ := s.runtime.Schedules.GetForObligation(obligationID)
	writeJSON(w, 200, map[string]any{"view": view, "payments": s.runtime.Payments.List(obligationID), "schedule_items": items, "disputes": s.runtime.Disputes.ListForObligation(obligationID), "payment_claims": s.runtime.PaymentClaims.ListForObligation(r.Context(), obligationID)})
}

func (s *Server) getSchedule(w http.ResponseWriter, r *http.Request) {
	orgID, _ := pathID(r, "organizationID")
	requestID, _ := pathID(r, "requestID")
	if _, _, _, ok := s.requireOrganizationAccess(w, r, orgID, access.PermissionReadFinancial); !ok {
		return
	}
	v, err := s.runtime.Credit.GetForSupplier(requestID, orgID)
	if err != nil || v.Obligation == nil {
		writeProblem(w, 404, "obligation_not_found", "obligation was not found")
		return
	}
	s.runtime.Schedules.Evaluate(time.Now().UTC())
	schedule, items, err := s.runtime.Schedules.GetForObligation(v.Obligation.ID)
	if err != nil {
		writeProblem(w, 404, "schedule_not_found", err.Error())
		return
	}
	writeJSON(w, 200, map[string]any{"schedule": schedule, "items": items})
}

func (s *Server) createSchedule(w http.ResponseWriter, r *http.Request) {
	orgID, _ := pathID(r, "organizationID")
	requestID, _ := pathID(r, "requestID")
	_, user, _, ok := s.requireOrganizationAccess(w, r, orgID, access.PermissionManageFinancial)
	if !ok {
		return
	}
	if !s.requireCSRF(w, r) {
		return
	}
	v, err := s.runtime.Credit.GetForSupplier(requestID, orgID)
	if err != nil || v.Obligation == nil {
		writeProblem(w, 404, "obligation_not_found", "obligation was not found")
		return
	}
	var in scheduleInput
	if err := decodeJSON(w, r, &in); err != nil {
		writeProblem(w, 400, "invalid_request", err.Error())
		return
	}
	custom := make([]schedules.CustomItem, 0, len(in.CustomItems))
	for _, item := range in.CustomItems {
		custom = append(custom, schedules.CustomItem{AmountKobo: ledger.Money(item.AmountKobo), DueDate: item.DueDate})
	}
	if _, existingItems, existingErr := s.runtime.Schedules.GetForObligation(v.Obligation.ID); existingErr == nil {
		for _, item := range existingItems {
			if item.AllocatedKobo > 0 {
				writeProblem(w, 409, "schedule_locked", "schedule has allocated payments")
				return
			}
		}
		if err := s.runtime.Schedules.DeleteIfEmpty(v.Obligation.ID); err != nil {
			writeProblem(w, 409, "schedule_locked", err.Error())
			return
		}
	}
	schedule, items, err := s.runtime.Schedules.Create(schedules.CreateInput{ObligationID: v.Obligation.ID, PrincipalKobo: v.Obligation.PrincipalKobo, ScheduleType: in.ScheduleType, Count: in.Count, InstalmentAmountKobo: ledger.Money(in.InstalmentAmountKobo), StartDate: in.StartDate, DueHour: in.DueHour, DueMinute: in.DueMinute, Timezone: in.Timezone, GraceHours: in.GraceHours, Cadence: in.Cadence, MonthEndPolicy: in.MonthEndPolicy, CustomItems: custom, AllocationPolicy: in.AllocationPolicy})
	if err != nil {
		writeProblem(w, 422, "schedule_invalid", err.Error())
		return
	}
	s.auditCredit(user.ID, orgID, "repayment.schedule.created", schedule.ID)
	writeJSON(w, 201, map[string]any{"schedule": schedule, "items": items})
}

func (s *Server) getBuyerSchedule(w http.ResponseWriter, r *http.Request) {
	_, user, ok := s.requireAuth(w, r)
	if !ok {
		return
	}
	requestID, _ := pathID(r, "requestID")
	v, err := s.runtime.Credit.GetForBuyer(requestID, user.ID)
	if err != nil || v.Obligation == nil {
		writeProblem(w, 404, "obligation_not_found", "obligation was not found")
		return
	}
	s.runtime.Schedules.Evaluate(time.Now().UTC())
	schedule, items, err := s.runtime.Schedules.GetForObligation(v.Obligation.ID)
	if err != nil {
		writeProblem(w, 404, "schedule_not_found", err.Error())
		return
	}
	writeJSON(w, 200, map[string]any{"schedule": schedule, "items": items})
}

func (s *Server) createTradeLine(w http.ResponseWriter, r *http.Request) {
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
	resolver, ok := s.runtime.Mandates.(mandates.TradeLineResolver)
	if !ok {
		writeProblem(w, http.StatusServiceUnavailable, "mandate_verification_unavailable", "trade-line mandate verification is unavailable")
		return
	}
	mandate, err := resolver.ResolveTradeLineMandate(r.Context(), in.MandateID, in.BuyerUserID, in.BuyerBusinessID)
	if err != nil || mandate.Status != mandates.Active {
		writeProblem(w, http.StatusUnprocessableEntity, "mandate_inactive", "an active mandate owned by the selected buyer business is required")
		return
	}
	if mandate.AmountCeiling < in.ApprovedLimitKobo {
		writeProblem(w, http.StatusUnprocessableEntity, "mandate_ceiling_insufficient", "the mandate ceiling does not cover the proposed trade-line limit")
		return
	}
	line, err := s.runtime.TradeLines.CreateLine(tradelines.CreateLineInput{SupplierOrganizationID: orgID, BuyerUserID: in.BuyerUserID, BuyerBusinessID: in.BuyerBusinessID, ApprovedLimitKobo: ledger.Money(in.ApprovedLimitKobo), Cadence: in.Cadence, DefaultGraceHours: in.DefaultGraceHours, StartAt: in.StartAt, EndAt: in.EndAt, MandateID: mandate.ID, MandateActive: true, MandateVerified: true, TermsVersion: in.TermsVersion})
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
	writeJSON(w, 200, map[string]any{"trade_lines": s.runtime.TradeLines.ListForSupplier(orgID)})
}
func (s *Server) getTradeLine(w http.ResponseWriter, r *http.Request) {
	orgID, _ := pathID(r, "organizationID")
	lineID, _ := pathID(r, "lineID")
	if _, _, _, ok := s.requireOrganizationAccess(w, r, orgID, access.PermissionReadOrganization); !ok {
		return
	}
	line, ok := s.runtime.TradeLines.Get(lineID)
	if !ok || line.SupplierOrganizationID != orgID {
		writeProblem(w, 404, "trade_line_not_found", "trade line was not found")
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
	line, exists := s.runtime.TradeLines.Get(lineID)
	if !exists || line.SupplierOrganizationID != orgID {
		writeProblem(w, 404, "trade_line_not_found", "Trade line was not found")
		return
	}
	var input tradeLineLimitInput
	if err := decodeJSON(w, r, &input); err != nil {
		writeProblem(w, 400, "invalid_request", err.Error())
		return
	}
	updated, err := s.runtime.TradeLines.ReduceLimit(lineID, ledger.Money(input.ApprovedLimitKobo), input.ExpectedVersion)
	if err != nil {
		writeProblem(w, 409, "trade_line_limit_change_failed", err.Error())
		return
	}
	s.auditCredit(user.ID, orgID, "trade_line.limit_reduced", lineID)
	writeJSON(w, 200, map[string]any{"trade_line": updated})
}
func (s *Server) reserveDrawdown(w http.ResponseWriter, r *http.Request) {
	orgID, _ := pathID(r, "organizationID")
	lineID, _ := pathID(r, "lineID")
	_, user, _, ok := s.requireOrganizationAccess(w, r, orgID, access.PermissionManageFinancial)
	if !ok {
		return
	}
	if !s.requireCSRF(w, r) {
		return
	}
	line, exists := s.runtime.TradeLines.Get(lineID)
	if !exists || line.SupplierOrganizationID != orgID {
		writeProblem(w, 404, "trade_line_not_found", "trade line was not found")
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
	drawdown, reservation, updated, err := s.runtime.TradeLines.ReserveDrawdown(tradelines.CreateDrawdownInput{LineID: lineID, PrincipalKobo: ledger.Money(in.PrincipalKobo), GoodsDescription: in.GoodsDescription, InvoiceReference: in.InvoiceReference, InvoiceDocumentHash: in.InvoiceDocumentHash, DueDate: in.DueDate, CollectionAt: in.CollectionAt, IdempotencyKey: in.IdempotencyKey, ExpiresAt: in.ExpiresAt})
	if err != nil {
		writeProblem(w, 409, "drawdown_reservation_failed", err.Error())
		return
	}
	s.auditCredit(user.ID, orgID, "trade_line.drawdown_reserved", drawdown.ID)
	_, _ = s.runtime.Notifications.Emit(r.Context(), notifications.Event{ID: "drawdown-confirmation-required:" + drawdown.ID, Type: "TradeLineDrawdownConfirmationRequired", RecipientID: updated.BuyerUserID, Priority: notifications.PriorityCritical, AmountKobo: int64(drawdown.PrincipalKobo), Currency: "NGN", Reference: drawdown.ID, NextAction: "Review and confirm the exact purchase terms", SecurePath: "/buyer/trade-lines"})
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
	drawdown, line, err := s.runtime.TradeLines.ConfirmDrawdown(drawdownID, user.ID, in.AgreementHash)
	if err != nil || line.ID != lineID {
		writeProblem(w, 409, "drawdown_confirmation_failed", errString(err, "drawdown confirmation failed"))
		return
	}
	s.auditCredit(user.ID, "", "trade_line.drawdown_confirmed", drawdown.ID)
	_, _ = s.runtime.Notifications.Emit(r.Context(), notifications.Event{ID: "drawdown-confirmed:" + drawdown.ID, Type: "TradeLineDrawdownConfirmed", RecipientID: line.SupplierOrganizationID, Priority: notifications.PriorityCritical, AmountKobo: int64(drawdown.PrincipalKobo), Currency: "NGN", Reference: drawdown.ID, NextAction: "Release the goods only when ready", SecurePath: "/app/trade-lines/" + line.ID})
	_, _ = s.runtime.Notifications.Emit(r.Context(), notifications.Event{ID: "drawdown-safe-to-release:" + drawdown.ID, Type: "TradeLineDrawdownSafeToRelease", RecipientID: line.SupplierOrganizationID, Priority: notifications.PriorityCritical, AmountKobo: int64(drawdown.PrincipalKobo), Currency: "NGN", Reference: drawdown.ID, NextAction: "Record release evidence when the goods leave", SecurePath: "/app/trade-lines/" + line.ID})
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
	line, exists := s.runtime.TradeLines.Get(lineID)
	if !exists || line.SupplierOrganizationID != orgID {
		writeProblem(w, 404, "trade_line_not_found", "trade line was not found")
		return
	}
	drawdown, updated, err := s.runtime.TradeLines.ReleaseDrawdown(tradelines.ReleaseInput{DrawdownID: drawdownID, SupplierOrganizationID: orgID, ActorID: user.ID, DeliveryMethod: in.DeliveryMethod, Notes: in.Notes, EvidenceReference: in.EvidenceReference})
	if err != nil {
		writeProblem(w, 409, "drawdown_release_failed", err.Error())
		return
	}
	s.auditCredit(user.ID, orgID, "trade_line.drawdown_goods_released", drawdown.ID)
	_, _ = s.runtime.Notifications.Emit(r.Context(), notifications.Event{ID: "drawdown-released:" + drawdown.ID, Type: "TradeLineDrawdownGoodsReleased", RecipientID: updated.BuyerUserID, Priority: notifications.PriorityCritical, AmountKobo: int64(drawdown.PrincipalKobo), Currency: "NGN", Reference: drawdown.ID, NextAction: "Confirm receipt or report an issue", SecurePath: "/buyer/trade-lines"})
	_, _ = s.runtime.Notifications.Emit(r.Context(), notifications.Event{ID: "drawdown-receipt-required:" + drawdown.ID, Type: "TradeLineDrawdownReceiptRequired", RecipientID: updated.BuyerUserID, Priority: notifications.PriorityCritical, AmountKobo: int64(drawdown.PrincipalKobo), Currency: "NGN", Reference: drawdown.ID, NextAction: "Record whether the goods arrived without an issue", SecurePath: "/buyer/trade-lines"})
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
	drawdown, line, err := s.runtime.TradeLines.RecordDrawdownReceipt(tradelines.ReceiptInput{DrawdownID: drawdownID, BuyerUserID: user.ID, State: in.State, IssueReason: in.IssueReason})
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
	_, _ = s.runtime.Notifications.Emit(r.Context(), notifications.Event{ID: "drawdown-receipt:" + drawdown.ID + ":" + in.State, Type: eventType, RecipientID: line.SupplierOrganizationID, Priority: notifications.PriorityCritical, AmountKobo: int64(drawdown.PrincipalKobo), Currency: "NGN", Reference: drawdown.ID, NextAction: "Review the drawdown status", SecurePath: "/app/trade-lines/" + line.ID})
	if in.State == "no_issue" {
		s.auditCredit(user.ID, line.SupplierOrganizationID, "trade_line.drawdown_obligation_activated", drawdown.ID)
		_, _ = s.runtime.Notifications.Emit(r.Context(), notifications.Event{ID: "drawdown-receipt-confirmed:" + drawdown.ID, Type: "TradeLineDrawdownReceiptConfirmed", RecipientID: line.SupplierOrganizationID, Priority: notifications.PriorityCritical, AmountKobo: int64(drawdown.PrincipalKobo), Currency: "NGN", Reference: drawdown.ID, NextAction: "The obligation and schedule are now active", SecurePath: "/app/trade-lines/" + line.ID})
	}
	writeJSON(w, 200, map[string]any{"drawdown": drawdown, "trade_line": line})
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
	line, exists := s.runtime.TradeLines.Get(lineID)
	if !exists || line.SupplierOrganizationID != orgID {
		writeProblem(w, 404, "trade_line_not_found", "trade line was not found")
		return
	}
	drawdown, updated, err := s.runtime.TradeLines.CancelDrawdown(drawdownID, user.ID)
	if err != nil {
		writeProblem(w, 409, "drawdown_cancel_failed", err.Error())
		return
	}
	s.auditCredit(user.ID, orgID, "trade_line.drawdown_cancelled", drawdown.ID)
	_, _ = s.runtime.Notifications.Emit(r.Context(), notifications.Event{ID: "drawdown-cancelled:" + drawdown.ID, Type: "TradeLineDrawdownCancelled", RecipientID: updated.BuyerUserID, Priority: notifications.PriorityCritical, AmountKobo: int64(drawdown.PrincipalKobo), Currency: "NGN", Reference: drawdown.ID, NextAction: "Review the released trade-line capacity", SecurePath: "/buyer/trade-lines"})
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
	line, exists := s.runtime.TradeLines.Get(lineID)
	if !exists || line.BuyerUserID != user.ID {
		writeProblem(w, http.StatusNotFound, "trade_line_not_found", "trade line was not found")
		return
	}
	drawdown, updated, err := s.runtime.TradeLines.CancelDrawdown(drawdownID, user.ID)
	if err != nil {
		writeProblem(w, http.StatusConflict, "drawdown_cancel_failed", err.Error())
		return
	}
	s.auditCredit(user.ID, line.SupplierOrganizationID, "trade_line.drawdown_cancelled", drawdown.ID)
	_, _ = s.runtime.Notifications.Emit(r.Context(), notifications.Event{ID: "drawdown-cancelled:" + drawdown.ID, Type: "TradeLineDrawdownCancelled", RecipientID: line.SupplierOrganizationID, Priority: notifications.PriorityCritical, AmountKobo: int64(drawdown.PrincipalKobo), Currency: "NGN", Reference: drawdown.ID, NextAction: "Review the released capacity", SecurePath: "/app/trade-lines/" + line.ID})
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
	line, exists := s.runtime.TradeLines.Get(lineID)
	if !exists || line.SupplierOrganizationID != orgID {
		writeProblem(w, 404, "trade_line_not_found", "trade line was not found")
		return
	}
	line, err := s.runtime.TradeLines.Suspend(lineID, in.Reason)
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
	line, exists := s.runtime.TradeLines.Get(lineID)
	if !exists || line.SupplierOrganizationID != orgID {
		writeProblem(w, 404, "trade_line_not_found", "trade line was not found")
		return
	}
	line, err := s.runtime.TradeLines.Resume(lineID)
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
	line, exists := s.runtime.TradeLines.Get(lineID)
	if !exists || line.SupplierOrganizationID != orgID {
		writeProblem(w, 404, "trade_line_not_found", "trade line was not found")
		return
	}
	statement, err := s.runtime.TradeLines.Statement(lineID)
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
	line, exists := s.runtime.TradeLines.Get(lineID)
	if !exists || line.BuyerUserID != user.ID {
		writeProblem(w, 404, "trade_line_not_found", "trade line was not found")
		return
	}
	statement, err := s.runtime.TradeLines.Statement(lineID)
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
	line, exists := s.runtime.TradeLines.Get(lineID)
	if !exists || line.SupplierOrganizationID != organizationID {
		writeProblem(w, http.StatusNotFound, "trade_line_not_found", "trade line was not found")
		return
	}
	s.renderDrawdownAgreementDocument(w, line, r.PathValue("drawdownID"))
}

func (s *Server) getBuyerDrawdownAgreementDocument(w http.ResponseWriter, r *http.Request) {
	_, user, ok := s.requireAuth(w, r)
	if !ok {
		return
	}
	lineID, _ := pathID(r, "lineID")
	line, exists := s.runtime.TradeLines.Get(lineID)
	if !exists || line.BuyerUserID != user.ID {
		writeProblem(w, http.StatusNotFound, "trade_line_not_found", "trade line was not found")
		return
	}
	s.renderDrawdownAgreementDocument(w, line, r.PathValue("drawdownID"))
}

func (s *Server) renderDrawdownAgreementDocument(w http.ResponseWriter, line tradelines.TradeLine, drawdownID string) {
	statement, err := s.runtime.TradeLines.Statement(line.ID)
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
	w.Header().Set("Content-Disposition", "inline; filename=\"kredit-drawdown-agreement-"+drawdown.ID+".html\"")
	w.Header().Set("X-Agreement-Hash", drawdown.AgreementHash)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(document)
}
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
type decisionInput struct {
	Outcome               string `json:"outcome"`
	ValidPrincipalKobo    int64  `json:"valid_principal_kobo"`
	AdjustmentKobo        int64  `json:"adjustment_kobo"`
	RemainingDisputedKobo int64  `json:"remaining_disputed_kobo"`
	Reason                string `json:"reason"`
}

func (s *Server) collectionEligibility(w http.ResponseWriter, r *http.Request) {
	orgID, _ := pathID(r, "organizationID")
	requestID, _ := pathID(r, "requestID")
	if _, _, _, ok := s.requireOrganizationAccess(w, r, orgID, access.PermissionReadFinancial); !ok {
		return
	}
	v, err := s.runtime.Credit.GetForSupplier(requestID, orgID)
	if err != nil || v.Obligation == nil {
		writeProblem(w, 404, "obligation_not_found", "obligation was not found")
		return
	}
	eligibility, err := s.runtime.Collections.Eligibility(v.Obligation.ID, time.Now().UTC())
	if err != nil {
		writeProblem(w, 409, "eligibility_failed", err.Error())
		return
	}
	writeJSON(w, 200, eligibility)
}

func (s *Server) startCollection(w http.ResponseWriter, r *http.Request) {
	orgID, _ := pathID(r, "organizationID")
	requestID, _ := pathID(r, "requestID")
	_, user, _, ok := s.requireOrganizationAccess(w, r, orgID, access.PermissionManageFinancial)
	if !ok {
		return
	}
	if !s.requireSupplierReady(w, orgID, user.ID, "starting a live collection") {
		return
	}
	if !s.requireCSRF(w, r) {
		return
	}
	v, err := s.runtime.Credit.GetForSupplier(requestID, orgID)
	if err != nil || v.Obligation == nil {
		writeProblem(w, 404, "obligation_not_found", "obligation was not found")
		return
	}
	var in collectionStartInput
	if err := decodeJSON(w, r, &in); err != nil {
		writeProblem(w, 400, "invalid_request", err.Error())
		return
	}
	if in.IdempotencyKey == "" {
		in.IdempotencyKey = r.Header.Get("Idempotency-Key")
	}
	attempt, err := s.runtime.Collections.Start(r.Context(), v.Obligation.ID, in.IdempotencyKey, time.Now().UTC())
	if err != nil {
		writeProblem(w, 409, "collection_start_failed", err.Error())
		return
	}
	s.auditCredit(user.ID, orgID, "collection.started", attempt.ID)
	state, _ := s.runtime.Credit.CollectionState(v.Obligation.ID)
	_, _ = s.runtime.Notifications.Emit(r.Context(), notifications.Event{ID: "collection-submitted:" + attempt.ID, Type: "CollectionSubmitted", RecipientID: state.BuyerUserID, Priority: notifications.PriorityCritical, AmountKobo: int64(attempt.RequestedAmountKobo), Currency: state.Currency, Reference: attempt.ID, NextAction: "Review the collection status", SecurePath: "/buyer/credit-requests/" + requestID})
	writeJSON(w, 202, map[string]any{"attempt": attempt})
}

func (s *Server) listCollections(w http.ResponseWriter, r *http.Request) {
	orgID, _ := pathID(r, "organizationID")
	requestID, _ := pathID(r, "requestID")
	if _, _, _, ok := s.requireOrganizationAccess(w, r, orgID, access.PermissionReadFinancial); !ok {
		return
	}
	v, err := s.runtime.Credit.GetForSupplier(requestID, orgID)
	if err != nil || v.Obligation == nil {
		writeProblem(w, 404, "obligation_not_found", "obligation was not found")
		return
	}
	writeJSON(w, 200, map[string]any{"attempts": s.runtime.Collections.ListAttempts(v.Obligation.ID)})
}

func (s *Server) retryCollection(w http.ResponseWriter, r *http.Request) {
	orgID, _ := pathID(r, "organizationID")
	attemptID, _ := pathID(r, "attemptID")
	_, user, _, ok := s.requireOrganizationAccess(w, r, orgID, access.PermissionManageFinancial)
	if !ok {
		return
	}
	if !s.requireCSRF(w, r) {
		return
	}
	attempt, exists := s.runtime.Collections.GetAttempt(attemptID)
	if !exists || !s.runtime.Credit.ObligationBelongsToOrganization(attempt.ObligationID, orgID) {
		writeProblem(w, 404, "collection_not_found", "collection attempt was not found")
		return
	}
	retried, err := s.runtime.Collections.Retry(r.Context(), attemptID, time.Now().UTC())
	if err != nil {
		writeProblem(w, 409, "collection_retry_failed", err.Error())
		return
	}
	s.auditCredit(user.ID, orgID, "collection.retried", retried.ID)
	writeJSON(w, 202, map[string]any{"attempt": retried})
}

func (s *Server) reconcileCollection(w http.ResponseWriter, r *http.Request) {
	orgID, _ := pathID(r, "organizationID")
	attemptID, _ := pathID(r, "attemptID")
	_, user, _, ok := s.requireOrganizationAccess(w, r, orgID, access.PermissionManageFinancial)
	if !ok {
		return
	}
	if !s.requireCSRF(w, r) {
		return
	}
	attempt, exists := s.runtime.Collections.GetAttempt(attemptID)
	if !exists || !s.runtime.Credit.ObligationBelongsToOrganization(attempt.ObligationID, orgID) {
		writeProblem(w, 404, "collection_not_found", "collection attempt was not found")
		return
	}
	resolved, err := s.runtime.Collections.Reconcile(r.Context(), attemptID)
	if err != nil {
		writeProblem(w, 409, "collection_reconcile_failed", err.Error())
		return
	}
	s.auditCredit(user.ID, orgID, "collection.reconciled", attemptID)
	writeJSON(w, 200, map[string]any{"attempt": resolved})
}

func (s *Server) collectionWebhook(w http.ResponseWriter, r *http.Request) {
	var event collections.Webhook
	if err := decodeJSON(w, r, &event); err != nil {
		writeProblem(w, 400, "invalid_webhook", err.Error())
		return
	}
	attempt, err := s.runtime.Collections.ProcessWebhook(r.Context(), event)
	if err != nil {
		writeProblem(w, 400, "collection_webhook_rejected", err.Error())
		return
	}
	writeJSON(w, 200, map[string]any{"attempt": attempt})
}

func (s *Server) openDispute(w http.ResponseWriter, r *http.Request) {
	orgID, _ := pathID(r, "organizationID")
	requestID, _ := pathID(r, "requestID")
	_, user, _, ok := s.requireOrganizationAccess(w, r, orgID, access.PermissionReadFinancial)
	if !ok {
		return
	}
	if !s.requireCSRF(w, r) {
		return
	}
	v, err := s.runtime.Credit.GetForSupplier(requestID, orgID)
	if err != nil || v.Obligation == nil {
		writeProblem(w, 404, "obligation_not_found", "obligation was not found")
		return
	}
	var in disputeInput
	if err := decodeJSON(w, r, &in); err != nil {
		writeProblem(w, 400, "invalid_request", err.Error())
		return
	}
	dispute, err := s.runtime.Disputes.Open(disputes.OpenInput{ObligationID: v.Obligation.ID, OpenedBy: user.ID, DisputedAmountKobo: ledger.Money(in.DisputedAmountKobo), Reason: in.Reason, Explanation: in.Explanation, CollectionEffect: in.CollectionEffect})
	if err != nil {
		writeProblem(w, 422, "dispute_invalid", err.Error())
		return
	}
	s.auditCredit(user.ID, orgID, "dispute.opened", dispute.ID)
	writeJSON(w, 201, map[string]any{"dispute": dispute})
}
func (s *Server) openBuyerDispute(w http.ResponseWriter, r *http.Request) {
	_, user, ok := s.requireAuth(w, r)
	if !ok {
		return
	}
	if !s.requireCSRF(w, r) {
		return
	}
	requestID, _ := pathID(r, "requestID")
	v, err := s.runtime.Credit.GetForBuyer(requestID, user.ID)
	if err != nil || v.Obligation == nil {
		writeProblem(w, 404, "obligation_not_found", "obligation was not found")
		return
	}
	var in disputeInput
	if err := decodeJSON(w, r, &in); err != nil {
		writeProblem(w, 400, "invalid_request", err.Error())
		return
	}
	dispute, err := s.runtime.Disputes.Open(disputes.OpenInput{ObligationID: v.Obligation.ID, OpenedBy: user.ID, DisputedAmountKobo: ledger.Money(in.DisputedAmountKobo), Reason: in.Reason, Explanation: in.Explanation, CollectionEffect: in.CollectionEffect})
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
	dispute, evidence, decisions, err := s.runtime.Disputes.Get(disputeID)
	if err != nil || dispute.BuyerUserID != user.ID {
		writeProblem(w, 404, "dispute_not_found", "dispute was not found")
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
	dispute, _, _, err := s.runtime.Disputes.Get(disputeID)
	if err != nil || dispute.BuyerUserID != user.ID {
		writeProblem(w, 404, "dispute_not_found", "dispute was not found")
		return
	}
	var in evidenceInput
	if err := decodeJSON(w, r, &in); err != nil {
		writeProblem(w, 400, "invalid_request", err.Error())
		return
	}
	evidence, err := s.runtime.Disputes.AddEvidence(disputeID, user.ID, in.DocumentID, in.Statement)
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
	writeJSON(w, 200, map[string]any{"disputes": s.runtime.Disputes.ListForOrganization(orgID)})
}
func (s *Server) getDispute(w http.ResponseWriter, r *http.Request) {
	orgID, _ := pathID(r, "organizationID")
	disputeID, _ := pathID(r, "disputeID")
	if _, _, _, ok := s.requireOrganizationAccess(w, r, orgID, access.PermissionReadFinancial); !ok {
		return
	}
	dispute, evidence, decisions, err := s.runtime.Disputes.Get(disputeID)
	if err != nil || dispute.SupplierOrganizationID != orgID {
		writeProblem(w, 404, "dispute_not_found", "dispute was not found")
		return
	}
	writeJSON(w, 200, map[string]any{"dispute": dispute, "evidence": evidence, "decisions": decisions})
}
func (s *Server) addDisputeEvidence(w http.ResponseWriter, r *http.Request) {
	orgID, _ := pathID(r, "organizationID")
	disputeID, _ := pathID(r, "disputeID")
	_, user, _, ok := s.requireOrganizationAccess(w, r, orgID, access.PermissionReadFinancial)
	if !ok {
		return
	}
	if !s.requireCSRF(w, r) {
		return
	}
	dispute, _, _, err := s.runtime.Disputes.Get(disputeID)
	if err != nil || dispute.SupplierOrganizationID != orgID {
		writeProblem(w, 404, "dispute_not_found", "dispute was not found")
		return
	}
	var in evidenceInput
	if err := decodeJSON(w, r, &in); err != nil {
		writeProblem(w, 400, "invalid_request", err.Error())
		return
	}
	evidence, err := s.runtime.Disputes.AddEvidence(disputeID, user.ID, in.DocumentID, in.Statement)
	if err != nil {
		writeProblem(w, 422, "evidence_invalid", err.Error())
		return
	}
	s.auditCredit(user.ID, orgID, "dispute.evidence_added", disputeID)
	writeJSON(w, 201, map[string]any{"evidence": evidence})
}
func (s *Server) decideDispute(w http.ResponseWriter, r *http.Request) {
	orgID, _ := pathID(r, "organizationID")
	disputeID, _ := pathID(r, "disputeID")
	_, user, _, ok := s.requireOrganizationAccess(w, r, orgID, access.PermissionManageFinancial)
	if !ok {
		return
	}
	if !s.requireCSRF(w, r) {
		return
	}
	dispute, _, _, err := s.runtime.Disputes.Get(disputeID)
	if err != nil || dispute.SupplierOrganizationID != orgID {
		writeProblem(w, 404, "dispute_not_found", "dispute was not found")
		return
	}
	var in decisionInput
	if err := decodeJSON(w, r, &in); err != nil {
		writeProblem(w, 400, "invalid_request", err.Error())
		return
	}
	updated, decision, err := s.runtime.Disputes.Decide(disputes.DecideInput{DisputeID: disputeID, ReviewerID: user.ID, Outcome: in.Outcome, ValidPrincipalKobo: ledger.Money(in.ValidPrincipalKobo), AdjustmentKobo: ledger.Money(in.AdjustmentKobo), RemainingDisputedKobo: ledger.Money(in.RemainingDisputedKobo), Reason: in.Reason})
	if err != nil {
		writeProblem(w, 409, "dispute_decision_failed", err.Error())
		return
	}
	s.auditCredit(user.ID, orgID, "dispute.decided", disputeID)
	writeJSON(w, 200, map[string]any{"dispute": updated, "decision": decision})
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
	v, err := s.runtime.Credit.GetForSupplier(requestID, orgID)
	if err != nil || v.Obligation == nil {
		writeProblem(w, 404, "obligation_not_found", "obligation was not found")
		return
	}
	var in operationInput
	if err := decodeJSON(w, r, &in); err != nil {
		writeProblem(w, 400, "invalid_request", err.Error())
		return
	}
	if ledger.Money(in.AmountKobo) > v.Obligation.OutstandingKobo {
		writeProblem(w, 422, "operation_invalid", "write-off exceeds outstanding balance")
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
	v, err := s.runtime.Credit.GetForSupplier(requestID, orgID)
	if err != nil || v.Obligation == nil {
		writeProblem(w, 404, "obligation_not_found", "obligation was not found")
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
func (s *Server) listOperationActions(w http.ResponseWriter, r *http.Request) {
	orgID, _ := pathID(r, "organizationID")
	if _, _, _, ok := s.requireOrganizationAccess(w, r, orgID, access.PermissionReadAudit); !ok {
		return
	}
	writeJSON(w, 200, map[string]any{"actions": s.runtime.Operations.ListForOrganization(orgID)})
}

func (s *Server) whatsappWebhook(w http.ResponseWriter, r *http.Request) {
	var event whatsapp.Event
	if err := decodeJSON(w, r, &event); err != nil {
		writeProblem(w, 400, "invalid_webhook", err.Error())
		return
	}
	command, err := s.runtime.WhatsApp.Handle(r.Context(), event)
	if err != nil {
		writeProblem(w, 401, "whatsapp_webhook_rejected", err.Error())
		return
	}
	response := map[string]any{"command": command}
	if command.RequiresConfirmation {
		response["confirmation_summary"] = whatsapp.ConfirmationSummary(command)
		response["secure_action_required"] = true
	}
	writeJSON(w, 200, response)
}
func (s *Server) myNotifications(w http.ResponseWriter, r *http.Request) {
	_, user, ok := s.requireAuth(w, r)
	if !ok {
		return
	}
	writeJSON(w, 200, map[string]any{"notifications": s.runtime.Notifications.ListDeliveries(user.ID)})
}
