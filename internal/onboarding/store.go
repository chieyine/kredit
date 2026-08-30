package onboarding

import (
	"errors"
	"strings"
	"sync"
	"time"

	"kredit/internal/identifier"
)

const (
	CurrentTermsVersion   = "supplier-terms-v1"
	CurrentPrivacyVersion = "privacy-v1"
	StateIncomplete       = "incomplete"
	StateProviderReview   = "provider_review"
	StatePilotReady       = "pilot_ready"
)

type Profile struct {
	OrganizationID                string    `json:"organization_id"`
	Version                       int64     `json:"version"`
	AuthorizedRepresentativeName  string    `json:"authorized_representative_name"`
	AuthorizedRepresentativeTitle string    `json:"authorized_representative_title"`
	OwnerEmailVerifiedAt          time.Time `json:"owner_email_verified_at,omitempty"`
	OwnerPhoneVerifiedAt          time.Time `json:"owner_phone_verified_at,omitempty"`
	KYBState                      string    `json:"kyb_state"`
	KYBProviderReference          string    `json:"kyb_provider_reference,omitempty"`
	KYBReasonCode                 string    `json:"kyb_reason_code,omitempty"`
	KYBSubmittedAt                time.Time `json:"kyb_submitted_at,omitempty"`
	KYBDecidedAt                  time.Time `json:"kyb_decided_at,omitempty"`
	KYBExpiresAt                  time.Time `json:"kyb_expires_at,omitempty"`
	SettlementState               string    `json:"settlement_state"`
	SettlementProvider            string    `json:"settlement_provider,omitempty"`
	SettlementProviderReference   string    `json:"settlement_provider_reference,omitempty"`
	SettlementBankName            string    `json:"settlement_bank_name,omitempty"`
	SettlementAccountName         string    `json:"settlement_account_name,omitempty"`
	SettlementAccountLast4        string    `json:"settlement_account_last4,omitempty"`
	SettlementReasonCode          string    `json:"settlement_reason_code,omitempty"`
	SettlementChangedAt           time.Time `json:"settlement_changed_at,omitempty"`
	BillingState                  string    `json:"billing_state"`
	BillingMethod                 string    `json:"billing_method,omitempty"`
	BillingProviderReference      string    `json:"billing_provider_reference,omitempty"`
	BillingCycle                  string    `json:"billing_cycle,omitempty"`
	BillingChangedAt              time.Time `json:"billing_changed_at,omitempty"`
	DefaultCreditLimitKobo        int64     `json:"default_credit_limit_kobo,omitempty"`
	DefaultPaymentDays            int       `json:"default_payment_days,omitempty"`
	DefaultGraceHours             int       `json:"default_grace_hours,omitempty"`
	DefaultCreditPolicyUpdatedAt  time.Time `json:"default_credit_policy_updated_at,omitempty"`
	TermsVersion                  string    `json:"terms_version,omitempty"`
	TermsAcceptedAt               time.Time `json:"terms_accepted_at,omitempty"`
	PrivacyVersion                string    `json:"privacy_version,omitempty"`
	PrivacyAcceptedAt             time.Time `json:"privacy_accepted_at,omitempty"`
	OwnerMFAVerifiedAt            time.Time `json:"owner_mfa_verified_at,omitempty"`
	FinanceMFAComplete            bool      `json:"finance_mfa_complete"`
	ReadinessState                string    `json:"readiness_state"`
	ReadinessChangedAt            time.Time `json:"readiness_changed_at"`
	CreatedAt                     time.Time `json:"created_at"`
	UpdatedAt                     time.Time `json:"updated_at"`
}

type Requirement struct {
	Code       string `json:"code"`
	Label      string `json:"label"`
	Complete   bool   `json:"complete"`
	ManagePath string `json:"manage_path,omitempty"`
}

type Summary struct {
	State        string        `json:"state"`
	Ready        bool          `json:"ready"`
	Version      int64         `json:"version"`
	Requirements []Requirement `json:"requirements"`
	Missing      []Requirement `json:"missing"`
}

