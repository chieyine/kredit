package web

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"kredit/internal/auth"
	"kredit/internal/config"
	"kredit/internal/idempotency"
	"kredit/internal/observability"
	platformlogging "kredit/internal/platform/logging"

	"go.opentelemetry.io/otel/attribute"
)

type Server struct {
	config       config.Config
	logger       *slog.Logger
	runtime      *Runtime
	startedAt    time.Time
	mux          *http.ServeMux
	rateMu       sync.Mutex
	rate         map[string]rateWindow
	ratePrunedAt time.Time
}

const (
	rateLimitPerMinute = 120
	// rateWindowTTL bounds how long an idle client entry stays in the rate
	// table so the map cannot grow without limit.
	rateWindowTTL = 5 * time.Minute
)

type rateWindow struct {
	started time.Time
	count   int
}

func NewServer(cfg config.Config, logger *slog.Logger) *Server {
	return NewServerWithRuntime(cfg, logger, NewRuntime(cfg))
}

func NewServerWithRuntime(cfg config.Config, logger *slog.Logger, runtime *Runtime) *Server {
	s := &Server{config: cfg, logger: logger, runtime: runtime, startedAt: time.Now().UTC(), mux: http.NewServeMux(), rate: make(map[string]rateWindow)}
	s.registerRoutes()
	return s
}

func (s *Server) Handler() http.Handler {
	handler := http.Handler(s.mux)
	handler = s.withBodyLimit(handler)
	handler = s.withIdempotency(handler)
	handler = s.withRateLimit(handler)
	handler = s.withSecurityHeaders(handler)
	handler = s.withPanicRecovery(handler)
	handler = s.withRequestContext(handler)
	return handler
}

// withIdempotency makes retries safe for every financial mutation. The
// Idempotency-Key is mandatory on those routes; allowing a caller to omit it
// would make a timeout indistinguishable from a failed money operation.
func (s *Server) withIdempotency(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
		if key == "" && requiresIdempotencyKey(r) {
			writeProblem(w, http.StatusBadRequest, "idempotency_key_required", "Idempotency-Key is required for this operation")
			return
		}
		if key == "" || r.Method == http.MethodGet || r.Method == http.MethodHead || r.Method == http.MethodOptions || s.runtime.Idempotency == nil {
			next.ServeHTTP(w, r)
			return
		}
		if r.Body == nil {
			r.Body = io.NopCloser(strings.NewReader(""))
		}
		body, err := io.ReadAll(io.LimitReader(r.Body, 2<<20+1))
		if err != nil {
			writeProblem(w, http.StatusBadRequest, "invalid_request", "request body could not be read")
			return
		}
		if len(body) > 2<<20 {
			writeProblem(w, http.StatusRequestEntityTooLarge, "request_too_large", "request body exceeds the 2 MiB limit")
			return
		}
		r.Body = io.NopCloser(bytes.NewReader(body))
		scope := r.Method + " " + safeRequestPath(r.URL.Path)
		if token := sessionTokenFromRequest(r); token != "" {
			if _, user, err := s.runtime.Auth.SessionFromToken(token); err == nil {
				scope += " user:" + user.ID
			}
		}
		record, existing, err := s.runtime.Idempotency.Reserve(r.Context(), scope, key, idempotency.HashRequest(r.Method, r.URL.Path, body))
		if err != nil {
			if strings.Contains(strings.ToLower(err.Error()), "different request") {
				writeProblem(w, http.StatusConflict, "idempotency_conflict", "idempotency key was reused for a different request")
				return
			}
			writeProblem(w, http.StatusServiceUnavailable, "idempotency_unavailable", "idempotency service is unavailable")
			return
		}
		if existing {
			if record.CompletedAt.IsZero() {
				writeProblem(w, http.StatusConflict, "idempotency_in_progress", "an identical request is already in progress")
				return
			}
			if record.Status == 0 {
				record.Status = http.StatusOK
			}
			w.Header().Set("X-Idempotent-Replay", "true")
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(record.Status)
			_, _ = w.Write(record.ResponseBody)
			return
		}

		recorder := newIdempotencyRecorder(w)
		next.ServeHTTP(recorder, r)
		if err := s.runtime.Idempotency.Complete(r.Context(), scope, key, recorder.status, recorder.body.Bytes()); err != nil {
			s.logger.Error("idempotency response persistence failed", "scope", scope, "error", err)
		}
	})
}

