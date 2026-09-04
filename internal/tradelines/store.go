package tradelines

import (
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
)

const (
	LineProposed                  = "PROPOSED"
	LinePendingBuyer              = "PENDING_BUYER_ACCEPTANCE"
	LinePendingMandate            = "PENDING_MANDATE"
	LineActive                    = "ACTIVE"
	LineSuspended                 = "SUSPENDED"
	LineExpired                   = "EXPIRED"
	LineClosed                    = "CLOSED"
	DrawdownPending               = "PENDING_BUYER_CONFIRMATION"
	DrawdownConfirmed             = "BUYER_CONFIRMED"
	DrawdownGoodsReleased         = "GOODS_RELEASED"
	DrawdownReceiptIssue          = "RECEIPT_ISSUE_REPORTED"
	DrawdownActivated             = "ACTIVATED"
	DrawdownCancelled             = "CANCELLED"
	DrawdownExpired               = "EXPIRED"
	ReservationPending            = "PENDING"
	ReservationConfirmed          = "CONFIRMED"
	ReservationReleasedToSupplier = "RELEASED_TO_SUPPLIER"
	ReservationConverted          = "CONVERTED"
	ReservationExpired            = "EXPIRED"
	ReservationReleased           = "RELEASED"
)

type TradeLine struct {
	ID                     string       `json:"id"`
	SupplierOrganizationID string       `json:"supplier_organization_id"`
	BuyerUserID            string       `json:"buyer_user_id"`
	BuyerBusinessID        string       `json:"buyer_business_id"`
	ApprovedLimitKobo      ledger.Money `json:"approved_limit_kobo"`
	CurrentExposureKobo    ledger.Money `json:"current_exposure_kobo"`
	ReservedPendingKobo    ledger.Money `json:"reserved_pending_kobo"`
	AvailableLimitKobo     ledger.Money `json:"available_limit_kobo"`
	Cadence                string       `json:"cadence"`
	DefaultGraceHours      int          `json:"default_grace_hours"`
	StartAt                time.Time    `json:"start_at"`
	EndAt                  time.Time    `json:"end_at"`
	State                  string       `json:"state"`
	MandateID              string       `json:"mandate_id,omitempty"`
	MandateActive          bool         `json:"mandate_active"`
	SuspensionReason       string       `json:"suspension_reason,omitempty"`
	TermsVersion           string       `json:"terms_version"`
	CreatedAt              time.Time    `json:"created_at"`
	UpdatedAt              time.Time    `json:"updated_at"`
	Version                int64        `json:"version"`
}

type CreateLineInput struct {
	SupplierOrganizationID string
	BuyerUserID            string
	BuyerBusinessID        string
	ApprovedLimitKobo      ledger.Money
	Cadence                string
	DefaultGraceHours      int
	StartAt                time.Time
	EndAt                  time.Time
	MandateID              string
	MandateActive          bool
	MandateVerified        bool
	TermsVersion           string
}
type CreateDrawdownInput struct {
	FeeTerms            *ledger.FeeTerms `json:"-"`
	LineID              string
	PrincipalKobo       ledger.Money
	GoodsDescription    string
	InvoiceReference    string
	InvoiceDocumentHash string
	DueDate             string
	CollectionAt        time.Time
	IdempotencyKey      string
	ExpiresAt           time.Time
}
type Drawdown struct {
	OutstandingKobo          *ledger.Money    `json:"outstanding_kobo,omitempty"`
	FeeTerms                 *ledger.FeeTerms `json:"fee_terms,omitempty"`
	ID                       string           `json:"id"`
	TradeLineID              string           `json:"trade_line_id"`
	PrincipalKobo            ledger.Money     `json:"principal_kobo"`
	GoodsDescription         string           `json:"goods_description"`
	InvoiceReference         string           `json:"invoice_reference,omitempty"`
	InvoiceDocumentHash      string           `json:"invoice_document_hash,omitempty"`
	DueDate                  string           `json:"due_date"`
	CollectionAt             time.Time        `json:"collection_at"`
	GraceHours               int              `json:"grace_hours"`
	TermsVersion             string           `json:"terms_version"`
	AgreementHash            string           `json:"agreement_hash"`
	State                    string           `json:"state"`
	ReservationID            string           `json:"reservation_id"`
	ObligationID             string           `json:"obligation_id,omitempty"`
	BuyerConfirmedAt         time.Time        `json:"buyer_confirmed_at,omitempty"`
	ReleaseActorID           string           `json:"release_actor_id,omitempty"`
	DeliveryMethod           string           `json:"delivery_method,omitempty"`
	ReleaseNotes             string           `json:"release_notes,omitempty"`
	ReleaseEvidenceReference string           `json:"release_evidence_reference,omitempty"`
	ReleasedAt               time.Time        `json:"released_at,omitempty"`
	ReceiptState             string           `json:"receipt_state,omitempty"`
	ReceiptActorID           string           `json:"receipt_actor_id,omitempty"`
	ReceiptIssueReason       string           `json:"receipt_issue_reason,omitempty"`
	ReceiptDisputeID         string           `json:"receipt_dispute_id,omitempty"`
	ReceiptAt                time.Time        `json:"receipt_at,omitempty"`
	ActivatedAt              time.Time        `json:"activated_at,omitempty"`
	CreatedAt                time.Time        `json:"created_at"`
}
type Reservation struct {
	ID             string       `json:"id"`
	TradeLineID    string       `json:"trade_line_id"`
	DrawdownID     string       `json:"drawdown_id"`
	AmountKobo     ledger.Money `json:"amount_kobo"`
	State          string       `json:"state"`
	ExpiresAt      time.Time    `json:"expires_at"`
	IdempotencyKey string       `json:"idempotency_key"`
	CreatedAt      time.Time    `json:"created_at"`
}
type Statement struct {
	Line        TradeLine  `json:"line"`
	Drawdowns   []Drawdown `json:"drawdowns"`
	GeneratedAt time.Time  `json:"generated_at"`
}

