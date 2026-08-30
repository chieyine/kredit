package ledger

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"sync"
	"time"
)

type Money int64

const (
	AccountTradeReceivable           = "TRADE_RECEIVABLE_CONTROL"
	AccountPrincipalOriginated       = "PRINCIPAL_ORIGINATED_CONTROL"
	AccountSupplierFeeReceivable     = "SUPPLIER_FEE_RECEIVABLE"
	AccountPlatformServiceRevenue    = "PLATFORM_SERVICE_REVENUE"
	AccountVoluntarySettlement       = "VOLUNTARY_SETTLEMENT_CONTROL"
	AccountCollectionSettlement      = "COLLECTION_SETTLEMENT_CONTROL"
	AccountPlatformCollectionRevenue = "PLATFORM_COLLECTION_REVENUE"
	AccountReturnsAdjustment         = "RETURNS_ADJUSTMENT_CONTROL"
	AccountWriteOff                  = "WRITE_OFF_CONTROL"
)

type Posting struct {
	Account string `json:"account"`
	Debit   Money  `json:"debit"`
	Credit  Money  `json:"credit"`
}

type Transaction struct {
	ID             string    `json:"id"`
	EventType      string    `json:"event_type"`
	ReferenceType  string    `json:"reference_type"`
	ReferenceID    string    `json:"reference_id"`
	IdempotencyKey string    `json:"idempotency_key"`
	EffectiveAt    time.Time `json:"effective_at"`
	RecordedAt     time.Time `json:"recorded_at"`
	Postings       []Posting `json:"postings"`
}

// Service is the financial journal boundary. Implementations must preserve
// the append-only/idempotent semantics of Store; the runtime can therefore
// use the in-memory implementation for local development and PostgreSQL for
// every durable environment.
type Service interface {
	PostPayment(paymentID string, amount Money, source string, effectiveAt time.Time, idempotencyKey string) (Transaction, error)
	PostPaymentReversal(paymentID string, amount Money, source string, effectiveAt time.Time, idempotencyKey string) (Transaction, error)
	PostCollectionFee(paymentID string, amount Money, effectiveAt time.Time, idempotencyKey string) (Transaction, error)
	PostCollectionFeeReversal(paymentID string, amount Money, effectiveAt time.Time, idempotencyKey string) (Transaction, error)
	PostAdjustment(referenceID string, amount Money, adjustmentType string, effectiveAt time.Time, idempotencyKey string) (Transaction, error)
	PostFeeWaiver(referenceID string, amount Money, effectiveAt time.Time, idempotencyKey string) (Transaction, error)
	PostActivation(obligationID string, principal Money, effectiveAt time.Time, idempotencyKey string) (Transaction, error)
	GetByReference(referenceID string) ([]Transaction, error)
}

type Store struct {
	mu           sync.RWMutex
	transactions map[string]Transaction
	byReference  map[string]string
	now          func() time.Time
	newID        func() string
}

func NewStore() *Store {
	return &Store{transactions: make(map[string]Transaction), byReference: make(map[string]string), now: func() time.Time { return time.Now().UTC() }, newID: newIdentifier}
}

func (s *Store) PostPayment(paymentID string, amount Money, source string, effectiveAt time.Time, idempotencyKey string) (Transaction, error) {
	if paymentID == "" || amount <= 0 || idempotencyKey == "" {
		return Transaction{}, errors.New("payment, positive amount, and idempotency key are required")
	}
	settlementAccount := AccountVoluntarySettlement
	if source == "kredit_collection" {
		settlementAccount = AccountCollectionSettlement
	} else if !isRecognizedPaymentSource(source) {
		return Transaction{}, errors.New("invalid payment source")
	}
	return s.post(Transaction{EventType: "payment_recognized", ReferenceType: "payment", ReferenceID: paymentID, IdempotencyKey: idempotencyKey, EffectiveAt: effectiveAt, Postings: []Posting{{Account: settlementAccount, Debit: amount}, {Account: AccountTradeReceivable, Credit: amount}}})
}