func requiresIdempotencyKey(r *http.Request) bool {
	if r == nil || (r.Method != http.MethodPost && r.Method != http.MethodPut && r.Method != http.MethodPatch && r.Method != http.MethodDelete) {
		return false
	}
	path := r.URL.Path
	if strings.HasSuffix(path, "/credit-requests") || strings.HasSuffix(path, "/buyer-invitations") {
		return true
	}
	for _, suffix := range []string{
		"/credit-requests/", "/payments", "/reverse", "/collection", "/retry", "/reconcile", "/disputes", "/decide", "/write-off", "/fee-waiver", "/drawdowns", "/activate", "/suspend", "/resume", "/exports", "/trade-lines/",
		// These routes mutate financial state or create durable authority and
		// therefore must be safe to replay after a client timeout.
		"/accept", "/release", "/receipt", "/adjust", "/settlement", "/mandates", "/members", "/confirm", "/send", "/evidence", "/schedule", "/documents", "/payment-claims",
		"/onboarding/", "/notification-preferences", "/recovery-codes", "/account-recovery/", "/privacy-requests", "/support-cases", "/product-feedback",
		"/ops/commands", "/ops/cases/", "/ops/team/",
	} {
		if strings.Contains(path, suffix) {
			return true
		}
	}
	return false
}

type idempotencyRecorder struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
	body        bytes.Buffer
}

func newIdempotencyRecorder(writer http.ResponseWriter) *idempotencyRecorder {
	return &idempotencyRecorder{ResponseWriter: writer, status: http.StatusOK}
}

