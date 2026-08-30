package credit

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"kredit/internal/identifier"
	"kredit/internal/ledger"
	"kredit/internal/mandates"
	"kredit/internal/payments"
)

const (
	Draft                      = "DRAFT"
	Sent                       = "SENT"
	BuyerReviewing             = "BUYER_REVIEWING"
	BuyerAccepted              = "BUYER_ACCEPTED"
	VerificationPending        = "VERIFICATION_PENDING"
	ReadyToRelease             = "READY_TO_RELEASE"
	GoodsReleased              = "GOODS_RELEASED"
	ReceiptConfirmationPending = "RECEIPT_CONFIRMATION_PENDING"
	Active                     = "ACTIVE"
	Cancelled                  = "CANCELLED"
	Declined                   = "DECLINED"
)

type CreditRequest struct {
	ID                     string         `json:"id"`
	SupplierOrganizationID string         `json:"supplier_organization_id"`
	SupplierLegalName      string         `json:"supplier_legal_name"`
	SupplierTradingName    string         `json:"supplier_trading_name,omitempty"`
	BuyerUserID            string         `json:"buyer_user_id"`
	BuyerBusinessID        string         `json:"buyer_business_id"`
	BuyerLegalName         string         `json:"buyer_legal_name"`
	BuyerTradingName       string         `json:"buyer_trading_name,omitempty"`
	PrincipalKobo          ledger.Money   `json:"principal_kobo"`
	Currency               string         `json:"currency"`
	GoodsDescription       string         `json:"goods_description"`
	InvoiceReference       string         `json:"invoice_reference,omitempty"`
	InvoiceDocumentHash    string         `json:"invoice_document_hash,omitempty"`
	DueDate                string         `json:"due_date"`
	GraceHours             int            `json:"grace_hours"`
	CollectionAt           time.Time      `json:"collection_at"`
	ScheduleType           string         `json:"schedule_type"`
	ScheduleCount          int            `json:"schedule_count,omitempty"`
	ScheduleCadence        string         `json:"schedule_cadence"`
	MonthEndPolicy         string         `json:"month_end_policy,omitempty"`
	CustomScheduleItems    []ScheduleTerm `json:"custom_schedule_items,omitempty"`
	State                  string         `json:"state"`
	AgreementVersionID     string         `json:"agreement_version_id,omitempty"`
	MandateID              string         `json:"mandate_id,omitempty"`
	AcceptanceID           string         `json:"acceptance_id,omitempty"`
	ReleaseID              string         `json:"release_id,omitempty"`
	ReceiptID              string         `json:"receipt_id,omitempty"`
	ObligationID           string         `json:"obligation_id,omitempty"`
	CreatedBy              string         `json:"created_by"`
	CreatedAt              time.Time      `json:"created_at"`
	UpdatedAt              time.Time      `json:"updated_at"`
	Version                int64          `json:"version"`
	RequiresEnhancedReview bool           `json:"requires_enhanced_review,omitempty"`
}

type ScheduleTerm struct {
	AmountKobo ledger.Money `json:"amount_kobo"`
	DueDate    string       `json:"due_date"`
}

type CreateInput struct {
	SupplierOrganizationID string
	SupplierLegalName      string
	SupplierTradingName    string
	BuyerUserID            string
	BuyerBusinessID        string
	BuyerLegalName         string
	BuyerTradingName       string
	PrincipalKobo          ledger.Money
	Currency               string
	GoodsDescription       string
	InvoiceReference       string
	InvoiceDocumentHash    string
	DueDate                string
	GraceHours             int
	CollectionAt           time.Time
	ScheduleType           string
	ScheduleCount          int
	ScheduleCadence        string
	MonthEndPolicy         string
	CustomScheduleItems    []ScheduleTerm
	CreatedBy              string
}

type UpdateDraftInput struct {
	ExpectedVersion     int64
	PrincipalKobo       ledger.Money
	GoodsDescription    string
	InvoiceReference    string
	InvoiceDocumentHash string
	DueDate             string
	GraceHours          int
	CollectionAt        time.Time
	ScheduleType        string
	ScheduleCount       int
	ScheduleCadence     string
	MonthEndPolicy      string
	CustomScheduleItems []ScheduleTerm
}

type AgreementVersion struct {
	ID              string          `json:"id"`
	CreditRequestID string          `json:"credit_request_id"`
	Version         int             `json:"version"`
	CanonicalJSON   json.RawMessage `json:"canonical_json"`
	DocumentHash    string          `json:"document_hash"`
	PrincipalKobo   ledger.Money    `json:"principal_kobo"`
	DueDate         string          `json:"due_date"`
	GraceHours      int             `json:"grace_hours"`
	CollectionAt    time.Time       `json:"collection_at"`
	TermsVersion    string          `json:"terms_version"`
	PrivacyVersion  string          `json:"privacy_version"`
	CreatedBy       string          `json:"created_by"`
	CreatedAt       time.Time       `json:"created_at"`
}

type Acceptance struct {
	ID                  string    `json:"id"`
	AgreementVersionID  string    `json:"agreement_version_id"`
	CreditRequestID     string    `json:"credit_request_id"`
	AcceptingUserID     string    `json:"accepting_user_id"`
	PersonID            string    `json:"person_id"`
	BusinessID          string    `json:"business_id"`
	AcceptanceMethod    string    `json:"acceptance_method"`
	AuthenticationLevel string    `json:"authentication_level"`
	AgreementHash       string    `json:"agreement_hash"`
	MandateProviderID   string    `json:"mandate_provider_id"`
	AcceptedAt          time.Time `json:"accepted_at"`
}

type GoodsRelease struct {
	ID              string    `json:"id"`
	CreditRequestID string    `json:"credit_request_id"`
	SupplierActorID string    `json:"supplier_actor_id"`
	DeliveryMethod  string    `json:"delivery_method"`
	Notes           string    `json:"notes,omitempty"`
	ReleasedAt      time.Time `json:"released_at"`
}

