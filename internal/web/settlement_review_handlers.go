package web

import (
	"encoding/json"
	"kredit/internal/access"
	"net/http"
	"strings"
	"time"
)

func (s *Server) listSettlementReview(w http.ResponseWriter, r *http.Request) {
	_, user, _, ok := s.requirePlatformAccess(w, r, access.PermissionPlatformSettings)
	if !ok {
		return
	}
	if s.runtime.Database == nil {
		writeProblem(w, 503, "review_unavailable", "Bank review is unavailable.")
		return
	}
	tx, err := s.runtime.Database.Raw().Begin(r.Context())
	if err != nil {
		writeProblem(w, 503, "review_unavailable", "Bank review is unavailable.")
		return
	}
	defer func() { _ = tx.Rollback(r.Context()) }()
	_, err = tx.Exec(r.Context(), `SELECT set_config('app.current_user_id',$1,true)`, user.ID)
	var raw json.RawMessage
	if err == nil {
		err = tx.QueryRow(r.Context(), `SELECT app.settlement_registration_review()`).Scan(&raw)
	}
	if err == nil {
		err = tx.Commit(r.Context())
	}
	if err != nil {
		writeProblem(w, 503, "review_unavailable", "Bank review could not be loaded.")
		return
	}
	s.auditPlatformRead(r, user.ID, "settlement.review.viewed", "settlement_registration", "")
	writeJSON(w, 200, raw)
}

func (s *Server) reviewSettlementDestination(w http.ResponseWriter, r *http.Request) {
	session, user, _, ok := s.requirePlatformAccess(w, r, access.PermissionPlatformSettings)
	if !ok || !s.requireFreshMFA(w, session) || !s.requireCSRF(w, r) {
		return
	}
	org, err := pathID(r, "organizationID")
	if err != nil {
		writeProblem(w, 400, "invalid_business", "Invalid business reference.")
		return
	}
	var in struct {
		Version   int64  `json:"version"`
		Reference string `json:"provider_reference"`
		Action    string `json:"action"`
		Reason    string `json:"reason"`
	}
	if !decodeJSONRequest(w, r, &in) {
		return
	}
	in.Reason = strings.TrimSpace(in.Reason)
	if in.Version <= 0 || in.Reference == "" || len(in.Reference) > 128 || len(in.Reason) < 20 || len(in.Reason) > 2000 || (in.Action != "approve" && in.Action != "reject") {
		writeProblem(w, 422, "invalid_review", "Choose a decision and record the ownership evidence reviewed.")
		return
	}
	p, _, err := s.runtime.Onboarding.Get(org)
	if err != nil || p.Version != in.Version || p.SettlementProviderReference != in.Reference || (p.SettlementState != "pending_verification" && p.SettlementState != "provider_review") {
		writeProblem(w, 409, "review_conflict", "The bank details changed. Refresh before reviewing.")
		return
	}
	state := "rejected"
	if in.Action == "approve" {
		if s.runtime.Database == nil || s.runtime.Settlement == nil || s.runtime.Settlement.Name() != p.SettlementProvider {
			writeProblem(w, 409, "provider_changed", "The original bank provider is not active. Review its connection first.")
			return
		}
		if p.KYBState != "approved" || (!p.KYBExpiresAt.IsZero() && !time.Now().Before(p.KYBExpiresAt)) {
			writeProblem(w, 409, "business_verification_required", "Complete business verification before approving its bank account.")
			return
		}
		tx, err := s.runtime.Database.Raw().Begin(r.Context())
		if err != nil {
			writeProblem(w, 503, "review_unavailable", "Bank evidence is unavailable.")
			return
		}
		defer func() { _ = tx.Rollback(r.Context()) }()
		_, err = tx.Exec(r.Context(), `SELECT set_config('app.current_organization_id',$1,true)`, org)
		var found bool
		if err == nil {
			err = tx.QueryRow(r.Context(), `SELECT EXISTS(SELECT 1 FROM app.settlement_registrations WHERE organization_id=$1::uuid AND provider=$2 AND connection_identity=$3 AND state='REGISTERED' AND result->>'provider_reference'=$4 AND result->>'account_name'=$5 AND account_last4=$6)`, org, p.SettlementProvider, s.runtime.Settlement.ConnectionIdentity(), in.Reference, p.SettlementAccountName, p.SettlementAccountLast4).Scan(&found)
		}
		if err != nil || !found {
			writeProblem(w, 409, "bank_evidence_missing", "The original provider registration could not be confirmed. Do not approve this account.")
			return
		}
		// Release the evidence transaction before the onboarding store acquires one.
		if err = tx.Commit(r.Context()); err != nil {
			writeProblem(w, 503, "review_unavailable", "Bank evidence is unavailable.")
			return
		}
		state = "verified"
	}
	next, summary, err := s.runtime.Onboarding.RecordSettlementDecisionForReference(org, user.ID, in.Reference, in.Version, state, in.Reason)
	s.finishOnboardingChange(w, r, user, org, "supplier.onboarding.settlement.reviewed", next, summary, err)
}

