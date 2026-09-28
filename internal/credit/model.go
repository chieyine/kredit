package credit

import (
	"encoding/json"
	"kredit/internal/ledger"
	"kredit/internal/legalpublication"
	"kredit/internal/mandates"
	"time"
)

type CreditRequest struct {
	SystemAcceptanceID     string           `json:"system_acceptance_id,omitempty"`
	FeeTerms               *ledger.FeeTerms `json:"fee_terms,omitempty"`
	CollectionPolicy       string           `json:"collection_policy,omitempty"`
	ID                     string           `json:"id"`
	SupplierOrganizationID string           `json:"supplier_organization_id"`
	SupplierLegalName      string           `json:"supplier_legal_name"`
	SupplierTradingName    string           `json:"supplier_trading_name,omitempty"`
	BuyerUserID            string           `json:"buyer_user_id"`
	BuyerBusinessID        string           `json:"buyer_business_id"`
	BuyerLegalName         string           `json:"buyer_legal_name"`
	BuyerTradingName       string           `json:"buyer_trading_name,omitempty"`
	PrincipalKobo          ledger.Money     `json:"principal_kobo"`
	Currency               string           `json:"currency"`
	GoodsDescription       string           `json:"goods_description"`
	InvoiceReference       string           `json:"invoice_reference,omitempty"`
	InvoiceDocumentID      string           `json:"invoice_document_id,omitempty"`
	InvoiceDocumentHash    string           `json:"invoice_document_hash,omitempty"`
	DueDate                string           `json:"due_date"`
	GraceHours             int              `json:"grace_hours"`
	CollectionAt           time.Time        `json:"collection_at"`
	ScheduleType           string           `json:"schedule_type"`
	ScheduleCount          int              `json:"schedule_count,omitempty"`
	ScheduleCadence        string           `json:"schedule_cadence"`
	MonthEndPolicy         string           `json:"month_end_policy,omitempty"`
	CustomScheduleItems    []ScheduleTerm   `json:"custom_schedule_items,omitempty"`
	State                  string           `json:"state"`
	AgreementVersionID     string           `json:"agreement_version_id,omitempty"`
	MandateID              string           `json:"mandate_id,omitempty"`
	AcceptanceID           string           `json:"acceptance_id,omitempty"`
	ReleaseID              string           `json:"release_id,omitempty"`
	ReceiptID              string           `json:"receipt_id,omitempty"`
	ObligationID           string           `json:"obligation_id,omitempty"`
	CreatedBy              string           `json:"created_by"`
	CreatedAt              time.Time        `json:"created_at"`
	UpdatedAt              time.Time        `json:"updated_at"`
	Version                int64            `json:"version"`
	RequiresEnhancedReview bool             `json:"requires_enhanced_review,omitempty"`
	DeemedAcceptedAt       *time.Time       `json:"deemed_accepted_at,omitempty"`
}

type ScheduleTerm struct {
	AmountKobo ledger.Money `json:"amount_kobo"`
	DueDate    string       `json:"due_date"`
}