type ReceiptConfirmation struct {
	ID              string    `json:"id"`
	CreditRequestID string    `json:"credit_request_id"`
	BuyerUserID     string    `json:"buyer_user_id"`
	State           string    `json:"state"`
	IssueReason     string    `json:"issue_reason,omitempty"`
	ReceivedAt      time.Time `json:"received_at"`
}

type Obligation struct {
	ID                     string       `json:"id"`
	CreditRequestID        string       `json:"credit_request_id"`
	AgreementVersionID     string       `json:"agreement_version_id"`
	SupplierOrganizationID string       `json:"supplier_organization_id"`
	BuyerBusinessID        string       `json:"buyer_business_id"`
	PrincipalKobo          ledger.Money `json:"principal_kobo"`
	Currency               string       `json:"currency"`
	LifecycleStatus        string       `json:"lifecycle_status"`
	PaymentStatus          string       `json:"payment_status"`
	OutstandingKobo        ledger.Money `json:"outstanding_kobo"`
	BaseFeeKobo            ledger.Money `json:"base_fee_kobo"`
	LedgerTransactionID    string       `json:"ledger_transaction_id"`
	ActivatedAt            time.Time    `json:"activated_at"`
}

type View struct {
	Request    CreditRequest         `json:"request"`
	Agreement  AgreementVersion      `json:"agreement"`
	Acceptance *Acceptance           `json:"acceptance,omitempty"`
	Mandate    *mandates.Mandate     `json:"mandate,omitempty"`
	Release    *GoodsRelease         `json:"release,omitempty"`
	Receipts   []ReceiptConfirmation `json:"receipts"`
	Obligation *Obligation           `json:"obligation,omitempty"`
}

type CollectionState struct {
	ID                     string
	SupplierOrganizationID string
	BuyerUserID            string
	BuyerBusinessID        string
	Currency               string
	Active                 bool
	OutstandingKobo        ledger.Money
	MandateActive          bool
	MandateRemainingKobo   ledger.Money
	CollectionEnabled      bool
	ComplianceHold         bool
	BuyerPaymentHold       bool
	ProviderSupported      bool
	DisputedBlockedKobo    ledger.Money
	Version                int64
}

type TradeLineActivationInput struct {
	DrawdownID             string
	TradeLineID            string
	SupplierOrganizationID string
	BuyerUserID            string
	BuyerBusinessID        string
	MandateID              string
	PrincipalKobo          ledger.Money
	GoodsDescription       string
	InvoiceReference       string
	InvoiceDocumentHash    string
	DueDate                string
	GraceHours             int
	CollectionAt           time.Time
	TermsVersion           string
	DrawdownAgreementHash  string
	BuyerConfirmedAt       time.Time
	ReleaseActorID         string
	DeliveryMethod         string
	ReleaseNotes           string
	ReleasedAt             time.Time
	ReceiptActorID         string
	ReceiptAt              time.Time
}

type agreementCanonical struct {
	SourceType          string         `json:"source_type,omitempty"`
	SourceID            string         `json:"source_id,omitempty"`
	TradeLineID         string         `json:"trade_line_id,omitempty"`
	AcceptedSourceHash  string         `json:"accepted_source_hash,omitempty"`
	SupplierLegalName   string         `json:"supplier_legal_name"`
	SupplierTradingName string         `json:"supplier_trading_name,omitempty"`
	BuyerLegalName      string         `json:"buyer_legal_name"`
	BuyerTradingName    string         `json:"buyer_trading_name,omitempty"`
	BuyerBusinessID     string         `json:"buyer_business_id"`
	GoodsDescription    string         `json:"goods_description"`
	InvoiceReference    string         `json:"invoice_reference,omitempty"`
	InvoiceDocumentHash string         `json:"invoice_document_hash,omitempty"`
	PrincipalKobo       ledger.Money   `json:"principal_kobo"`
	Currency            string         `json:"currency"`
	DueDate             string         `json:"due_date"`
	GraceHours          int            `json:"grace_hours"`
	CollectionAt        time.Time      `json:"collection_at"`
	ScheduleType        string         `json:"schedule_type"`
	ScheduleCount       int            `json:"schedule_count,omitempty"`
	ScheduleCadence     string         `json:"schedule_cadence"`
	MonthEndPolicy      string         `json:"month_end_policy,omitempty"`
	CustomScheduleItems []ScheduleTerm `json:"custom_schedule_items,omitempty"`
	MandateDisclosure   string         `json:"mandate_disclosure"`
	BaseFeeDisclosure   string         `json:"base_fee_disclosure"`
	TermsVersion        string         `json:"terms_version"`
	PrivacyVersion      string         `json:"privacy_version"`
}

type Store struct {
	mu                      sync.RWMutex
	mandates                mandates.Provider
	ledger                  ledger.Service
	requests                map[string]*CreditRequest
	agreements              map[string]*AgreementVersion
	acceptances             map[string]*Acceptance
	mandateMap              map[string]*mandates.Mandate
	releases                map[string]*GoodsRelease
	receipts                map[string][]*ReceiptConfirmation
	obligations             map[string]*Obligation
	now                     func() time.Time
	newID                   func() string
	onActivated             func(CreditRequest, Obligation)
	creationGuard           func(CreateInput) error
	enhancedReviewThreshold ledger.Money
	maxActiveExposureKobo   ledger.Money
}