type ReleaseInput struct {
	DrawdownID             string
	SupplierOrganizationID string
	ActorID                string
	DeliveryMethod         string
	Notes                  string
	EvidenceReference      string
}

type ReceiptInput struct {
	DrawdownID  string
	BuyerUserID string
	State       string
	IssueReason string
}

type ActivationInput struct {
	Drawdown Drawdown
	Line     TradeLine
}

type Service interface {
	SetLineGuard(func(CreateLineInput) error)
	SetMaxDrawdownsPerLineDay(int)
	SetMaxActiveExposure(ledger.Money)
	SetActivationHandler(func(ActivationInput) (string, error))
	CreateLine(CreateLineInput) (TradeLine, error)
	Get(string) (TradeLine, bool)
	ListForSupplier(string) []TradeLine
	ListForBuyer(string) []TradeLine
	ReserveDrawdown(CreateDrawdownInput) (Drawdown, Reservation, TradeLine, error)
	ConfirmDrawdown(string, string, string) (Drawdown, TradeLine, error)
	ReleaseDrawdown(ReleaseInput) (Drawdown, TradeLine, error)
	RecordDrawdownReceipt(ReceiptInput) (Drawdown, TradeLine, error)
	CancelDrawdown(string, string, string) (Drawdown, TradeLine, error)
	UpdateOutstanding(string, ledger.Money) (TradeLine, error)
	Suspend(string, string) (TradeLine, error)
	Resume(string) (TradeLine, error)
	ReduceLimit(string, ledger.Money, int64) (TradeLine, error)
	SetMandateState(string, string, bool) (TradeLine, error)
	Statement(string) (Statement, error)
}

var _ Service = (*Store)(nil)

type Store struct {
	mu                     sync.RWMutex
	lines                  map[string]*TradeLine
	drawdowns              map[string]*Drawdown
	reservations           map[string]*Reservation
	byKey                  map[string]string
	byLine                 map[string][]string
	now                    func() time.Time
	newID                  func() string
	lineGuard              func(CreateLineInput) error
	maxDrawdownsPerLineDay int
	maxActiveExposureKobo  ledger.Money
	activationHandler      func(ActivationInput) (string, error)
}

func NewStore() *Store {
	return &Store{lines: map[string]*TradeLine{}, drawdowns: map[string]*Drawdown{}, reservations: map[string]*Reservation{}, byKey: map[string]string{}, byLine: map[string][]string{}, now: func() time.Time { return time.Now().UTC() }, newID: newIdentifier}
}

func (s *Store) SetLineGuard(guard func(CreateLineInput) error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lineGuard = guard
}
func (s *Store) SetMaxDrawdownsPerLineDay(max int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if max > 0 {
		s.maxDrawdownsPerLineDay = max
	}
}

func (s *Store) SetMaxActiveExposure(threshold ledger.Money) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.maxActiveExposureKobo = threshold
}

func (s *Store) SetActivationHandler(handler func(ActivationInput) (string, error)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.activationHandler = handler
}