func (r *idempotencyRecorder) WriteHeader(status int) {
	if r.wroteHeader {
		return
	}
	r.wroteHeader = true
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func (r *idempotencyRecorder) Write(body []byte) (int, error) {
	if !r.wroteHeader {
		r.WriteHeader(http.StatusOK)
	}
	_, _ = r.body.Write(body)
	return r.ResponseWriter.Write(body)
}

func (s *Server) registerRoutes() {
	for _, prefix := range []string{"", "/api/v1"} {
		s.mux.HandleFunc(prefix+"/healthz", s.health)
		s.mux.HandleFunc(prefix+"/readyz", s.ready)
		s.mux.HandleFunc(prefix+"/meta", s.meta)
	}
	s.mux.HandleFunc("POST /api/v1/auth/otp/challenges", s.requestOTP)
	s.mux.HandleFunc("POST /api/v1/auth/otp/verify", s.verifyOTP)
	s.mux.HandleFunc("POST /api/v1/auth/logout", s.logout)
	s.mux.HandleFunc("GET /api/v1/secure-link", s.resolveSecureLink)
	s.mux.HandleFunc("GET /api/v1/me", s.me)
	s.mux.HandleFunc("POST /api/v1/me/product-feedback", s.submitProductFeedback)
	s.mux.HandleFunc("POST /api/v1/mfa/totp/enroll", s.enrollTOTP)
	s.mux.HandleFunc("POST /api/v1/mfa/totp/verify", s.verifyTOTP)
	s.mux.HandleFunc("POST /api/v1/me/recovery-codes/regenerate", s.regenerateRecoveryCodes)
	s.mux.HandleFunc("POST /api/v1/account-recovery/requests", s.requestAccountRecovery)
	s.mux.HandleFunc("POST /api/v1/account-recovery/requests/{requestID}/evidence", s.addAccountRecoveryEvidence)
	s.mux.HandleFunc("POST /api/v1/account-recovery/requests/{requestID}/complete", s.completeAccountRecovery)
	s.mux.HandleFunc("POST /api/v1/me/account-recovery/{requestID}/cancel", s.cancelAccountRecovery)
	s.mux.HandleFunc("GET /api/v1/ops/account-recovery", s.listRecoveryReviews)
	s.mux.HandleFunc("POST /api/v1/ops/account-recovery/{requestID}/review", s.reviewAccountRecovery)
	s.mux.HandleFunc("GET /api/v1/me/notification-preferences", s.getNotificationPreferences)
	s.mux.HandleFunc("PUT /api/v1/me/notification-preferences", s.updateNotificationPreferences)
	s.mux.HandleFunc("GET /api/v1/me/privacy-requests", s.listMyPrivacyRequests)
	s.mux.HandleFunc("POST /api/v1/me/privacy-requests", s.createPrivacyRequest)
	s.mux.HandleFunc("GET /api/v1/me/privacy-requests/{requestID}/export", s.downloadPrivacyExport)
	s.mux.HandleFunc("GET /api/v1/ops/privacy-requests", s.listPrivacyReviews)
	s.mux.HandleFunc("POST /api/v1/ops/privacy-requests/{requestID}/decide", s.decidePrivacyRequest)
	s.mux.HandleFunc("POST /api/v1/ops/privacy-requests/{requestID}/complete", s.completePrivacyRequest)
	s.mux.HandleFunc("GET /api/v1/organizations", s.listOrganizations)
	s.mux.HandleFunc("POST /api/v1/organizations", s.createOrganization)
	s.mux.HandleFunc("GET /api/v1/organizations/{organizationID}", s.getOrganization)
	s.mux.HandleFunc("GET /api/v1/organizations/{organizationID}/members", s.listMembers)
	s.mux.HandleFunc("POST /api/v1/organizations/{organizationID}/members", s.inviteMember)
	s.mux.HandleFunc("PATCH /api/v1/organizations/{organizationID}/members/{userID}", s.changeMemberRole)
	s.mux.HandleFunc("GET /api/v1/organizations/{organizationID}/audit-events", s.listAuditEvents)
	s.mux.HandleFunc("GET /api/v1/organizations/{organizationID}/onboarding", s.getSupplierOnboarding)
	s.mux.HandleFunc("POST /api/v1/organizations/{organizationID}/onboarding/contacts/challenges", s.requestOnboardingContactOTP)
	s.mux.HandleFunc("POST /api/v1/organizations/{organizationID}/onboarding/contacts/verify", s.verifyOnboardingContact)
	s.mux.HandleFunc("PATCH /api/v1/organizations/{organizationID}/onboarding/representative", s.updateSupplierRepresentative)
	s.mux.HandleFunc("POST /api/v1/organizations/{organizationID}/onboarding/kyb", s.submitSupplierKYB)
	s.mux.HandleFunc("PUT /api/v1/organizations/{organizationID}/onboarding/settlement", s.updateSupplierSettlement)
	s.mux.HandleFunc("PUT /api/v1/organizations/{organizationID}/onboarding/billing", s.updateSupplierBilling)
	s.mux.HandleFunc("PUT /api/v1/organizations/{organizationID}/onboarding/credit-policy", s.updateSupplierCreditPolicy)
	s.mux.HandleFunc("POST /api/v1/organizations/{organizationID}/onboarding/consents", s.acceptSupplierConsents)
	s.mux.HandleFunc("POST /api/v1/organizations/{organizationID}/documents", s.uploadDocument)
	s.mux.HandleFunc("POST /api/v1/organizations/{organizationID}/documents/upload-slot", s.createDocumentUploadSlot)
	s.mux.HandleFunc("POST /api/v1/organizations/{organizationID}/documents/{documentID}/complete", s.completeDocumentUpload)
	s.mux.HandleFunc("GET /api/v1/organizations/{organizationID}/documents/{documentID}/download", s.documentDownload)
	s.mux.HandleFunc("POST /api/v1/organizations/{organizationID}/support-cases", s.openSupportCase)
	s.mux.HandleFunc("GET /api/v1/organizations/{organizationID}/support-cases", s.listSupportCases)
	s.mux.HandleFunc("PATCH /api/v1/organizations/{organizationID}/support-cases/{caseID}", s.transitionSupportCase)
	s.mux.HandleFunc("POST /api/v1/buyer/relationships/consents", s.recordBuyerConsent)
	s.mux.HandleFunc("GET /api/v1/buyer/relationships/consents", s.listBuyerConsents)
	s.mux.HandleFunc("POST /api/v1/organizations/{organizationID}/buyer-invitations", s.createBuyerInvitation)
	s.mux.HandleFunc("GET /api/v1/buyer-invitations/{token}", s.previewBuyerInvitation)
	s.mux.HandleFunc("POST /api/v1/buyer-invitations/{token}/otp", s.requestBuyerInvitationOTP)
	s.mux.HandleFunc("POST /api/v1/buyer-invitations/{token}/accept", s.acceptBuyerInvitation)
	s.mux.HandleFunc("GET /api/v1/buyer/me", s.buyerPortal)
	s.mux.HandleFunc("GET /api/v1/buyer/credit-requests", s.listBuyerCreditRequests)
	s.mux.HandleFunc("GET /api/v1/buyer/mandates", s.listBuyerMandates)
	s.mux.HandleFunc("POST /api/v1/buyer/mandates/{mandateID}/cancel", s.cancelBuyerMandate)
	s.mux.HandleFunc("POST /api/v1/buyer/mandates/{mandateID}/restore", s.restoreBuyerMandate)
	s.mux.HandleFunc("GET /api/v1/buyer/trade-lines", s.listBuyerTradeLines)
	s.mux.HandleFunc("GET /api/v1/buyer/trade-lines/{lineID}/statement", s.buyerTradeLineStatement)
	s.mux.HandleFunc("GET /api/v1/buyer/trade-lines/{lineID}/drawdowns/{drawdownID}/agreement-document", s.getBuyerDrawdownAgreementDocument)
	s.mux.HandleFunc("POST /api/v1/buyer/trade-lines/{lineID}/drawdowns/{drawdownID}/cancel", s.cancelBuyerDrawdown)
	s.mux.HandleFunc("GET /api/v1/buyer/disputes", s.listBuyerDisputes)
	s.mux.HandleFunc("GET /api/v1/buyer/disputes/{disputeID}", s.getBuyerDispute)
	s.mux.HandleFunc("POST /api/v1/buyer/disputes/{disputeID}/evidence", s.addBuyerDisputeEvidence)
	s.mux.HandleFunc("POST /api/v1/organizations/{organizationID}/credit-requests", s.createCreditRequest)
	s.mux.HandleFunc("GET /api/v1/organizations/{organizationID}/credit-requests", s.listCreditRequests)
	s.mux.HandleFunc("GET /api/v1/organizations/{organizationID}/payments", s.listOrganizationPayments)
	s.mux.HandleFunc("GET /api/v1/organizations/{organizationID}/collections", s.listOrganizationCollections)
	s.mux.HandleFunc("GET /api/v1/organizations/{organizationID}/overdue", s.listOrganizationOverdue)
	s.mux.HandleFunc("GET /api/v1/organizations/{organizationID}/customers", s.listOrganizationCustomers)
	s.mux.HandleFunc("GET /api/v1/organizations/{organizationID}/credit-requests/{requestID}", s.getCreditRequest)
	s.mux.HandleFunc("PATCH /api/v1/organizations/{organizationID}/credit-requests/{requestID}", s.updateDraftCreditRequest)
	s.mux.HandleFunc("POST /api/v1/organizations/{organizationID}/credit-requests/{requestID}/send", s.sendCreditRequest)
	s.mux.HandleFunc("POST /api/v1/organizations/{organizationID}/credit-requests/{requestID}/cancel", s.cancelCreditRequest)
	s.mux.HandleFunc("POST /api/v1/organizations/{organizationID}/credit-requests/{requestID}/release", s.releaseCreditRequest)
	s.mux.HandleFunc("GET /api/v1/buyer/credit-requests/{requestID}", s.getBuyerCreditRequest)
	s.mux.HandleFunc("GET /api/v1/buyer/credit-requests/{requestID}/agreement", s.getBuyerAgreement)
	s.mux.HandleFunc("POST /api/v1/buyer/credit-requests/{requestID}/mandate", s.authorizeCreditMandate)
	s.mux.HandleFunc("POST /api/v1/buyer/credit-requests/{requestID}/accept", s.acceptCreditRequest)
	s.mux.HandleFunc("POST /api/v1/buyer/credit-requests/{requestID}/decline", s.declineCreditRequest)
	s.mux.HandleFunc("POST /api/v1/buyer/credit-requests/{requestID}/receipt", s.recordCreditReceipt)
	s.mux.HandleFunc("POST /api/v1/organizations/{organizationID}/credit-requests/{requestID}/payments", s.recordPayment)
	s.mux.HandleFunc("GET /api/v1/organizations/{organizationID}/credit-requests/{requestID}/payments", s.listPayments)
	s.mux.HandleFunc("GET /api/v1/organizations/{organizationID}/credit-requests/{requestID}/reconciliation", s.reconcilePayments)
	s.mux.HandleFunc("GET /api/v1/organizations/{organizationID}/credit-requests/{requestID}/agreement-document", s.getSupplierAgreementDocument)
	s.mux.HandleFunc("POST /api/v1/organizations/{organizationID}/credit-requests/{requestID}/payments/{paymentID}/reverse", s.reversePayment)
	s.mux.HandleFunc("GET /api/v1/buyer/credit-requests/{requestID}/payments", s.listBuyerPayments)
	s.mux.HandleFunc("GET /api/v1/buyer/credit-requests/{requestID}/agreement-document", s.getBuyerAgreementDocument)
	s.mux.HandleFunc("GET /api/v1/buyer/obligations/{obligationID}", s.getBuyerObligation)
	s.mux.HandleFunc("POST /api/v1/buyer/credit-requests/{requestID}/payment-claims", s.createBuyerPaymentClaim)
	s.mux.HandleFunc("GET /api/v1/buyer/payment-claims", s.listBuyerPaymentClaims)
	s.mux.HandleFunc("GET /api/v1/organizations/{organizationID}/credit-requests/{requestID}/payment-claims", s.listPaymentClaims)
	s.mux.HandleFunc("GET /api/v1/organizations/{organizationID}/payment-claims", s.listOrganizationPaymentClaims)
	s.mux.HandleFunc("POST /api/v1/organizations/{organizationID}/payment-claims/{claimID}/decide", s.decidePaymentClaim)
	s.mux.HandleFunc("GET /api/v1/public/receipts/{token}", s.publicReceipt)
	s.mux.HandleFunc("POST /api/v1/buyer/credit-requests/{requestID}/payment-link", s.createPaymentLink)
	s.mux.HandleFunc("GET /api/v1/public/payment-intents/{token}", s.publicPaymentIntent)
	s.mux.HandleFunc("GET /api/v1/organizations/{organizationID}/credit-requests/{requestID}/schedule", s.getSchedule)
	s.mux.HandleFunc("POST /api/v1/organizations/{organizationID}/credit-requests/{requestID}/schedule", s.createSchedule)
	s.mux.HandleFunc("GET /api/v1/buyer/credit-requests/{requestID}/schedule", s.getBuyerSchedule)
	s.mux.HandleFunc("POST /api/v1/organizations/{organizationID}/trade-lines", s.createTradeLine)
	s.mux.HandleFunc("GET /api/v1/organizations/{organizationID}/trade-lines", s.listTradeLines)
	s.mux.HandleFunc("GET /api/v1/organizations/{organizationID}/trade-lines/{lineID}", s.getTradeLine)
	s.mux.HandleFunc("GET /api/v1/organizations/{organizationID}/trade-lines/{lineID}/drawdowns/{drawdownID}/agreement-document", s.getSupplierDrawdownAgreementDocument)
	s.mux.HandleFunc("PATCH /api/v1/organizations/{organizationID}/trade-lines/{lineID}", s.reduceTradeLineLimit)
	s.mux.HandleFunc("POST /api/v1/organizations/{organizationID}/trade-lines/{lineID}/drawdowns", s.reserveDrawdown)
	s.mux.HandleFunc("POST /api/v1/organizations/{organizationID}/trade-lines/{lineID}/drawdowns/{drawdownID}/release", s.releaseDrawdown)
	s.mux.HandleFunc("POST /api/v1/organizations/{organizationID}/trade-lines/{lineID}/drawdowns/{drawdownID}/cancel", s.cancelDrawdown)
	s.mux.HandleFunc("POST /api/v1/organizations/{organizationID}/trade-lines/{lineID}/suspend", s.suspendTradeLine)
	s.mux.HandleFunc("POST /api/v1/organizations/{organizationID}/trade-lines/{lineID}/resume", s.resumeTradeLine)
	s.mux.HandleFunc("GET /api/v1/organizations/{organizationID}/trade-lines/{lineID}/statement", s.tradeLineStatement)
	s.mux.HandleFunc("POST /api/v1/buyer/trade-lines/{lineID}/drawdowns/{drawdownID}/confirm", s.confirmDrawdown)
	s.mux.HandleFunc("POST /api/v1/buyer/trade-lines/{lineID}/drawdowns/{drawdownID}/receipt", s.receiptDrawdown)
	s.mux.HandleFunc("GET /api/v1/organizations/{organizationID}/credit-requests/{requestID}/collection/eligibility", s.collectionEligibility)
	s.mux.HandleFunc("POST /api/v1/organizations/{organizationID}/credit-requests/{requestID}/collection", s.startCollection)
	s.mux.HandleFunc("GET /api/v1/organizations/{organizationID}/credit-requests/{requestID}/collections", s.listCollections)
	s.mux.HandleFunc("POST /api/v1/organizations/{organizationID}/collections/{attemptID}/retry", s.retryCollection)
	s.mux.HandleFunc("POST /api/v1/organizations/{organizationID}/collections/{attemptID}/reconcile", s.reconcileCollection)
	s.mux.HandleFunc("POST /api/v1/webhooks/collection/{provider}", s.collectionWebhook)
	s.mux.HandleFunc("POST /api/v1/organizations/{organizationID}/credit-requests/{requestID}/disputes", s.openDispute)
	s.mux.HandleFunc("GET /api/v1/organizations/{organizationID}/disputes", s.listDisputes)
	s.mux.HandleFunc("GET /api/v1/organizations/{organizationID}/disputes/{disputeID}", s.getDispute)
	s.mux.HandleFunc("POST /api/v1/organizations/{organizationID}/disputes/{disputeID}/evidence", s.addDisputeEvidence)
	s.mux.HandleFunc("POST /api/v1/organizations/{organizationID}/disputes/{disputeID}/decide", s.decideDispute)
	s.mux.HandleFunc("POST /api/v1/buyer/credit-requests/{requestID}/disputes", s.openBuyerDispute)
	s.mux.HandleFunc("POST /api/v1/organizations/{organizationID}/credit-requests/{requestID}/write-off", s.writeOffObligation)
	s.mux.HandleFunc("POST /api/v1/organizations/{organizationID}/credit-requests/{requestID}/fee-waiver", s.waiveObligationFee)
	s.mux.HandleFunc("GET /api/v1/organizations/{organizationID}/operations", s.listOperationActions)
	s.mux.HandleFunc("POST /api/v1/webhooks/messaging/whatsapp", s.whatsappWebhook)
	s.mux.HandleFunc("GET /api/v1/me/notifications", s.myNotifications)
	s.mux.HandleFunc("GET /api/v1/organizations/{organizationID}/reports/receivables", s.reportReceivables)
	s.mux.HandleFunc("GET /api/v1/organizations/{organizationID}/provider-status", s.providerStatus)
	s.mux.HandleFunc("GET /api/v1/organizations/{organizationID}/readiness", s.readinessStatus)
	s.mux.HandleFunc("GET /api/v1/ops/metrics", s.metricsStatus)
	s.mux.HandleFunc("GET /api/v1/ops/metrics/prometheus", s.metricsPrometheus)
	s.mux.HandleFunc("GET /api/v1/ops/overview", s.operationsOverview)
	s.mux.HandleFunc("GET /api/v1/ops/analytics/scorecard", s.operationsAnalyticsScorecard)
	s.mux.HandleFunc("GET /api/v1/ops/jobs", s.operationsJobs)
	s.mux.HandleFunc("GET /api/v1/ops/provider-events", s.operationsProviderEvents)
	s.mux.HandleFunc("POST /api/v1/ops/commands/preview", s.previewOperationsCommand)
	s.mux.HandleFunc("POST /api/v1/ops/commands", s.executeOperationsCommand)
	s.mux.HandleFunc("GET /api/v1/ops/diagnostics", s.operationsDiagnostics)
	s.mux.HandleFunc("GET /api/v1/ops/search", s.operationsSearch)
	s.mux.HandleFunc("GET /api/v1/ops/users", s.operationsUsers)
	s.mux.HandleFunc("GET /api/v1/ops/organizations", s.operationsOrganizations)
	s.mux.HandleFunc("GET /api/v1/ops/money", s.operationsMoney)
	s.mux.HandleFunc("GET /api/v1/ops/cases", s.operationsCases)
	s.mux.HandleFunc("PATCH /api/v1/ops/cases/{caseID}", s.transitionOperationsCase)
	s.mux.HandleFunc("GET /api/v1/ops/disputes", s.operationsDisputes)
	s.mux.HandleFunc("POST /api/v1/ops/disputes/{disputeID}/decide", s.decideOperationsDispute)
	s.mux.HandleFunc("GET /api/v1/ops/team", s.operationsTeam)
	s.mux.HandleFunc("POST /api/v1/ops/team/{userID}/roles", s.grantOperationsRole)
	s.mux.HandleFunc("DELETE /api/v1/ops/team/roles/{assignmentID}", s.revokeOperationsRole)
	s.mux.HandleFunc("GET /api/v1/ops/audit", s.operationsAudit)
	s.mux.HandleFunc("GET /api/v1/ops/cases/{caseID}", s.operationsCase)
	s.mux.HandleFunc("GET /api/v1/ops/disputes/{disputeID}", s.operationsDispute)
	s.mux.HandleFunc("GET /api/v1/organizations/{organizationID}/reports/ageing", s.reportAgeing)
	s.mux.HandleFunc("GET /api/v1/organizations/{organizationID}/reports/fees", s.reportFees)
	s.mux.HandleFunc("POST /api/v1/organizations/{organizationID}/reports/exports", s.exportReceivables)
	s.mux.HandleFunc("GET /api/v1/buyer/history", s.buyerHistory)
	s.mux.HandleFunc("GET /api/v1/organizations/{organizationID}/customers/{buyerUserID}/history", s.supplierCustomerHistory)
	s.mux.HandleFunc("GET /api/v1/organizations/{organizationID}/customers/{buyerUserID}/statement", s.supplierCustomerStatement)
	s.mux.HandleFunc("POST /api/v1/buyer/history/corrections", s.openCorrection)
	s.mux.HandleFunc("GET /api/v1/organizations/{organizationID}/corrections", s.listCorrections)
	s.mux.HandleFunc("POST /api/v1/organizations/{organizationID}/corrections/{correctionID}/decide", s.decideCorrection)
}

func (s *Server) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "service": "kredit-api", "version": s.config.Version})
}

