package ledger

import (
	"errors"
	"fmt"
	"time"
)

// FeeTerms is copied into an offer before its agreement hash is produced.
// nil represents the historical 50/50 basis-point contract, never today's policy.
type FeeTerms struct {
	PolicyRevision int64 `json:"policy_revision"`
	BaseBPS        int64 `json:"base_bps"`
	CollectionBPS  int64 `json:"collection_bps"`
	MinFeeKobo     Money `json:"min_fee_kobo,omitempty"`
}

func (f *FeeTerms) Rates() (int64, int64) {
	if f == nil {
		return 50, 50
	}
	return f.BaseBPS, f.CollectionBPS
}
func (f *FeeTerms) Clone() *FeeTerms {
	if f == nil {
		return nil
	}
	c := *f
	return &c
}
func (f *FeeTerms) Validate() error {
	a, b := f.Rates()
	if a < 0 || a > 1000 || b < 0 || b > 1000 {
		return errors.New("fee rates must be between 0 and 1000 basis points")
	}
	if f != nil && f.MinFeeKobo < 0 {
		return errors.New("minimum fee cannot be negative")
	}
	return nil
}
func (f *FeeTerms) Base(principal Money) (Money, error) {
	a, _ := f.Rates()
	fee, err := FeeAtRate(principal, a)
	if err != nil {
		return 0, err
	}
	if f != nil && f.MinFeeKobo > 0 && principal > 0 {
		if fee < f.MinFeeKobo {
			if principal < f.MinFeeKobo {
				return principal, nil
			}
			return f.MinFeeKobo, nil
		}
	}
	return fee, nil
}
func (f *FeeTerms) Collection(amount Money) (Money, error) {
	_, b := f.Rates()
	return FeeAtRate(amount, b)
}
func (f *FeeTerms) Disclosure() string {
	a, b := f.Rates()
	return fmt.Sprintf("%d.%02d%% supplier base service fee on activated principal; an additional %d.%02d%% only on amounts Kredit successfully collects at or after the permitted collection time", a/100, a%100, b/100, b%100)
}
func FeeAtRate(amount Money, bps int64) (Money, error) {
	if amount < 0 || bps < 0 || bps > 1000 {
		return 0, errors.New("invalid amount or fee rate")
	}
	// Split first so every valid int64 principal is safe without multiplication overflow.
	return (amount/10000)*Money(bps) + (amount%10000)*Money(bps)/10000, nil
}
func ActivateWithFee(s Service, id string, principal, fee Money, at time.Time, key string) (Transaction, error) {
	if v, ok := s.(interface {
		PostActivationWithFee(string, Money, Money, time.Time, string) (Transaction, error)
	}); ok {
		return v.PostActivationWithFee(id, principal, fee, at, key)
	}
	standard, err := BaseFee(principal)
	if err != nil || standard != fee {
		return Transaction{}, errors.New("ledger does not support the recorded fee terms")
	}
	return s.PostActivation(id, principal, at, key)
}
func activationTransaction(id string, principal, fee Money, at time.Time, key string) (Transaction, error) {
	if id == "" || principal <= 0 || fee < 0 || fee > principal || key == "" {
		return Transaction{}, errors.New("invalid activation amounts")
	}
	postings := []Posting{{Account: AccountTradeReceivable, Debit: principal}, {Account: AccountPrincipalOriginated, Credit: principal}}
	if fee > 0 {
		postings = append(postings, Posting{Account: AccountSupplierFeeReceivable, Debit: fee}, Posting{Account: AccountPlatformServiceRevenue, Credit: fee})
	}
	return Transaction{EventType: "principal_activated", ReferenceType: "obligation", ReferenceID: id, IdempotencyKey: key, EffectiveAt: at, Postings: postings}, nil
}
func (s *Store) PostActivationWithFee(id string, principal, fee Money, at time.Time, key string) (Transaction, error) {
	t, err := activationTransaction(id, principal, fee, at, key)
	if err != nil {
		return t, err
	}
	return s.post(t)
}
func (s *PostgresStore) PostActivationWithFee(id string, principal, fee Money, at time.Time, key string) (Transaction, error) {
	t, err := activationTransaction(id, principal, fee, at, key)
	if err != nil {
		return t, err
	}
	return s.post(t)
}
