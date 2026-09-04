package web

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"kredit/internal/access"
	"kredit/internal/audit"
	"kredit/internal/auth"
	"kredit/internal/identity"
	"kredit/internal/notifications"
	"kredit/internal/onboarding"
)

type representativeRequest struct {
	ExpectedVersion int64  `json:"expected_version"`
	Name            string `json:"name"`
	Title           string `json:"title"`
}
type providerReferenceRequest struct {
	ExpectedVersion int64 `json:"expected_version"`
}
type settlementRequest struct {
	ExpectedVersion   int64  `json:"expected_version"`
	Provider          string `json:"provider"`
	ProviderReference string `json:"provider_reference"`
	BankName          string `json:"bank_name"`
	AccountName       string `json:"account_name"`
	AccountLast4      string `json:"account_last4"`
}
type billingRequest struct {
	ExpectedVersion   int64  `json:"expected_version"`
	Method            string `json:"method"`
	ProviderReference string `json:"provider_reference"`
	Cycle             string `json:"cycle"`
}
type policyRequest struct {
	ExpectedVersion int64 `json:"expected_version"`
	CreditLimitKobo int64 `json:"credit_limit_kobo"`
	PaymentDays     int   `json:"payment_days"`
	GraceHours      int   `json:"grace_hours"`
}
type supplierConsentRequest struct {
	ExpectedVersion int64  `json:"expected_version"`
	TermsVersion    string `json:"terms_version"`
	PrivacyVersion  string `json:"privacy_version"`
}
type onboardingContactRequest struct {
	Identifier string `json:"identifier"`
	Channel    string `json:"channel"`
}
type onboardingContactVerifyRequest struct {
	Identifier  string `json:"identifier"`
	Channel     string `json:"channel"`
	ChallengeID string `json:"challenge_id"`
	Code        string `json:"code"`
}

func (s *Server) getSupplierOnboarding(w http.ResponseWriter, r *http.Request) {
	orgID, err := pathID(r, "organizationID")
	if err != nil {
		writeProblem(w, 400, "invalid_path", err.Error())
		return
	}
	_, user, membership, ok := s.requireOrganizationAccess(w, r, orgID, access.PermissionReadOrganization)
	if !ok {
		return
	}
	s.syncOnboardingSecurity(orgID, user.ID)
	profile, summary, err := s.runtime.Onboarding.Get(orgID)
	if err != nil {
		writeProblem(w, 404, "onboarding_not_found", err.Error())
		return
	}
	profile = visibleOnboardingProfile(profile, membership.Role)
	writeJSON(w, 200, map[string]any{"profile": profile, "readiness": summary, "current_terms_version": onboarding.CurrentTermsVersion, "current_privacy_version": onboarding.CurrentPrivacyVersion, "permissions": onboardingPermissions(membership.Role)})
}

func (s *Server) requestOnboardingContactOTP(w http.ResponseWriter, r *http.Request) {
	orgID, _ := pathID(r, "organizationID")
	session, user, membership, ok := s.requireOrganizationAccess(w, r, orgID, access.PermissionReadOrganization)
	if !ok {
		return
	}
	if membership.Role != access.RoleOwner {
		writeProblem(w, 403, "owner_required", "only the organization owner may verify owner contacts")
		return
	}
	if !s.requireFreshMFA(w, session) || !s.requireCSRF(w, r) {
		return
	}
	var in onboardingContactRequest
	if err := decodeJSON(w, r, &in); err != nil {
		return
	}
	challenge, code, err := s.runtime.Auth.RequestOTP(in.Identifier, in.Channel, "supplier_contact_verification")
	if err != nil {
		writeProblem(w, 422, "contact_verification_failed", err.Error())
		return
	}
	// A swallowed delivery failure returns 202 for a code that never arrives and
	// leaves the supplier stuck on an onboarding step with no way to know why.
	// The login OTP route already reports this; report it here too.
	if err := s.runtime.Notifications.SendOTP(r.Context(), in.Identifier, in.Channel, code); err != nil {
		writeProblem(w, http.StatusServiceUnavailable, "otp_delivery_unavailable", "verification code delivery is unavailable")
		return
	}
	s.runtime.Audit.Append(audit.Event{ActorUserID: user.ID, OrganizationID: orgID, Action: "supplier.onboarding.contact_verification_requested", ResourceType: "supplier_onboarding", ResourceID: orgID, Outcome: "success", RequestID: requestIDFromContext(r.Context()), Metadata: map[string]string{"channel": in.Channel}})
	payload := map[string]any{"challenge_id": challenge.ID, "expires_at": challenge.ExpiresAt}
	if s.config.Environment == "development" {
		payload["development_code"] = code
	}
	writeJSON(w, 202, payload)
}