func (s *Server) ready(w http.ResponseWriter, r *http.Request) {
	if s.config.Environment == "production" || s.config.Environment == "staging" {
		if s.runtime.Database == nil {
			writeProblem(w, http.StatusServiceUnavailable, "database_unavailable", "database connection is not configured")
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		defer cancel()
		if err := s.runtime.Database.Ping(ctx); err != nil {
			writeProblem(w, http.StatusServiceUnavailable, "database_unavailable", "database health check failed")
			return
		}
		if err := s.runtime.Database.CheckSchema(ctx); err != nil {
			writeProblem(w, http.StatusServiceUnavailable, "database_schema_unavailable", "database migrations are incomplete")
			return
		}
		if err := s.runtime.Database.CheckPersistenceContract(ctx); err != nil {
			writeProblem(w, http.StatusServiceUnavailable, "persistence_contract_incomplete", "database persistence contract is incomplete")
			return
		}
		if !s.runtime.DurableDomainReady() {
			writeProblem(w, http.StatusServiceUnavailable, "durable_repositories_unavailable", "durable domain repositories are not fully wired")
			return
		}
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "service": "kredit-api", "version": s.config.Version})
}

func (s *Server) meta(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"service":     "kredit-api",
		"version":     s.config.Version,
		"environment": s.config.Environment,
		"timezone":    s.config.Timezone,
		"currency":    s.config.Currency,
		"money_unit":  s.config.MoneyUnit,
	})
}