func (s *Store) PostPaymentReversal(paymentID string, amount Money, source string, effectiveAt time.Time, idempotencyKey string) (Transaction, error) {
	if paymentID == "" || amount <= 0 || idempotencyKey == "" {
		return Transaction{}, errors.New("payment, positive amount, and idempotency key are required")
	}
	settlementAccount := AccountVoluntarySettlement
	if source == "kredit_collection" {
		settlementAccount = AccountCollectionSettlement
	} else if !isRecognizedPaymentSource(source) {
		return Transaction{}, errors.New("invalid payment source")
	}
	return s.post(Transaction{EventType: "payment_reversed", ReferenceType: "payment", ReferenceID: paymentID, IdempotencyKey: idempotencyKey, EffectiveAt: effectiveAt, Postings: []Posting{{Account: AccountTradeReceivable, Debit: amount}, {Account: settlementAccount, Credit: amount}}})
}

func isRecognizedPaymentSource(source string) bool {
	switch source {
	case "integrated_voluntary", "supplier_recorded_transfer", "buyer_payment_claim", "cash_recorded", "adjustment":
		return true
	default:
		return false
	}
}

func (s *Store) PostCollectionFee(paymentID string, amount Money, effectiveAt time.Time, idempotencyKey string) (Transaction, error) {
	if paymentID == "" || amount <= 0 || idempotencyKey == "" {
		return Transaction{}, errors.New("payment, positive amount, and idempotency key are required")
	}
	return s.post(Transaction{EventType: "collection_fee_accrued", ReferenceType: "payment", ReferenceID: paymentID, IdempotencyKey: idempotencyKey, EffectiveAt: effectiveAt, Postings: []Posting{{Account: AccountSupplierFeeReceivable, Debit: amount}, {Account: AccountPlatformCollectionRevenue, Credit: amount}}})
}

func (s *Store) PostCollectionFeeReversal(paymentID string, amount Money, effectiveAt time.Time, idempotencyKey string) (Transaction, error) {
	if paymentID == "" || amount <= 0 || idempotencyKey == "" {
		return Transaction{}, errors.New("payment, positive amount, and idempotency key are required")
	}
	return s.post(Transaction{EventType: "collection_fee_reversed", ReferenceType: "payment", ReferenceID: paymentID, IdempotencyKey: idempotencyKey, EffectiveAt: effectiveAt, Postings: []Posting{{Account: AccountPlatformCollectionRevenue, Debit: amount}, {Account: AccountSupplierFeeReceivable, Credit: amount}}})
}

func (s *Store) PostAdjustment(referenceID string, amount Money, adjustmentType string, effectiveAt time.Time, idempotencyKey string) (Transaction, error) {
	if referenceID == "" || amount <= 0 || idempotencyKey == "" {
		return Transaction{}, errors.New("adjustment, positive amount, and idempotency key are required")
	}
	account := AccountReturnsAdjustment
	if adjustmentType == "write_off" {
		account = AccountWriteOff
	} else if adjustmentType != "dispute_adjustment" {
		return Transaction{}, errors.New("invalid adjustment type")
	}
	return s.post(Transaction{EventType: adjustmentType, ReferenceType: "dispute", ReferenceID: referenceID, IdempotencyKey: idempotencyKey, EffectiveAt: effectiveAt, Postings: []Posting{{Account: account, Debit: amount}, {Account: AccountTradeReceivable, Credit: amount}}})
}

func (s *Store) PostFeeWaiver(referenceID string, amount Money, effectiveAt time.Time, idempotencyKey string) (Transaction, error) {
	if referenceID == "" || amount <= 0 || idempotencyKey == "" {
		return Transaction{}, errors.New("fee waiver, positive amount, and idempotency key are required")
	}
	return s.post(Transaction{EventType: "fee_waived", ReferenceType: "obligation", ReferenceID: referenceID, IdempotencyKey: idempotencyKey, EffectiveAt: effectiveAt, Postings: []Posting{{Account: AccountPlatformServiceRevenue, Debit: amount}, {Account: AccountSupplierFeeReceivable, Credit: amount}}})
}

