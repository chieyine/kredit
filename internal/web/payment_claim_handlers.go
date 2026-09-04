package web

import (
	"net/http"
	"strings"
	"time"

	"kredit/internal/access"
	"kredit/internal/audit"
	"kredit/internal/ledger"
	"kredit/internal/notifications"
	"kredit/internal/paymentclaims"
	"kredit/internal/publictoken"
)

type paymentClaimInput struct {
	AmountKobo          int64     `json:"amount_kobo"`
	PaidAt              time.Time `json:"paid_at"`
	SourceAccountMasked string    `json:"source_account_masked"`
	TransferReference   string    `json:"transfer_reference"`
	EvidenceDocumentID  string    `json:"evidence_document_id"`
}

type paymentClaimDecisionInput struct {
	Decision string `json:"decision"`
	Reason   string `json:"reason"`
}

func (s *Server) createBuyerPaymentClaim(w http.ResponseWriter, r *http.Request) {
	enabled := s.runtime.PaymentClaimsEnabled
	if s.runtime.BusinessPolicies != nil {
		policy, err := s.runtime.BusinessPolicies.Read(r.Context())
		if err != nil {
			writeProblem(w, 503, "policy_unavailable", "Business policies could not be checked")
			return
		}
		enabled = policy.Values.PaymentClaims
	}
	if !enabled {
		writeProblem(w, http.StatusConflict, "feature_disabled", "Off-platform payment claims are disabled")
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
	view, err := s.runtime.Credit.GetForBuyer(requestID, user.ID)
	if err != nil || view.Obligation == nil {
		writeProblem(w, 404, "obligation_not_found", "Active obligation was not found")
		return
	}
	var input paymentClaimInput
	if err := decodeJSON(w, r, &input); err != nil {
		writeProblem(w, 400, "invalid_request", err.Error())
		return
	}
	claim, err := s.runtime.PaymentClaims.Create(r.Context(), paymentclaims.CreateInput{ObligationID: view.Obligation.ID, BuyerUserID: user.ID, AmountKobo: ledger.Money(input.AmountKobo), PaidAt: input.PaidAt, SourceAccountMasked: input.SourceAccountMasked, TransferReference: input.TransferReference, EvidenceDocumentID: input.EvidenceDocumentID, IdempotencyKey: r.Header.Get("Idempotency-Key")})
	if err != nil {
		writeProblem(w, 409, "payment_claim_failed", err.Error())
		return
	}
	s.runtime.Audit.Append(audit.Event{ActorUserID: user.ID, OrganizationID: claim.SupplierOrganizationID, Action: "payment_claim.opened", ResourceType: "payment_claim", ResourceID: claim.ID, Outcome: "success"})
	_, _ = s.runtime.EmitNotification(r.Context(), notifications.Event{ID: "payment-claim:" + claim.ID, Type: "BuyerPaymentClaimed", OrganizationID: claim.SupplierOrganizationID, Priority: notifications.PriorityCritical, AmountKobo: int64(claim.AmountKobo), Currency: claim.Currency, Reference: claim.ID, NextAction: "Review the payment evidence", SecurePath: "/app/payments"})
	writeJSON(w, 201, map[string]any{"payment_claim": claim})
}

func (s *Server) listBuyerPaymentClaims(w http.ResponseWriter, r *http.Request) {
	_, user, ok := s.requireAuth(w, r)
	if !ok {
		return
	}
	financialRows1, readErr1 := s.runtime.readPaymentClaimsForBuyer(r.Context(), user.ID)
	if financialReadError(w, readErr1) {
		return
	}
	writeJSON(w, 200, map[string]any{"payment_claims": financialRows1})
}

func (s *Server) listPaymentClaims(w http.ResponseWriter, r *http.Request) {
	orgID, _ := pathID(r, "organizationID")
	requestID, _ := pathID(r, "requestID")
	if _, _, _, ok := s.requireOrganizationAccess(w, r, orgID, access.PermissionReadFinancial); !ok {
		return
	}
	view, err := s.runtime.Credit.GetForSupplier(requestID, orgID)
	if err != nil || view.Obligation == nil {
		writeProblem(w, 404, "obligation_not_found", "Obligation was not found")
		return
	}
	financialRows2, readErr2 := s.runtime.readPaymentClaimsForObligation(r.Context(), view.Obligation.ID)
	if financialReadError(w, readErr2) {
		return
	}
	writeJSON(w, 200, map[string]any{"payment_claims": financialRows2})
}

func (s *Server) listOrganizationPaymentClaims(w http.ResponseWriter, r *http.Request) {
	orgID, _ := pathID(r, "organizationID")
	if _, _, _, ok := s.requireOrganizationAccess(w, r, orgID, access.PermissionReadFinancial); !ok {
		return
	}
	financialRows3, readErr3 := s.runtime.readPaymentClaimsForSupplier(r.Context(), orgID)
	if financialReadError(w, readErr3) {
		return
	}
	writeJSON(w, 200, map[string]any{"payment_claims": financialRows3})
}

func (s *Server) decidePaymentClaim(w http.ResponseWriter, r *http.Request) {
	orgID, _ := pathID(r, "organizationID")
	claimID, _ := pathID(r, "claimID")
	_, user, _, ok := s.requireOrganizationAccess(w, r, orgID, access.PermissionManageFinancial)
	if !ok {
		return
	}
	if !s.requireCSRF(w, r) {
		return
	}
	claim, err := s.runtime.PaymentClaims.Get(r.Context(), claimID)
	if err != nil || claim.SupplierOrganizationID != orgID {
		writeProblem(w, 404, "payment_claim_not_found", "Payment claim was not found")
		return
	}
	var input paymentClaimDecisionInput
	if err := decodeJSON(w, r, &input); err != nil {
		writeProblem(w, 400, "invalid_request", err.Error())
		return
	}
	decision := strings.ToLower(strings.TrimSpace(input.Decision))
	if decision == paymentclaims.Confirmed {
		claim, err = s.runtime.PaymentClaims.Confirm(r.Context(), claim.ID, user.ID, input.Reason, s.runtime.Payments)
	} else {
		claim, err = s.runtime.PaymentClaims.Decide(r.Context(), claim.ID, user.ID, decision, input.Reason, "")
	}
	if err != nil {
		writeProblem(w, 409, "payment_claim_decision_failed", err.Error())
		return
	}
	s.runtime.Audit.Append(audit.Event{ActorUserID: user.ID, OrganizationID: orgID, Action: "payment_claim." + decision, ResourceType: "payment_claim", ResourceID: claim.ID, Outcome: "success"})
	_, _ = s.runtime.EmitNotification(r.Context(), notifications.Event{ID: "payment-claim-decision:" + claim.ID, Type: "PaymentClaimDecision", RecipientID: claim.BuyerUserID, Priority: notifications.PriorityCritical, AmountKobo: int64(claim.AmountKobo), Currency: claim.Currency, Reference: claim.ID, NextAction: "Review the supplier decision", SecurePath: "/buyer/history"})
	paymentID := claim.PaymentID
	response := map[string]any{"payment_claim": claim}
	if paymentID != "" {
		if token, issueErr := s.issuePublicToken("receipt", paymentID, 365*24*time.Hour); issueErr == nil {
			response["receipt_url"] = strings.TrimRight(s.config.PublicBaseURL, "/") + "/receipt/" + token
		}
	}
	writeJSON(w, 200, response)
}

func (s *Server) publicReceipt(w http.ResponseWriter, r *http.Request) {
	token := r.PathValue("token")
	paymentID, err := publictoken.Parse(runtimeDomainKey(s.config.TokenHashKey, s.config.SessionSigningKey, "public-receipts"), token, "receipt", time.Now().UTC())
	if err != nil {
		writeProblem(w, http.StatusGone, "receipt_unavailable", "Receipt link is invalid or expired")
		return
	}
	payment, err := s.runtime.getPayment(r.Context(), paymentID)
	if err != nil {
		writeProblem(w, 404, "receipt_not_found", "Receipt was not found")
		return
	}
	writeJSON(w, 200, map[string]any{"receipt": map[string]any{"reference": payment.ID, "amount_kobo": payment.AmountKobo, "currency": payment.Currency, "source_type": payment.SourceType, "state": payment.State, "paid_at": payment.PaidAt, "recognized_at": payment.RecognizedAt}})
}

func (s *Server) createPaymentLink(w http.ResponseWriter, r *http.Request) {
	_, user, ok := s.requireAuth(w, r)
	if !ok {
		return
	}
	if !s.requireCSRF(w, r) {
		return
	}
	requestID, _ := pathID(r, "requestID")
	view, err := s.runtime.Credit.GetForBuyer(requestID, user.ID)
	if err != nil || view.Obligation == nil {
		writeProblem(w, 404, "obligation_not_found", "Active obligation was not found")
		return
	}
	token, err := s.issuePublicToken("payment", requestID, time.Hour)
	if err != nil {
		writeProblem(w, 500, "payment_link_failed", "Payment link could not be created")
		return
	}
	_, _ = s.runtime.Reports.Track("payment_link.created", requestID, "product_improvement", map[string]string{"surface": "buyer_portal"})
	writeJSON(w, 201, map[string]any{"payment_url": strings.TrimRight(s.config.PublicBaseURL, "/") + "/pay/" + token, "expires_in_seconds": 3600})
}

func (s *Server) publicPaymentIntent(w http.ResponseWriter, r *http.Request) {
	requestID, err := publictoken.Parse(runtimeDomainKey(s.config.TokenHashKey, s.config.SessionSigningKey, "public-payments"), r.PathValue("token"), "payment", time.Now().UTC())
	if err != nil {
		writeProblem(w, 410, "payment_link_unavailable", "Payment link is invalid or expired")
		return
	}
	view, err := s.runtime.Credit.GetPublic(requestID)
	if err != nil || view.Obligation == nil {
		writeProblem(w, 404, "obligation_not_found", "Payment obligation was not found")
		return
	}
	writeJSON(w, 200, map[string]any{"payment_intent": map[string]any{"reference": view.Request.ID, "supplier_name": view.Request.SupplierTradingName, "description": view.Request.GoodsDescription, "amount_kobo": view.Obligation.OutstandingKobo, "currency": view.Obligation.Currency, "payment_status": view.Obligation.PaymentStatus, "provider_action": "Sign in to the buyer portal to choose an approved payment method."}})
}

func (s *Server) issuePublicToken(purpose, id string, duration time.Duration) (string, error) {
	return publictoken.Issue(runtimeDomainKey(s.config.TokenHashKey, s.config.SessionSigningKey, "public-"+purpose+"s"), purpose, id, time.Now().UTC().Add(duration))
}