func (s *Server) withSecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-Permitted-Cross-Domain-Policies", "none")
		w.Header().Set("X-DNS-Prefetch-Control", "off")
		w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; frame-ancestors 'none'; base-uri 'self'; form-action 'self'")
		w.Header().Set("Cross-Origin-Opener-Policy", "same-origin")
		w.Header().Set("Cross-Origin-Resource-Policy", "same-origin")
		if s.config.Environment != "development" {
			w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Cache-Control", "no-store")
		next.ServeHTTP(w, r)
	})
}

func (s *Server) withBodyLimit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") && r.Body != nil {
			r.Body = http.MaxBytesReader(w, r.Body, 2<<20)
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) withPanicRecovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if recovered := recover(); recovered != nil {
				s.logger.Error("http panic recovered", "request_id", requestIDFromContext(r.Context()), "panic", platformlogging.Redact(fmt.Sprint(recovered)))
				writeProblem(w, http.StatusInternalServerError, "internal_error", "an unexpected server error occurred")
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func (s *Server) withRateLimit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/healthz") || strings.HasPrefix(r.URL.Path, "/api/v1/healthz") {
			next.ServeHTTP(w, r)
			return
		}
		key := clientIP(r)
		now := time.Now()
		s.rateMu.Lock()
		s.pruneRateWindows(now)
		window := s.rate[key]
		if window.started.IsZero() || now.Sub(window.started) >= time.Minute {
			window = rateWindow{started: now}
		}
		window.count++
		s.rate[key] = window
		s.rateMu.Unlock()
		if window.count > rateLimitPerMinute {
			w.Header().Set("Retry-After", "60")
			writeProblem(w, http.StatusTooManyRequests, "rate_limited", "too many requests")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) pruneRateWindows(now time.Time) {
	if now.Sub(s.ratePrunedAt) < time.Minute {
		return
	}
	s.ratePrunedAt = now
	for key, window := range s.rate {
		if now.Sub(window.started) >= rateWindowTTL {
			delete(s.rate, key)
		}
	}
}

