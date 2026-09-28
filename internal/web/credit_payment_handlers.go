package web

import (
	"context"
	"kredit/internal/access"
	"kredit/internal/db"
	"kredit/internal/ledger"
	"kredit/internal/notifications"
	"kredit/internal/payments"
	"net/http"
	"strings"
	"time"
)

func (s *Server) recordPayment(w http.ResponseWriter, r *http.Request) {
	orgID, _ := pathID(r, "organizationID")
	requestID, _ := pathID(r, "requestID")
	_, user, _, ok := s.requireOrganizationAccess(w, r, orgID, access.PermissionManageFinancial)
	if !ok {
		return
	}
	r = r.WithContext(db.WithTenantContext(r.Context(), user.ID, orgID))
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
	if in.SourceType == payments.SourceCollected {
		writeProblem(w, http.StatusUnprocessableEntity, "payment_source_invalid", "collection payments can only be recorded from a verified provider collection")
		return
	}
	v, err := s.runtime.getCreditForSupplier(r.Context(), requestID, orgID)
	if err != nil || v.Obligation == nil {
		writeProblem(w, 404, "obligation_not_found", "We could not find an open sale for that.")
		return
	}
	recorder, ok := s.runtime.Payments.(interface {
		RecordContext(context.Context, payments.RecordInput) (payments.Payment, payments.Allocation, error)
	})
	if !ok {
		writeProblem(w, 503, "payment_unavailable", "Payment recording is temporarily unavailable. No payment has been confirmed.")
		return
	}
	p, a, err := recorder.RecordContext(r.Context(), payments.RecordInput{ObligationID: v.Obligation.ID, SourceType: in.SourceType, AmountKobo: ledger.Money(in.AmountKobo), Currency: in.Currency, Provider: in.Provider, ProviderReference: in.ProviderReference, PaidAt: in.PaidAt, RecordedBy: user.ID, IdempotencyKey: in.IdempotencyKey})
	if err != nil {
		writeProblem(w, 409, "payment_record_failed", err.Error())
		return
	}
	s.auditCredit(user.ID, orgID, "payment.recorded", p.ID)
	_, _ = s.runtime.EmitNotification(r.Context(), notifications.Event{ID: "payment-recorded:" + p.ID, Type: "PaymentRecorded", RecipientID: p.BuyerUserID, Priority: notifications.PriorityCritical, AmountKobo: int64(p.AmountKobo), Currency: p.Currency, Reference: p.ID, NextAction: "Review your payment history", SecurePath: "/workspace/purchases/orders/" + requestID})
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
	r = r.WithContext(db.WithOrganizationContext(r.Context(), orgID))
	v, err := s.runtime.getCreditForSupplier(r.Context(), requestID, orgID)
	if err != nil || v.Obligation == nil {
		writeProblem(w, 404, "obligation_not_found", "We could not find that sale.")
		return
	}
	financialRows, readErr := s.runtime.readPayments(r.Context(), v.Obligation.ID)
	if financialReadError(w, readErr) {
		return
	}
	writeJSON(w, 200, map[string]any{"payments": financialRows, "outstanding_kobo": v.Obligation.OutstandingKobo})
}

