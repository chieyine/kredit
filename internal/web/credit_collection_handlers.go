package web

import (
	"kredit/internal/access"
	"kredit/internal/collections"
	"kredit/internal/db"
	"kredit/internal/notifications"
	"net/http"
	"time"
)

func (s *Server) collectionEligibility(w http.ResponseWriter, r *http.Request) {
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
	ctx := db.WithOrganizationContext(r.Context(), orgID)
	eligibility, err := s.runtime.collectionEligibilityContext(ctx, v.Obligation.ID, time.Now().UTC())
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
	r = r.WithContext(db.WithTenantContext(r.Context(), user.ID, orgID))
	if !s.requireSupplierReady(w, r, orgID, user.ID, "starting a live collection") {
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
	if s.runtime.Database == nil {
		state, _ := s.runtime.Credit.CollectionState(v.Obligation.ID)
		_, _ = s.runtime.EmitNotification(r.Context(), notifications.Event{ID: "collection-submitted:" + attempt.ID, Type: "CollectionSubmitted", RecipientID: state.BuyerUserID, Priority: notifications.PriorityCritical, AmountKobo: int64(attempt.RequestedAmountKobo), Currency: state.Currency, Reference: attempt.ID, NextAction: "Review the collection status", SecurePath: "/workspace/purchases/orders/" + requestID})
	}
	writeJSON(w, 202, map[string]any{"attempt": attempt})
}

func (s *Server) listCollections(w http.ResponseWriter, r *http.Request) {
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
	financialRows, readErr := s.runtime.readCollectionsAttemptsContext(db.WithOrganizationContext(r.Context(), orgID), v.Obligation.ID)
	if financialReadError(w, readErr) {
		return
	}
	writeJSON(w, 200, map[string]any{"attempts": financialRows})
}

func (s *Server) retryCollection(w http.ResponseWriter, r *http.Request) {
	orgID, _ := pathID(r, "organizationID")
	attemptID, _ := pathID(r, "attemptID")
	_, user, _, ok := s.requireOrganizationAccess(w, r, orgID, access.PermissionManageFinancial)
	if !ok {
		return
	}
	r = r.WithContext(db.WithTenantContext(r.Context(), user.ID, orgID))
	if !s.requireCSRF(w, r) {
		return
	}
	attempt, exists := s.runtime.getCollectionAttemptContext(r.Context(), attemptID)
	if !exists || !s.runtime.Credit.ObligationBelongsToOrganization(attempt.ObligationID, orgID) {
		writeProblem(w, 404, "collection_not_found", "We could not find that bank debit.")
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
	r = r.WithContext(db.WithTenantContext(r.Context(), user.ID, orgID))
	if !s.requireCSRF(w, r) {
		return
	}
	attempt, exists := s.runtime.getCollectionAttemptContext(r.Context(), attemptID)
	if !exists || !s.runtime.Credit.ObligationBelongsToOrganization(attempt.ObligationID, orgID) {
		writeProblem(w, 404, "collection_not_found", "We could not find that bank debit.")
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
	attempt, err := s.runtime.Collections.SignalWebhook(r.Context(), event)
	if err != nil {
		writeProblem(w, 400, "collection_webhook_rejected", err.Error())
		return
	}
	writeJSON(w, 202, map[string]any{"attempt": attempt, "reconciliation_required": true})
}