// Service is the credit aggregate boundary. It keeps HTTP and dependent
// financial services independent from the development store while allowing a
// PostgreSQL-backed implementation to preserve the same lifecycle rules.
type Service interface {
	SetCreationGuard(func(CreateInput) error)
	SetEnhancedReviewThreshold(ledger.Money)
	SetMaxActiveExposure(ledger.Money)
	SetActivationHook(func(CreditRequest, Obligation))
	ActivateTradeLineDrawdown(TradeLineActivationInput) (View, *ledger.Transaction, error)
	Create(CreateInput) (CreditRequest, error)
	UpdateDraft(string, string, UpdateDraftInput) (CreditRequest, error)
	Send(string, string) (View, error)
	Cancel(string, string) (View, error)
	Review(string, string) (View, error)
	Decline(string, string) (View, error)
	AuthorizeMandate(context.Context, string, string) (View, error)
	SetMandate(string, string, mandates.Mandate) (View, error)
	Accept(string, string, string, string, string, string, bool, bool) (View, error)
	Release(string, string, string, string, string) (View, error)
	RecordReceipt(string, string, string, string) (View, *ledger.Transaction, error)
	GetForSupplier(string, string) (View, error)
	GetForBuyer(string, string) (View, error)
	GetPublic(string) (View, error)
	GetByObligationForBuyer(string, string) (View, error)
	ListForSupplier(string) []View
	ListForBuyer(string) []View
	PaymentSnapshot(string) (payments.ObligationSnapshot, error)
	ApplyPayment(string, ledger.Money) error
	ApplyAdjustment(string, ledger.Money) error
	CollectionState(string) (CollectionState, error)
	CollectionStateForOrganization(string, string) (CollectionState, error)
	ObligationBelongsToOrganization(string, string) bool
}

var _ Service = (*Store)(nil)

func (s *Store) SetCreationGuard(guard func(CreateInput) error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.creationGuard = guard
}

func (s *Store) SetEnhancedReviewThreshold(threshold ledger.Money) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.enhancedReviewThreshold = threshold
}

func (s *Store) SetMaxActiveExposure(threshold ledger.Money) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.maxActiveExposureKobo = threshold
}

func (s *Store) SetActivationHook(hook func(CreditRequest, Obligation)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.onActivated = hook
}

func NewStore(mandateProvider mandates.Provider, ledgerStore ledger.Service) *Store {
	return &Store{mandates: mandateProvider, ledger: ledgerStore, requests: make(map[string]*CreditRequest), agreements: make(map[string]*AgreementVersion), acceptances: make(map[string]*Acceptance), mandateMap: make(map[string]*mandates.Mandate), releases: make(map[string]*GoodsRelease), receipts: make(map[string][]*ReceiptConfirmation), obligations: make(map[string]*Obligation), now: func() time.Time { return time.Now().UTC() }, newID: newIdentifier}
}