func (s *Server) reconcilePayments(w http.ResponseWriter, r *http.Request) {
	orgID, _ := pathID(r, "organizationID")
	requestID, _ := pathID(r, "requestID")
	if _, _, _, ok := s.requireOrganizationAccess(w, r, orgID, access.PermissionReadFinancial); !ok {
		return
	}
	r = r.WithContext(db.WithOrganizationContext(r.Context(), orgID))
	v, err := s.runtime.getCreditForSupplier(r.Context(), requestID, orgID)
	if err != nil || v.Obligation == nil {
		writeProblem(w, 404, "obligation_not_found", "We could not find that sale.")
		return
	}
	rebuilder, ok := s.runtime.Payments.(interface {
		RebuildContext(context.Context, string) (ledger.Money, error)
	})
	if !ok {
		writeProblem(w, 503, "payment_unavailable", "Payment reconciliation is temporarily unavailable. Please try again.")
		return
	}
	rebuilt, err := rebuilder.RebuildContext(r.Context(), v.Obligation.ID)
	if err != nil {
		writeProblem(w, 409, "reconciliation_failed", err.Error())
		return
	}
	if refreshed, refreshErr := s.runtime.getCreditForSupplier(r.Context(), requestID, orgID); refreshErr == nil && refreshed.Obligation != nil {
		v = refreshed
	}
	financialRows, readErr := s.runtime.readPayments(r.Context(), v.Obligation.ID)
	if financialReadError(w, readErr) {
		return
	}
	paymentsForObligation := financialRows
	transactions := []ledger.Transaction{}
	for _, payment := range paymentsForObligation {
		posted, err := s.runtime.Ledger.GetByReference(payment.ID)
		if err != nil {
			writeProblem(w, 503, "ledger_unavailable", "Your money history is not available right now. Please try again shortly.")
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
	r = r.WithContext(db.WithTenantContext(r.Context(), user.ID, orgID))
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
	v, err := s.runtime.getCreditForSupplier(r.Context(), requestID, orgID)
	if err != nil || v.Obligation == nil {
		writeProblem(w, 404, "obligation_not_found", "We could not find that sale.")
		return
	}
	contextPayments, ok := s.runtime.Payments.(interface {
		GetContext(context.Context, string) (payments.Payment, error)
		ReverseContext(context.Context, string, string, string) (payments.Payment, error)
	})
	if !ok {
		writeProblem(w, 503, "payment_unavailable", "Payment reversal is temporarily unavailable. Please try again.")
		return
	}
	p, err := contextPayments.GetContext(r.Context(), paymentID)
	if err != nil || p.ObligationID != v.Obligation.ID {
		writeProblem(w, 404, "payment_not_found", "We could not find that payment.")
		return
	}
	p, err = contextPayments.ReverseContext(r.Context(), paymentID, user.ID, in.Reason)
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
	r = r.WithContext(db.WithTenantContext(r.Context(), user.ID, ""))
	requestID, _ := pathID(r, "requestID")
	v, err := s.runtime.Credit.GetForBuyer(requestID, user.ID)
	if err != nil || v.Obligation == nil {
		writeProblem(w, 404, "obligation_not_found", "We could not find that sale.")
		return
	}
	financialRows, readErr := s.runtime.readPayments(r.Context(), v.Obligation.ID)
	if financialReadError(w, readErr) {
		return
	}
	writeJSON(w, 200, map[string]any{"payments": financialRows, "outstanding_kobo": v.Obligation.OutstandingKobo})
}
func (s *Server) getBuyerObligation(w http.ResponseWriter, r *http.Request) {
	_, user, ok := s.requireAuth(w, r)
	if !ok {
		return
	}
	r = r.WithContext(db.WithTenantContext(r.Context(), user.ID, ""))
	obligationID, _ := pathID(r, "obligationID")
	view, err := s.runtime.Credit.GetByObligationForBuyer(obligationID, user.ID)
	if err != nil || view.Obligation == nil {
		writeProblem(w, 404, "obligation_not_found", "We could not find that sale.")
		return
	}
	_, items, scheduleErr := s.runtime.Schedules.ForContext(r.Context()).GetForObligation(obligationID)
	if financialReadError(w, scheduleErr) {
		return
	}
	paymentRows, err := s.runtime.readPayments(r.Context(), obligationID)
	if financialReadError(w, err) {
		return
	}
	obligationDisputes, err := s.runtime.readDisputesForObligation(r.Context(), obligationID)
	if financialReadError(w, err) {
		return
	}
	claims, err := s.runtime.readPaymentClaimsForObligation(r.Context(), obligationID)
	if financialReadError(w, err) {
		return
	}
	notices, noticeErr := s.collectionNotices(r.Context(), user.ID, obligationID)
	if financialReadError(w, noticeErr) {
		return
	}
	writeJSON(w, 200, map[string]any{"view": view, "payments": paymentRows, "schedule_items": items, "disputes": obligationDisputes, "payment_claims": claims, "collection_notices": notices})
}

func (s *Server) getSchedule(w http.ResponseWriter, r *http.Request) {
	orgID, _ := pathID(r, "organizationID")
	requestID, _ := pathID(r, "requestID")
	if _, _, _, ok := s.requireOrganizationAccess(w, r, orgID, access.PermissionReadFinancial); !ok {
		return
	}
	v, err := s.runtime.getCreditForSupplier(r.Context(), requestID, orgID)
	if err != nil || v.Obligation == nil {
		writeProblem(w, 404, "obligation_not_found", "We could not find that sale.")
		return
	}
	schedule, items, err := s.runtime.Schedules.ForContext(r.Context()).GetForObligationAt(v.Obligation.ID, time.Now().UTC())
	if err != nil {
		financialReadError(w, err)
		return
	}
	writeJSON(w, 200, map[string]any{"schedule": schedule, "items": items})
}

func (s *Server) createSchedule(w http.ResponseWriter, r *http.Request) {
	orgID, _ := pathID(r, "organizationID")
	requestID, _ := pathID(r, "requestID")
	if _, _, _, ok := s.requireOrganizationAccess(w, r, orgID, access.PermissionManageFinancial); !ok {
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
	// Activation derives the schedule from the accepted agreement. Replacing it
	// here would bypass the buyer's acceptance and could accelerate a bank debit.
	writeProblem(w, 409, "accepted_schedule_locked", "Payment days you already agreed cannot be changed here. Somebody at Kredit must review it, and your customer must agree to it. Contact support to ask.")
}

func (s *Server) getBuyerSchedule(w http.ResponseWriter, r *http.Request) {
	_, user, ok := s.requireAuth(w, r)
	if !ok {
		return
	}
	requestID, _ := pathID(r, "requestID")
	v, err := s.runtime.Credit.GetForBuyer(requestID, user.ID)
	if err != nil || v.Obligation == nil {
		writeProblem(w, 404, "obligation_not_found", "We could not find that sale.")
		return
	}
	schedule, items, err := s.runtime.Schedules.ForContext(r.Context()).GetForObligationAt(v.Obligation.ID, time.Now().UTC())
	if err != nil {
		financialReadError(w, err)
		return
	}
	writeJSON(w, 200, map[string]any{"schedule": schedule, "items": items})
}
