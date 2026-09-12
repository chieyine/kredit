package web

import (
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
	"kredit/internal/organizations"
	"kredit/internal/settlement"
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
	ExpectedVersion int64  `json:"expected_version"`
	BankCode        string `json:"bank_code"`
	AccountNumber   string `json:"account_number"`
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
	if err := s.syncOnboardingSecurity(orgID, user.ID); err != nil {
		writeProblem(w, 503, "onboarding_security_unavailable", "Account security requirements could not be checked")
		return
	}
	profile, summary, err := s.runtime.Onboarding.Get(orgID)
	if err != nil {
		writeProblem(w, 404, "onboarding_not_found", err.Error())
		return
	}
	profile = visibleOnboardingProfile(profile, membership.Role)
	writeJSON(w, 200, map[string]any{"profile": profile, "readiness": summary, "current_terms_version": summary.CurrentTermsVersion, "current_privacy_version": summary.CurrentPrivacyVersion, "permissions": onboardingPermissions(membership.Role)})
}

func (s *Server) requestOnboardingContactOTP(w http.ResponseWriter, r *http.Request) {
	orgID, _ := pathID(r, "organizationID")
	session, user, membership, ok := s.requireOrganizationAccess(w, r, orgID, access.PermissionReadOrganization)
	if !ok {
		return
	}
	if membership.Role != access.RoleOwner {
		writeProblem(w, 403, "owner_required", "Only the business owner can confirm the owner's phone or email.")
		return
	}
	if !s.requireFreshMFA(w, session) || !s.requireCSRF(w, r) {
		return
	}
	var in onboardingContactRequest
	if !decodeJSONRequest(w, r, &in) {
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
	if err := s.runtime.Notifications.SendOTP(r.Context(), challenge.TargetValue, challenge.TargetType, code); err != nil {
		writeProblem(w, http.StatusServiceUnavailable, "otp_delivery_unavailable", "We cannot send codes right now. Please try again shortly.")
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
		writeProblem(w, 403, "owner_required", "Only the business owner can confirm the owner's phone or email.")
		return
	}
	if !s.requireFreshMFA(w, session) || !s.requireCSRF(w, r) {
		return
	}
	var in onboardingContactVerifyRequest
	if !decodeJSONRequest(w, r, &in) {
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
		writeProblem(w, http.StatusNotFound, "organization_not_found", "We could not find that business.")
		return
	}
	verification, providerErr := s.runtime.Identity.CreateBusinessVerification(identity.WithActor(r.Context(), user.ID), identity.BusinessVerificationInput{RequireReview: true, SubjectID: orgID, LegalName: organization.LegalName, BusinessType: organization.BusinessType, Address: organization.BusinessAddress, Registration: organization.RegistrationInfo})
	if providerErr != nil {
		writeProblem(w, http.StatusServiceUnavailable, "kyb_provider_unavailable", providerErr.Error())
		return
	}
	p, sum, err := s.runtime.Onboarding.SubmitKYB(orgID, user.ID, identity.RoutedReference(verification.Provider, verification.ProviderID), in.ExpectedVersion)
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
	verification, err := identity.GetRouted(identity.WithActor(r.Context(), user.ID), s.runtime.Identity, profile.KYBProviderReference)
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
	case "expired":
		state, reason = "expired", "provider_expired"
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

	if s.runtime.Settlement == nil || s.runtime.Database == nil {
		writeProblem(w, 503, "settlement_unavailable", "Bank registration is not configured. Contact support.")
		return
	}
	profile, _, err := s.runtime.Onboarding.Get(orgID)
	if err != nil || profile.Version != in.ExpectedVersion || in.ExpectedVersion <= 0 {
		writeProblem(w, 409, "onboarding_conflict", "Your settings changed. Refresh before submitting.")
		return
	}
	banks, err := s.runtime.Settlement.Banks(r.Context())
	if err != nil {
		writeProblem(w, 503, "banks_unavailable", "Banks could not be loaded. Try again later.")
		return
	}
	bankName := ""
	for _, bank := range banks {
		if bank.Code == in.BankCode {
			bankName = bank.Name
			break
		}
	}
	if bankName == "" {
		writeProblem(w, 422, "invalid_bank", "Choose a bank from the current list.")
		return
	}
	if err = settlement.ValidateInput(settlement.Input{Reference: "validation", OrganizationID: orgID, BankCode: in.BankCode, AccountNumber: in.AccountNumber}); err != nil {
		writeProblem(w, 422, "invalid_bank_account", err.Error())
		return
	}
	result, err := settlement.Register(r.Context(), s.runtime.Database.Raw(), s.config.SettingsEncryptionKey, s.runtime.Settlement, settlement.Input{OrganizationID: orgID, BankCode: in.BankCode, AccountNumber: in.AccountNumber})
	in.AccountNumber = ""
	if err != nil {
		writeProblem(w, 409, "bank_registration_unconfirmed", "The bank registration is unconfirmed. Contact support before submitting a different account.")
		return
	}
	p, sum, err := s.runtime.Onboarding.UpdateSettlement(orgID, user.ID, onboarding.SettlementInput{ExpectedVersion: in.ExpectedVersion, Provider: s.runtime.Settlement.Name(), ProviderReference: result.ProviderReference, BankName: bankName, AccountName: result.AccountName, AccountLast4: result.AccountLast4})

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
		writeProblem(w, 403, "owner_required", "Only the business owner can accept these terms.")
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
		writeProblem(w, 403, "step_up_required", "Please prove it is really you first. Open your authenticator app and enter the code.")
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
	if s.runtime.Database == nil && (strings.Contains(action, "settlement") || strings.Contains(action, "billing")) {
		_, _ = s.runtime.EmitNotification(r.Context(), notifications.Event{ID: action + ":" + orgID + ":" + fmt.Sprint(p.Version), Type: "SupplierSensitiveSettingChanged", OrganizationID: orgID, Priority: notifications.PriorityCritical, Reference: action, NextAction: "Review the change in supplier settings.", SecurePath: "/app/onboarding"})
	}
	if strings.Contains(action, "kyb") {
		_, _ = s.runtime.EmitNotification(r.Context(), notifications.Event{ID: "supplier-kyb:" + orgID + ":" + fmt.Sprint(p.Version), Type: "SupplierVerificationOutcome", OrganizationID: orgID, Priority: notifications.PriorityCritical, Reference: p.KYBState, NextAction: "Review your business verification result.", SecurePath: "/app/onboarding"})
	}
	if sum.Ready && p.ReadinessChangedAt.Equal(p.UpdatedAt) {
		_, _ = s.runtime.EmitNotification(r.Context(), notifications.Event{ID: "supplier-pilot-ready:" + orgID + ":" + fmt.Sprint(p.Version), Type: "SupplierPilotReady", OrganizationID: orgID, Priority: notifications.PriorityCritical, Reference: orgID, NextAction: "Invite your team or create a credit request.", SecurePath: "/app/onboarding"})
	}
	writeJSON(w, 200, map[string]any{"profile": p, "readiness": sum})
}

func (s *Server) syncOnboardingSecurity(orgID, actor string) error {
	var members []organizations.Membership
	if source, ok := s.runtime.Organizations.(interface {
		ReadMembers(string) ([]organizations.Membership, error)
	}); ok {
		var err error
		members, err = source.ReadMembers(orgID)
		if err != nil {
			return err
		}
	} else {
		members = s.runtime.Organizations.ListMembers(orgID)
	}
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
	_, _, err := s.runtime.Onboarding.SyncSecurity(orgID, actor, ownerMFA, financeMFA)
	return err
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
	if err := s.syncOnboardingSecurity(organizationID, actorUserID); err != nil {
		writeProblem(w, 503, "onboarding_security_unavailable", "Account security requirements could not be checked")
		return false
	}
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
	writeProblem(w, http.StatusConflict, "supplier_not_ready", fmt.Sprintf("Finish these setup steps before %s: %s", action, strings.Join(codes, ", ")))
	return false
}

func ownerContactEvidence(user auth.User) (bool, bool) { return user.Email != "", user.Phone != "" }
