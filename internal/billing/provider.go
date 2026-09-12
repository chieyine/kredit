package billing

import (
	"context"
	"time"
)

// FeeProvider is independent of trade repayments. A seller explicitly gives a
// separate authorization; a buyer mandate can never pay a seller's platform bill.
type FeeProvider interface {
	Name() string
	CreateFeeCustomer(context.Context, FeeCustomer) (string, error)
	FeeCustomerIdentity(context.Context, string) (string, error)
	CreateFeeAuthorization(context.Context, FeeAuthorization) (FeeAuthorization, error)
	ReadFeeAuthorization(context.Context, string) (FeeAuthorization, error)
	SubmitFeeDebit(context.Context, FeeDebitRequest) (FeeDebitResult, error)
	ReadFeeDebit(context.Context, FeeDebitRequest) (FeeDebitResult, error)
}
type FeeCustomer struct {
	BusinessName string `json:"business_name"`
	Email        string `json:"email"`
	Phone        string `json:"phone"`
	Address      string `json:"address"`
	BVN          string `json:"bvn"`
}
type FeeAuthorization struct {
	ID        string    `json:"id"`
	Reference string    `json:"reference"`
	Customer  string    `json:"customer"`
	Ceiling   int64     `json:"ceiling_kobo"`
	StartsAt  time.Time `json:"starts_at"`
	EndsAt    time.Time `json:"ends_at"`
	URL       string    `json:"authorization_url"`
	Ready     bool      `json:"ready"`
}
type FeeDebitRequest struct {
	Mandate   string `json:"mandate"`
	Reference string `json:"reference"`
	Amount    int64  `json:"amount_kobo"`
}
type FeeDebitResult struct {
	State  string `json:"state"`
	Amount int64  `json:"amount_kobo"`
}