func (s *Store) ActivateTradeLineDrawdown(input TradeLineActivationInput) (View, *ledger.Transaction, error) {
	if strings.TrimSpace(input.DrawdownID) == "" || strings.TrimSpace(input.TradeLineID) == "" || strings.TrimSpace(input.SupplierOrganizationID) == "" || strings.TrimSpace(input.BuyerUserID) == "" || strings.TrimSpace(input.BuyerBusinessID) == "" || input.PrincipalKobo <= 0 || strings.TrimSpace(input.GoodsDescription) == "" || strings.TrimSpace(input.DrawdownAgreementHash) == "" {
		return View{}, nil, errors.New("complete accepted drawdown terms are required")
	}
	if _, err := time.Parse("2006-01-02", input.DueDate); err != nil || input.CollectionAt.IsZero() || input.GraceHours < 0 || input.GraceHours > 720 {
		return View{}, nil, errors.New("valid drawdown due, collection, and grace terms are required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if existing := s.requests[input.DrawdownID]; existing != nil {
		view := s.viewLocked(existing)
		if view.Obligation == nil {
			return View{}, nil, errors.New("drawdown activation is incomplete")
		}
		return view, nil, nil
	}
	if s.ledger == nil {
		return View{}, nil, errors.New("ledger unavailable")
	}
	now := s.now()
	request := &CreditRequest{ID: input.DrawdownID, SupplierOrganizationID: input.SupplierOrganizationID, SupplierLegalName: input.SupplierOrganizationID, BuyerUserID: input.BuyerUserID, BuyerBusinessID: input.BuyerBusinessID, BuyerLegalName: input.BuyerBusinessID, PrincipalKobo: input.PrincipalKobo, Currency: "NGN", GoodsDescription: strings.TrimSpace(input.GoodsDescription), InvoiceReference: strings.TrimSpace(input.InvoiceReference), InvoiceDocumentHash: strings.TrimSpace(input.InvoiceDocumentHash), DueDate: input.DueDate, GraceHours: input.GraceHours, CollectionAt: input.CollectionAt, ScheduleType: "one_time", ScheduleCount: 1, ScheduleCadence: "custom", State: Active, MandateID: input.MandateID, CreatedBy: input.ReleaseActorID, CreatedAt: now, UpdatedAt: now, Version: 1}
	canonical := agreementCanonical{SourceType: "trade_line_drawdown", SourceID: input.DrawdownID, TradeLineID: input.TradeLineID, AcceptedSourceHash: input.DrawdownAgreementHash, SupplierLegalName: request.SupplierLegalName, BuyerLegalName: request.BuyerLegalName, BuyerBusinessID: request.BuyerBusinessID, GoodsDescription: request.GoodsDescription, InvoiceReference: request.InvoiceReference, InvoiceDocumentHash: request.InvoiceDocumentHash, PrincipalKobo: request.PrincipalKobo, Currency: request.Currency, DueDate: request.DueDate, GraceHours: request.GraceHours, CollectionAt: request.CollectionAt, ScheduleType: request.ScheduleType, ScheduleCount: 1, ScheduleCadence: request.ScheduleCadence, MandateDisclosure: "Buyer confirmed this drawdown under the active trade-line mandate.", BaseFeeDisclosure: "0.5% supplier base service fee on activated principal.", TermsVersion: scheduleDefault(input.TermsVersion, "terms-v1"), PrivacyVersion: "privacy-v1"}
	data, _ := json.Marshal(canonical)
	hash := sha256.Sum256(data)
	agreement := &AgreementVersion{ID: s.newID(), CreditRequestID: request.ID, Version: 1, CanonicalJSON: data, DocumentHash: hex.EncodeToString(hash[:]), PrincipalKobo: request.PrincipalKobo, DueDate: request.DueDate, GraceHours: request.GraceHours, CollectionAt: request.CollectionAt, TermsVersion: canonical.TermsVersion, PrivacyVersion: canonical.PrivacyVersion, CreatedBy: input.ReleaseActorID, CreatedAt: now}
	acceptance := &Acceptance{ID: s.newID(), AgreementVersionID: agreement.ID, CreditRequestID: request.ID, AcceptingUserID: input.BuyerUserID, PersonID: input.BuyerUserID, BusinessID: input.BuyerBusinessID, AcceptanceMethod: "trade_line_drawdown_confirmation", AuthenticationLevel: "aal1", AgreementHash: input.DrawdownAgreementHash, MandateProviderID: input.MandateID, AcceptedAt: input.BuyerConfirmedAt}
	release := &GoodsRelease{ID: s.newID(), CreditRequestID: request.ID, SupplierActorID: input.ReleaseActorID, DeliveryMethod: input.DeliveryMethod, Notes: input.ReleaseNotes, ReleasedAt: input.ReleasedAt}
	receipt := &ReceiptConfirmation{ID: s.newID(), CreditRequestID: request.ID, BuyerUserID: input.ReceiptActorID, State: "confirmed", ReceivedAt: input.ReceiptAt}
	fee, _ := ledger.BaseFee(request.PrincipalKobo)
	obligation := &Obligation{ID: s.newID(), CreditRequestID: request.ID, AgreementVersionID: agreement.ID, SupplierOrganizationID: request.SupplierOrganizationID, BuyerBusinessID: request.BuyerBusinessID, PrincipalKobo: request.PrincipalKobo, Currency: request.Currency, LifecycleStatus: "ACTIVE", PaymentStatus: "UNPAID", OutstandingKobo: request.PrincipalKobo, BaseFeeKobo: fee, ActivatedAt: now}
	tx, err := s.ledger.PostActivation(obligation.ID, obligation.PrincipalKobo, now, "trade-line-drawdown:"+input.DrawdownID+":activation")
	if err != nil {
		return View{}, nil, err
	}
	obligation.LedgerTransactionID = tx.ID
	request.AgreementVersionID, request.AcceptanceID, request.ReleaseID, request.ReceiptID, request.ObligationID = agreement.ID, acceptance.ID, release.ID, receipt.ID, obligation.ID
	s.requests[request.ID] = request
	s.agreements[agreement.ID] = agreement
	s.acceptances[acceptance.ID] = acceptance
	s.releases[release.ID] = release
	s.receipts[request.ID] = []*ReceiptConfirmation{receipt}
	s.obligations[request.ID] = obligation
	if s.onActivated != nil {
		s.onActivated(*request, *obligation)
	}
	view := s.viewLocked(request)
	return view, &tx, nil
}

func (s *Store) Create(input CreateInput) (CreditRequest, error) {
	if err := validateCreateInput(input); err != nil {
		return CreditRequest{}, err
	}
	s.mu.RLock()
	guard := s.creationGuard
	s.mu.RUnlock()
	if guard != nil {
		if err := guard(input); err != nil {
			return CreditRequest{}, err
		}
	}
	now := s.now()
	s.mu.RLock()
	enhancedReview := s.enhancedReviewThreshold > 0 && input.PrincipalKobo >= s.enhancedReviewThreshold
	s.mu.RUnlock()
	request := &CreditRequest{ID: s.newID(), SupplierOrganizationID: input.SupplierOrganizationID, SupplierLegalName: strings.TrimSpace(input.SupplierLegalName), SupplierTradingName: strings.TrimSpace(input.SupplierTradingName), BuyerUserID: input.BuyerUserID, BuyerBusinessID: input.BuyerBusinessID, BuyerLegalName: strings.TrimSpace(input.BuyerLegalName), BuyerTradingName: strings.TrimSpace(input.BuyerTradingName), PrincipalKobo: input.PrincipalKobo, Currency: "NGN", GoodsDescription: strings.TrimSpace(input.GoodsDescription), InvoiceReference: strings.TrimSpace(input.InvoiceReference), InvoiceDocumentHash: strings.TrimSpace(input.InvoiceDocumentHash), DueDate: input.DueDate, GraceHours: input.GraceHours, CollectionAt: input.CollectionAt, ScheduleType: scheduleDefault(input.ScheduleType, "one_time"), ScheduleCount: input.ScheduleCount, ScheduleCadence: scheduleDefault(input.ScheduleCadence, "custom"), MonthEndPolicy: input.MonthEndPolicy, CustomScheduleItems: append([]ScheduleTerm(nil), input.CustomScheduleItems...), State: Draft, CreatedBy: input.CreatedBy, CreatedAt: now, UpdatedAt: now, Version: 1, RequiresEnhancedReview: enhancedReview}
	s.mu.Lock()
	s.requests[request.ID] = request
	s.mu.Unlock()
	return cloneRequest(*request), nil
}

func (s *Store) UpdateDraft(requestID, actorID string, input UpdateDraftInput) (CreditRequest, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	r := s.requests[requestID]
	if r == nil {
		return CreditRequest{}, errors.New("credit request not found")
	}
	if r.CreatedBy != actorID {
		return CreditRequest{}, errors.New("only the request creator may amend this draft")
	}
	if r.State != Draft {
		return CreditRequest{}, fmt.Errorf("credit request cannot be amended from %s", r.State)
	}
	if input.ExpectedVersion != r.Version {
		return CreditRequest{}, errors.New("credit request version conflict")
	}
	if strings.TrimSpace(input.ScheduleType) == "" {
		input.ScheduleType = r.ScheduleType
		input.ScheduleCount = r.ScheduleCount
		input.ScheduleCadence = r.ScheduleCadence
		input.MonthEndPolicy = r.MonthEndPolicy
		input.CustomScheduleItems = append([]ScheduleTerm(nil), r.CustomScheduleItems...)
	}
	validation := CreateInput{SupplierOrganizationID: r.SupplierOrganizationID, SupplierLegalName: r.SupplierLegalName, BuyerUserID: r.BuyerUserID, BuyerBusinessID: r.BuyerBusinessID, BuyerLegalName: r.BuyerLegalName, CreatedBy: r.CreatedBy, PrincipalKobo: input.PrincipalKobo, Currency: r.Currency, GoodsDescription: input.GoodsDescription, DueDate: input.DueDate, GraceHours: input.GraceHours, CollectionAt: input.CollectionAt, ScheduleType: input.ScheduleType, ScheduleCount: input.ScheduleCount, ScheduleCadence: input.ScheduleCadence, MonthEndPolicy: input.MonthEndPolicy, CustomScheduleItems: input.CustomScheduleItems}
	if err := validateCreateInput(validation); err != nil {
		return CreditRequest{}, err
	}
	r.PrincipalKobo = input.PrincipalKobo
	r.GoodsDescription = strings.TrimSpace(input.GoodsDescription)
	r.InvoiceReference = strings.TrimSpace(input.InvoiceReference)
	r.InvoiceDocumentHash = strings.TrimSpace(input.InvoiceDocumentHash)
	r.DueDate = input.DueDate
	r.GraceHours = input.GraceHours
	r.CollectionAt = input.CollectionAt
	r.ScheduleType = scheduleDefault(input.ScheduleType, "one_time")
	r.ScheduleCount = input.ScheduleCount
	r.ScheduleCadence = scheduleDefault(input.ScheduleCadence, "custom")
	r.MonthEndPolicy = input.MonthEndPolicy
	r.CustomScheduleItems = append([]ScheduleTerm(nil), input.CustomScheduleItems...)
	r.UpdatedAt = s.now()
	r.Version++
	return cloneRequest(*r), nil
}

func (s *Store) Send(requestID, actorID string) (View, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	r := s.requests[requestID]
	if r == nil {
		return View{}, errors.New("credit request not found")
	}
	if actorID == "" || r.CreatedBy != actorID {
		return View{}, errors.New("only the request creator may send this credit request")
	}
	if r.State != Draft {
		return View{}, fmt.Errorf("credit request cannot be sent from %s", r.State)
	}
	canonical := agreementCanonical{SupplierLegalName: r.SupplierLegalName, SupplierTradingName: r.SupplierTradingName, BuyerLegalName: r.BuyerLegalName, BuyerTradingName: r.BuyerTradingName, BuyerBusinessID: r.BuyerBusinessID, GoodsDescription: r.GoodsDescription, InvoiceReference: r.InvoiceReference, InvoiceDocumentHash: r.InvoiceDocumentHash, PrincipalKobo: r.PrincipalKobo, Currency: r.Currency, DueDate: r.DueDate, GraceHours: r.GraceHours, CollectionAt: r.CollectionAt, ScheduleType: r.ScheduleType, ScheduleCount: r.ScheduleCount, ScheduleCadence: r.ScheduleCadence, MonthEndPolicy: r.MonthEndPolicy, CustomScheduleItems: append([]ScheduleTerm(nil), r.CustomScheduleItems...), MandateDisclosure: "Buyer authorizes a mandate up to the accepted principal.", BaseFeeDisclosure: "0.5% base service fee on activated principal.", TermsVersion: "terms-v1", PrivacyVersion: "privacy-v1"}
	data, _ := json.Marshal(canonical)
	hash := sha256.Sum256(data)
	agreement := &AgreementVersion{ID: newIdentifier(), CreditRequestID: r.ID, Version: 1, CanonicalJSON: data, DocumentHash: hex.EncodeToString(hash[:]), PrincipalKobo: r.PrincipalKobo, DueDate: r.DueDate, GraceHours: r.GraceHours, CollectionAt: r.CollectionAt, TermsVersion: "terms-v1", PrivacyVersion: "privacy-v1", CreatedBy: actorID, CreatedAt: s.now()}
	s.agreements[agreement.ID] = agreement
	r.AgreementVersionID = agreement.ID
	r.State = Sent
	r.UpdatedAt = s.now()
	r.Version++
	return s.viewLocked(r), nil
}

func (s *Store) Cancel(requestID, actorID string) (View, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	r := s.requests[requestID]
	if r == nil {
		return View{}, errors.New("credit request not found")
	}
	if r.CreatedBy != actorID {
		return View{}, errors.New("only the request creator may cancel this credit request")
	}
	if r.State != Draft && r.State != Sent && r.State != BuyerReviewing {
		return View{}, fmt.Errorf("credit request cannot be cancelled from %s", r.State)
	}
	r.State = Cancelled
	r.UpdatedAt = s.now()
	r.Version++
	return s.viewLocked(r), nil
}

func (s *Store) Review(requestID, buyerUserID string) (View, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	r := s.requests[requestID]
	if r == nil {
		return View{}, errors.New("credit request not found")
	}
	if r.BuyerUserID != buyerUserID {
		return View{}, errors.New("buyer mismatch")
	}
	if r.State == Sent {
		r.State = BuyerReviewing
		r.UpdatedAt = s.now()
		r.Version++
	}
	if r.State != BuyerReviewing && r.State != VerificationPending && r.State != ReadyToRelease && r.State != BuyerAccepted && r.State != ReceiptConfirmationPending && r.State != Active {
		return View{}, fmt.Errorf("credit request cannot be reviewed from %s", r.State)
	}
	return s.viewLocked(r), nil
}

func (s *Store) Decline(requestID, buyerUserID string) (View, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	r := s.requests[requestID]
	if r == nil {
		return View{}, errors.New("credit request not found")
	}
	if r.BuyerUserID != buyerUserID {
		return View{}, errors.New("buyer mismatch")
	}
	if r.State != Sent && r.State != BuyerReviewing {
		return View{}, fmt.Errorf("credit request cannot be declined from %s", r.State)
	}
	r.State = Declined
	r.UpdatedAt = s.now()
	r.Version++
	return s.viewLocked(r), nil
}

func (s *Store) AuthorizeMandate(ctx context.Context, requestID, buyerUserID string) (View, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	r := s.requests[requestID]
	if r == nil {
		return View{}, errors.New("credit request not found")
	}
	if r.BuyerUserID != buyerUserID {
		return View{}, errors.New("buyer mismatch")
	}
	if r.State != BuyerReviewing {
		return View{}, fmt.Errorf("mandate cannot be authorized from %s", r.State)
	}
	if r.AgreementVersionID == "" {
		return View{}, errors.New("agreement not ready")
	}
	if s.mandates == nil {
		return View{}, errors.New("mandate provider unavailable")
	}
	m, err := s.mandates.CreateAuthorizationSession(ctx, mandates.AuthorizationInput{UserID: buyerUserID, BusinessID: r.BuyerBusinessID, AmountCeiling: int64(r.PrincipalKobo), Purpose: r.ID})
	if err != nil {
		return View{}, err
	}
	cp := m
	s.mandateMap[m.ID] = &cp
	s.mandateMap[m.ProviderID] = &cp
	r.MandateID = m.ID
	r.UpdatedAt = s.now()
	r.Version++
	return s.viewLocked(r), nil
}

func (s *Store) SetMandate(requestID, buyerUserID string, mandate mandates.Mandate) (View, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	r := s.requests[requestID]
	if r == nil || r.BuyerUserID != buyerUserID {
		return View{}, errors.New("credit request not found")
	}
	if mandate.UserID != buyerUserID || mandate.BusinessID != r.BuyerBusinessID || mandate.ID == "" || mandate.ProviderID == "" {
		return View{}, errors.New("mandate ownership does not match credit request")
	}
	cp := mandate
	s.mandateMap[mandate.ID], s.mandateMap[mandate.ProviderID] = &cp, &cp
	r.MandateID, r.UpdatedAt, r.Version = mandate.ID, s.now(), r.Version+1
	return s.viewLocked(r), nil
}

func (s *Store) Accept(requestID, buyerUserID, agreementID, agreementHash, mandateID, authLevel string, identityVerified, authorityVerified bool) (View, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	r := s.requests[requestID]
	if r == nil {
		return View{}, errors.New("credit request not found")
	}
	if r.BuyerUserID != buyerUserID {
		return View{}, errors.New("buyer mismatch")
	}
	if r.State != BuyerReviewing {
		return View{}, fmt.Errorf("credit request cannot be accepted from %s", r.State)
	}
	a := s.agreements[r.AgreementVersionID]
	if a == nil || a.ID != agreementID || a.DocumentHash != agreementHash {
		return View{}, errors.New("agreement hash mismatch")
	}
	m := s.mandateMap[mandateID]
	if m == nil || m.Status != mandates.Active || m.AmountCeiling < int64(r.PrincipalKobo) || m.UserID != buyerUserID || m.BusinessID != r.BuyerBusinessID {
		return View{}, errors.New("active mandate required")
	}
	if !identityVerified || !authorityVerified {
		return View{}, errors.New("verified identity and authority required")
	}
	accept := &Acceptance{ID: newIdentifier(), AgreementVersionID: a.ID, CreditRequestID: r.ID, AcceptingUserID: buyerUserID, PersonID: buyerUserID, BusinessID: r.BuyerBusinessID, AcceptanceMethod: "explicit_web_action", AuthenticationLevel: authLevel, AgreementHash: agreementHash, MandateProviderID: m.ProviderID, AcceptedAt: s.now()}
	s.acceptances[accept.ID] = accept
	r.AcceptanceID = accept.ID
	r.State = BuyerAccepted
	r.State = ReadyToRelease
	r.UpdatedAt = s.now()
	r.Version++
	return s.viewLocked(r), nil
}

func (s *Store) Release(requestID, supplierOrgID, actorID, deliveryMethod, notes string) (View, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	r := s.requests[requestID]
	if r == nil {
		return View{}, errors.New("credit request not found")
	}
	if r.SupplierOrganizationID != supplierOrgID {
		return View{}, errors.New("supplier mismatch")
	}
	if r.State != ReadyToRelease {
		return View{}, fmt.Errorf("credit request cannot be released from %s", r.State)
	}
	rel := &GoodsRelease{ID: newIdentifier(), CreditRequestID: r.ID, SupplierActorID: actorID, DeliveryMethod: strings.TrimSpace(deliveryMethod), Notes: strings.TrimSpace(notes), ReleasedAt: s.now()}
	if rel.DeliveryMethod == "" {
		return View{}, errors.New("delivery method required")
	}
	s.releases[rel.ID] = rel
	r.ReleaseID = rel.ID
	r.State = ReceiptConfirmationPending
	r.UpdatedAt = s.now()
	r.Version++
	return s.viewLocked(r), nil
}

func (s *Store) RecordReceipt(requestID, buyerUserID, state, issueReason string) (View, *ledger.Transaction, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	r := s.requests[requestID]
	if r == nil {
		return View{}, nil, errors.New("credit request not found")
	}
	if r.BuyerUserID != buyerUserID {
		return View{}, nil, errors.New("buyer mismatch")
	}
	if r.State != ReceiptConfirmationPending {
		return View{}, nil, fmt.Errorf("receipt cannot be recorded from %s", r.State)
	}
	if state != "confirmed" && state != "issue_raised" {
		return View{}, nil, errors.New("invalid receipt state")
	}
	receipt := &ReceiptConfirmation{ID: newIdentifier(), CreditRequestID: r.ID, BuyerUserID: buyerUserID, State: state, IssueReason: strings.TrimSpace(issueReason), ReceivedAt: s.now()}
	if state == "issue_raised" {
		s.receipts[r.ID] = append(s.receipts[r.ID], receipt)
		r.ReceiptID = receipt.ID
		r.UpdatedAt = s.now()
		r.Version++
		return s.viewLocked(r), nil, nil
	}
	if s.ledger == nil {
		return View{}, nil, errors.New("ledger unavailable")
	}
	fee, _ := ledger.BaseFee(r.PrincipalKobo)
	prospective := &Obligation{ID: newIdentifier(), CreditRequestID: r.ID, AgreementVersionID: r.AgreementVersionID, SupplierOrganizationID: r.SupplierOrganizationID, BuyerBusinessID: r.BuyerBusinessID, PrincipalKobo: r.PrincipalKobo, Currency: r.Currency, LifecycleStatus: "ACTIVE", PaymentStatus: "UNPAID", OutstandingKobo: r.PrincipalKobo, BaseFeeKobo: fee, ActivatedAt: s.now()}
	if s.maxActiveExposureKobo > 0 {
		var exposure ledger.Money
		for _, existing := range s.obligations {
			if existing.BuyerBusinessID == r.BuyerBusinessID && existing.LifecycleStatus == "ACTIVE" {
				var addErr error
				exposure, addErr = ledger.CheckedAdd(exposure, existing.OutstandingKobo)
				if addErr != nil {
					return View{}, nil, errors.New("buyer active exposure is too large")
				}
			}
		}
		totalExposure, addErr := ledger.CheckedAdd(exposure, prospective.OutstandingKobo)
		if addErr != nil || totalExposure > s.maxActiveExposureKobo {
			return View{}, nil, errors.New("buyer active exposure exceeds configured pilot limit")
		}
	}
	tx, err := s.ledger.PostActivation(r.ID, r.PrincipalKobo, s.now(), r.ID+":activation")
	if err != nil {
		return View{}, nil, err
	}
	s.receipts[r.ID] = append(s.receipts[r.ID], receipt)
	r.ReceiptID = receipt.ID
	prospective.LedgerTransactionID = tx.ID
	obligation := prospective
	s.obligations[r.ID] = obligation
	r.ObligationID = obligation.ID
	r.State = Active
	r.UpdatedAt = s.now()
	r.Version++
	if s.onActivated != nil {
		s.onActivated(*r, *obligation)
	}
	return s.viewLocked(r), &tx, nil
}

func (s *Store) GetForSupplier(requestID, orgID string) (View, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	r := s.requests[requestID]
	if r == nil || r.SupplierOrganizationID != orgID {
		return View{}, errors.New("credit request not found")
	}
	return s.viewLocked(r), nil
}
func (s *Store) GetForBuyer(requestID, buyerUserID string) (View, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	r := s.requests[requestID]
	if r == nil || r.BuyerUserID != buyerUserID {
		return View{}, errors.New("credit request not found")
	}
	if r.State == Sent {
		r.State = BuyerReviewing
		r.UpdatedAt = s.now()
		r.Version++
	}
	return s.viewLocked(r), nil
}
func (s *Store) GetPublic(requestID string) (View, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	r := s.requests[requestID]
	if r == nil {
		return View{}, errors.New("credit request not found")
	}
	return s.viewLocked(r), nil
}
func (s *Store) GetByObligationForBuyer(obligationID, buyerUserID string) (View, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, r := range s.requests {
		if r.ObligationID == obligationID && r.BuyerUserID == buyerUserID {
			return s.viewLocked(r), nil
		}
	}
	return View{}, errors.New("obligation not found")
}
func (s *Store) ListForSupplier(orgID string) []View {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := []View{}
	for _, r := range s.requests {
		if r.SupplierOrganizationID == orgID {
			out = append(out, s.viewLocked(r))
		}
	}
	return out
}

// ListForBuyer returns the buyer's complete credit-request history. Callers
// should apply presentation rules (for example, only exposing a buyer's own
// relationship) at the API boundary.
func (s *Store) ListForBuyer(buyerUserID string) []View {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := []View{}
	for _, r := range s.requests {
		if r.BuyerUserID == buyerUserID {
			out = append(out, s.viewLocked(r))
		}
	}
	return out
}

func (s *Store) viewLocked(r *CreditRequest) View {
	v := View{Request: cloneRequest(*r), Receipts: []ReceiptConfirmation{}}
	if a := s.agreements[r.AgreementVersionID]; a != nil {
		v.Agreement = cloneAgreement(*a)
	}
	if a := s.acceptances[r.AcceptanceID]; a != nil {
		cp := *a
		v.Acceptance = &cp
	}
	if m := s.mandateMap[r.MandateID]; m != nil {
		cp := *m
		v.Mandate = &cp
	}
	if rel := s.releases[r.ReleaseID]; rel != nil {
		cp := *rel
		v.Release = &cp
	}
	for _, receipt := range s.receipts[r.ID] {
		v.Receipts = append(v.Receipts, *receipt)
	}
	if o := s.obligations[r.ID]; o != nil {
		cp := *o
		v.Obligation = &cp
	}
	return v
}

func validateCreateInput(in CreateInput) error {
	for name, value := range map[string]string{"supplier organization": in.SupplierOrganizationID, "buyer user": in.BuyerUserID, "buyer business": in.BuyerBusinessID, "created by": in.CreatedBy, "supplier name": in.SupplierLegalName, "buyer name": in.BuyerLegalName, "goods description": in.GoodsDescription, "due date": in.DueDate} {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("%s is required", name)
		}
	}
	if in.PrincipalKobo <= 0 {
		return errors.New("principal must be positive")
	}
	if in.Currency != "" && strings.ToUpper(in.Currency) != "NGN" {
		return errors.New("currency must be NGN")
	}
	if _, err := time.Parse("2006-01-02", in.DueDate); err != nil {
		return errors.New("due date must be YYYY-MM-DD")
	}
	if in.GraceHours < 0 || in.GraceHours > 720 {
		return errors.New("grace hours must be between 0 and 720")
	}
	if in.CollectionAt.IsZero() {
		return errors.New("collection time is required")
	}
	if err := validateScheduleTerms(in.PrincipalKobo, in.ScheduleType, in.ScheduleCount, in.ScheduleCadence, in.MonthEndPolicy, in.CustomScheduleItems); err != nil {
		return err
	}
	return nil
}