type Revision struct {
	ID             string    `json:"id"`
	OrganizationID string    `json:"organization_id"`
	ProfileVersion int64     `json:"profile_version"`
	ChangeType     string    `json:"change_type"`
	ActorUserID    string    `json:"actor_user_id,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
}

type RepresentativeInput struct {
	ExpectedVersion int64
	Name, Title     string
}
type SettlementInput struct {
	ExpectedVersion                                                  int64
	Provider, ProviderReference, BankName, AccountName, AccountLast4 string
}
type BillingInput struct {
	ExpectedVersion                  int64
	Method, ProviderReference, Cycle string
}
type CreditPolicyInput struct {
	ExpectedVersion int64
	PaymentDays     int
	GraceHours      int
	CreditLimitKobo int64
}

type Service interface {
	Ensure(organizationID, actorUserID string, emailVerified, phoneVerified bool) (Profile, error)
	Get(organizationID string) (Profile, Summary, error)
	UpdateRepresentative(organizationID, actorUserID string, input RepresentativeInput) (Profile, Summary, error)
	RecordContactVerified(organizationID, actorUserID, channel string) (Profile, Summary, error)
	SubmitKYB(organizationID, actorUserID, providerReference string, expectedVersion int64) (Profile, Summary, error)
	RecordKYBDecision(organizationID, actorUserID, state, reason string, expiresAt time.Time) (Profile, Summary, error)
	UpdateSettlement(organizationID, actorUserID string, input SettlementInput) (Profile, Summary, error)
	RecordSettlementDecision(organizationID, actorUserID, state, reason string) (Profile, Summary, error)
	UpdateBilling(organizationID, actorUserID string, input BillingInput) (Profile, Summary, error)
	UpdateCreditPolicy(organizationID, actorUserID string, input CreditPolicyInput) (Profile, Summary, error)
	AcceptConsents(organizationID, actorUserID string, expectedVersion int64, termsVersion, privacyVersion string) (Profile, Summary, error)
	SyncSecurity(organizationID, actorUserID string, ownerMFA, financeMFAComplete bool) (Profile, Summary, error)
	Reconcile(now time.Time) []Profile
}

type Store struct {
	mu        sync.RWMutex
	profiles  map[string]*Profile
	revisions map[string][]Revision
	now       func() time.Time
}

func NewStore() *Store {
	return &Store{profiles: map[string]*Profile{}, revisions: map[string][]Revision{}, now: func() time.Time { return time.Now().UTC() }}
}

func (s *Store) Ensure(org, actor string, email, phone bool) (Profile, error) {
	if strings.TrimSpace(org) == "" {
		return Profile{}, errors.New("organization is required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if p := s.profiles[org]; p != nil {
		return *p, nil
	}
	now := s.now()
	p := &Profile{OrganizationID: org, Version: 1, KYBState: "not_started", SettlementState: "not_started", BillingState: "not_started", FinanceMFAComplete: true, ReadinessState: StateIncomplete, ReadinessChangedAt: now, CreatedAt: now, UpdatedAt: now}
	if email {
		p.OwnerEmailVerifiedAt = now
	}
	if phone {
		p.OwnerPhoneVerifiedAt = now
	}
	s.profiles[org] = p
	s.revisionLocked(p, actor, "profile.created")
	return *p, nil
}

func (s *Store) Get(org string) (Profile, Summary, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	p := s.profiles[org]
	if p == nil {
		return Profile{}, Summary{}, errors.New("onboarding profile not found")
	}
	return *p, summarize(*p, s.now()), nil
}

func (s *Store) mutate(org, actor, change string, expected int64, fn func(*Profile) error) (Profile, Summary, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p := s.profiles[org]
	if p == nil {
		return Profile{}, Summary{}, errors.New("onboarding profile not found")
	}
	if expected > 0 && p.Version != int64(expected) {
		return Profile{}, Summary{}, errors.New("onboarding profile version conflict")
	}
	if err := fn(p); err != nil {
		return Profile{}, Summary{}, err
	}
	now := s.now()
	oldState := p.ReadinessState
	p.Version++
	p.UpdatedAt = now
	p.ReadinessState = summarize(*p, now).State
	if oldState != p.ReadinessState {
		p.ReadinessChangedAt = now
	}
	s.revisionLocked(p, actor, change)
	return *p, summarize(*p, now), nil
}

func (s *Store) UpdateRepresentative(org, actor string, in RepresentativeInput) (Profile, Summary, error) {
	return s.mutate(org, actor, "representative.updated", in.ExpectedVersion, func(p *Profile) error {
		p.AuthorizedRepresentativeName = strings.TrimSpace(in.Name)
		p.AuthorizedRepresentativeTitle = strings.TrimSpace(in.Title)
		if p.AuthorizedRepresentativeName == "" || p.AuthorizedRepresentativeTitle == "" {
			return errors.New("representative name and title are required")
		}
		return nil
	})
}
func (s *Store) RecordContactVerified(org, actor, channel string) (Profile, Summary, error) {
	return s.mutate(org, actor, "contact.verified", 0, func(p *Profile) error {
		if channel == "email" {
			p.OwnerEmailVerifiedAt = s.now()
		} else if channel == "phone" {
			p.OwnerPhoneVerifiedAt = s.now()
		} else {
			return errors.New("contact channel must be email or phone")
		}
		return nil
	})
}
func (s *Store) SubmitKYB(org, actor, ref string, expected int64) (Profile, Summary, error) {
	return s.mutate(org, actor, "kyb.submitted", expected, func(p *Profile) error {
		ref = strings.TrimSpace(ref)
		if ref == "" {
			return errors.New("KYB provider reference is required")
		}
		p.KYBProviderReference, p.KYBState, p.KYBReasonCode, p.KYBSubmittedAt = ref, "submitted", "", s.now()
		return nil
	})
}
func (s *Store) RecordKYBDecision(org, actor, state, reason string, expires time.Time) (Profile, Summary, error) {
	return s.mutate(org, actor, "kyb.decision", 0, func(p *Profile) error {
		if state != "provider_review" && state != "approved" && state != "rejected" && state != "expired" {
			return errors.New("invalid KYB provider state")
		}
		if p.KYBProviderReference == "" {
			return errors.New("KYB has not been submitted")
		}
		p.KYBState, p.KYBReasonCode, p.KYBDecidedAt, p.KYBExpiresAt = state, strings.TrimSpace(reason), s.now(), expires
		return nil
	})
}
func (s *Store) UpdateSettlement(org, actor string, in SettlementInput) (Profile, Summary, error) {
	return s.mutate(org, actor, "settlement.updated", in.ExpectedVersion, func(p *Profile) error {
		if strings.TrimSpace(in.Provider) == "" || strings.TrimSpace(in.ProviderReference) == "" || len(strings.TrimSpace(in.AccountLast4)) != 4 {
			return errors.New("provider reference and four masked account digits are required")
		}
		for _, r := range in.AccountLast4 {
			if r < '0' || r > '9' {
				return errors.New("account last4 must contain digits only")
			}
		}
		p.SettlementProvider, p.SettlementProviderReference, p.SettlementBankName, p.SettlementAccountName, p.SettlementAccountLast4 = strings.TrimSpace(in.Provider), strings.TrimSpace(in.ProviderReference), strings.TrimSpace(in.BankName), strings.TrimSpace(in.AccountName), strings.TrimSpace(in.AccountLast4)
		p.SettlementState, p.SettlementReasonCode, p.SettlementChangedAt = "pending_verification", "", s.now()
		return nil
	})
}
func (s *Store) RecordSettlementDecision(org, actor, state, reason string) (Profile, Summary, error) {
	return s.mutate(org, actor, "settlement.decision", 0, func(p *Profile) error {
		if state != "provider_review" && state != "verified" && state != "rejected" && state != "expired" {
			return errors.New("invalid settlement provider state")
		}
		if p.SettlementProviderReference == "" {
			return errors.New("settlement destination has not been submitted")
		}
		p.SettlementState, p.SettlementReasonCode, p.SettlementChangedAt = state, strings.TrimSpace(reason), s.now()
		return nil
	})
}
func (s *Store) UpdateBilling(org, actor string, in BillingInput) (Profile, Summary, error) {
	return s.mutate(org, actor, "billing.updated", in.ExpectedVersion, func(p *Profile) error {
		if in.Method != "split_settlement" && in.Method != "authorized_debit" && in.Method != "consolidated_invoice" {
			return errors.New("invalid billing method")
		}
		if strings.TrimSpace(in.ProviderReference) == "" {
			return errors.New("billing provider reference is required")
		}
		p.BillingMethod, p.BillingProviderReference, p.BillingCycle, p.BillingState, p.BillingChangedAt = in.Method, strings.TrimSpace(in.ProviderReference), strings.TrimSpace(in.Cycle), "configured", s.now()
		return nil
	})
}
func (s *Store) UpdateCreditPolicy(org, actor string, in CreditPolicyInput) (Profile, Summary, error) {
	return s.mutate(org, actor, "credit_policy.updated", in.ExpectedVersion, func(p *Profile) error {
		if in.CreditLimitKobo <= 0 || in.PaymentDays < 1 || in.PaymentDays > 365 || in.GraceHours < 0 || in.GraceHours > 720 {
			return errors.New("valid credit limit, payment days, and grace hours are required")
		}
		p.DefaultCreditLimitKobo, p.DefaultPaymentDays, p.DefaultGraceHours, p.DefaultCreditPolicyUpdatedAt = in.CreditLimitKobo, in.PaymentDays, in.GraceHours, s.now()
		return nil
	})
}
func (s *Store) AcceptConsents(org, actor string, expected int64, terms, privacy string) (Profile, Summary, error) {
	return s.mutate(org, actor, "consents.accepted", expected, func(p *Profile) error {
		if terms != CurrentTermsVersion || privacy != CurrentPrivacyVersion {
			return errors.New("current terms and privacy versions must be accepted")
		}
		now := s.now()
		p.TermsVersion, p.PrivacyVersion, p.TermsAcceptedAt, p.PrivacyAcceptedAt = terms, privacy, now, now
		return nil
	})
}
func (s *Store) SyncSecurity(org, actor string, ownerMFA, financeMFA bool) (Profile, Summary, error) {
	s.mu.RLock()
	current := s.profiles[org]
	if current != nil && (!current.OwnerMFAVerifiedAt.IsZero()) == ownerMFA && current.FinanceMFAComplete == financeMFA {
		profile := *current
		s.mu.RUnlock()
		return profile, summarize(profile, s.now()), nil
	}
	s.mu.RUnlock()
	return s.mutate(org, actor, "security.synced", 0, func(p *Profile) error {
		if ownerMFA && p.OwnerMFAVerifiedAt.IsZero() {
			p.OwnerMFAVerifiedAt = s.now()
		} else {
			p.OwnerMFAVerifiedAt = time.Time{}
		}
		p.FinanceMFAComplete = financeMFA
		return nil
	})
}

func (s *Store) Reconcile(now time.Time) []Profile {
	s.mu.Lock()
	defer s.mu.Unlock()
	var changed []Profile
	for _, p := range s.profiles {
		expired := (p.KYBState == "approved" && !p.KYBExpiresAt.IsZero() && !now.Before(p.KYBExpiresAt))
		if expired {
			p.KYBState = "expired"
		}
		state := summarize(*p, now).State
		if expired || state != p.ReadinessState {
			p.ReadinessState = state
			p.ReadinessChangedAt = now
			p.UpdatedAt = now
			p.Version++
			s.revisionLocked(p, "system", "requirements.reconciled")
			changed = append(changed, *p)
		}
	}
	return changed
}

func summarize(p Profile, now time.Time) Summary {
	kybApproved := p.KYBState == "approved" && (p.KYBExpiresAt.IsZero() || now.Before(p.KYBExpiresAt))
	reqs := []Requirement{
		{Code: "business_identity", Label: "Business identity and authorised representative", Complete: p.AuthorizedRepresentativeName != "" && p.AuthorizedRepresentativeTitle != "", ManagePath: "/app/onboarding"},
		{Code: "email_verified", Label: "Owner email verified", Complete: !p.OwnerEmailVerifiedAt.IsZero(), ManagePath: "/app/onboarding"},
		{Code: "phone_verified", Label: "Owner phone verified", Complete: !p.OwnerPhoneVerifiedAt.IsZero(), ManagePath: "/app/onboarding"},
		{Code: "kyb_approved", Label: "Business verification approved", Complete: kybApproved, ManagePath: "/app/onboarding"},
		{Code: "settlement_verified", Label: "Settlement destination verified", Complete: p.SettlementState == "verified", ManagePath: "/app/settings/settlement"},
		{Code: "billing_configured", Label: "Billing method configured", Complete: p.BillingState == "configured", ManagePath: "/app/settings/billing"},
		{Code: "credit_policy", Label: "Default credit policy configured", Complete: p.DefaultCreditPolicyUpdatedAt.IsZero() == false, ManagePath: "/app/settings/credit-policy"},
		{Code: "current_consents", Label: "Current terms and privacy accepted", Complete: p.TermsVersion == CurrentTermsVersion && p.PrivacyVersion == CurrentPrivacyVersion, ManagePath: "/app/onboarding"},
		{Code: "owner_mfa", Label: "Owner MFA active", Complete: !p.OwnerMFAVerifiedAt.IsZero(), ManagePath: "/app/settings/security"},
		{Code: "finance_mfa", Label: "Every active finance user has MFA", Complete: p.FinanceMFAComplete, ManagePath: "/app/team"},
	}
	missing := []Requirement{}
	for _, r := range reqs {
		if !r.Complete {
			missing = append(missing, r)
		}
	}
	state := StateIncomplete
	if len(missing) == 0 {
		state = StatePilotReady
	} else if p.KYBState == "rejected" || p.SettlementState == "rejected" {
		state = "rejected"
	} else if p.KYBState == "expired" || p.SettlementState == "expired" || p.BillingState == "expired" {
		state = "expired"
	} else if p.KYBState == "submitted" || p.KYBState == "provider_review" || p.SettlementState == "pending_verification" || p.SettlementState == "provider_review" {
		state = StateProviderReview
	}
	return Summary{State: state, Ready: state == StatePilotReady, Version: p.Version, Requirements: reqs, Missing: missing}
}

func (s *Store) revisionLocked(p *Profile, actor, change string) {
	s.revisions[p.OrganizationID] = append(s.revisions[p.OrganizationID], Revision{ID: identifier.New(), OrganizationID: p.OrganizationID, ProfileVersion: p.Version, ChangeType: change, ActorUserID: actor, CreatedAt: s.now()})
}