func (s *Server) verifyOnboardingContact(w http.ResponseWriter, r *http.Request) {
	orgID, _ := pathID(r, "organizationID")
	session, user, membership, ok := s.requireOrganizationAccess(w, r, orgID, access.PermissionReadOrganization)
	if !ok {
		return
	}
	if membership.Role != access.RoleOwner {
		writeProblem(w, 403, "owner_required", "only the organization owner may verify owner contacts")
		return
	}
	if !s.requireFreshMFA(w, session) || !s.requireCSRF(w, r) {
		return
	}
	var in onboardingContactVerifyRequest
	if err := decodeJSON(w, r, &in); err != nil {
		return
	}
	if err := s.runtime.Auth.VerifyAndAttachIdentifier(user.ID, in.ChallengeID, in.Code, in.Channel, in.Identifier); err != nil {
		writeProblem(w, 422, "contact_verification_failed", err.Error())
		return
	}
	p, summary, err := s.runtime.Onboarding.RecordContactVerified(orgID, user.ID, in.Channel)
	s.finishOnboardingChange(w, r, user, orgID, "supplier.onboarding.contact.verified", p, summary, err)
}

func (s *Server) updateSupplierRepresentative(w http.ResponseWriter, r *http.Request) {
	orgID, _ := pathID(r, "organizationID")
	session, user, _, ok := s.requireOrganizationAccess(w, r, orgID, access.PermissionManageOrganization)
	if !ok {
		return
	}
	if !s.requireFreshMFA(w, session) {
		return
	}
	if !s.requireCSRF(w, r) {
		return
	}
	var in representativeRequest
	if !decodeJSONRequest(w, r, &in) {
		return
	}
	p, sum, err := s.runtime.Onboarding.UpdateRepresentative(orgID, user.ID, onboarding.RepresentativeInput{ExpectedVersion: in.ExpectedVersion, Name: in.Name, Title: in.Title})
	s.finishOnboardingChange(w, r, user, orgID, "supplier.onboarding.representative.updated", p, sum, err)
}
func (s *Server) submitSupplierKYB(w http.ResponseWriter, r *http.Request) {
	orgID, _ := pathID(r, "organizationID")
	session, user, _, ok := s.requireOrganizationAccess(w, r, orgID, access.PermissionManageOrganization)
	if !ok {
		return
	}
	if !s.requireFreshMFA(w, session) {
		return
	}
	if !s.requireCSRF(w, r) {
		return
	}
	var in providerReferenceRequest
	if !decodeJSONRequest(w, r, &in) {
		return
	}
	organization, exists := s.runtime.Organizations.Get(orgID)
	if !exists {
		writeProblem(w, http.StatusNotFound, "organization_not_found", "organization was not found")
		return
	}
	verification, providerErr := s.runtime.Identity.CreateBusinessVerification(r.Context(), identity.BusinessVerificationInput{SubjectID: orgID, LegalName: organization.LegalName, BusinessType: organization.BusinessType, Address: organization.BusinessAddress, Registration: organization.RegistrationInfo})
	if providerErr != nil {
		writeProblem(w, http.StatusServiceUnavailable, "kyb_provider_unavailable", providerErr.Error())
		return
	}
	p, sum, err := s.runtime.Onboarding.SubmitKYB(orgID, user.ID, verification.ProviderID, in.ExpectedVersion)
	if err == nil && (verification.State == "verified" || verification.State == "approved") {
		p, sum, err = s.runtime.Onboarding.RecordKYBDecisionForReference(orgID, user.ID, p.KYBProviderReference, p.Version, "approved", "provider_approved", verification.ExpiresAt)
	} else if err == nil && verification.State == "rejected" {
		p, sum, err = s.runtime.Onboarding.RecordKYBDecisionForReference(orgID, user.ID, p.KYBProviderReference, p.Version, "rejected", "provider_rejected", verification.ExpiresAt)
	} else if err == nil && verification.State != "submitted" {
		p, sum, err = s.runtime.Onboarding.RecordKYBDecisionForReference(orgID, user.ID, p.KYBProviderReference, p.Version, "provider_review", "", verification.ExpiresAt)
	}
	s.finishOnboardingChange(w, r, user, orgID, "supplier.onboarding.kyb.submitted", p, sum, err)
}