func validateScheduleTerms(principal ledger.Money, scheduleType string, count int, cadence, monthEndPolicy string, custom []ScheduleTerm) error {
	scheduleType = scheduleDefault(scheduleType, "one_time")
	switch scheduleType {
	case "one_time":
		return nil
	case "equal":
		if count < 2 || count > 60 {
			return errors.New("equal instalment count must be between 2 and 60")
		}
		if cadence != "weekly" && cadence != "fortnightly" && cadence != "monthly" {
			return errors.New("equal instalment cadence must be weekly, fortnightly, or monthly")
		}
		if cadence == "monthly" && monthEndPolicy != "last_day" && monthEndPolicy != "cap" {
			return errors.New("monthly instalments require a documented month-end policy")
		}
		return nil
	case "custom":
		if len(custom) < 2 || len(custom) > 60 {
			return errors.New("custom schedule must contain 2 to 60 items")
		}
		var total ledger.Money
		var previous time.Time
		for _, item := range custom {
			due, err := time.Parse("2006-01-02", item.DueDate)
			if err != nil || item.AmountKobo <= 0 {
				return errors.New("custom schedule items require a positive amount and YYYY-MM-DD due date")
			}
			if !previous.IsZero() && !due.After(previous) {
				return errors.New("custom schedule due dates must be ordered")
			}
			previous = due
			var addErr error
			total, addErr = ledger.CheckedAdd(total, item.AmountKobo)
			if addErr != nil {
				return errors.New("custom schedule total is too large")
			}
		}
		if total != principal {
			return errors.New("custom schedule amounts must equal principal")
		}
		return nil
	default:
		return errors.New("schedule type must be one_time, equal, or custom")
	}
}