type CreateInput struct {
	FeeTerms               *ledger.FeeTerms `json:"-"`
	CollectionPolicy       string           `json:"collection_policy,omitempty"`
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
	InvoiceDocumentID      string `json:"invoice_document_id,omitempty"`
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
	ReplaceInvoice      bool
	CollectionPolicy    string `json:"collection_policy,omitempty"`
	ExpectedVersion     int64
	PrincipalKobo       ledger.Money
	GoodsDescription    string
	InvoiceReference    string
	InvoiceDocumentID   string `json:"invoice_document_id,omitempty"`
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
	CollectionPolicy string          `json:"collection_policy,omitempty"`
	ID               string          `json:"id"`
	CreditRequestID  string          `json:"credit_request_id"`
	Version          int             `json:"version"`
	CanonicalJSON    json.RawMessage `json:"canonical_json"`
	CanonicalBytes   []byte          `json:"canonical_bytes,omitempty"`
	DocumentHash     string          `json:"document_hash"`
	PrincipalKobo    ledger.Money    `json:"principal_kobo"`
	DueDate          string          `json:"due_date"`
	GraceHours       int             `json:"grace_hours"`
	CollectionAt     time.Time       `json:"collection_at"`
	TermsVersion     string          `json:"terms_version"`
	PrivacyVersion   string          `json:"privacy_version"`
	CreatedBy        string          `json:"created_by"`
	CreatedAt        time.Time       `json:"created_at"`
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
	ID              string `json:"id"`
	CreditRequestID string `json:"credit_request_id"`
	SupplierActorID string `json:"supplier_actor_id"`
	DeliveryMethod  string `json:"delivery_method"`
	Notes           string `json:"notes,omitempty"`
	// WaybillNumber is a distinct piece of delivery evidence. It is never
	// derived from the free-text release notes: a note is not a waybill and
	// presenting one as the other would corrupt the release evidence trail.
	WaybillNumber    string    `json:"waybill_number,omitempty"`
	ReleasedAt       time.Time `json:"released_at"`
	DeemedAcceptedAt time.Time `json:"deemed_accepted_at,omitzero"`
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
	PurchasingActions []string              `json:"purchasing_actions"`
	Request           CreditRequest         `json:"request"`
	Agreement         AgreementVersion      `json:"agreement"`
	Acceptance        *Acceptance           `json:"acceptance,omitempty"`
	Mandate           *mandates.Mandate     `json:"mandate,omitempty"`
	Release           *GoodsRelease         `json:"release,omitempty"`
	Receipts          []ReceiptConfirmation `json:"receipts"`
	Obligation        *Obligation           `json:"obligation,omitempty"`
}

type CollectionState struct {
	CollectionPolicy       string
	MandateReference       string
	MandateProvider        string
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
	LegalVersions          *legalpublication.Versions `json:"-"`
	FeeTerms               *ledger.FeeTerms           `json:"-"`
	CollectionPolicy       string                     `json:"collection_policy,omitempty"`
	DrawdownID             string
	TradeLineID            string
	SupplierOrganizationID string
	BuyerUserID            string
	BuyerBusinessID        string
	MandateID              string
	PrincipalKobo          ledger.Money
	GoodsDescription       string
	InvoiceReference       string
	InvoiceDocumentID      string `json:"invoice_document_id,omitempty"`
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
	TradeLineTermsVersion string           `json:"trade_line_terms_version,omitempty"`
	FeeTerms              *ledger.FeeTerms `json:"fee_terms,omitempty"`
	CollectionPolicy      string           `json:"collection_policy,omitempty"`
	SourceType            string           `json:"source_type,omitempty"`
	SourceID              string           `json:"source_id,omitempty"`
	TradeLineID           string           `json:"trade_line_id,omitempty"`
	AcceptedSourceHash    string           `json:"accepted_source_hash,omitempty"`
	SupplierLegalName     string           `json:"supplier_legal_name"`
	SupplierTradingName   string           `json:"supplier_trading_name,omitempty"`
	BuyerLegalName        string           `json:"buyer_legal_name"`
	BuyerTradingName      string           `json:"buyer_trading_name,omitempty"`
	BuyerBusinessID       string           `json:"buyer_business_id"`
	GoodsDescription      string           `json:"goods_description"`
	InvoiceReference      string           `json:"invoice_reference,omitempty"`
	InvoiceDocumentID     string           `json:"invoice_document_id,omitempty"`
	InvoiceDocumentHash   string           `json:"invoice_document_hash,omitempty"`
	PrincipalKobo         ledger.Money     `json:"principal_kobo"`
	Currency              string           `json:"currency"`
	DueDate               string           `json:"due_date"`
	GraceHours            int              `json:"grace_hours"`
	CollectionAt          time.Time        `json:"collection_at"`
	ScheduleType          string           `json:"schedule_type"`
	ScheduleCount         int              `json:"schedule_count,omitempty"`
	ScheduleCadence       string           `json:"schedule_cadence"`
	MonthEndPolicy        string           `json:"month_end_policy,omitempty"`
	CustomScheduleItems   []ScheduleTerm   `json:"custom_schedule_items,omitempty"`
	MandateDisclosure     string           `json:"mandate_disclosure"`
	BaseFeeDisclosure     string           `json:"base_fee_disclosure"`
	TermsVersion          string           `json:"terms_version"`
	PrivacyVersion        string           `json:"privacy_version"`
}