func (s *Server) reconcileSupplierKYB(w http.ResponseWriter, r *http.Request) {
	orgID, _ := pathID(r, "organizationID")
	session, user, _, ok := s.requireOrganizationAccess(w, r, orgID, access.PermissionManageOrganization)
	if !ok || !s.requireFreshMFA(w, session) || !s.requireCSRF(w, r) {
		return
	}
	profile, _, err := s.runtime.Onboarding.Get(orgID)
	if err != nil || strings.TrimSpace(profile.KYBProviderReference) == "" {
		writeProblem(w, http.StatusConflict, "kyb_not_submitted", "business verification has not been submitted")
		return
	}
	verification, err := s.runtime.Identity.GetVerification(r.Context(), profile.KYBProviderReference)
	if err != nil {
		writeProblem(w, http.StatusServiceUnavailable, "kyb_provider_unavailable", "business verification status is temporarily unavailable")
		return
	}
	if verification.ProviderID != profile.KYBProviderReference || verification.SubjectID != orgID {
		writeProblem(w, http.StatusConflict, "kyb_provider_identity_mismatch", "the provider result does not match this business verification")
		return
	}
	state, reason := "provider_review", ""
	switch strings.ToLower(strings.TrimSpace(verification.State)) {
	case "verified", "approved":
		state, reason = "approved", "provider_approved"
	case "rejected", "failed":
		state, reason = "rejected", "provider_rejected"
	case "pending", "submitted", "in_review", "provider_review":
	default:
		writeProblem(w, http.StatusConflict, "kyb_provider_state_invalid", "the provider returned an unsupported verification state")
		return
	}
	updated, summary, err := s.runtime.Onboarding.RecordKYBDecisionForReference(orgID, user.ID, profile.KYBProviderReference, profile.Version, state, reason, verification.ExpiresAt)
	s.finishOnboardingChange(w, r, user, orgID, "supplier.onboarding.kyb.reconciled", updated, summary, err)
}
func (s *Server) updateSupplierSettlement(w http.ResponseWriter, r *http.Request) {
	orgID, _ := pathID(r, "organizationID")
	session, user, _, ok := s.requireOrganizationAccess(w, r, orgID, access.PermissionManageFinancial)
	if !ok {
		return
	}
	if !s.requireFreshMFA(w, session) {
		return
	}
	if !s.requireCSRF(w, r) {
		return
	}
	var in settlementRequest
	if !decodeJSONRequest(w, r, &in) {
		return
	}
	p, sum, err := s.runtime.Onboarding.UpdateSettlement(orgID, user.ID, onboarding.SettlementInput{ExpectedVersion: in.ExpectedVersion, Provider: in.Provider, ProviderReference: in.ProviderReference, BankName: in.BankName, AccountName: in.AccountName, AccountLast4: in.AccountLast4})
	if err == nil && s.config.Environment == "development" {
		p, sum, err = s.runtime.Onboarding.RecordSettlementDecision(orgID, user.ID, "verified", "development_provider_verified")
	}
	s.finishOnboardingChange(w, r, user, orgID, "supplier.onboarding.settlement.updated", p, sum, err)
}
func (s *Server) updateSupplierBilling(w http.ResponseWriter, r *http.Request) {
	orgID, _ := pathID(r, "organizationID")
	session, user, _, ok := s.requireOrganizationAccess(w, r, orgID, access.PermissionManageFinancial)
	if !ok {
		return
	}
	if !s.requireFreshMFA(w, session) {
		return
	}
	if !s.requireCSRF(w, r) {
		return
	}
	var in billingRequest
	if !decodeJSONRequest(w, r, &in) {
		return
	}
	p, sum, err := s.runtime.Onboarding.UpdateBilling(orgID, user.ID, onboarding.BillingInput{ExpectedVersion: in.ExpectedVersion, Method: in.Method, ProviderReference: in.ProviderReference, Cycle: in.Cycle})
	s.finishOnboardingChange(w, r, user, orgID, "supplier.onboarding.billing.updated", p, sum, err)
}
func (s *Server) updateSupplierCreditPolicy(w http.ResponseWriter, r *http.Request) {
	orgID, _ := pathID(r, "organizationID")
	session, user, _, ok := s.requireOrganizationAccess(w, r, orgID, access.PermissionManageOrganization)
	if !ok {
		return
	}
	if !s.requireFreshMFA(w, session) {
		return
	}
	if !s.requireCSRF(w, r) {
		return
	}
	var in policyRequest
	if !decodeJSONRequest(w, r, &in) {
		return
	}
	p, sum, err := s.runtime.Onboarding.UpdateCreditPolicy(orgID, user.ID, onboarding.CreditPolicyInput{ExpectedVersion: in.ExpectedVersion, CreditLimitKobo: in.CreditLimitKobo, PaymentDays: in.PaymentDays, GraceHours: in.GraceHours})
	s.finishOnboardingChange(w, r, user, orgID, "supplier.onboarding.credit_policy.updated", p, sum, err)
}
func (s *Server) acceptSupplierConsents(w http.ResponseWriter, r *http.Request) {
	orgID, _ := pathID(r, "organizationID")
	session, user, m, ok := s.requireOrganizationAccess(w, r, orgID, access.PermissionReadOrganization)
	if !ok {
		return
	}
	if m.Role != access.RoleOwner {
		writeProblem(w, 403, "owner_required", "only the organization owner may accept supplier terms")
		return
	}
	if !s.requireFreshMFA(w, session) {
		return
	}
	if !s.requireCSRF(w, r) {
		return
	}
	var in supplierConsentRequest
	if !decodeJSONRequest(w, r, &in) {
		return
	}
	p, sum, err := s.runtime.Onboarding.AcceptConsents(orgID, user.ID, in.ExpectedVersion, in.TermsVersion, in.PrivacyVersion)
	s.finishOnboardingChange(w, r, user, orgID, "supplier.onboarding.consents.accepted", p, sum, err)
}