// clientIP resolves the requesting client address. When the request arrives
// from a private/loopback address the deployment is behind a reverse proxy,
// so X-Forwarded-For carries the real client (the ingress overwrites this
// header in production). Direct public connections are keyed on RemoteAddr
// so callers cannot rotate their identity with a spoofed header.
func clientIP(r *http.Request) string {
	remote := remoteHost(r)
	if forwarded := strings.TrimSpace(r.Header.Get("X-Forwarded-For")); forwarded != "" && isPrivateOrLoopback(remote) {
		for _, candidate := range strings.Split(forwarded, ",") {
			candidate = strings.TrimSpace(candidate)
			if parsed := net.ParseIP(candidate); parsed != nil {
				return candidate
			}
		}
	}
	if remote != "" {
		return remote
	}
	return "unknown"
}

func remoteHost(r *http.Request) string {
	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil && host != "" {
		return host
	}
	if r.RemoteAddr != "" {
		return r.RemoteAddr
	}
	return ""
}

func isPrivateOrLoopback(host string) bool {
	parsed := net.ParseIP(host)
	if parsed == nil {
		return false
	}
	return parsed.IsLoopback() || parsed.IsPrivate() || parsed.IsLinkLocalUnicast()
}

func (s *Server) metricsStatus(w http.ResponseWriter, r *http.Request) {
	if !s.requireMetricsAccess(w, r) {
		return
	}
	if s.runtime.Metrics == nil {
		writeJSON(w, http.StatusOK, map[string]any{"counters": map[string]uint64{}})
		return
	}
	writeJSON(w, http.StatusOK, s.runtime.Metrics.Snapshot())
}