func cloneRequest(r CreditRequest) CreditRequest {
	r.CustomScheduleItems = append([]ScheduleTerm(nil), r.CustomScheduleItems...)
	return r
}
func cloneAgreement(a AgreementVersion) AgreementVersion {
	a.CanonicalJSON = append(json.RawMessage(nil), a.CanonicalJSON...)
	return a
}
func newIdentifier() string { return identifier.New() }

func scheduleDefault(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return strings.TrimSpace(value)
}

func PrintableAgreement(v View) string {
	r := v.Request
	return fmt.Sprintf("KREDIT CREDIT AGREEMENT\nRequest: %s\nSupplier: %s\nBuyer: %s\nPrincipal: %d %s\nDue date: %s\nGrace hours: %d\nCollection: %s\nGoods: %s\nBase fee: 0.5%% of activated principal\nAgreement hash: %s\nTerms version: %s\n", r.ID, r.SupplierLegalName, r.BuyerLegalName, r.PrincipalKobo, r.Currency, r.DueDate, r.GraceHours, r.CollectionAt.UTC().Format(time.RFC3339), r.GoodsDescription, v.Agreement.DocumentHash, v.Agreement.TermsVersion)
}

func (s *Store) PaymentSnapshot(obligationID string) (payments.ObligationSnapshot, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	o := s.obligations[obligationID]
	if o == nil {
		return payments.ObligationSnapshot{}, errors.New("obligation not found")
	}
	r := s.requests[o.CreditRequestID]
	if r == nil {
		return payments.ObligationSnapshot{}, errors.New("credit request not found")
	}
	return payments.ObligationSnapshot{ID: o.ID, BuyerUserID: r.BuyerUserID, SupplierOrganizationID: o.SupplierOrganizationID, PrincipalKobo: o.PrincipalKobo, OutstandingKobo: o.OutstandingKobo, CollectionAt: r.CollectionAt, Currency: o.Currency}, nil
}