func (s *Server) requireFreshMFA(w http.ResponseWriter, session auth.Session) bool {
	if session.AuthenticationLevel != auth.AAL2 || session.MFAVerifiedAt.IsZero() || time.Since(session.MFAVerifiedAt) > 15*time.Minute {
		writeProblem(w, 403, "step_up_required", "recent MFA verification is required for this sensitive action")
		return false
	}
	return true
}
func (s *Server) finishOnboardingChange(w http.ResponseWriter, r *http.Request, user auth.User, orgID, action string, p onboarding.Profile, sum onboarding.Summary, err error) {
	if err != nil {
		code := 422
		if strings.Contains(err.Error(), "version conflict") {
			code = 409
		}
		writeProblem(w, code, "onboarding_update_failed", err.Error())
		return
	}
	s.runtime.Audit.Append(audit.Event{ActorUserID: user.ID, OrganizationID: orgID, Action: action, ResourceType: "supplier_onboarding", ResourceID: orgID, Outcome: "success", RequestID: requestIDFromContext(r.Context()), Metadata: map[string]string{"readiness_state": sum.State}})
	if strings.Contains(action, "settlement") || strings.Contains(action, "billing") {
		_, _ = s.runtime.EmitNotification(context.Background(), notifications.Event{ID: action + ":" + orgID + ":" + fmt.Sprint(p.Version), Type: "SupplierSensitiveSettingChanged", RecipientID: user.ID, Email: user.Email, Phone: user.Phone, OrganizationID: orgID, Priority: notifications.PriorityCritical, Reference: action, NextAction: "Review the change in supplier settings.", SecurePath: "/app/onboarding"})
	}
	if strings.Contains(action, "kyb") {
		_, _ = s.runtime.EmitNotification(context.Background(), notifications.Event{ID: "supplier-kyb:" + orgID + ":" + fmt.Sprint(p.Version), Type: "SupplierVerificationOutcome", RecipientID: user.ID, Email: user.Email, Phone: user.Phone, OrganizationID: orgID, Priority: notifications.PriorityCritical, Reference: p.KYBState, NextAction: "Review your business verification result.", SecurePath: "/app/onboarding"})
	}
	if sum.Ready && p.ReadinessChangedAt.Equal(p.UpdatedAt) {
		_, _ = s.runtime.EmitNotification(context.Background(), notifications.Event{ID: "supplier-pilot-ready:" + orgID + ":" + fmt.Sprint(p.Version), Type: "SupplierPilotReady", RecipientID: user.ID, Email: user.Email, Phone: user.Phone, OrganizationID: orgID, Priority: notifications.PriorityCritical, Reference: orgID, NextAction: "Invite your team or create a credit request.", SecurePath: "/app/onboarding"})
	}
	writeJSON(w, 200, map[string]any{"profile": p, "readiness": sum})
}