func (s *Server) metricsPrometheus(w http.ResponseWriter, r *http.Request) {
	if !s.requireMetricsAccess(w, r) {
		return
	}
	if s.runtime.Metrics == nil {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4")
		w.WriteHeader(http.StatusOK)
		return
	}
	w.Header().Set("Content-Type", "text/plain; version=0.0.4")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusOK)
	_, _ = io.WriteString(w, s.runtime.Metrics.Prometheus())
}

func (s *Server) requireMetricsAccess(w http.ResponseWriter, r *http.Request) bool {
	session, _, ok := s.requireAuth(w, r)
	if !ok {
		return false
	}
	if session.AuthenticationLevel != auth.AAL2 {
		s.recordSecurityEvent(r, "authorization.metrics_step_up_required", "metrics", "denied", "warning")
		writeProblem(w, http.StatusForbidden, "step_up_required", "step-up authentication is required")
		return false
	}
	return true
}

func (s *Server) withRequestContext(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := strings.TrimSpace(r.Header.Get("X-Request-ID"))
		if !validRequestID(requestID) {
			requestID = newRequestID()
		}
		traceID := traceIDFromRequest(r)
		w.Header().Set("X-Request-ID", requestID)
		w.Header().Set("X-Trace-ID", traceID)
		ctx := context.WithValue(r.Context(), requestIDKey, requestID)
		ctx = context.WithValue(ctx, traceIDKey, traceID)
		ctx = observability.ExtractTraceContext(ctx, map[string]string{"traceparent": r.Header.Get("traceparent")})
		ctx, span := s.runtime.Tracer.Start(ctx, "http.request", attribute.String("http.method", r.Method))
		r = r.WithContext(ctx)
		observer := &responseObserver{ResponseWriter: w}
		started := time.Now()
		next.ServeHTTP(observer, r)
		duration := time.Since(started)
		status := observer.status
		if status == 0 {
			status = http.StatusOK
		}
		if s.runtime.Metrics != nil {
			s.runtime.Metrics.Inc("http_requests_total")
			s.runtime.Metrics.Inc(fmt.Sprintf("http_responses_%dxx", status/100))
			s.runtime.Metrics.ObserveDuration("http_request_duration", duration)
		}
		route := r.Pattern
		if route == "" {
			route = safeRequestPath(r.URL.Path)
		}
		span.SetAttributes(attribute.String("http.route", safeRequestPath(route)), attribute.Int("http.status_code", status), attribute.Int64("http.duration_ms", duration.Milliseconds()))
		span.End()
		attributes := []any{"method", r.Method, "path", safeRequestPath(r.URL.RequestURI()), "request_id", requestID, "trace_id", traceID, "status", status, "duration_ms", duration.Milliseconds()}
		if token := sessionTokenFromRequest(r); token != "" {
			if session, user, err := s.runtime.Auth.SessionFromToken(token); err == nil {
				attributes = append(attributes, "user_id", user.ID, "authentication_level", session.AuthenticationLevel)
			}
		}
		s.logger.Info("http request", attributes...)
	})
}