func (s *Store) PostActivation(obligationID string, principal Money, effectiveAt time.Time, idempotencyKey string) (Transaction, error) {
	if obligationID == "" || principal <= 0 || idempotencyKey == "" {
		return Transaction{}, errors.New("obligation, positive principal, and idempotency key are required")
	}
	baseFee, err := BaseFee(principal)
	if err != nil {
		return Transaction{}, err
	}
	return s.post(Transaction{EventType: "principal_activated", ReferenceType: "obligation", ReferenceID: obligationID, IdempotencyKey: idempotencyKey, EffectiveAt: effectiveAt, Postings: []Posting{
		{Account: AccountTradeReceivable, Debit: principal},
		{Account: AccountPrincipalOriginated, Credit: principal},
		{Account: AccountSupplierFeeReceivable, Debit: baseFee},
		{Account: AccountPlatformServiceRevenue, Credit: baseFee},
	}})
}

func (s *Store) post(transaction Transaction) (Transaction, error) {
	if err := validateTransaction(transaction); err != nil {
		return Transaction{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if existingID := s.byReference[transaction.IdempotencyKey]; existingID != "" {
		existing := s.transactions[existingID]
		if !sameTransactionIntent(existing, transaction) {
			return Transaction{}, errors.New("idempotency key was reused for a different ledger transaction")
		}
		return cloneTransaction(existing), nil
	}
	if transaction.EffectiveAt.IsZero() {
		transaction.EffectiveAt = s.now()
	}
	transaction.RecordedAt = s.now()
	transaction.ID = s.newID()
	transaction.Postings = append([]Posting(nil), transaction.Postings...)
	s.transactions[transaction.ID] = transaction
	s.byReference[transaction.IdempotencyKey] = transaction.ID
	return cloneTransaction(transaction), nil
}

func sameTransactionIntent(existing, requested Transaction) bool {
	if existing.EventType != requested.EventType || existing.ReferenceType != requested.ReferenceType || existing.ReferenceID != requested.ReferenceID || len(existing.Postings) != len(requested.Postings) {
		return false
	}
	for index := range existing.Postings {
		if existing.Postings[index] != requested.Postings[index] {
			return false
		}
	}
	return true
}

func (s *Store) GetByReference(referenceID string) ([]Transaction, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]Transaction, 0)
	for _, transaction := range s.transactions {
		if transaction.ReferenceID == referenceID {
			result = append(result, cloneTransaction(transaction))
		}
	}
	return result, nil
}

func BaseFee(principal Money) (Money, error) {
	if principal < 0 {
		return 0, errors.New("principal cannot be negative")
	}
	// 0.5% = 5 basis points of one thousand (integer kobo arithmetic).
	if principal > (Money(^uint64(0)>>1))/5 {
		return 0, errors.New("principal is too large")
	}
	return (principal * 5) / 1000, nil
}

// CheckedAdd performs money arithmetic without allowing signed integer
// overflow to turn a large positive balance negative (or vice versa).
func CheckedAdd(left, right Money) (Money, error) {
	if (right > 0 && left > Money(math.MaxInt64)-right) || (right < 0 && left < Money(math.MinInt64)-right) {
		return 0, errors.New("money amount is too large")
	}
	return left + right, nil
}

func validateTransaction(transaction Transaction) error {
	if transaction.EventType == "" || transaction.ReferenceID == "" || transaction.IdempotencyKey == "" || len(transaction.Postings) == 0 {
		return errors.New("transaction metadata and postings are required")
	}
	var debits, credits Money
	for _, posting := range transaction.Postings {
		if posting.Account == "" || posting.Debit < 0 || posting.Credit < 0 || (posting.Debit > 0 && posting.Credit > 0) || (posting.Debit == 0 && posting.Credit == 0) {
			return errors.New("each posting must have exactly one positive side")
		}
		var err error
		debits, err = CheckedAdd(debits, posting.Debit)
		if err != nil {
			return errors.New("ledger debit total is too large")
		}
		credits, err = CheckedAdd(credits, posting.Credit)
		if err != nil {
			return errors.New("ledger credit total is too large")
		}
	}
	if debits != credits {
		return fmt.Errorf("unbalanced transaction: debits=%d credits=%d", debits, credits)
	}
	return nil
}

func cloneTransaction(transaction Transaction) Transaction {
	transaction.Postings = append([]Posting(nil), transaction.Postings...)
	return transaction
}

func newIdentifier() string {
	var value [16]byte
	if _, err := rand.Read(value[:]); err != nil {
		return fmt.Sprintf("ledger-fallback-%d", time.Now().UnixNano())
	}
	return "ledger-" + hex.EncodeToString(value[:])
}