func (s *Server) syncOnboardingSecurity(orgID, actor string) {
	members := s.runtime.Organizations.ListMembers(orgID)
	ownerMFA := false
	financeMFA := true
	for _, m := range members {
		if m.Status != "active" {
			continue
		}
		if m.Role == access.RoleOwner {
			ownerMFA = s.runtime.Auth.IsMFAEnrolled(m.UserID)
		}
		if m.Role == access.RoleFinance && !s.runtime.Auth.IsMFAEnrolled(m.UserID) {
			financeMFA = false
		}
	}
	_, _, _ = s.runtime.Onboarding.SyncSecurity(orgID, actor, ownerMFA, financeMFA)
}
func visibleOnboardingProfile(p onboarding.Profile, role access.Role) onboarding.Profile {
	if role != access.RoleOwner && role != access.RoleAdministrator && role != access.RoleFinance {
		p.KYBProviderReference = ""
		p.SettlementProviderReference = ""
		p.BillingProviderReference = ""
		p.SettlementAccountName = ""
	}
	return p
}
func onboardingPermissions(role access.Role) map[string]bool {
	return map[string]bool{"business": access.Can(role, access.PermissionManageOrganization), "settlement": access.Can(role, access.PermissionManageFinancial), "billing": access.Can(role, access.PermissionManageFinancial), "credit_policy": access.Can(role, access.PermissionManageOrganization), "consents": role == access.RoleOwner}
}

func (s *Server) requireSupplierReady(w http.ResponseWriter, organizationID, actorUserID, action string) bool {
	s.syncOnboardingSecurity(organizationID, actorUserID)
	_, summary, err := s.runtime.Onboarding.Get(organizationID)
	if err != nil {
		writeProblem(w, http.StatusConflict, "supplier_onboarding_required", "Complete supplier onboarding before "+action+". Open /app/onboarding to continue.")
		return false
	}
	if summary.Ready {
		return true
	}
	codes := make([]string, 0, len(summary.Missing))
	for _, requirement := range summary.Missing {
		codes = append(codes, requirement.Code)
	}
	writeProblem(w, http.StatusConflict, "supplier_not_ready", fmt.Sprintf("Complete these onboarding steps before %s: %s", action, strings.Join(codes, ", ")))
	return false
}

func ownerContactEvidence(user auth.User) (bool, bool) { return user.Email != "", user.Phone != "" }