func (s *Store) CreateLine(input CreateLineInput) (TradeLine, error) {
	s.mu.RLock()
	guard := s.lineGuard
	s.mu.RUnlock()
	if guard != nil {
		if err := guard(input); err != nil {
			return TradeLine{}, err
		}
	}
	if strings.TrimSpace(input.SupplierOrganizationID) == "" || strings.TrimSpace(input.BuyerUserID) == "" || strings.TrimSpace(input.BuyerBusinessID) == "" {
		return TradeLine{}, errors.New("supplier, buyer user, and buyer business are required")
	}
	if input.ApprovedLimitKobo <= 0 {
		return TradeLine{}, errors.New("approved limit must be positive")
	}
	if input.MandateActive && !input.MandateVerified {
		return TradeLine{}, errors.New("active mandate must be verified by the server")
	}
	if input.Cadence == "" {
		input.Cadence = "friday"
	}
	if input.DefaultGraceHours < 0 || input.DefaultGraceHours > 720 {
		return TradeLine{}, errors.New("grace hours must be between 0 and 720")
	}
	if input.StartAt.IsZero() {
		input.StartAt = s.now()
	}
	if input.EndAt.IsZero() || !input.EndAt.After(input.StartAt) {
		return TradeLine{}, errors.New("end time must be after start time")
	}
	state := LinePendingMandate
	if input.MandateID != "" && input.MandateActive {
		state = LineActive
	}
	line := &TradeLine{ID: s.newID(), SupplierOrganizationID: input.SupplierOrganizationID, BuyerUserID: input.BuyerUserID, BuyerBusinessID: input.BuyerBusinessID, ApprovedLimitKobo: input.ApprovedLimitKobo, AvailableLimitKobo: input.ApprovedLimitKobo, Cadence: input.Cadence, DefaultGraceHours: input.DefaultGraceHours, StartAt: input.StartAt, EndAt: input.EndAt, State: state, MandateID: input.MandateID, MandateActive: input.MandateActive, TermsVersion: defaultTerms(input.TermsVersion), CreatedAt: s.now(), UpdatedAt: s.now(), Version: 1}
	s.mu.Lock()
	if s.maxActiveExposureKobo > 0 {
		var exposure ledger.Money
		for _, existing := range s.lines {
			if existing.BuyerBusinessID == input.BuyerBusinessID && existing.State != LineClosed && existing.State != LineExpired {
				lineExposure, addErr := ledger.CheckedAdd(existing.CurrentExposureKobo, existing.ReservedPendingKobo)
				if addErr != nil {
					s.mu.Unlock()
					return TradeLine{}, errors.New("buyer active exposure is too large")
				}
				exposure, addErr = ledger.CheckedAdd(exposure, lineExposure)
				if addErr != nil {
					s.mu.Unlock()
					return TradeLine{}, errors.New("buyer active exposure is too large")
				}
			}
		}
		totalExposure, addErr := ledger.CheckedAdd(exposure, input.ApprovedLimitKobo)
		if addErr != nil || totalExposure > s.maxActiveExposureKobo {
			s.mu.Unlock()
			return TradeLine{}, errors.New("buyer active exposure exceeds configured pilot limit")
		}
	}
	s.lines[line.ID] = line
	s.mu.Unlock()
	return cloneLine(*line), nil
}

func (s *Store) Get(lineID string) (TradeLine, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	line, ok := s.lines[lineID]
	if !ok {
		return TradeLine{}, false
	}
	return cloneLine(*line), true
}
func (s *Store) ListForSupplier(orgID string) []TradeLine {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := []TradeLine{}
	for _, line := range s.lines {
		if line.SupplierOrganizationID == orgID {
			out = append(out, cloneLine(*line))
		}
	}
	return out
}

func (s *Store) ListForBuyer(buyerUserID string) []TradeLine {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := []TradeLine{}
	for _, line := range s.lines {
		if line.BuyerUserID == buyerUserID {
			out = append(out, cloneLine(*line))
		}
	}
	return out
}