func (s *Store) ApplyPayment(obligationID string, delta ledger.Money) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	o := s.obligations[obligationID]
	if o == nil {
		return errors.New("obligation not found")
	}
	next, err := ledger.CheckedAdd(o.OutstandingKobo, delta)
	if err != nil {
		return errors.New("payment would create invalid outstanding balance")
	}
	if next < 0 || next > o.PrincipalKobo {
		return errors.New("payment would create invalid outstanding balance")
	}
	o.OutstandingKobo = next
	if request := s.requests[o.CreditRequestID]; request != nil {
		request.Version++
		request.UpdatedAt = s.now()
	}
	if next == 0 {
		o.PaymentStatus = "PAID"
	} else if next < o.PrincipalKobo {
		o.PaymentStatus = "PARTIALLY_PAID"
	} else {
		o.PaymentStatus = "UNPAID"
	}
	return nil
}

func (s *Store) ApplyAdjustment(obligationID string, reduction ledger.Money) error {
	if reduction <= 0 {
		return errors.New("adjustment must be positive")
	}
	return s.ApplyPayment(obligationID, -reduction)
}

func (s *Store) CollectionState(obligationID string) (CollectionState, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	o := s.obligations[obligationID]
	if o == nil {
		return CollectionState{}, errors.New("obligation not found")
	}
	r := s.requests[o.CreditRequestID]
	if r == nil {
		return CollectionState{}, errors.New("credit request not found")
	}
	mandate := s.mandateMap[r.MandateID]
	remaining := ledger.Money(0)
	active := false
	if mandate != nil {
		active = mandate.Status == mandates.Active
		remaining = ledger.Money(mandate.AmountCeiling)
	}
	return CollectionState{ID: o.ID, SupplierOrganizationID: o.SupplierOrganizationID, BuyerUserID: r.BuyerUserID, BuyerBusinessID: r.BuyerBusinessID, Currency: o.Currency, Active: o.LifecycleStatus == "ACTIVE", OutstandingKobo: o.OutstandingKobo, MandateActive: active, MandateRemainingKobo: remaining, CollectionEnabled: true, ProviderSupported: true, Version: r.Version}, nil
}

func (s *Store) CollectionStateForOrganization(obligationID, organizationID string) (CollectionState, error) {
	state, err := s.CollectionState(obligationID)
	if err != nil {
		return CollectionState{}, err
	}
	if state.SupplierOrganizationID != organizationID {
		return CollectionState{}, errors.New("obligation does not belong to organization")
	}
	return state, nil
}

func (s *Store) ObligationBelongsToOrganization(obligationID, organizationID string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	o := s.obligations[obligationID]
	return o != nil && o.SupplierOrganizationID == organizationID
}