func (s *Server) permitSettlementRegistrationRetry(w http.ResponseWriter, r *http.Request) {
	session, user, _, ok := s.requirePlatformAccess(w, r, access.PermissionPlatformSettings)
	if !ok || !s.requireFreshMFA(w, session) || !s.requireCSRF(w, r) {
		return
	}
	org, err := pathID(r, "organizationID")
	if err != nil {
		writeProblem(w, 400, "invalid_business", "Invalid business reference.")
		return
	}
	id := r.PathValue("registrationID")
	var in struct {
		Reason string `json:"reason"`
	}
	if !decodeJSONRequest(w, r, &in) {
		return
	}
	in.Reason = strings.TrimSpace(in.Reason)
	if len(id) != 64 || len(in.Reason) < 20 || len(in.Reason) > 2000 {
		writeProblem(w, 422, "invalid_review", "Record how the provider confirmed that no bank registration was created.")
		return
	}
	if s.runtime.Database == nil {
		writeProblem(w, 503, "review_unavailable", "Bank review is unavailable.")
		return
	}
	tx, err := s.runtime.Database.Raw().Begin(r.Context())
	if err != nil {
		writeProblem(w, 503, "review_unavailable", "Bank review is unavailable.")
		return
	}
	defer func() { _ = tx.Rollback(r.Context()) }()
	_, err = tx.Exec(r.Context(), `SELECT set_config('app.current_organization_id',$1,true),set_config('app.current_user_id',$2,true)`, org, user.ID)
	if err != nil {
		writeProblem(w, 503, "review_unavailable", "Bank review is unavailable.")
		return
	}
	tag, err := tx.Exec(r.Context(), `UPDATE app.settlement_registrations SET state='READY',updated_at=now() WHERE id=$1 AND organization_id=$2::uuid AND state='STARTED' AND created_at<now()-interval '1 minute'`, id, org)
	if err != nil || tag.RowsAffected() != 1 {
		writeProblem(w, 409, "review_conflict", "This registration changed or is still active. Refresh before continuing.")
		return
	}
	metadata, _ := json.Marshal(map[string]string{"reason": in.Reason, "effect": "seller_may_retry_only_after_provider_confirms_not_created"})
	_, err = tx.Exec(r.Context(), `INSERT INTO app.audit_events(actor_user_id,organization_id,action,resource_type,resource_id,outcome,severity,metadata) VALUES($1::uuid,$2::uuid,'settlement.registration.retry_permitted','settlement_registration',$3,'success','high',$4::jsonb)`, user.ID, org, id, metadata)
	if err == nil {
		err = tx.Commit(r.Context())
	}
	if err != nil {
		writeProblem(w, 503, "review_unavailable", "The decision was not confirmed. Refresh before retrying.")
		return
	}
	writeJSON(w, 200, map[string]bool{"retry_permitted": true})
}
