package buyers

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"sync"
	"time"

	"kredit/internal/identity"
)

type Invitation struct {
	ID                   string    `json:"id"`
	OrganizationID       string    `json:"organization_id"`
	TargetType           string    `json:"target_type"`
	ProposedLegalName    string    `json:"proposed_legal_name"`
	ProposedTradingName  string    `json:"proposed_trading_name,omitempty"`
	ProposedBusinessType string    `json:"proposed_business_type"`
	ProposedAddress      string    `json:"proposed_address"`
	ProposedIndustry     string    `json:"proposed_industry"`
	Status               string    `json:"status"`
	ExpiresAt            time.Time `json:"expires_at"`
	CreatedAt            time.Time `json:"created_at"`
	AcceptedAt           time.Time `json:"accepted_at,omitempty"`
	AcceptedByUserID     string    `json:"accepted_by_user_id,omitempty"`
}

type InvitationPreview struct {
	ID                   string    `json:"id"`
	OrganizationID       string    `json:"organization_id"`
	TargetType           string    `json:"target_type"`
	ProposedLegalName    string    `json:"proposed_legal_name"`
	ProposedTradingName  string    `json:"proposed_trading_name,omitempty"`
	ProposedBusinessType string    `json:"proposed_business_type"`
	ProposedAddress      string    `json:"proposed_address"`
	ProposedIndustry     string    `json:"proposed_industry"`
	Status               string    `json:"status"`
	ExpiresAt            time.Time `json:"expires_at"`
}

type CreateInvitationInput struct {
	Target          string
	TargetType      string
	LegalName       string
	TradingName     string
	BusinessType    string
	BusinessAddress string
	Industry        string
}

type AcceptInput struct {
	FullName        string
	LegalName       string
	TradingName     string
	BusinessType    string
	BusinessAddress string
	Industry        string
}

