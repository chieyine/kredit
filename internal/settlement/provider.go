// Package settlement keeps destination registration separate from evidence that
// a seller has actually received a payment.
package settlement

import (
	"context"
	"errors"
	"regexp"
)

var ErrUnknown = errors.New("bank registration is unconfirmed; review the original request before retrying")
var digits = regexp.MustCompile(`^[0-9]{10}$`)
var bankCode = regexp.MustCompile(`^[0-9]{3,6}$`)

type Input struct {
	Reference      string `json:"reference"`
	OrganizationID string `json:"organization_id"`
	BankCode       string `json:"bank_code"`
	AccountNumber  string `json:"account_number"`
}
type Destination struct {
	ProviderReference string `json:"provider_reference"`
	BankCode          string `json:"bank_code"`
	AccountName       string `json:"account_name"`
	AccountLast4      string `json:"account_last4"`
}
type Provider interface {
	Name() string
	Banks(context.Context) ([]Bank, error)
	// ConnectionIdentity distinguishes different accounts at the same provider.
	// It is a keyed fingerprint, never an API credential.
	ConnectionIdentity() string
	CreateDestination(context.Context, Input) (Destination, error)
}
type Bank struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

func ValidateBanks(banks []Bank) error {
	if len(banks) == 0 || len(banks) > 2000 {
		return errors.New("bank list is unavailable")
	}
	seen := map[string]bool{}
	for _, bank := range banks {
		if !bankCode.MatchString(bank.Code) || bank.Name == "" || len(bank.Name) > 200 || seen[bank.Code] {
			return errors.New("invalid bank list")
		}
		seen[bank.Code] = true
	}
	return nil
}
func ValidateInput(in Input) error {
	if in.Reference == "" || in.OrganizationID == "" || !bankCode.MatchString(in.BankCode) || !digits.MatchString(in.AccountNumber) {
		return errors.New("enter a valid bank code and ten-digit account number")
	}
	return nil
}
func ValidateDestination(in Input, d Destination) error {
	if ValidateInput(in) != nil {
		return ErrUnknown
	}
	if d.ProviderReference == "" || len(d.ProviderReference) > 128 || d.BankCode != in.BankCode || d.AccountLast4 != in.AccountNumber[6:] || d.AccountName == "" || len(d.AccountName) > 200 {
		return ErrUnknown
	}
	return nil
}
