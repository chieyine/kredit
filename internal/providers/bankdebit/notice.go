package bankdebit

import (
	"context"
	"kredit/internal/collections"
	"kredit/internal/settlement"
	"net/http"
)

type Notice struct {
	EventID          string `json:"event_id"`
	Type             string `json:"type"`
	PaymentReference string `json:"payment_reference,omitempty"`
	MandateReference string `json:"mandate_reference,omitempty"`
}
type NativeClient interface {
	Enroller
	Banks(context.Context) ([]settlement.Bank, error)
	collections.Provider
	ParseNotice(http.Header, []byte) (Notice, error)
	LocalReference(context.Context, string) (string, error)
}