func (s *Store) ReserveDrawdown(input CreateDrawdownInput) (Drawdown, Reservation, TradeLine, error) {
	if input.LineID == "" || input.PrincipalKobo <= 0 || strings.TrimSpace(input.GoodsDescription) == "" || input.IdempotencyKey == "" {
		return Drawdown{}, Reservation{}, TradeLine{}, errors.New("line, positive principal, goods, and idempotency key are required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.expireReservationsLocked()
	if existing := s.byKey[input.IdempotencyKey]; existing != "" {
		d := s.drawdowns[existing]
		r := s.reservations[d.ReservationID]
		return cloneDrawdown(*d), cloneReservation(*r), cloneLine(*s.lines[input.LineID]), nil
	}
	line := s.lines[input.LineID]
	if line == nil {
		return Drawdown{}, Reservation{}, TradeLine{}, errors.New("trade line not found")
	}
	if s.maxDrawdownsPerLineDay > 0 {
		today := s.now().UTC().Format("2006-01-02")
		count := 0
		for _, drawdownID := range s.byLine[line.ID] {
			if existing := s.drawdowns[drawdownID]; existing != nil && existing.CreatedAt.UTC().Format("2006-01-02") == today {
				count++
			}
		}
		if count >= s.maxDrawdownsPerLineDay {
			return Drawdown{}, Reservation{}, TradeLine{}, errors.New("daily pilot drawdown limit reached")
		}
	}
	if err := s.eligibleLocked(line); err != nil {
		return Drawdown{}, Reservation{}, TradeLine{}, err
	}
	if input.PrincipalKobo > line.AvailableLimitKobo {
		return Drawdown{}, Reservation{}, TradeLine{}, errors.New("drawdown exceeds available limit")
	}
	expires := input.ExpiresAt
	if expires.IsZero() {
		expires = s.now().Add(24 * time.Hour)
	}
	if !expires.After(s.now()) {
		return Drawdown{}, Reservation{}, TradeLine{}, errors.New("reservation expiry must be in the future")
	}
	if strings.TrimSpace(input.DueDate) == "" {
		input.DueDate = s.now().AddDate(0, 1, 0).Format("2006-01-02")
	}
	due, err := time.Parse("2006-01-02", input.DueDate)
	if err != nil {
		return Drawdown{}, Reservation{}, TradeLine{}, errors.New("due date must use YYYY-MM-DD")
	}
	if input.CollectionAt.IsZero() {
		input.CollectionAt = due.Add(time.Duration(line.DefaultGraceHours) * time.Hour)
	}
	if input.CollectionAt.Before(due) {
		return Drawdown{}, Reservation{}, TradeLine{}, errors.New("collection time cannot be before due date")
	}
	drawdown := &Drawdown{FeeTerms: input.FeeTerms.Clone(), ID: s.newID(), TradeLineID: line.ID, PrincipalKobo: input.PrincipalKobo, GoodsDescription: strings.TrimSpace(input.GoodsDescription), InvoiceReference: strings.TrimSpace(input.InvoiceReference), InvoiceDocumentHash: strings.TrimSpace(input.InvoiceDocumentHash), DueDate: input.DueDate, CollectionAt: input.CollectionAt, GraceHours: line.DefaultGraceHours, TermsVersion: line.TermsVersion, State: DrawdownPending, CreatedAt: s.now()}
	drawdown.AgreementHash = drawdownHash(*drawdown, *line)
	reservation := &Reservation{ID: s.newID(), TradeLineID: line.ID, DrawdownID: drawdown.ID, AmountKobo: input.PrincipalKobo, State: ReservationPending, ExpiresAt: expires, IdempotencyKey: input.IdempotencyKey, CreatedAt: s.now()}
	drawdown.ReservationID = reservation.ID
	s.drawdowns[drawdown.ID] = drawdown
	s.reservations[reservation.ID] = reservation
	s.byKey[input.IdempotencyKey] = drawdown.ID
	s.byLine[line.ID] = append(s.byLine[line.ID], drawdown.ID)
	reserved, addErr := ledger.CheckedAdd(line.ReservedPendingKobo, input.PrincipalKobo)
	if addErr != nil {
		delete(s.drawdowns, drawdown.ID)
		delete(s.reservations, reservation.ID)
		delete(s.byKey, input.IdempotencyKey)
		s.byLine[line.ID] = s.byLine[line.ID][:len(s.byLine[line.ID])-1]
		return Drawdown{}, Reservation{}, TradeLine{}, errors.New("reserved trade-line amount is too large")
	}
	line.ReservedPendingKobo = reserved
	s.recalculateLocked(line)
	line.UpdatedAt = s.now()
	line.Version++
	return cloneDrawdown(*drawdown), cloneReservation(*reservation), cloneLine(*line), nil
}

func (s *Store) ConfirmDrawdown(drawdownID, buyerUserID, agreementHash string) (Drawdown, TradeLine, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.expireReservationsLocked()
	d := s.drawdowns[drawdownID]
	if d == nil {
		return Drawdown{}, TradeLine{}, errors.New("drawdown not found")
	}
	line := s.lines[d.TradeLineID]
	if line == nil || line.BuyerUserID != buyerUserID {
		return Drawdown{}, TradeLine{}, errors.New("buyer mismatch")
	}
	if d.State != DrawdownPending && !d.BuyerConfirmedAt.IsZero() && agreementHash == d.AgreementHash && VerifyAgreementHash(*d, *line) {
		return cloneDrawdown(*d), cloneLine(*line), nil
	}
	if d.State != DrawdownPending {
		return Drawdown{}, TradeLine{}, fmt.Errorf("drawdown cannot be confirmed from %s", d.State)
	}
	if err := s.eligibleLocked(line); err != nil {
		return Drawdown{}, TradeLine{}, err
	}
	if strings.TrimSpace(agreementHash) == "" || agreementHash != d.AgreementHash || !VerifyAgreementHash(*d, *line) {
		return Drawdown{}, TradeLine{}, errors.New("drawdown agreement hash mismatch")
	}
	r := s.reservations[d.ReservationID]
	if r == nil || r.State != ReservationPending {
		return Drawdown{}, TradeLine{}, errors.New("reservation is not pending")
	}
	d.State = DrawdownConfirmed
	d.BuyerConfirmedAt = s.now()
	r.State = ReservationConfirmed
	line.UpdatedAt = s.now()
	line.Version++
	return cloneDrawdown(*d), cloneLine(*line), nil
}

func (s *Store) ReleaseDrawdown(input ReleaseInput) (Drawdown, TradeLine, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.expireReservationsLocked()
	d := s.drawdowns[input.DrawdownID]
	if d == nil {
		return Drawdown{}, TradeLine{}, errors.New("drawdown not found")
	}
	line := s.lines[d.TradeLineID]
	if line == nil || line.SupplierOrganizationID != input.SupplierOrganizationID {
		return Drawdown{}, TradeLine{}, errors.New("trade line not found")
	}
	if d.State != DrawdownConfirmed && !d.ReleasedAt.IsZero() && d.ReleaseActorID == input.ActorID && d.DeliveryMethod == strings.TrimSpace(input.DeliveryMethod) && d.ReleaseEvidenceReference == strings.TrimSpace(input.EvidenceReference) {
		return cloneDrawdown(*d), cloneLine(*line), nil
	}
	if d.State != DrawdownConfirmed {
		return Drawdown{}, TradeLine{}, fmt.Errorf("drawdown cannot be released from %s", d.State)
	}
	if err := s.eligibleLocked(line); err != nil {
		return Drawdown{}, TradeLine{}, err
	}
	r := s.reservations[d.ReservationID]
	if r == nil || r.State != ReservationConfirmed {
		return Drawdown{}, TradeLine{}, errors.New("confirmed reservation required")
	}
	if strings.TrimSpace(input.ActorID) == "" || strings.TrimSpace(input.DeliveryMethod) == "" || strings.TrimSpace(input.EvidenceReference) == "" {
		return Drawdown{}, TradeLine{}, errors.New("release actor, delivery method, and evidence reference are required")
	}
	d.State = DrawdownGoodsReleased
	d.ReleaseActorID = input.ActorID
	d.DeliveryMethod = strings.TrimSpace(input.DeliveryMethod)
	d.ReleaseNotes = strings.TrimSpace(input.Notes)
	d.ReleaseEvidenceReference = strings.TrimSpace(input.EvidenceReference)
	d.ReleasedAt = s.now()
	r.State = ReservationReleasedToSupplier
	line.UpdatedAt = s.now()
	line.Version++
	return cloneDrawdown(*d), cloneLine(*line), nil
}

func (s *Store) RecordDrawdownReceipt(input ReceiptInput) (Drawdown, TradeLine, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	d := s.drawdowns[input.DrawdownID]
	if d == nil {
		return Drawdown{}, TradeLine{}, errors.New("drawdown not found")
	}
	line := s.lines[d.TradeLineID]
	if line == nil || line.BuyerUserID != input.BuyerUserID {
		return Drawdown{}, TradeLine{}, errors.New("buyer mismatch")
	}
	if input.State == "no_issue" && d.State == DrawdownActivated && d.ReceiptState == input.State && d.ReceiptActorID == input.BuyerUserID {
		return cloneDrawdown(*d), cloneLine(*line), nil
	}
	if input.State == "issue_reported" && d.State == DrawdownReceiptIssue && d.ReceiptState == input.State && d.ReceiptActorID == input.BuyerUserID && d.ReceiptIssueReason == strings.TrimSpace(input.IssueReason) {
		return cloneDrawdown(*d), cloneLine(*line), nil
	}
	if d.State != DrawdownGoodsReleased {
		return Drawdown{}, TradeLine{}, fmt.Errorf("drawdown receipt cannot be recorded from %s", d.State)
	}
	if input.State == "no_issue" {
		if err := s.eligibleLocked(line); err != nil {
			return Drawdown{}, TradeLine{}, err
		}
	}
	if input.State != "no_issue" && input.State != "issue_reported" {
		return Drawdown{}, TradeLine{}, errors.New("receipt state must be no_issue or issue_reported")
	}
	if input.State == "issue_reported" && strings.TrimSpace(input.IssueReason) == "" {
		return Drawdown{}, TradeLine{}, errors.New("receipt issue reason is required")
	}
	d.ReceiptState = input.State
	d.ReceiptActorID = input.BuyerUserID
	d.ReceiptIssueReason = strings.TrimSpace(input.IssueReason)
	d.ReceiptAt = s.now()
	if input.State == "issue_reported" {
		if d.ReceiptDisputeID == "" {
			d.ReceiptDisputeID = s.newID()
		}
		d.State = DrawdownReceiptIssue
		line.UpdatedAt = s.now()
		line.Version++
		return cloneDrawdown(*d), cloneLine(*line), nil
	}
	r := s.reservations[d.ReservationID]
	if r == nil || r.State != ReservationReleasedToSupplier {
		return Drawdown{}, TradeLine{}, errors.New("released reservation required")
	}
	if s.maxActiveExposureKobo > 0 {
		var exposure ledger.Money
		for _, existing := range s.lines {
			if existing.BuyerBusinessID == line.BuyerBusinessID {
				var addErr error
				exposure, addErr = ledger.CheckedAdd(exposure, existing.CurrentExposureKobo)
				if addErr != nil {
					return Drawdown{}, TradeLine{}, errors.New("buyer active exposure is too large")
				}
			}
		}
		totalExposure, addErr := ledger.CheckedAdd(exposure, d.PrincipalKobo)
		if addErr != nil || totalExposure > s.maxActiveExposureKobo {
			return Drawdown{}, TradeLine{}, errors.New("buyer active exposure exceeds configured pilot limit")
		}
	}
	if s.activationHandler == nil {
		return Drawdown{}, TradeLine{}, errors.New("drawdown activation is unavailable")
	}
	currentExposure, addErr := ledger.CheckedAdd(line.CurrentExposureKobo, d.PrincipalKobo)
	if addErr != nil {
		return Drawdown{}, TradeLine{}, errors.New("trade-line exposure is too large")
	}
	obligationID, err := s.activationHandler(ActivationInput{Drawdown: cloneDrawdown(*d), Line: cloneLine(*line)})
	if err != nil {
		return Drawdown{}, TradeLine{}, err
	}
	if strings.TrimSpace(obligationID) == "" {
		return Drawdown{}, TradeLine{}, errors.New("drawdown activation did not create an obligation")
	}
	line.ReservedPendingKobo -= d.PrincipalKobo
	line.CurrentExposureKobo = currentExposure
	r.State = ReservationConverted
	d.State = DrawdownActivated
	d.ObligationID = obligationID
	principal := d.PrincipalKobo
	d.OutstandingKobo = &principal
	d.ActivatedAt = s.now()
	s.recalculateLocked(line)
	line.UpdatedAt = s.now()
	line.Version++
	return cloneDrawdown(*d), cloneLine(*line), nil
}

func (s *Store) CancelDrawdown(authorizedLineID, drawdownID, actorID string) (Drawdown, TradeLine, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	d := s.drawdowns[drawdownID]
	if d == nil {
		return Drawdown{}, TradeLine{}, errors.New("drawdown not found")
	}
	line := s.lines[d.TradeLineID]
	if line == nil || authorizedLineID == "" || d.TradeLineID != authorizedLineID || strings.TrimSpace(actorID) == "" {
		return Drawdown{}, TradeLine{}, errors.New("drawdown cancellation actor is invalid")
	}
	if d.State == DrawdownCancelled {
		return cloneDrawdown(*d), cloneLine(*line), nil
	}
	if d.State != DrawdownPending && d.State != DrawdownConfirmed {
		return Drawdown{}, TradeLine{}, fmt.Errorf("drawdown cannot be cancelled from %s", d.State)
	}
	r := s.reservations[d.ReservationID]
	if r == nil || (r.State != ReservationPending && r.State != ReservationConfirmed) {
		return Drawdown{}, TradeLine{}, errors.New("releasable reservation required")
	}
	d.State = DrawdownCancelled
	r.State = ReservationReleased
	line.ReservedPendingKobo -= d.PrincipalKobo
	s.recalculateLocked(line)
	line.UpdatedAt = s.now()
	line.Version++
	return cloneDrawdown(*d), cloneLine(*line), nil
}

func (s *Store) UpdateOutstanding(drawdownID string, outstanding ledger.Money) (TradeLine, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	d := s.drawdowns[drawdownID]
	if d == nil {
		return TradeLine{}, errors.New("drawdown not found")
	}
	line := s.lines[d.TradeLineID]
	if line == nil {
		return TradeLine{}, errors.New("trade line not found")
	}
	if d.State != DrawdownActivated || outstanding < 0 || outstanding > d.PrincipalKobo {
		return TradeLine{}, errors.New("invalid activated outstanding")
	}
	previous := d.PrincipalKobo
	if d.OutstandingKobo != nil {
		previous = *d.OutstandingKobo
	}
	line.CurrentExposureKobo += outstanding - previous
	d.OutstandingKobo = &outstanding
	s.recalculateLocked(line)
	line.UpdatedAt = s.now()
	line.Version++
	return cloneLine(*line), nil
}

func (s *Store) Suspend(lineID, reason string) (TradeLine, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	line := s.lines[lineID]
	if line == nil {
		return TradeLine{}, errors.New("trade line not found")
	}
	if strings.TrimSpace(reason) == "" {
		return TradeLine{}, errors.New("suspension reason is required")
	}
	if line.State == LineClosed || line.State == LineExpired {
		return TradeLine{}, errors.New("closed or expired line cannot be suspended")
	}
	line.State = LineSuspended
	line.SuspensionReason = strings.TrimSpace(reason)
	line.UpdatedAt = s.now()
	line.Version++
	return cloneLine(*line), nil
}
func (s *Store) Resume(lineID string) (TradeLine, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	line := s.lines[lineID]
	if line == nil {
		return TradeLine{}, errors.New("trade line not found")
	}
	if line.State != LineSuspended {
		return TradeLine{}, fmt.Errorf("line cannot resume from %s", line.State)
	}
	if !line.MandateActive {
		return TradeLine{}, errors.New("active mandate required")
	}
	if !s.now().Before(line.EndAt) {
		line.State = LineExpired
		return TradeLine{}, errors.New("trade line has expired")
	}
	line.State = LineActive
	line.SuspensionReason = ""
	line.UpdatedAt = s.now()
	line.Version++
	return cloneLine(*line), nil
}
func (s *Store) ReduceLimit(lineID string, limit ledger.Money, expectedVersion int64) (TradeLine, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	line := s.lines[lineID]
	if line == nil {
		return TradeLine{}, errors.New("trade line not found")
	}
	if expectedVersion <= 0 || line.Version != expectedVersion {
		return TradeLine{}, errors.New("trade line version conflict")
	}
	if limit <= 0 || limit >= line.ApprovedLimitKobo {
		return TradeLine{}, errors.New("supplier may only reduce the approved limit; increases require fresh buyer acceptance")
	}
	if limit < line.CurrentExposureKobo+line.ReservedPendingKobo {
		return TradeLine{}, errors.New("limit cannot be reduced below active and reserved exposure")
	}
	line.ApprovedLimitKobo = limit
	line.UpdatedAt = s.now()
	line.Version++
	s.recalculateLocked(line)
	return cloneLine(*line), nil
}
func (s *Store) SetMandateState(lineID string, mandateID string, active bool) (TradeLine, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	line := s.lines[lineID]
	if line == nil {
		return TradeLine{}, errors.New("trade line not found")
	}
	line.MandateID = mandateID
	line.MandateActive = active
	if !active && line.State == LineActive {
		line.State = LineSuspended
		line.SuspensionReason = "mandate_inactive"
	}
	if active && line.State == LinePendingMandate && s.now().Before(line.EndAt) {
		line.State = LineActive
		line.SuspensionReason = ""
	}
	s.recalculateLocked(line)
	line.UpdatedAt = s.now()
	line.Version++
	return cloneLine(*line), nil
}
func (s *Store) Statement(lineID string) (Statement, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	line := s.lines[lineID]
	if line == nil {
		return Statement{}, errors.New("trade line not found")
	}
	out := Statement{Line: cloneLine(*line), Drawdowns: []Drawdown{}, GeneratedAt: s.now()}
	for _, id := range s.byLine[lineID] {
		if d := s.drawdowns[id]; d != nil {
			out.Drawdowns = append(out.Drawdowns, cloneDrawdown(*d))
		}
	}
	return out, nil
}

func (s *Store) eligibleLocked(line *TradeLine) error {
	if line.State != LineActive {
		return fmt.Errorf("trade line is %s", line.State)
	}
	if !line.MandateActive {
		return errors.New("active mandate required")
	}
	if !s.now().Before(line.EndAt) {
		line.State = LineExpired
		return errors.New("trade line has expired")
	}
	return nil
}
func (s *Store) expireReservationsLocked() {
	now := s.now()
	for _, reservation := range s.reservations {
		if (reservation.State != ReservationPending && reservation.State != ReservationConfirmed) || now.Before(reservation.ExpiresAt) {
			continue
		}
		reservation.State = ReservationExpired
		if line := s.lines[reservation.TradeLineID]; line != nil {
			line.ReservedPendingKobo -= reservation.AmountKobo
			if line.ReservedPendingKobo < 0 {
				line.ReservedPendingKobo = 0
			}
			s.recalculateLocked(line)
			line.UpdatedAt = now
			line.Version++
		}
		if d := s.drawdowns[reservation.DrawdownID]; d != nil && (d.State == DrawdownPending || d.State == DrawdownConfirmed) {
			d.State = DrawdownExpired
		}
	}
}

const currentFeeDisclosure = "0.5% supplier base service fee on activated principal; an additional 0.5% only on amounts Kredit successfully collects at or after the permitted collection time (up to 1% total on collected principal)"
const legacyFeeDisclosure = "0.5% supplier base service fee on activated principal; 1% collection fee only on amounts Kredit successfully collects"

func drawdownHash(drawdown Drawdown, line TradeLine) string {
	if drawdown.FeeTerms != nil {
		return drawdownHashWithFee(drawdown, line, drawdown.FeeTerms.Disclosure())
	}
	return drawdownHashWithFee(drawdown, line, currentFeeDisclosure)
}

func drawdownHashWithFee(drawdown Drawdown, line TradeLine, fee string) string {
	canonical := struct {
		DrawdownID             string       `json:"drawdown_id"`
		TradeLineID            string       `json:"trade_line_id"`
		SupplierOrganizationID string       `json:"supplier_organization_id"`
		BuyerUserID            string       `json:"buyer_user_id"`
		BuyerBusinessID        string       `json:"buyer_business_id"`
		PrincipalKobo          ledger.Money `json:"principal_kobo"`
		Currency               string       `json:"currency"`
		GoodsDescription       string       `json:"goods_description"`
		InvoiceReference       string       `json:"invoice_reference,omitempty"`
		InvoiceDocumentHash    string       `json:"invoice_document_hash,omitempty"`
		DueDate                string       `json:"due_date"`
		CollectionAt           time.Time    `json:"collection_at"`
		GraceHours             int          `json:"grace_hours"`
		RepaymentCadence       string       `json:"repayment_cadence"`
		MandateID              string       `json:"mandate_id"`
		TermsVersion           string       `json:"terms_version"`
		FeeDisclosure          string       `json:"fee_disclosure"`
	}{drawdown.ID, drawdown.TradeLineID, line.SupplierOrganizationID, line.BuyerUserID, line.BuyerBusinessID, drawdown.PrincipalKobo, "NGN", drawdown.GoodsDescription, drawdown.InvoiceReference, drawdown.InvoiceDocumentHash, drawdown.DueDate, drawdown.CollectionAt.UTC(), drawdown.GraceHours, line.Cadence, line.MandateID, drawdown.TermsVersion, fee}
	payload, _ := json.Marshal(canonical)
	hash := sha256.Sum256(payload)
	return hex.EncodeToString(hash[:])
}

// VerifyAgreementHash proves that the stored digest still represents the
// complete immutable drawdown terms and trade-line parties.
func VerifyAgreementHash(drawdown Drawdown, line TradeLine) bool {
	return DrawdownFeeDisclosure(drawdown, line) != ""
}

func (s *Store) recalculateLocked(line *TradeLine) {
	line.AvailableLimitKobo = line.ApprovedLimitKobo - line.CurrentExposureKobo - line.ReservedPendingKobo
	if line.AvailableLimitKobo < 0 {
		line.AvailableLimitKobo = 0
	}
}
func cloneLine(v TradeLine) TradeLine { return v }
func cloneDrawdown(v Drawdown) Drawdown {
	v.FeeTerms = v.FeeTerms.Clone()
	if v.OutstandingKobo != nil {
		n := *v.OutstandingKobo
		v.OutstandingKobo = &n
	}
	return v
}
func cloneReservation(v Reservation) Reservation { return v }
func defaultTerms(v string) string {
	if strings.TrimSpace(v) == "" {
		return "terms-v1"
	}
	return v
}
func newIdentifier() string { return identifier.New() }

// Preserve the exact fee wording bound by older accepted hashes. New agreements
// use the corrected disclosure; rendering never silently rewrites accepted terms.
func DrawdownFeeDisclosure(drawdown Drawdown, line TradeLine) string {
	if drawdown.AgreementHash == "" {
		return ""
	}
	if drawdown.FeeTerms != nil {
		fee := drawdown.FeeTerms.Disclosure()
		if drawdownHashWithFee(drawdown, line, fee) == drawdown.AgreementHash {
			return fee
		}
		return ""
	}
	for _, fee := range []string{currentFeeDisclosure, legacyFeeDisclosure} {
		if drawdownHashWithFee(drawdown, line, fee) == drawdown.AgreementHash {
			return fee
		}
	}
	return ""
}

// ApplyObligationDelta keeps the development store's balance and capacity in
// one critical section, using the same line-before-credit lock order as activation.
func (s *Store) ApplyObligationDelta(obligationID string, delta ledger.Money, apply func() error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, d := range s.drawdowns {
		if d.ObligationID != obligationID || d.State != DrawdownActivated {
			continue
		}
		previous := d.PrincipalKobo
		if d.OutstandingKobo != nil {
			previous = *d.OutstandingKobo
		}
		next, err := ledger.CheckedAdd(previous, delta)
		if err != nil || next < 0 || next > d.PrincipalKobo {
			return errors.New("invalid drawdown balance")
		}
		line := s.lines[d.TradeLineID]
		exposure, err := ledger.CheckedAdd(line.CurrentExposureKobo, delta)
		if err != nil || exposure < 0 {
			return errors.New("invalid line exposure")
		}
		if err := apply(); err != nil {
			return err
		}
		d.OutstandingKobo = &next
		line.CurrentExposureKobo = exposure
		s.recalculateLocked(line)
		line.Version++
		line.UpdatedAt = s.now()
		return nil
	}
	return apply()
}
