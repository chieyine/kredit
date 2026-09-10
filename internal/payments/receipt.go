package payments

import (
	"context"
	"kredit/internal/ledger"
	"time"
)

// Receipt contains only the fields authorized by a signed public receipt link.
// Buyer identities, organization IDs and provider references stay private.
type Receipt struct {
	Reference    string       `json:"reference"`
	AmountKobo   ledger.Money `json:"amount_kobo"`
	Currency     string       `json:"currency"`
	SourceType   string       `json:"source_type"`
	State        string       `json:"state"`
	PaidAt       time.Time    `json:"paid_at"`
	RecognizedAt time.Time    `json:"recognized_at"`
}

// PublicReceiptContext must only be called after verifying a receipt token.
// Ordinary GetContext remains tenant-scoped, even for this payment ID.
func (s *PostgresStore) PublicReceiptContext(ctx context.Context, id string) (Receipt, error) {
	var receipt Receipt
	err := s.pool.QueryRow(ctx, `SELECT reference::text,amount_kobo,currency,source_type,state,paid_at,recognized_at FROM app.public_payment_receipt($1::uuid)`, id).Scan(
		&receipt.Reference, &receipt.AmountKobo, &receipt.Currency, &receipt.SourceType, &receipt.State, &receipt.PaidAt, &receipt.RecognizedAt)
	return receipt, err
}

func (s *Store) PublicReceiptContext(ctx context.Context, id string) (Receipt, error) {
	payment, err := s.GetContext(ctx, id)
	if err != nil {
		return Receipt{}, err
	}
	return Receipt{Reference: payment.ID, AmountKobo: payment.AmountKobo, Currency: payment.Currency, SourceType: payment.SourceType, State: payment.State, PaidAt: payment.PaidAt, RecognizedAt: payment.RecognizedAt}, nil
}