func safeRequestPath(path string) string {
	return platformlogging.SafePath(path)
}

func validRequestID(value string) bool {
	if value == "" || len(value) > 128 {
		return false
	}
	for _, character := range value {
		if character < 0x21 || character > 0x7e {
			return false
		}
	}
	return true
}

type contextKey string

const (
	requestIDKey contextKey = "request_id"
	traceIDKey   contextKey = "trace_id"
)

func requestIDFromContext(ctx context.Context) string {
	requestID, _ := ctx.Value(requestIDKey).(string)
	return requestID
}

func traceIDFromRequest(r *http.Request) string {
	if r != nil {
		parts := strings.Split(r.Header.Get("traceparent"), "-")
		if len(parts) == 4 && len(parts[1]) == 32 && isHex(parts[1]) && len(parts[2]) == 16 && isHex(parts[2]) {
			return strings.ToLower(parts[1])
		}
	}
	return newTraceID()
}

func isHex(value string) bool {
	for _, character := range value {
		if (character < '0' || character > '9') && (character < 'a' || character > 'f') && (character < 'A' || character > 'F') {
			return false
		}
	}
	return true
}

func newRequestID() string {
	var value [16]byte
	if _, err := rand.Read(value[:]); err != nil {
		return fmt.Sprintf("fallback-%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(value[:])
}

func newTraceID() string {
	var value [16]byte
	if _, err := rand.Read(value[:]); err != nil {
		return fmt.Sprintf("%032x", time.Now().UnixNano())
	}
	return hex.EncodeToString(value[:])
}

type responseObserver struct {
	http.ResponseWriter
	status int
	bytes  int
}

func (o *responseObserver) WriteHeader(status int) {
	if o.status != 0 {
		return
	}
	o.status = status
	o.ResponseWriter.WriteHeader(status)
}

func (o *responseObserver) Write(body []byte) (int, error) {
	if o.status == 0 {
		o.WriteHeader(http.StatusOK)
	}
	count, err := o.ResponseWriter.Write(body)
	o.bytes += count
	return count, err
}

func (o *responseObserver) Unwrap() http.ResponseWriter { return o.ResponseWriter }

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