type Person struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	FullName  string    `json:"full_name"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

type Business struct {
	ID              string    `json:"id"`
	OwnerUserID     string    `json:"owner_user_id"`
	LegalName       string    `json:"legal_name"`
	TradingName     string    `json:"trading_name,omitempty"`
	BusinessType    string    `json:"business_type"`
	BusinessAddress string    `json:"business_address"`
	Industry        string    `json:"industry"`
	Status          string    `json:"status"`
	CreatedAt       time.Time `json:"created_at"`
}

type Representative struct {
	ID              string    `json:"id"`
	BusinessID      string    `json:"business_id"`
	PersonID        string    `json:"person_id"`
	RoleTitle       string    `json:"role_title"`
	AuthorityType   string    `json:"authority_type"`
	AuthorityStatus string    `json:"authority_status"`
	CreatedAt       time.Time `json:"created_at"`
}

type VerificationCase struct {
	ID                string            `json:"id"`
	SubjectType       string            `json:"subject_type"`
	SubjectID         string            `json:"subject_id"`
	Provider          string            `json:"provider"`
	ProviderReference string            `json:"provider_reference"`
	VerificationLevel int               `json:"verification_level"`
	State             string            `json:"state"`
	Reasons           []string          `json:"reasons,omitempty"`
	SafeResult        map[string]string `json:"safe_result,omitempty"`
	StartedAt         time.Time         `json:"started_at"`
	CompletedAt       time.Time         `json:"completed_at,omitempty"`
	ExpiresAt         time.Time         `json:"expires_at,omitempty"`
}

type Consent struct {
	ID          string    `json:"id"`
	UserID      string    `json:"user_id"`
	ConsentType string    `json:"consent_type"`
	Version     string    `json:"version"`
	AcceptedAt  time.Time `json:"accepted_at"`
}

type BankAccountReference struct {
	ID                string    `json:"id"`
	OwnerType         string    `json:"owner_type"`
	OwnerID           string    `json:"owner_id"`
	Provider          string    `json:"provider"`
	ProviderReference string    `json:"provider_reference"`
	BankCode          string    `json:"bank_code,omitempty"`
	MaskedAccount     string    `json:"masked_account,omitempty"`
	AccountNameResult string    `json:"account_name_result,omitempty"`
	AccountType       string    `json:"account_type,omitempty"`
	OwnershipState    string    `json:"ownership_state"`
	Active            bool      `json:"active"`
	CreatedAt         time.Time `json:"created_at"`
}

type Portal struct {
	Person            Person                 `json:"person"`
	Business          Business               `json:"business"`
	Representative    Representative         `json:"representative"`
	VerificationCases []VerificationCase     `json:"verification_cases"`
	Consents          []Consent              `json:"consents"`
	BankAccounts      []BankAccountReference `json:"bank_accounts"`
}

type CreateInvitationResult struct {
	Invitation Invitation
	RawToken   string
}

type Customer struct {
	BuyerUserID     string `json:"buyer_user_id"`
	BuyerBusinessID string `json:"buyer_business_id"`
	LegalName       string `json:"legal_name"`
	TradingName     string `json:"trading_name,omitempty"`
	Industry        string `json:"industry"`
	Status          string `json:"status"`
}

type invitationRecord struct {
	Invitation     Invitation
	targetValue    string
	tokenHash      string
	lookupAttempts int
}

type Store struct {
	mu              sync.RWMutex
	tokenHashKey    []byte
	identity        identity.IdentityProvider
	invitations     map[string]*invitationRecord
	persons         map[string]*Person
	personsByUser   map[string]string
	businesses      map[string]*Business
	representatives map[string]*Representative
	verifications   map[string]*VerificationCase
	consents        map[string][]*Consent
	bankAccounts    map[string][]*BankAccountReference
	now             func() time.Time
	newID           func() string
	invitationGuard func(CreateInvitationInput) error
	acceptanceGuard func(AcceptInput) error
}

// Service is the buyer identity and invitation boundary consumed by HTTP
// handlers. The development store and PostgreSQL implementation intentionally
// share this contract so the runtime cannot accidentally mix persistence
// semantics between environments.
type Service interface {
	SetInvitationGuard(func(CreateInvitationInput) error)
	SetAcceptanceGuard(func(AcceptInput) error)
	CountBusinesses() int
	CreateInvitation(string, string, CreateInvitationInput) (CreateInvitationResult, error)
	Preview(string) (InvitationPreview, error)
	InvitationTarget(string) (string, string, error)
	Accept(context.Context, string, string, AcceptInput) (Portal, error)
	Portal(string) (Portal, error)
	ListCustomers(string) []Customer
	AddBankAccountReference(string, string, string, BankAccountReference) (BankAccountReference, error)
}

func (s *Store) ListCustomers(organizationID string) []Customer {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := []Customer{}
	seen := map[string]bool{}
	for _, record := range s.invitations {
		invitation := record.Invitation
		if invitation.OrganizationID != organizationID || invitation.Status != "accepted" || seen[invitation.AcceptedByUserID] {
			continue
		}
		for _, business := range s.businesses {
			if business.OwnerUserID == invitation.AcceptedByUserID {
				result = append(result, Customer{BuyerUserID: business.OwnerUserID, BuyerBusinessID: business.ID, LegalName: business.LegalName, TradingName: business.TradingName, Industry: business.Industry, Status: business.Status})
				seen[invitation.AcceptedByUserID] = true
				break
			}
		}
	}
	return result
}

var _ Service = (*Store)(nil)

func (s *Store) SetInvitationGuard(guard func(CreateInvitationInput) error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.invitationGuard = guard
}

func (s *Store) SetAcceptanceGuard(guard func(AcceptInput) error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.acceptanceGuard = guard
}

func (s *Store) CountBusinesses() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.businesses)
}

func NewStore(tokenHashKey string, provider identity.IdentityProvider) *Store {
	if tokenHashKey == "" {
		tokenHashKey = "development-only-change-me"
	}
	return &Store{
		tokenHashKey:    []byte(tokenHashKey),
		identity:        provider,
		invitations:     make(map[string]*invitationRecord),
		persons:         make(map[string]*Person),
		personsByUser:   make(map[string]string),
		businesses:      make(map[string]*Business),
		representatives: make(map[string]*Representative),
		verifications:   make(map[string]*VerificationCase),
		consents:        make(map[string][]*Consent),
		bankAccounts:    make(map[string][]*BankAccountReference),
		now:             func() time.Time { return time.Now().UTC() },
		newID:           randomID,
	}
}

func (s *Store) CreateInvitation(actorUserID, organizationID string, input CreateInvitationInput) (CreateInvitationResult, error) {
	if actorUserID == "" || organizationID == "" {
		return CreateInvitationResult{}, errors.New("actor and organization are required")
	}
	if err := validateInvitationInput(input); err != nil {
		return CreateInvitationResult{}, err
	}
	s.mu.RLock()
	guard := s.invitationGuard
	s.mu.RUnlock()
	if guard != nil {
		if err := guard(input); err != nil {
			return CreateInvitationResult{}, err
		}
	}
	rawToken, err := randomToken()
	if err != nil {
		return CreateInvitationResult{}, err
	}
	now := s.now()
	invitation := Invitation{ID: s.newID(), OrganizationID: organizationID, TargetType: input.TargetType, ProposedLegalName: strings.TrimSpace(input.LegalName), ProposedTradingName: strings.TrimSpace(input.TradingName), ProposedBusinessType: strings.TrimSpace(input.BusinessType), ProposedAddress: strings.TrimSpace(input.BusinessAddress), ProposedIndustry: strings.TrimSpace(input.Industry), Status: "pending", ExpiresAt: now.Add(7 * 24 * time.Hour), CreatedAt: now}
	record := &invitationRecord{Invitation: invitation, targetValue: normalizeTarget(input.Target), tokenHash: s.hashToken(rawToken)}
	s.mu.Lock()
	s.invitations[record.tokenHash] = record
	s.mu.Unlock()
	return CreateInvitationResult{Invitation: invitation, RawToken: rawToken}, nil
}

func (s *Store) Preview(rawToken string) (InvitationPreview, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	record, err := s.lookupLocked(rawToken)
	if err != nil {
		return InvitationPreview{}, err
	}
	record.lookupAttempts++
	if record.lookupAttempts > 10 {
		return InvitationPreview{}, errors.New("invitation lookup rate limit exceeded")
	}
	invitation := record.Invitation
	return InvitationPreview{ID: invitation.ID, OrganizationID: invitation.OrganizationID, TargetType: invitation.TargetType, ProposedLegalName: invitation.ProposedLegalName, ProposedTradingName: invitation.ProposedTradingName, ProposedBusinessType: invitation.ProposedBusinessType, ProposedAddress: invitation.ProposedAddress, ProposedIndustry: invitation.ProposedIndustry, Status: invitation.Status, ExpiresAt: invitation.ExpiresAt}, nil
}

func (s *Store) InvitationTarget(rawToken string) (string, string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	record, err := s.lookupLocked(rawToken)
	if err != nil {
		return "", "", err
	}
	if record.Invitation.Status != "pending" {
		return "", "", errors.New("invitation is no longer available")
	}
	return record.Invitation.TargetType, record.targetValue, nil
}

func (s *Store) Accept(ctx context.Context, rawToken, userID string, input AcceptInput) (Portal, error) {
	if userID == "" || strings.TrimSpace(input.FullName) == "" {
		return Portal{}, errors.New("authenticated user and full name are required")
	}
	if s.identity == nil {
		return Portal{}, errors.New("identity provider is not configured")
	}
	s.mu.RLock()
	guard := s.acceptanceGuard
	s.mu.RUnlock()
	if guard != nil {
		if err := guard(input); err != nil {
			return Portal{}, err
		}
	}
	s.mu.Lock()
	record, err := s.lookupLocked(rawToken)
	if err != nil {
		s.mu.Unlock()
		return Portal{}, err
	}
	if record.Invitation.Status != "pending" {
		s.mu.Unlock()
		return Portal{}, errors.New("invitation is no longer available")
	}
	invitation := record.Invitation
	legalName := defaultValue(input.LegalName, invitation.ProposedLegalName)
	tradingName := defaultValue(input.TradingName, invitation.ProposedTradingName)
	businessType := defaultValue(input.BusinessType, invitation.ProposedBusinessType)
	address := defaultValue(input.BusinessAddress, invitation.ProposedAddress)
	industry := defaultValue(input.Industry, invitation.ProposedIndustry)
	if legalName == "" || businessType == "" || address == "" || industry == "" {
		s.mu.Unlock()
		return Portal{}, errors.New("complete buyer business details are required")
	}
	now := s.now()
	personID := s.newID()
	businessID := s.newID()
	representativeID := s.newID()
	person := &Person{ID: personID, UserID: userID, FullName: strings.TrimSpace(input.FullName), Status: "invited", CreatedAt: now}
	business := &Business{ID: businessID, OwnerUserID: userID, LegalName: legalName, TradingName: tradingName, BusinessType: businessType, BusinessAddress: address, Industry: industry, Status: "pending_verification", CreatedAt: now}
	representative := &Representative{ID: representativeID, BusinessID: businessID, PersonID: personID, RoleTitle: "authorised representative", AuthorityType: "buyer_acceptance", AuthorityStatus: "pending", CreatedAt: now}
	s.persons[person.ID] = person
	s.personsByUser[userID] = person.ID
	s.businesses[business.ID] = business
	s.representatives[representative.ID] = representative
	s.mu.Unlock()
	accepted := false
	defer func() {
		if accepted {
			return
		}
		s.mu.Lock()
		delete(s.persons, person.ID)
		delete(s.personsByUser, userID)
		delete(s.businesses, business.ID)
		delete(s.representatives, representative.ID)
		s.mu.Unlock()
	}()

	personSession, err := s.identity.CreatePersonVerification(ctx, identity.PersonVerificationInput{SubjectID: person.ID, FullName: person.FullName})
	if err != nil {
		return Portal{}, err
	}
	businessSession, err := s.identity.CreateBusinessVerification(ctx, identity.BusinessVerificationInput{SubjectID: business.ID, LegalName: business.LegalName, BusinessType: business.BusinessType, Address: business.BusinessAddress})
	if err != nil {
		return Portal{}, err
	}
	authoritySession, err := s.identity.CreateAuthorityVerification(ctx, identity.AuthorityVerificationInput{SubjectID: representative.ID, PersonID: person.ID, BusinessID: business.ID, RoleTitle: representative.RoleTitle})
	if err != nil {
		return Portal{}, err
	}
	if !verificationComplete(personSession) || !verificationComplete(businessSession) || !verificationComplete(authoritySession) {
		return Portal{}, errors.New("identity, business, and authority verification must complete before onboarding")
	}
	s.mu.Lock()
	if record.Invitation.Status != "pending" {
		s.mu.Unlock()
		return Portal{}, errors.New("invitation was accepted by another session")
	}
	record.Invitation.Status = "accepted"
	record.Invitation.AcceptedAt = s.now()
	record.Invitation.AcceptedByUserID = userID
	person.Status = "verified"
	business.Status = "verified"
	representative.AuthorityStatus = "verified"
	s.addVerificationLocked(person.ID, "person", personSession)
	s.addVerificationLocked(business.ID, "business", businessSession)
	s.addVerificationLocked(representative.ID, "authority", authoritySession)
	s.addConsentLocked(userID, "buyer_portal", "v1")
	s.addConsentLocked(userID, "identity_verification", "v1")
	s.addConsentLocked(userID, "privacy_notice", "v1")
	portal := s.portalLocked(userID)
	accepted = true
	s.mu.Unlock()
	return portal, nil
}

func verificationComplete(session identity.VerificationSession) bool {
	return strings.EqualFold(session.State, "verified") && session.VerificationLevel >= 2
}

func (s *Store) Portal(userID string) (Portal, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	portal, ok := s.portalForUserLocked(userID)
	if !ok {
		return Portal{}, errors.New("buyer portal profile not found")
	}
	return portal, nil
}

func (s *Store) AddBankAccountReference(userID, ownerType, ownerID string, reference BankAccountReference) (BankAccountReference, error) {
	if userID == "" || ownerID == "" || reference.Provider == "" || reference.ProviderReference == "" {
		return BankAccountReference{}, errors.New("bank account provider reference is required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	reference.ID = s.newID()
	reference.OwnerType = ownerType
	reference.OwnerID = ownerID
	reference.Active = true
	reference.OwnershipState = defaultValue(reference.OwnershipState, "pending")
	reference.CreatedAt = s.now()
	s.bankAccounts[userID] = append(s.bankAccounts[userID], &reference)
	return reference, nil
}

func (s *Store) lookupLocked(rawToken string) (*invitationRecord, error) {
	if strings.TrimSpace(rawToken) == "" {
		return nil, errors.New("invitation token is required")
	}
	record, ok := s.invitations[s.hashToken(rawToken)]
	if !ok || !s.now().Before(record.Invitation.ExpiresAt) {
		return nil, errors.New("invitation is invalid or expired")
	}
	return record, nil
}

func (s *Store) addVerificationLocked(subjectID, subjectType string, session identity.VerificationSession) {
	verification := &VerificationCase{ID: s.newID(), SubjectType: subjectType, SubjectID: subjectID, Provider: session.Provider, ProviderReference: session.ProviderID, VerificationLevel: session.VerificationLevel, State: session.State, SafeResult: cloneMap(session.SafeResult), StartedAt: s.now(), CompletedAt: s.now(), ExpiresAt: session.ExpiresAt}
	s.verifications[verification.ID] = verification
}

func (s *Store) addConsentLocked(userID, consentType, version string) {
	s.consents[userID] = append(s.consents[userID], &Consent{ID: s.newID(), UserID: userID, ConsentType: consentType, Version: version, AcceptedAt: s.now()})
}

func (s *Store) portalLocked(userID string) Portal {
	portal, _ := s.portalForUserLocked(userID)
	return portal
}

func (s *Store) portalForUserLocked(userID string) (Portal, bool) {
	personID := s.personsByUser[userID]
	person, ok := s.persons[personID]
	if !ok {
		return Portal{}, false
	}
	var business *Business
	var representative *Representative
	for _, candidate := range s.representatives {
		if candidate.PersonID == person.ID {
			representative = candidate
			business = s.businesses[candidate.BusinessID]
			break
		}
	}
	if business == nil || representative == nil {
		return Portal{}, false
	}
	verificationCases := make([]VerificationCase, 0)
	for _, verification := range s.verifications {
		if verification.SubjectID == person.ID || verification.SubjectID == business.ID || verification.SubjectID == representative.ID {
			verificationCases = append(verificationCases, cloneVerification(*verification))
		}
	}
	consents := make([]Consent, 0, len(s.consents[userID]))
	for _, consent := range s.consents[userID] {
		consents = append(consents, *consent)
	}
	accounts := make([]BankAccountReference, 0, len(s.bankAccounts[userID]))
	for _, account := range s.bankAccounts[userID] {
		accounts = append(accounts, *account)
	}
	return Portal{Person: clonePerson(*person), Business: cloneBusiness(*business), Representative: cloneRepresentative(*representative), VerificationCases: verificationCases, Consents: consents, BankAccounts: accounts}, true
}

func validateInvitationInput(input CreateInvitationInput) error {
	if normalizeTarget(input.Target) == "" || (input.TargetType != "phone" && input.TargetType != "email") {
		return errors.New("valid buyer target and target type are required")
	}
	if strings.TrimSpace(input.LegalName) == "" || strings.TrimSpace(input.BusinessType) == "" || strings.TrimSpace(input.BusinessAddress) == "" || strings.TrimSpace(input.Industry) == "" {
		return errors.New("buyer legal name, business type, address, and industry are required")
	}
	return nil
}

func normalizeTarget(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if strings.Contains(value, "@") {
		return value
	}
	for _, separator := range []string{" ", "-", "(", ")"} {
		value = strings.ReplaceAll(value, separator, "")
	}
	return value
}

func defaultValue(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return strings.TrimSpace(fallback)
	}
	return strings.TrimSpace(value)
}

func (s *Store) hashToken(token string) string {
	mac := hmac.New(sha256.New, s.tokenHashKey)
	_, _ = mac.Write([]byte(token))
	return hex.EncodeToString(mac.Sum(nil))
}

func randomToken() (string, error) {
	value := make([]byte, 32)
	if _, err := rand.Read(value); err != nil {
		return "", err
	}
	return hex.EncodeToString(value), nil
}

func randomID() string {
	token, err := randomToken()
	if err != nil {
		return hex.EncodeToString([]byte(time.Now().String()))[:32]
	}
	return token[:32]
}

func cloneMap(input map[string]string) map[string]string {
	if input == nil {
		return nil
	}
	output := make(map[string]string, len(input))
	for key, value := range input {
		output[key] = value
	}
	return output
}

func clonePerson(value Person) Person                         { return value }
func cloneBusiness(value Business) Business                   { return value }
func cloneRepresentative(value Representative) Representative { return value }
func cloneVerification(value VerificationCase) VerificationCase {
	value.SafeResult = cloneMap(value.SafeResult)
	value.Reasons = append([]string(nil), value.Reasons...)
	return value
}
